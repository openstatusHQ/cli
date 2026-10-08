package api

import (
	"net/http"
	"net/http/httptest"
	"runtime"
	"testing"

	"github.com/openstatusHQ/cli/internal/version"
)

// roundTrip sends req through the CLI transport and returns the headers the
// server received.
func roundTrip(t *testing.T, req *http.Request) http.Header {
	t.Helper()
	var got http.Header
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = r.Header.Clone()
	}))
	defer srv.Close()

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

func TestUsageOptOutValues(t *testing.T) {
	for v, want := range map[string]bool{"": false, "0": false, "false": false, "1": true, "true": true} {
		t.Setenv("OPENSTATUS_NO_TELEMETRY", "")
		t.Setenv("DO_NOT_TRACK", v)
		if got := UsageOptOut(); got != want {
			t.Errorf("DO_NOT_TRACK=%q: UsageOptOut() = %v, want %v", v, got, want)
		}
	}
}
