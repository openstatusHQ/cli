package login

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/openstatusHQ/cli/internal/api"
	"github.com/openstatusHQ/cli/internal/auth"
	output "github.com/openstatusHQ/cli/internal/cli"
)

// fakeAPI implements the parts of the OAuth server and /v1 the flow touches,
// enforcing PKCE and the redirect checks the real one does.
type fakeAPI struct {
	*httptest.Server
	t             *testing.T
	challenge     string
	redirectURI   string
	whoamiStatus  int
	registerCalls atomic.Int64
	revoked       atomic.Value
}

func newFakeAPI(t *testing.T) *fakeAPI {
	t.Helper()
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	f := &fakeAPI{t: t, whoamiStatus: http.StatusOK}

	mux := http.NewServeMux()
	mux.HandleFunc("POST /oauth/register", func(w http.ResponseWriter, r *http.Request) {
		f.registerCalls.Add(1)
		var body struct {
			RedirectURIs []string `json:"redirect_uris"`
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		if len(body.RedirectURIs) != 1 || body.RedirectURIs[0] != registeredRedirectURI {
			t.Errorf("redirect_uris = %v", body.RedirectURIs)
		}
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"client_id":"client-1"}`))
	})
	mux.HandleFunc("POST /oauth/token", func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		sum := sha256.Sum256([]byte(r.PostForm.Get("code_verifier")))
		if base64.RawURLEncoding.EncodeToString(sum[:]) != f.challenge {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(`{"error":"invalid_grant","error_description":"PKCE verification failed"}`))
			return
		}
		if r.PostForm.Get("code") != "the-code" || r.PostForm.Get("redirect_uri") != f.redirectURI || r.PostForm.Get("client_id") != "client-1" {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(`{"error":"invalid_grant"}`))
			return
		}
		_, _ = w.Write([]byte(`{"access_token":"os_oat_a","refresh_token":"refresh_a","expires_in":3600,"token_type":"bearer","scope":"write"}`))
	})
	mux.HandleFunc("POST /oauth/revoke", func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		f.revoked.Store(r.PostForm.Get("token"))
	})
	mux.HandleFunc("GET /v1/whoami", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("x-openstatus-key") != "os_oat_a" {
			t.Errorf("whoami key = %q", r.Header.Get("x-openstatus-key"))
		}
		w.WriteHeader(f.whoamiStatus)
		_, _ = w.Write([]byte(`{"name":"Acme <Inc>","slug":"acme","plan":"team"}`))
	})
	f.Server = httptest.NewServer(mux)
	t.Cleanup(f.Close)

	prevBase, prevV1 := api.BaseURL, api.APIBaseURL
	api.BaseURL, api.APIBaseURL = f.URL, f.URL+"/v1"
	t.Cleanup(func() { api.BaseURL, api.APIBaseURL = prevBase, prevV1 })
	return f
}

// resultSeen is the page the browser ended up on.
type resultSeen struct {
	status int
	body   string
}

// browser stands in for the user approving in the dashboard: it reads the
// authorize URL and calls back with whatever query respond builds.
func (f *fakeAPI) browser(respond func(authorize url.Values) url.Values, seen *atomic.Pointer[resultSeen]) func(string) error {
	return func(target string) error {
		u, err := url.Parse(target)
		if err != nil {
			return err
		}
		q := u.Query()
		f.challenge = q.Get("code_challenge")
		f.redirectURI = q.Get("redirect_uri")
		go func() {
			res, err := http.Get(q.Get("redirect_uri") + "?" + respond(q).Encode())
			if err != nil {
				f.t.Errorf("callback: %v", err)
				return
			}
			body, _ := io.ReadAll(res.Body)
			res.Body.Close()
			if seen != nil {
				seen.Store(&resultSeen{status: res.StatusCode, body: string(body)})
			}
		}()
		return nil
	}
}

func approve(authorize url.Values) url.Values {
	return url.Values{"code": {"the-code"}, "state": {authorize.Get("state")}}
}

func newFlow(open func(string) error) *browserFlow {
	return &browserFlow{httpClient: http.DefaultClient, stderr: io.Discard, open: open, timeout: 5 * time.Second}
}

func TestBrowserLogin_SavesLoginAndShowsResult(t *testing.T) {
	f := newFakeAPI(t)
	var page atomic.Pointer[resultSeen]

	workspace, err := newFlow(f.browser(approve, &page)).run(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if workspace.Slug != "acme" {
		t.Errorf("workspace = %+v", workspace)
	}

	creds, err := auth.LoadOAuth()
	if err != nil || creds == nil {
		t.Fatalf("creds = %+v, %v", creds, err)
	}
	if creds.Issuer != f.URL || creds.ClientID != "client-1" || creds.AccessToken != "os_oat_a" || creds.RefreshToken != "refresh_a" {
		t.Errorf("creds = %+v", creds)
	}
	if time.Until(creds.ExpiresAt) < 50*time.Minute {
		t.Errorf("ExpiresAt = %v", creds.ExpiresAt)
	}

	// Server shutdown waits for the handler, so the page has been sent.
	deadline := time.Now().Add(2 * time.Second)
	for page.Load() == nil && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	result := page.Load()
	if result == nil {
		t.Fatal("browser got no result page")
	}
	if result.status != http.StatusOK || !strings.Contains(result.body, "You&#39;re logged in") || !strings.Contains(result.body, "acme") {
		t.Errorf("page %d: %s", result.status, result.body)
	}
}

func TestBrowserLogin_AuthorizeURL(t *testing.T) {
	f := newFakeAPI(t)
	var q url.Values
	flow := newFlow(f.browser(func(authorize url.Values) url.Values {
		q = authorize
		return approve(authorize)
	}, nil))
	if _, err := flow.run(context.Background()); err != nil {
		t.Fatal(err)
	}

	checks := map[string]string{
		"response_type":         "code",
		"client_id":             "client-1",
		"code_challenge_method": "S256",
		"scope":                 "write",
	}
	for k, want := range checks {
		if got := q.Get(k); got != want {
			t.Errorf("%s = %q, want %q", k, got, want)
		}
	}
	if !strings.HasPrefix(q.Get("redirect_uri"), "http://127.0.0.1:") || !strings.HasSuffix(q.Get("redirect_uri"), callbackPath) {
		t.Errorf("redirect_uri = %q", q.Get("redirect_uri"))
	}
	if len(q.Get("state")) != 43 {
		t.Errorf("state = %q", q.Get("state"))
	}
}

// A hostile page can hit the local port, but must not be able to end the login
// or plant a code.
func TestBrowserLogin_IgnoresCallbackWithWrongState(t *testing.T) {
	f := newFakeAPI(t)
	open := func(target string) error {
		u, _ := url.Parse(target)
		q := u.Query()
		f.challenge, f.redirectURI = q.Get("code_challenge"), q.Get("redirect_uri")
		go func() {
			res, err := http.Get(q.Get("redirect_uri") + "?code=evil&state=wrong")
			if err != nil {
				t.Errorf("callback: %v", err)
				return
			}
			res.Body.Close()
			if res.StatusCode != http.StatusBadRequest {
				t.Errorf("wrong state status = %d", res.StatusCode)
			}
			res, err = http.Get(q.Get("redirect_uri") + "?" + approve(q).Encode())
			if err != nil {
				t.Errorf("callback: %v", err)
				return
			}
			res.Body.Close()
		}()
		return nil
	}
	if _, err := newFlow(open).run(context.Background()); err != nil {
		t.Fatal(err)
	}
}

func TestBrowserLogin_Denied(t *testing.T) {
	f := newFakeAPI(t)
	deny := func(authorize url.Values) url.Values {
		return url.Values{"error": {"access_denied"}, "state": {authorize.Get("state")}}
	}
	_, err := newFlow(f.browser(deny, nil)).run(context.Background())
	var cliErr *output.Error
	if !errors.As(err, &cliErr) || cliErr.Kind != output.KindUnauthenticated || !strings.Contains(err.Error(), "denied") {
		t.Fatalf("err = %v", err)
	}
	if creds, _ := auth.LoadOAuth(); creds != nil {
		t.Errorf("saved %+v", creds)
	}
}

func TestBrowserLogin_RevokesTokensThatFailVerification(t *testing.T) {
	f := newFakeAPI(t)
	f.whoamiStatus = http.StatusForbidden
	_, err := newFlow(f.browser(approve, nil)).run(context.Background())
	if err == nil {
		t.Fatal("want error")
	}
	if f.revoked.Load() != "refresh_a" {
		t.Errorf("revoked = %v", f.revoked.Load())
	}
	if creds, _ := auth.LoadOAuth(); creds != nil {
		t.Errorf("saved %+v", creds)
	}
}

func TestBrowserLogin_TimesOut(t *testing.T) {
	newFakeAPI(t)
	flow := newFlow(func(string) error { return nil })
	flow.timeout = 50 * time.Millisecond
	_, err := flow.run(context.Background())
	var cliErr *output.Error
	if !errors.As(err, &cliErr) || cliErr.Kind != output.KindTimeout {
		t.Fatalf("err = %v", err)
	}
}

func TestBrowserLogin_CancelledByInterrupt(t *testing.T) {
	newFakeAPI(t)
	ctx, cancel := context.WithCancel(context.Background())
	flow := newFlow(func(string) error { cancel(); return nil })
	_, err := flow.run(ctx)
	var cliErr *output.Error
	if !errors.As(err, &cliErr) || cliErr.Kind != output.KindCancelled {
		t.Fatalf("err = %v", err)
	}
}

func TestBrowserLogin_ServerWithoutOAuthSuggestsToken(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	srv := httptest.NewServer(http.NotFoundHandler())
	defer srv.Close()
	prev := api.BaseURL
	api.BaseURL = srv.URL
	defer func() { api.BaseURL = prev }()

	_, err := newFlow(func(string) error { t.Error("opened a browser"); return nil }).run(context.Background())
	var cliErr *output.Error
	if !errors.As(err, &cliErr) || !strings.Contains(cliErr.Hint, "--with-token") {
		t.Fatalf("err = %v", err)
	}
}

func TestBrowserLogin_NoBrowserPrintsURL(t *testing.T) {
	newFakeAPI(t)
	var stderr strings.Builder
	flow := &browserFlow{httpClient: http.DefaultClient, stderr: &stderr, timeout: 50 * time.Millisecond}
	_, _ = flow.run(context.Background())
	if !strings.Contains(stderr.String(), "/oauth/authorize?") {
		t.Errorf("stderr = %q", stderr.String())
	}
}

func TestBrowserLogin_OpenFailureStillPrintsURL(t *testing.T) {
	newFakeAPI(t)
	var stderr strings.Builder
	flow := &browserFlow{
		httpClient: http.DefaultClient,
		stderr:     &stderr,
		timeout:    50 * time.Millisecond,
		open:       func(string) error { return fmt.Errorf("no browser") },
	}
	_, _ = flow.run(context.Background())
	if !strings.Contains(stderr.String(), "/oauth/authorize?") {
		t.Errorf("stderr = %q", stderr.String())
	}
}
