package api

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"runtime"
	"testing"

	"github.com/openstatusHQ/cli/internal/version"
)

// setBaseURL points the API base URL at u for the duration of the test.
func setBaseURL(t *testing.T, u string) {
	t.Helper()
	prev := BaseURL
	BaseURL = u
	t.Cleanup(func() { BaseURL = prev })
}

// roundTrip sends req through the CLI transport to a test server configured as
// the API, and returns the headers the server received.
func roundTrip(t *testing.T, req *http.Request) http.Header {
	t.Helper()
	var got http.Header
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = r.Header.Clone()
	}))
	defer srv.Close()
	setBaseURL(t, srv.URL)

	if req == nil {
		var err error
		if req, err = http.NewRequest(http.MethodGet, srv.URL, http.NoBody); err != nil {
			t.Fatal(err)
		}
	} else {
		req.URL, _ = req.URL.Parse(srv.URL)
	}
	resp, err := (&http.Client{Transport: NewTransport(nil)}).Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	return got
}

func TestCLITransport(t *testing.T) {
	t.Setenv("DO_NOT_TRACK", "")
	t.Setenv("OPENSTATUS_NO_TELEMETRY", "")
	t.Cleanup(func() { SetCommand("") })
	SetCommand("monitors apply")

	req, _ := http.NewRequest(http.MethodGet, "http://placeholder", http.NoBody)
	req.Header.Set("x-openstatus-key", "secret")
	got := roundTrip(t, req)

	if want := "openstatus-cli/" + version.Version + " (" + runtime.GOOS + "; " + runtime.GOARCH + ")"; got.Get("User-Agent") != want {
		t.Errorf("User-Agent = %q, want %q", got.Get("User-Agent"), want)
	}
	if got.Get(HeaderCLICommand) != "monitors apply" {
		t.Errorf("%s = %q, want %q", HeaderCLICommand, got.Get(HeaderCLICommand), "monitors apply")
	}
	if id := got.Get(HeaderCLIInvocation); len(id) != 32 || id != invocationID {
		t.Errorf("%s = %q, want the process invocation ID", HeaderCLIInvocation, id)
	}
	if got.Get("x-openstatus-key") != "secret" {
		t.Error("existing headers must be preserved")
	}
	if req.Header.Get("User-Agent") != "" {
		t.Error("transport must not mutate the caller's request")
	}
}

func TestCLITransportKeepsExistingUserAgent(t *testing.T) {
	req, _ := http.NewRequest(http.MethodGet, "http://placeholder", http.NoBody)
	req.Header.Set("User-Agent", "connect-go/1.20.0 (go1.25.0)")
	got := roundTrip(t, req)

	if want := userAgent + " connect-go/1.20.0 (go1.25.0)"; got.Get("User-Agent") != want {
		t.Errorf("User-Agent = %q, want %q", got.Get("User-Agent"), want)
	}
}

func TestCLITransportWithoutCommand(t *testing.T) {
	t.Setenv("DO_NOT_TRACK", "")
	t.Setenv("OPENSTATUS_NO_TELEMETRY", "")
	t.Cleanup(func() { SetCommand("") })
	SetCommand("")

	got := roundTrip(t, nil)
	if _, ok := got[http.CanonicalHeaderKey(HeaderCLICommand)]; ok {
		t.Errorf("%s should be omitted when no command is set", HeaderCLICommand)
	}
}

func TestCLITransportOptOut(t *testing.T) {
	t.Cleanup(func() { SetCommand("") })
	SetCommand("monitors apply")

	for _, env := range []string{"DO_NOT_TRACK", "OPENSTATUS_NO_TELEMETRY"} {
		t.Run(env, func(t *testing.T) {
			t.Setenv("DO_NOT_TRACK", "")
			t.Setenv("OPENSTATUS_NO_TELEMETRY", "")
			t.Setenv(env, "1")

			got := roundTrip(t, nil)
			for _, h := range []string{HeaderCLICommand, HeaderCLIInvocation} {
				if _, ok := got[http.CanonicalHeaderKey(h)]; ok {
					t.Errorf("%s should be omitted when %s is set", h, env)
				}
			}
			if got.Get("User-Agent") != userAgent {
				t.Errorf("User-Agent = %q, want %q", got.Get("User-Agent"), userAgent)
			}
		})
	}
}

func TestSendsUsage(t *testing.T) {
	t.Setenv("DO_NOT_TRACK", "")
	t.Setenv("OPENSTATUS_NO_TELEMETRY", "")
	checker, _ := url.Parse(PlayCheckerURL)
	cloud, _ := url.Parse(DefaultBaseURL + "/rpc/x")
	selfHosted, _ := url.Parse("https://api.example.com/rpc/x")
	other, _ := url.Parse("https://elsewhere.example.com/")

	tests := []struct {
		name string
		base string
		u    *url.URL
		want bool
	}{
		{"cloud api", DefaultBaseURL, cloud, true},
		{"cloud speed checker", DefaultBaseURL, checker, true},
		{"cloud other host", DefaultBaseURL, other, false},
		{"self-hosted api", "https://api.example.com", selfHosted, true},
		{"self-hosted speed checker", "https://api.example.com", checker, false},
		{"self-hosted cloud api", "https://api.example.com", cloud, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setBaseURL(t, tt.base)
			if got := sendsUsage(tt.u); got != tt.want {
				t.Errorf("sendsUsage(%s) with BaseURL %s = %v, want %v", tt.u, tt.base, got, tt.want)
			}
		})
	}
}

func TestUsageOptOutValues(t *testing.T) {
	for v, want := range map[string]bool{
		"": false, " ": false, "0": false, "false": false, "FALSE": false, "False": false,
		"no": false, "NO": false, "off": false, "Off": false, " 0 ": false,
		"1": true, "true": true, "TRUE": true, "yes": true, " 1 ": true,
	} {
		t.Setenv("OPENSTATUS_NO_TELEMETRY", "")
		t.Setenv("DO_NOT_TRACK", v)
		if got := UsageOptOut(); got != want {
			t.Errorf("DO_NOT_TRACK=%q: UsageOptOut() = %v, want %v", v, got, want)
		}
	}
}
