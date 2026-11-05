package auth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/openstatusHQ/cli/internal/api"
	output "github.com/openstatusHQ/cli/internal/cli"
	"github.com/openstatusHQ/cli/internal/config"
)

func setup(t *testing.T, handler http.HandlerFunc) *httptest.Server {
	t.Helper()
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	previous := api.BaseURL
	api.BaseURL = srv.URL
	t.Cleanup(func() { api.BaseURL = previous })
	return srv
}

func saved(issuer string, expiresIn time.Duration) *OAuthCredentials {
	return &OAuthCredentials{
		Issuer:       issuer,
		ClientID:     "client",
		AccessToken:  "os_oat_old",
		RefreshToken: "refresh_old",
		ExpiresAt:    time.Now().Add(expiresIn),
	}
}

func rotatingTokenHandler(calls *atomic.Int64) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		n := calls.Add(1)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"access_token":  fmt.Sprintf("os_oat_%d", n),
			"refresh_token": fmt.Sprintf("refresh_%d", n),
			"expires_in":    3600,
		})
	}
}

func TestResolveToken_FlagWinsOverSavedLogin(t *testing.T) {
	srv := setup(t, nil)
	if err := SaveOAuth(saved(srv.URL, time.Hour)); err != nil {
		t.Fatal(err)
	}
	got, err := ResolveToken(t.Context(), "from-flag")
	if err != nil || got != "from-flag" {
		t.Fatalf("got %q, %v", got, err)
	}
}

func TestResolveToken_FreshTokenMakesNoRequest(t *testing.T) {
	var calls atomic.Int64
	srv := setup(t, rotatingTokenHandler(&calls))
	if err := SaveOAuth(saved(srv.URL, time.Hour)); err != nil {
		t.Fatal(err)
	}
	got, err := ResolveToken(t.Context(), "")
	if err != nil || got != "os_oat_old" {
		t.Fatalf("got %q, %v", got, err)
	}
	if calls.Load() != 0 {
		t.Errorf("made %d requests", calls.Load())
	}
}

func TestResolveToken_RefreshesAndSavesRotatedToken(t *testing.T) {
	var form map[string]string
	srv := setup(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/oauth/token" {
			t.Errorf("path = %s", r.URL.Path)
		}
		_ = r.ParseForm()
		form = map[string]string{
			"grant_type":    r.PostForm.Get("grant_type"),
			"client_id":     r.PostForm.Get("client_id"),
			"refresh_token": r.PostForm.Get("refresh_token"),
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"access_token": "os_oat_new", "refresh_token": "refresh_new", "expires_in": 3600,
		})
	})
	// Within the margin, not yet expired.
	if err := SaveOAuth(saved(srv.URL, 30*time.Second)); err != nil {
		t.Fatal(err)
	}

	got, err := ResolveToken(t.Context(), "")
	if err != nil || got != "os_oat_new" {
		t.Fatalf("got %q, %v", got, err)
	}
	if form["grant_type"] != "refresh_token" || form["client_id"] != "client" || form["refresh_token"] != "refresh_old" {
		t.Errorf("form = %v", form)
	}
	creds, err := LoadOAuth()
	if err != nil {
		t.Fatal(err)
	}
	if creds.RefreshToken != "refresh_new" || time.Until(creds.ExpiresAt) < 50*time.Minute {
		t.Errorf("saved = %+v", creds)
	}
}

// A rotated-out refresh token kills the login, so concurrent commands must
// share one refresh rather than each saving their own.
func TestResolveToken_ConcurrentCommandsRefreshOnce(t *testing.T) {
	var calls atomic.Int64
	srv := setup(t, rotatingTokenHandler(&calls))
	if err := SaveOAuth(saved(srv.URL, -time.Minute)); err != nil {
		t.Fatal(err)
	}

	var wg sync.WaitGroup
	for range 10 {
		wg.Go(func() {
			if got, err := ResolveToken(t.Context(), ""); err != nil || got != "os_oat_1" {
				t.Errorf("got %q, %v", got, err)
			}
		})
	}
	wg.Wait()
	if calls.Load() != 1 {
		t.Errorf("refreshed %d times", calls.Load())
	}
}

func TestResolveToken_RevokedLoginAsksToLogInAgain(t *testing.T) {
	srv := setup(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":"invalid_grant","error_description":"Invalid refresh token"}`))
	})
	if err := SaveOAuth(saved(srv.URL, -time.Minute)); err != nil {
		t.Fatal(err)
	}
	_, err := ResolveToken(t.Context(), "")
	var cliErr *output.Error
	if !errors.As(err, &cliErr) || cliErr.Kind != output.KindUnauthenticated {
		t.Fatalf("err = %v", err)
	}
	if !strings.Contains(err.Error(), "openstatus login") {
		t.Errorf("err = %v", err)
	}
}

func TestResolveToken_ServerErrorKeepsUsableToken(t *testing.T) {
	srv := setup(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})
	if err := SaveOAuth(saved(srv.URL, 30*time.Second)); err != nil {
		t.Fatal(err)
	}
	got, err := ResolveToken(t.Context(), "")
	if err != nil || got != "os_oat_old" {
		t.Fatalf("got %q, %v", got, err)
	}

	if err := SaveOAuth(saved(srv.URL, -time.Minute)); err != nil {
		t.Fatal(err)
	}
	if _, err := ResolveToken(t.Context(), ""); err == nil {
		t.Fatal("expired token with failed refresh: want error")
	}
}

func TestResolveToken_RefusesTokensForAnotherAPI(t *testing.T) {
	var calls atomic.Int64
	setup(t, rotatingTokenHandler(&calls))
	if err := SaveOAuth(saved("https://other.example.com", -time.Minute)); err != nil {
		t.Fatal(err)
	}
	_, err := ResolveToken(t.Context(), "")
	if err == nil || !strings.Contains(err.Error(), "other.example.com") {
		t.Fatalf("err = %v", err)
	}
	if calls.Load() != 0 {
		t.Error("sent a refresh token to a different API")
	}
}

// A crashed process leaves the lock file behind, but not the lock.
func TestResolveToken_LeftoverLockFileDoesNotBlock(t *testing.T) {
	var calls atomic.Int64
	srv := setup(t, rotatingTokenHandler(&calls))
	if err := SaveOAuth(saved(srv.URL, -time.Minute)); err != nil {
		t.Fatal(err)
	}
	path, _ := config.OAuthPath()
	if err := os.WriteFile(path+".lock", nil, 0o600); err != nil {
		t.Fatal(err)
	}

	start := time.Now()
	if got, err := ResolveToken(t.Context(), ""); err != nil || got != "os_oat_1" {
		t.Fatalf("got %q, %v", got, err)
	}
	if elapsed := time.Since(start); elapsed > 5*time.Second {
		t.Errorf("took %v", elapsed)
	}
}

func TestResolveToken_CancelStopsWaitingForLock(t *testing.T) {
	var calls atomic.Int64
	srv := setup(t, rotatingTokenHandler(&calls))
	// Still usable, but a cancelled command must stop rather than carry on.
	if err := SaveOAuth(saved(srv.URL, 30*time.Second)); err != nil {
		t.Fatal(err)
	}
	path, _ := config.OAuthPath()
	unlock, err := lockFile(t.Context(), path+".lock")
	if err != nil {
		t.Fatal(err)
	}
	defer unlock()

	ctx, cancel := context.WithTimeout(t.Context(), 100*time.Millisecond)
	defer cancel()
	start := time.Now()
	_, err = ResolveToken(ctx, "")
	if err == nil {
		t.Fatal("want error")
	}
	if elapsed := time.Since(start); elapsed > 5*time.Second {
		t.Errorf("took %v", elapsed)
	}
	if calls.Load() != 0 {
		t.Errorf("made %d requests", calls.Load())
	}
}

func TestResolveToken_CancelDuringRefreshIsReported(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	srv := setup(t, func(w http.ResponseWriter, r *http.Request) {
		cancel()
		w.WriteHeader(http.StatusInternalServerError)
	})
	if err := SaveOAuth(saved(srv.URL, 30*time.Second)); err != nil {
		t.Fatal(err)
	}
	_, err := ResolveToken(ctx, "")
	var cliErr *output.Error
	if !errors.As(err, &cliErr) || cliErr.Kind != output.KindCancelled {
		t.Fatalf("err = %v", err)
	}
}

// readOnlyConfigDir makes the directory holding the saved login read-only.
func readOnlyConfigDir(t *testing.T) {
	t.Helper()
	if runtime.GOOS == "windows" || os.Geteuid() == 0 {
		t.Skip("needs directory permissions to be enforced")
	}
	path, _ := config.OAuthPath()
	dir := filepath.Dir(path)
	if err := os.Chmod(dir, 0o500); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(dir, 0o700) })
}

// Refreshing would use up a refresh token whose replacement can't be saved.
func TestResolveToken_UnwritableConfigDoesNotRefresh(t *testing.T) {
	for _, tt := range []struct {
		name       string
		lockExists bool
		expiresIn  time.Duration
		wantUsable bool
	}{
		{name: "expired", expiresIn: -time.Minute},
		{name: "expired with lock file", lockExists: true, expiresIn: -time.Minute},
		{name: "still usable", expiresIn: 30 * time.Second, wantUsable: true},
		{name: "still usable with lock file", lockExists: true, expiresIn: 30 * time.Second, wantUsable: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			var calls atomic.Int64
			srv := setup(t, rotatingTokenHandler(&calls))
			if err := SaveOAuth(saved(srv.URL, tt.expiresIn)); err != nil {
				t.Fatal(err)
			}
			if tt.lockExists {
				path, _ := config.OAuthPath()
				if err := os.WriteFile(path+".lock", nil, 0o600); err != nil {
					t.Fatal(err)
				}
			}
			readOnlyConfigDir(t)

			start := time.Now()
			got, err := ResolveToken(t.Context(), "")
			if elapsed := time.Since(start); elapsed > 5*time.Second {
				t.Errorf("took %v", elapsed)
			}
			if tt.wantUsable {
				if err != nil || got != "os_oat_old" {
					t.Errorf("got %q, %v", got, err)
				}
			} else if err == nil {
				t.Errorf("got %q, want error", got)
			}
			if calls.Load() != 0 {
				t.Errorf("made %d requests", calls.Load())
			}
		})
	}
}

func TestResolveToken_SaveFailureAfterRefreshKeepsNewToken(t *testing.T) {
	var calls atomic.Int64
	rotate := rotatingTokenHandler(&calls)
	srv := setup(t, func(w http.ResponseWriter, r *http.Request) {
		readOnlyConfigDir(t)
		rotate(w, r)
	})
	if err := SaveOAuth(saved(srv.URL, -time.Minute)); err != nil {
		t.Fatal(err)
	}
	got, err := ResolveToken(t.Context(), "")
	if err != nil || got != "os_oat_1" {
		t.Fatalf("got %q, %v", got, err)
	}
}

func TestSavingOneCredentialTypeReplacesTheOther(t *testing.T) {
	srv := setup(t, nil)
	tokenPath, _ := config.TokenPath()
	oauthPath, _ := config.OAuthPath()

	if err := SaveToken("os_key"); err != nil {
		t.Fatal(err)
	}
	if err := SaveOAuth(saved(srv.URL, time.Hour)); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(tokenPath); !os.IsNotExist(err) {
		t.Errorf("API key survived a browser login: %v", err)
	}

	if err := SaveToken("os_key"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(oauthPath); !os.IsNotExist(err) {
		t.Errorf("browser login survived saving an API key: %v", err)
	}
	if got, err := ResolveToken(t.Context(), ""); err != nil || got != "os_key" {
		t.Fatalf("got %q, %v", got, err)
	}
}

func TestSaveOAuth_IsPrivate(t *testing.T) {
	srv := setup(t, nil)
	if err := SaveOAuth(saved(srv.URL, time.Hour)); err != nil {
		t.Fatal(err)
	}
	path, _ := config.OAuthPath()
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if perm := info.Mode().Perm(); perm != 0o600 {
		t.Errorf("mode = %o", perm)
	}
	if entries, _ := os.ReadDir(filepath.Dir(path)); len(entries) != 1 {
		t.Errorf("left extra files behind: %v", entries)
	}
}

func TestRemoveToken_RemovesBothAndReportsWhetherAnyExisted(t *testing.T) {
	srv := setup(t, nil)
	if removed, err := RemoveToken(); err != nil || removed {
		t.Fatalf("empty: removed=%v, err=%v", removed, err)
	}
	if err := SaveOAuth(saved(srv.URL, time.Hour)); err != nil {
		t.Fatal(err)
	}
	if removed, err := RemoveToken(); err != nil || !removed {
		t.Fatalf("saved: removed=%v, err=%v", removed, err)
	}
	if creds, err := LoadOAuth(); err != nil || creds != nil {
		t.Fatalf("after removal: %+v, %v", creds, err)
	}
}

func TestLoadOAuth_CorruptFile(t *testing.T) {
	setup(t, nil)
	path, _ := config.OAuthPath()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("not json"), 0o600); err != nil {
		t.Fatal(err)
	}
	_, err := ResolveToken(t.Context(), "")
	if err == nil || !strings.Contains(err.Error(), "openstatus login") {
		t.Fatalf("err = %v", err)
	}
}
