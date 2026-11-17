package monitors_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"testing"

	output "github.com/openstatusHQ/cli/internal/cli"
	"github.com/openstatusHQ/cli/internal/monitors"
)

const regeneratedMonitor = `{"monitor":{"id":"7","name":"Nightly backup","period":"86400","grace":"1800","pingUrl":"https://api.openstatus.dev/heartbeat/new-token"}}`

type confirmRecorder struct {
	prompts []string
	answer  bool
	err     error
}

func (c *confirmRecorder) confirm(prompt string) (bool, error) {
	c.prompts = append(c.prompts, prompt)
	return c.answer, c.err
}

func regenerateRoutes(get pingsRoute, regenerate pingsRoute) map[string]pingsRoute {
	return map[string]pingsRoute{"GetMonitor": get, "RegenerateHeartbeatToken": regenerate}
}

func Test_regenerateHeartbeatToken_Rejections(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name      string
		get       pingsRoute
		wantKind  output.Kind
		wantInErr []string
	}{
		{
			name:      "HTTP monitor",
			get:       pingsRoute{http.StatusOK, pingsHTTPMonitor},
			wantKind:  output.KindInvalidArgument,
			wantInErr: []string{"monitor 7 is not a heartbeat monitor (type: http)", "only heartbeat monitors have a ping URL"},
		},
		{
			name:      "Not found",
			get:       pingsRoute{http.StatusNotFound, `{"code":"not_found","message":"Monitor not found"}`},
			wantKind:  output.KindNotFound,
			wantInErr: []string{"monitor 7 not found"},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			p := newPingsInterceptor(regenerateRoutes(tc.get, pingsRoute{http.StatusOK, regeneratedMonitor}))
			c := &confirmRecorder{answer: true}

			err := monitors.RegenerateHeartbeatTokenWithHTTPClient(context.Background(), p.client(), "test", "7", c.confirm)
			var cliErr *output.Error
			if !errors.As(err, &cliErr) || cliErr.Kind != tc.wantKind {
				t.Fatalf("expected kind %q, got %v", tc.wantKind, err)
			}
			for _, want := range tc.wantInErr {
				if !strings.Contains(err.Error(), want) {
					t.Errorf("expected %q in error: %v", want, err)
				}
			}
			if strings.Contains(err.Error(), "monitors logs") {
				t.Errorf("did not expect a logs hint: %v", err)
			}
			if len(c.prompts) != 0 {
				t.Errorf("expected no prompt, got %v", c.prompts)
			}
			if got := p.count("RegenerateHeartbeatToken"); got != 0 {
				t.Errorf("expected no regenerate call, got %d", got)
			}
		})
	}
}

// Not parallel: subtests redirect the process-wide os.Stdout/os.Stderr and toggle global output modes.
func Test_regenerateHeartbeatToken(t *testing.T) {
	heartbeat := pingsRoute{http.StatusOK, pingsHeartbeatMonitor}
	regenerated := pingsRoute{http.StatusOK, regeneratedMonitor}

	t.Run("Missing ID", func(t *testing.T) {
		p := newPingsInterceptor(regenerateRoutes(heartbeat, regenerated))
		var err error
		captureStderr(t, func() {
			err = monitors.RegenerateHeartbeatTokenWithHTTPClient(context.Background(), p.client(), "test", "", nil)
		})
		if err == nil || !strings.Contains(err.Error(), "monitor ID is required") {
			t.Fatalf("expected missing ID error, got %v", err)
		}
		if got := p.count(""); got != 0 {
			t.Errorf("expected no requests, got %d", got)
		}
	})

	t.Run("Declined", func(t *testing.T) {
		p := newPingsInterceptor(regenerateRoutes(heartbeat, regenerated))
		c := &confirmRecorder{answer: false}
		out := captureStdout(t, func() {
			if err := monitors.RegenerateHeartbeatTokenWithHTTPClient(context.Background(), p.client(), "test", "7", c.confirm); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
		if out != "" {
			t.Errorf("expected no output, got %q", out)
		}
		if got := p.count("RegenerateHeartbeatToken"); got != 0 {
			t.Errorf("expected no regenerate call, got %d", got)
		}
		if len(c.prompts) != 1 || !strings.Contains(c.prompts[0], "'Nightly backup' (7)") || !strings.Contains(c.prompts[0], "stops working immediately") {
			t.Errorf("unexpected prompt: %v", c.prompts)
		}
	})

	t.Run("Prompt fails", func(t *testing.T) {
		p := newPingsInterceptor(regenerateRoutes(heartbeat, regenerated))
		c := &confirmRecorder{err: output.UsageErrorf("confirmation required but stdin is not a terminal")}
		err := monitors.RegenerateHeartbeatTokenWithHTTPClient(context.Background(), p.client(), "test", "7", c.confirm)
		if err == nil || !strings.Contains(err.Error(), "failed to read input") {
			t.Fatalf("expected read error, got %v", err)
		}
		var cliErr *output.Error
		if !errors.As(err, &cliErr) || cliErr.Kind != output.KindUsage {
			t.Errorf("expected the usage error to be kept, got %v", err)
		}
		if got := p.count("RegenerateHeartbeatToken"); got != 0 {
			t.Errorf("expected no regenerate call, got %d", got)
		}
	})

	t.Run("Confirmed", func(t *testing.T) {
		p := newPingsInterceptor(regenerateRoutes(heartbeat, regenerated))
		c := &confirmRecorder{answer: true}
		out := captureStdout(t, func() {
			if err := monitors.RegenerateHeartbeatTokenWithHTTPClient(context.Background(), p.client(), "test", "7", c.confirm); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
		if got := p.count("RegenerateHeartbeatToken"); got != 1 {
			t.Errorf("expected one regenerate call, got %d", got)
		}
		for _, want := range []string{"Ping URL regenerated", "Nightly backup", "https://api.openstatus.dev/heartbeat/new-token", "previous URL no longer works"} {
			if !strings.Contains(out, want) {
				t.Errorf("expected %q in output, got:\n%s", want, out)
			}
		}
		if strings.Contains(out, "secret-ping-token") {
			t.Errorf("old ping URL printed:\n%s", out)
		}
	})

	t.Run("Auto-accept", func(t *testing.T) {
		p := newPingsInterceptor(regenerateRoutes(heartbeat, regenerated))
		captureStdout(t, func() {
			if err := monitors.RegenerateHeartbeatTokenWithHTTPClient(context.Background(), p.client(), "test", "7", nil); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
		if got := p.count("RegenerateHeartbeatToken"); got != 1 {
			t.Errorf("expected one regenerate call, got %d", got)
		}
	})

	t.Run("Quiet", func(t *testing.T) {
		output.SetQuietMode(true)
		t.Cleanup(func() { output.SetQuietMode(false) })

		p := newPingsInterceptor(regenerateRoutes(heartbeat, regenerated))
		out := captureStdout(t, func() {
			if err := monitors.RegenerateHeartbeatTokenWithHTTPClient(context.Background(), p.client(), "test", "7", nil); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
		if out != "https://api.openstatus.dev/heartbeat/new-token\n" {
			t.Errorf("expected only the URL, got %q", out)
		}
	})

	t.Run("JSON", func(t *testing.T) {
		output.SetJSONOutput(true)
		t.Cleanup(func() { output.SetJSONOutput(false) })

		p := newPingsInterceptor(regenerateRoutes(heartbeat, regenerated))
		out := captureStdout(t, func() {
			if err := monitors.RegenerateHeartbeatTokenWithHTTPClient(context.Background(), p.client(), "test", "7", nil); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
		var got map[string]string
		if err := json.Unmarshal([]byte(out), &got); err != nil {
			t.Fatalf("invalid JSON: %v\n%s", err, out)
		}
		want := map[string]string{"id": "7", "name": "Nightly backup", "ping_url": "https://api.openstatus.dev/heartbeat/new-token"}
		if len(got) != len(want) {
			t.Errorf("expected %v, got %v", want, got)
		}
		for k, v := range want {
			if got[k] != v {
				t.Errorf("%s = %q, want %q", k, got[k], v)
			}
		}
	})

	t.Run("Regenerate fails", func(t *testing.T) {
		p := newPingsInterceptor(regenerateRoutes(heartbeat, pingsRoute{http.StatusInternalServerError, `{"code":"internal","message":"boom"}`}))
		var err error
		out := captureStdout(t, func() {
			err = monitors.RegenerateHeartbeatTokenWithHTTPClient(context.Background(), p.client(), "test", "7", nil)
		})
		if err == nil {
			t.Fatal("expected an error")
		}
		if out != "" {
			t.Errorf("expected no output, got %q", out)
		}
	})

	t.Run("Empty ping URL", func(t *testing.T) {
		p := newPingsInterceptor(regenerateRoutes(heartbeat, pingsRoute{http.StatusOK, `{"monitor":{"id":"7","name":"Nightly backup"}}`}))
		var err error
		out := captureStdout(t, func() {
			err = monitors.RegenerateHeartbeatTokenWithHTTPClient(context.Background(), p.client(), "test", "7", nil)
		})
		if err == nil || !strings.Contains(err.Error(), "did not return a new ping URL") {
			t.Fatalf("expected empty URL error, got %v", err)
		}
		if out != "" {
			t.Errorf("expected no output, got %q", out)
		}
	})
}
