package login

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"html/template"
	"io"
	"net"
	"net/http"
	"net/url"
	"os/exec"
	"runtime"
	"sync"
	"time"

	"github.com/openstatusHQ/cli/internal/api"
	"github.com/openstatusHQ/cli/internal/auth"
	output "github.com/openstatusHQ/cli/internal/cli"
	"github.com/openstatusHQ/cli/internal/whoami"
)

const (
	oauthClientName = "OpenStatus CLI"
	callbackPath    = "/callback"
	// The server ignores the port when matching loopback redirects (RFC 8252
	// §7.3), so one registered URI covers whichever port is free.
	registeredRedirectURI = "http://127.0.0.1" + callbackPath

	// Long enough to sign in through an emailed magic link.
	browserLoginTimeout = 10 * time.Minute
)

// browserFlow holds what a browser login needs from its surroundings.
type browserFlow struct {
	httpClient *http.Client
	stderr     io.Writer
	// open launches the browser. Nil means print the URL only.
	open    func(url string) error
	timeout time.Duration
}

// run sends the user through the dashboard and saves the resulting login. The
// API's /oauth/authorize redirects to the dashboard consent page, which goes
// through /login first when there is no session.
func (f *browserFlow) run(ctx context.Context) (whoami.Whoami, error) {
	ctx, cancel := context.WithTimeout(ctx, f.timeout)
	defer cancel()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return whoami.Whoami{}, fmt.Errorf("could not start the local login server: %w", err)
	}
	defer listener.Close()
	redirectURI := fmt.Sprintf("http://%s%s", listener.Addr(), callbackPath)

	clientID, err := auth.RegisterOAuthClient(ctx, f.httpClient, api.BaseURL, oauthClientName, registeredRedirectURI)
	if errors.Is(err, auth.ErrOAuthUnsupported) {
		return whoami.Whoami{}, output.UsageErrorWithHint(
			fmt.Sprintf("%s does not support browser login", api.BaseURL),
			"Use an API token instead: openstatus login --with-token",
		)
	}
	if err != nil {
		return whoami.Whoami{}, fmt.Errorf("could not start login: %w", output.FormatError(err, "", ""))
	}

	state, err := randomToken()
	if err != nil {
		return whoami.Whoami{}, err
	}
	verifier, err := randomToken()
	if err != nil {
		return whoami.Whoami{}, err
	}
	authorizeURL := buildAuthorizeURL(api.BaseURL, clientID, redirectURI, state, verifier)

	type outcome struct {
		workspace whoami.Whoami
		err       error
	}
	results := make(chan outcome, 1)
	var once sync.Once

	mux := http.NewServeMux()
	mux.HandleFunc("GET "+callbackPath, func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query()
		// Any local page or process can reach this port, so an unexpected
		// request is dropped rather than allowed to end the login.
		if subtle.ConstantTimeCompare([]byte(query.Get("state")), []byte(state)) != 1 {
			http.Error(w, "This login request doesn't match the one in your terminal.", http.StatusBadRequest)
			return
		}
		handled := false
		once.Do(func() {
			handled = true
			workspace, err := f.complete(ctx, query, clientID, verifier, redirectURI)
			writeResultPage(w, workspace, err)
			results <- outcome{workspace, err}
		})
		if !handled {
			writeResultPage(w, whoami.Whoami{}, errors.New("already handled"))
		}
	})

	server := &http.Server{Handler: mux, ReadHeaderTimeout: 10 * time.Second}
	go func() { _ = server.Serve(listener) }()
	defer func() {
		// Let the result page finish sending before exiting.
		shutdownCtx, stop := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
		defer stop()
		_ = server.Shutdown(shutdownCtx)
	}()

	f.promptUser(authorizeURL)

	select {
	case result := <-results:
		return result.workspace, result.err
	case <-ctx.Done():
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return whoami.Whoami{}, output.NewError(output.KindTimeout, "timed out waiting for login in the browser. Run 'openstatus login' to try again")
		}
		return whoami.Whoami{}, output.NewError(output.KindCancelled, "login cancelled")
	}
}

func (f *browserFlow) promptUser(authorizeURL string) {
	if f.open == nil {
		fmt.Fprintf(f.stderr, "Open this URL in a browser on this machine to log in:\n\n  %s\n\n", authorizeURL)
	} else {
		fmt.Fprintln(f.stderr, "Opening your browser to log in to OpenStatus...")
		if err := f.open(authorizeURL); err != nil {
			fmt.Fprintln(f.stderr, "Could not open a browser.")
		}
		fmt.Fprintf(f.stderr, "If it didn't open, visit:\n\n  %s\n\n", authorizeURL)
	}
	fmt.Fprintln(f.stderr, "Waiting for you to approve the login. Without a browser on this machine, use 'openstatus login --with-token'.")
}

// complete turns an authenticated callback into a saved login.
func (f *browserFlow) complete(ctx context.Context, query url.Values, clientID, verifier, redirectURI string) (whoami.Whoami, error) {
	if code := query.Get("error"); code != "" {
		if code == "access_denied" {
			return whoami.Whoami{}, output.NewError(output.KindUnauthenticated, "login was denied in the browser")
		}
		oauthErr := &auth.OAuthError{Code: code, Description: query.Get("error_description")}
		return whoami.Whoami{}, output.NewError(output.KindUnauthenticated, "login failed: %w", oauthErr)
	}
	code := query.Get("code")
	if code == "" {
		return whoami.Whoami{}, errors.New("login failed: the browser returned no authorization code")
	}

	creds, err := auth.ExchangeOAuthCode(ctx, f.httpClient, api.BaseURL, clientID, code, verifier, redirectURI)
	if err != nil {
		return whoami.Whoami{}, fmt.Errorf("could not complete login: %w", output.FormatError(err, "", ""))
	}

	workspace, err := whoami.Fetch(ctx, f.httpClient, creds.AccessToken)
	if err == nil {
		err = auth.SaveOAuth(creds)
	}
	if err != nil {
		// Nothing holds these tokens now, so don't leave the grant live.
		_ = auth.RevokeOAuth(ctx, f.httpClient, creds)
		return whoami.Whoami{}, fmt.Errorf("could not complete login: %w", output.FormatError(err, "", ""))
	}
	return workspace, nil
}

func buildAuthorizeURL(issuer, clientID, redirectURI, state, verifier string) string {
	challenge := sha256.Sum256([]byte(verifier))
	return issuer + "/oauth/authorize?" + url.Values{
		"response_type":         {"code"},
		"client_id":             {clientID},
		"redirect_uri":          {redirectURI},
		"scope":                 {"write"},
		"state":                 {state},
		"code_challenge":        {base64.RawURLEncoding.EncodeToString(challenge[:])},
		"code_challenge_method": {"S256"},
	}.Encode()
}

// randomToken returns 256 random bits, base64url-encoded: 43 characters, which
// is the shortest PKCE verifier RFC 7636 allows.
func randomToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("could not generate a login token: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

var resultPage = template.Must(template.New("result").Parse(`<!doctype html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>OpenStatus CLI</title>
<style>
  :root { color-scheme: light dark; }
  body { font: 16px/1.5 system-ui, sans-serif; display: grid; place-items: center; min-height: 100vh; margin: 0; padding: 16px; box-sizing: border-box; text-align: center; }
  h1 { font-size: 1.5rem; margin: 0 0 8px; }
  p { margin: 0; opacity: .7; }
</style>
</head>
<body>
<main>
  <h1>{{.Title}}</h1>
  <p>{{.Message}}</p>
</main>
</body>
</html>
`))

func writeResultPage(w http.ResponseWriter, workspace whoami.Whoami, err error) {
	data := struct{ Title, Message string }{
		Title:   "You're logged in",
		Message: fmt.Sprintf("Connected to workspace %s. You can close this tab and return to your terminal.", workspace.Slug),
	}
	status := http.StatusOK
	if err != nil {
		// Details stay in the terminal; this page only says where to look.
		data.Title = "Login didn't complete"
		data.Message = "Return to your terminal for details."
		status = http.StatusBadRequest
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Referrer-Policy", "no-referrer")
	w.Header().Set("Content-Security-Policy", "default-src 'none'; style-src 'unsafe-inline'")
	w.WriteHeader(status)
	_ = resultPage.Execute(w, data)
}

// openBrowser launches the default browser. It starts the opener without
// waiting, since some keep running for as long as the browser does.
func openBrowser(target string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", target)
	case "windows":
		// Not `cmd /c start`, which would interpret the & in the URL.
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", target)
	default:
		cmd = exec.Command("xdg-open", target)
	}
	if err := cmd.Start(); err != nil {
		return err
	}
	go func() { _ = cmd.Wait() }()
	return nil
}
