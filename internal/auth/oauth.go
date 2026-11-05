package auth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/gofrs/flock"

	"github.com/openstatusHQ/cli/internal/api"
	output "github.com/openstatusHQ/cli/internal/cli"
	"github.com/openstatusHQ/cli/internal/config"
)

const (
	// refreshMargin renews a token this long before it expires, so it cannot
	// lapse between being resolved and reaching the API.
	refreshMargin = time.Minute

	// lockWait is how long a process waits for another one's refresh. It exceeds
	// the HTTP client timeout, the longest a holder should need.
	lockWait = time.Minute

	maxOAuthResponseBytes = 1 << 20
)

// OAuthCredentials is the result of a browser login.
type OAuthCredentials struct {
	// Issuer is the API origin that minted the tokens. They are only ever sent
	// back there.
	Issuer       string    `json:"issuer"`
	ClientID     string    `json:"client_id"`
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	ExpiresAt    time.Time `json:"expires_at"`
}

func (c *OAuthCredentials) needsRefresh(now time.Time) bool {
	return !now.Add(refreshMargin).Before(c.ExpiresAt)
}

// OAuthError is an RFC 6749 error response.
type OAuthError struct {
	Code        string
	Description string
}

func (e *OAuthError) Error() string {
	if e.Description != "" {
		return fmt.Sprintf("%s: %s", e.Code, e.Description)
	}
	return e.Code
}

// ErrOAuthUnsupported means the API has no OAuth endpoints, as on a self-hosted
// instance older than the OAuth server.
var ErrOAuthUnsupported = errors.New("this API does not support browser login")

// LoadOAuth returns the saved browser login, or nil if there is none.
func LoadOAuth() (*OAuthCredentials, error) {
	path, err := config.OAuthPath()
	if err != nil {
		return nil, fmt.Errorf("failed to determine credentials path: %w", err)
	}
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to read credentials file %s: %w", path, err)
	}
	var creds OAuthCredentials
	if err := json.Unmarshal(data, &creds); err != nil {
		return nil, output.NewError(output.KindUnauthenticated, "credentials file %s is corrupt. Run 'openstatus login' to log in again", path)
	}
	return &creds, nil
}

// SaveOAuth stores a browser login, replacing any saved API key.
func SaveOAuth(creds *OAuthCredentials) error {
	if err := writeOAuth(creds); err != nil {
		return err
	}
	tokenPath, err := config.TokenPath()
	if err != nil {
		return fmt.Errorf("failed to determine token path: %w", err)
	}
	if err := os.Remove(tokenPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove previous token: %w", err)
	}
	return nil
}

func writeOAuth(creds *OAuthCredentials) error {
	path, err := config.OAuthPath()
	if err != nil {
		return fmt.Errorf("failed to determine credentials path: %w", err)
	}
	data, err := json.Marshal(creds) //nolint:gosec // Written to a 0600 file, which is the point.
	if err != nil {
		return fmt.Errorf("failed to encode credentials: %w", err)
	}
	return writeSecret(path, data)
}

// resolveOAuthToken returns a usable access token, refreshing it if needed.
func resolveOAuthToken(ctx context.Context, hc *http.Client, creds *OAuthCredentials) (string, error) {
	// Tokens are sent to the configured API, so don't hand them to another one.
	if creds.Issuer != api.BaseURL {
		return "", output.NewError(output.KindUnauthenticated,
			"you are logged in to %s, but the CLI is targeting %s. Run 'openstatus login' to log in there", creds.Issuer, api.BaseURL)
	}
	if !creds.needsRefresh(time.Now()) {
		return creds.AccessToken, nil
	}

	// Refresh tokens rotate, so two processes refreshing at once could each
	// save a different one and leave a dead token on disk. Serialize them.
	path, err := config.OAuthPath()
	if err != nil {
		return "", fmt.Errorf("failed to determine credentials path: %w", err)
	}
	unlock, err := lockFile(ctx, path+".lock")
	if err != nil {
		return tokenWithoutRefresh(ctx, creds, err)
	}
	defer unlock()

	// Another process may have refreshed while this one waited.
	latest, err := LoadOAuth()
	if err != nil {
		return "", err
	}
	if latest == nil {
		return "", output.NewError(output.KindUnauthenticated, "you were logged out. Run 'openstatus login' to log in again")
	}
	if !latest.needsRefresh(time.Now()) {
		return latest.AccessToken, nil
	}

	// Refreshing uses up the saved refresh token, so don't start unless its
	// replacement can be saved. Rewriting the same login is a harmless check.
	if err := writeOAuth(latest); err != nil {
		return tokenWithoutRefresh(ctx, latest, err)
	}

	refreshed, err := refreshOAuth(ctx, hc, latest)
	if err != nil {
		var oauthErr *OAuthError
		if errors.As(err, &oauthErr) && (oauthErr.Code == "invalid_grant" || oauthErr.Code == "invalid_client") {
			return "", output.NewError(output.KindUnauthenticated, "your login has expired or was revoked. Run 'openstatus login' to log in again")
		}
		return tokenWithoutRefresh(ctx, latest, err)
	}
	// The old refresh token is gone either way, so failing here would throw away
	// a working access token as well.
	if err := writeOAuth(refreshed); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: could not save your refreshed login (%v). Run 'openstatus login' if later commands report an expired login\n", err)
	}
	return refreshed.AccessToken, nil
}

// tokenWithoutRefresh handles a refresh that couldn't happen. A failure there
// shouldn't fail commands while the current token still works.
func tokenWithoutRefresh(ctx context.Context, creds *OAuthCredentials, err error) (string, error) {
	if ctx.Err() == nil && time.Now().Before(creds.ExpiresAt) {
		return creds.AccessToken, nil
	}
	return "", fmt.Errorf("could not refresh your login: %w", output.FormatError(err, "", ""))
}

// lockFile takes an OS-level cross-process lock on path. The OS releases it when
// the holder exits, however it exits, so a crash can't leave the lock stuck. The
// file itself stays on disk: deleting it could give two processes separate locks.
func lockFile(ctx context.Context, path string) (unlock func(), err error) {
	waitCtx, cancel := context.WithTimeout(ctx, lockWait)
	defer cancel()

	lock := flock.New(path)
	if _, err := lock.TryLockContext(waitCtx, 50*time.Millisecond); err != nil {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		if errors.Is(err, context.DeadlineExceeded) {
			return nil, errors.New("timed out waiting for another openstatus process to refresh your login")
		}
		return nil, fmt.Errorf("failed to lock %s: %w", path, err)
	}
	return func() { _ = lock.Unlock() }, nil
}

// RegisterOAuthClient registers this CLI login as a public OAuth client
// (RFC 7591) and returns its client ID.
func RegisterOAuthClient(ctx context.Context, hc *http.Client, issuer, name, redirectURI string) (string, error) {
	body, err := json.Marshal(map[string]any{
		"client_name":                name,
		"redirect_uris":              []string{redirectURI},
		"token_endpoint_auth_method": "none",
		"grant_types":                []string{"authorization_code", "refresh_token"},
		"response_types":             []string{"code"},
	})
	if err != nil {
		return "", fmt.Errorf("failed to encode client registration: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, issuer+"/oauth/register", strings.NewReader(string(body)))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	var client struct {
		ClientID string `json:"client_id"`
	}
	if err := doOAuth(hc, req, &client); err != nil {
		return "", err
	}
	if client.ClientID == "" {
		return "", errors.New("the API did not return a client_id")
	}
	return client.ClientID, nil
}

// ExchangeOAuthCode trades an authorization code for tokens.
func ExchangeOAuthCode(ctx context.Context, hc *http.Client, issuer, clientID, code, verifier, redirectURI string) (*OAuthCredentials, error) {
	return requestTokens(ctx, hc, issuer, clientID, url.Values{
		"grant_type":    {"authorization_code"},
		"client_id":     {clientID},
		"code":          {code},
		"code_verifier": {verifier},
		"redirect_uri":  {redirectURI},
	})
}

func refreshOAuth(ctx context.Context, hc *http.Client, creds *OAuthCredentials) (*OAuthCredentials, error) {
	return requestTokens(ctx, hc, creds.Issuer, creds.ClientID, url.Values{
		"grant_type":    {"refresh_token"},
		"client_id":     {creds.ClientID},
		"refresh_token": {creds.RefreshToken},
	})
}

func requestTokens(ctx context.Context, hc *http.Client, issuer, clientID string, form url.Values) (*OAuthCredentials, error) {
	req, err := newOAuthFormRequest(ctx, issuer+"/oauth/token", form)
	if err != nil {
		return nil, err
	}
	var tokens struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		ExpiresIn    int64  `json:"expires_in"`
	}
	issuedAt := time.Now()
	if err := doOAuth(hc, req, &tokens); err != nil {
		return nil, err
	}
	if tokens.AccessToken == "" || tokens.RefreshToken == "" {
		return nil, errors.New("the API returned an incomplete token response")
	}
	return &OAuthCredentials{
		Issuer:       issuer,
		ClientID:     clientID,
		AccessToken:  tokens.AccessToken,
		RefreshToken: tokens.RefreshToken,
		ExpiresAt:    issuedAt.Add(time.Duration(tokens.ExpiresIn) * time.Second),
	}, nil
}

// RevokeOAuth ends the login on the server (RFC 7009).
func RevokeOAuth(ctx context.Context, hc *http.Client, creds *OAuthCredentials) error {
	req, err := newOAuthFormRequest(ctx, creds.Issuer+"/oauth/revoke", url.Values{
		"client_id": {creds.ClientID},
		"token":     {creds.RefreshToken},
	})
	if err != nil {
		return err
	}
	return doOAuth(hc, req, nil)
}

func newOAuthFormRequest(ctx context.Context, endpoint string, form url.Values) (*http.Request, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	return req, nil
}

// doOAuth sends req and decodes a success body into out, if out is non-nil.
func doOAuth(hc *http.Client, req *http.Request, out any) error {
	res, err := hc.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	body, err := io.ReadAll(io.LimitReader(res.Body, maxOAuthResponseBytes))
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}

	if res.StatusCode == http.StatusNotFound {
		return ErrOAuthUnsupported
	}
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		var oauthErr struct {
			Error       string `json:"error"`
			Description string `json:"error_description"`
		}
		if json.Unmarshal(body, &oauthErr) == nil && oauthErr.Error != "" {
			return &OAuthError{Code: oauthErr.Error, Description: oauthErr.Description}
		}
		return fmt.Errorf("unexpected response from the API (HTTP %d)", res.StatusCode)
	}
	if out == nil {
		return nil
	}
	if err := json.Unmarshal(body, out); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}
	return nil
}
