package monitors_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"slices"
	"strings"
	"sync"
	"testing"

	monitorv1 "buf.build/gen/go/openstatus/api/protocolbuffers/go/openstatus/monitor/v1"
	"google.golang.org/protobuf/encoding/protojson"

	output "github.com/openstatusHQ/cli/internal/cli"
	"github.com/openstatusHQ/cli/internal/monitors"
)

const (
	pingsHeartbeatMonitor = `{"monitor":{"heartbeat":{"id":"7","name":"Nightly backup","period":"86400","grace":"1800","pingUrl":"https://api.openstatus.dev/heartbeat/secret-ping-token"}}}`
	pingsHTTPMonitor      = `{"monitor":{"http":{"id":"7","name":"API","url":"https://example.com"}}}`
	pingsTwo              = `{"pings":[` +
		`{"id":"p2","monitorId":"7","timestamp":"1794794400000","kind":"HEARTBEAT_PING_KIND_FAIL","method":"POST","ip":"203.0.113.7","userAgent":"curl/8.7","headers":{"content-type":"application/json"},"body":"{\"exit\":1}","bodyTruncated":true},` +
		`{"id":"p1","monitorId":"7","timestamp":"1794790800000","kind":"HEARTBEAT_PING_KIND_SUCCESS","method":"GET"}` +
		`],"nextCursor":"c-next"}`
)

type pingsCall struct {
	path string
	body []byte
}

type pingsRoute struct {
	status int
	body   string
}

type pingsInterceptor struct {
	mu     sync.Mutex
	calls  []pingsCall
	routes map[string]pingsRoute
}

func newPingsInterceptor(routes map[string]pingsRoute) *pingsInterceptor {
	return &pingsInterceptor{routes: routes}
}

func (p *pingsInterceptor) client() *http.Client {
	return (&interceptorHTTPClient{f: func(req *http.Request) (*http.Response, error) {
		var body []byte
		if req.Body != nil {
			body, _ = io.ReadAll(req.Body)
		}
		p.mu.Lock()
		p.calls = append(p.calls, pingsCall{path: req.URL.Path, body: body})
		p.mu.Unlock()

		route := pingsRoute{status: http.StatusInternalServerError, body: `{"code":"internal","message":"unexpected call"}`}
		for suffix, r := range p.routes {
			if strings.HasSuffix(req.URL.Path, "/"+suffix) {
				route = r
			}
		}
		return &http.Response{
			StatusCode: route.status,
			Body:       io.NopCloser(bytes.NewReader([]byte(route.body))),
			Header:     http.Header{"Content-Type": []string{"application/json"}},
		}, nil
	}}).GetHTTPClient()
}

func (p *pingsInterceptor) count(rpc string) int {
	p.mu.Lock()
	defer p.mu.Unlock()
	n := 0
	for _, c := range p.calls {
		if rpc == "" || strings.HasSuffix(c.path, "/"+rpc) {
			n++
		}
	}
	return n
}

func (p *pingsInterceptor) pingsRequest(t *testing.T) *monitorv1.ListMonitorHeartbeatPingsRequest {
	t.Helper()
	p.mu.Lock()
	defer p.mu.Unlock()
	for _, c := range p.calls {
		if strings.HasSuffix(c.path, "/ListMonitorHeartbeatPings") {
			req := &monitorv1.ListMonitorHeartbeatPingsRequest{}
			if err := protojson.Unmarshal(c.body, req); err != nil {
				t.Fatalf("failed to decode request: %v", err)
			}
			return req
		}
	}
	t.Fatal("ListMonitorHeartbeatPings was not called")
	return nil
}

func okPings(body string) map[string]pingsRoute {
	return map[string]pingsRoute{"ListMonitorHeartbeatPings": {http.StatusOK, body}}
}

func int32Ptr(v int32) *int32 { return &v }

// Not parallel: the missing-ID case redirects the process-wide os.Stderr.
func Test_listHeartbeatPings_Validation(t *testing.T) {
	for _, tc := range []struct {
		name string
		id   string
		opts monitors.PingsOptions
		want string
	}{
		{"missing ID", "", monitors.PingsOptions{}, "monitor ID is required"},
		{"limit zero", "7", monitors.PingsOptions{Limit: int32Ptr(0)}, "--limit must be between 1 and 100"},
		{"limit too high", "7", monitors.PingsOptions{Limit: int32Ptr(101)}, "--limit must be between 1 and 100"},
		{"bad from", "7", monitors.PingsOptions{From: "yesterday"}, "invalid RFC 3339 timestamp"},
		{"from equals to", "7", monitors.PingsOptions{From: "2026-11-01T00:00:00Z", To: "2026-11-01T00:00:00Z"}, "--from must be before --to"},
		{"from after to", "7", monitors.PingsOptions{From: "2026-11-02T00:00:00Z", To: "2026-11-01T00:00:00Z"}, "--from must be before --to"},
		{"unknown kind", "7", monitors.PingsOptions{Kinds: []string{"maybe"}}, `invalid --kind "maybe"`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p := newPingsInterceptor(okPings(`{"pings":[]}`))
			var err error
			captureStderr(t, func() {
				err = monitors.ListHeartbeatPingsWithHTTPClient(context.Background(), p.client(), "test", tc.id, tc.opts)
			})
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("expected error containing %q, got %v", tc.want, err)
			}
			var cliErr *output.Error
			if !errors.As(err, &cliErr) || cliErr.Kind != output.KindUsage {
				t.Errorf("expected a usage error, got %v", err)
			}
			if got := p.count(""); got != 0 {
				t.Errorf("expected no requests, got %d", got)
			}
		})
	}
}

// Not parallel: subtests redirect the process-wide os.Stdout.
func Test_listHeartbeatPings_Request(t *testing.T) {
	for _, tc := range []struct {
		name  string
		opts  monitors.PingsOptions
		check func(t *testing.T, req *monitorv1.ListMonitorHeartbeatPingsRequest)
	}{
		{
			name: "both kinds send no filter",
			opts: monitors.PingsOptions{Kinds: []string{"success", "fail", "fail"}},
			check: func(t *testing.T, req *monitorv1.ListMonitorHeartbeatPingsRequest) {
				if len(req.GetKinds()) != 0 {
					t.Errorf("expected no kinds, got %v", req.GetKinds())
				}
			},
		},
		{
			name: "single kind is normalised",
			opts: monitors.PingsOptions{Kinds: []string{" FAIL "}},
			check: func(t *testing.T, req *monitorv1.ListMonitorHeartbeatPingsRequest) {
				want := []monitorv1.HeartbeatPingKind{monitorv1.HeartbeatPingKind_HEARTBEAT_PING_KIND_FAIL}
				if !slices.Equal(req.GetKinds(), want) {
					t.Errorf("expected %v, got %v", want, req.GetKinds())
				}
			},
		},
		{
			name: "set options are forwarded",
			opts: monitors.PingsOptions{Limit: int32Ptr(10), From: "2026-11-01T00:00:00Z", To: "2026-11-02T00:00:00Z", Cursor: "c-1"},
			check: func(t *testing.T, req *monitorv1.ListMonitorHeartbeatPingsRequest) {
				if req.GetId() != "7" || req.GetLimit() != 10 || req.GetCursor() != "c-1" {
					t.Errorf("unexpected request: %v", req)
				}
				if req.GetFromTimestamp() != 1793491200000 || req.GetToTimestamp() != 1793577600000 {
					t.Errorf("unexpected window: %d-%d", req.GetFromTimestamp(), req.GetToTimestamp())
				}
			},
		},
		{
			name: "unset options are omitted",
			opts: monitors.PingsOptions{},
			check: func(t *testing.T, req *monitorv1.ListMonitorHeartbeatPingsRequest) {
				if req.HasLimit() || req.HasCursor() || req.HasFromTimestamp() || req.HasToTimestamp() || len(req.GetKinds()) != 0 {
					t.Errorf("expected only the ID, got %v", req)
				}
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p := newPingsInterceptor(okPings(`{"pings":[]}`))
			captureStdout(t, func() {
				if err := monitors.ListHeartbeatPingsWithHTTPClient(context.Background(), p.client(), "test", "7", tc.opts); err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
			})
			tc.check(t, p.pingsRequest(t))
		})
	}
}

// Not parallel: subtests redirect os.Stdout/os.Stderr and toggle global output modes.
func Test_listHeartbeatPings_Output(t *testing.T) {
	t.Run("Table", func(t *testing.T) {
		p := newPingsInterceptor(okPings(pingsTwo))
		var stdout string
		stderr := captureStderr(t, func() {
			stdout = captureStdout(t, func() {
				if err := monitors.ListHeartbeatPingsWithHTTPClient(context.Background(), p.client(), "test", "7", monitors.PingsOptions{}); err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
			})
		})

		for _, want := range []string{"Time", "User-Agent", "2026-11-16T02:00:00Z", "● fail", "POST", "203.0.113.7", "curl/8.7", "● success", "GET"} {
			if !strings.Contains(stdout, want) {
				t.Errorf("expected %q in output, got:\n%s", want, stdout)
			}
		}
		for _, unwanted := range []string{"p1", "p2", "exit", "content-type"} {
			if strings.Contains(stdout, unwanted) {
				t.Errorf("did not expect %q in output, got:\n%s", unwanted, stdout)
			}
		}
		lines := strings.Split(strings.TrimSpace(stdout), "\n")
		if len(lines) != 3 {
			t.Fatalf("expected header plus 2 rows, got:\n%s", stdout)
		}
		if fields := strings.Fields(lines[2]); !slices.Equal(fields[len(fields)-2:], []string{"-", "-"}) {
			t.Errorf("expected dashes for missing IP and User-Agent, got %q", lines[2])
		}
		if !strings.Contains(stderr, "Use --cursor c-next to see the next page") {
			t.Errorf("expected cursor hint on stderr, got: %q", stderr)
		}
	})

	t.Run("JSON", func(t *testing.T) {
		output.SetJSONOutput(true)
		t.Cleanup(func() { output.SetJSONOutput(false) })

		p := newPingsInterceptor(okPings(pingsTwo))
		out := captureStdout(t, func() {
			if err := monitors.ListHeartbeatPingsWithHTTPClient(context.Background(), p.client(), "test", "7", monitors.PingsOptions{}); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})

		var got struct {
			Pings      []map[string]json.RawMessage `json:"pings"`
			NextCursor string                       `json:"next_cursor"`
		}
		if err := json.Unmarshal([]byte(out), &got); err != nil {
			t.Fatalf("invalid JSON: %v\n%s", err, out)
		}
		if got.NextCursor != "c-next" || len(got.Pings) != 2 {
			t.Fatalf("unexpected output: %s", out)
		}
		first := got.Pings[0]
		for key, want := range map[string]string{
			"id":             `"p2"`,
			"monitor_id":     `"7"`,
			"timestamp":      `"2026-11-16T02:00:00Z"`,
			"kind":           `"fail"`,
			"method":         `"POST"`,
			"ip":             `"203.0.113.7"`,
			"user_agent":     `"curl/8.7"`,
			"headers":        `{"content-type":"application/json"}`,
			"body":           `"{\"exit\":1}"`,
			"body_truncated": `true`,
		} {
			var compact bytes.Buffer
			if err := json.Compact(&compact, first[key]); err != nil || compact.String() != want {
				t.Errorf("pings[0].%s = %s, want %s", key, first[key], want)
			}
		}
		for _, key := range []string{"ip", "user_agent", "headers", "body"} {
			if _, ok := got.Pings[1][key]; ok {
				t.Errorf("expected pings[1] to omit %q: %s", key, out)
			}
		}
		if strings.Contains(out, "ping_url") || strings.Contains(out, "secret-ping-token") {
			t.Errorf("ping URL leaked into JSON output:\n%s", out)
		}
	})

	t.Run("JSON last page has no cursor", func(t *testing.T) {
		output.SetJSONOutput(true)
		t.Cleanup(func() { output.SetJSONOutput(false) })

		p := newPingsInterceptor(okPings(`{"pings":[{"id":"p1","monitorId":"7","timestamp":"1794790800000","kind":"HEARTBEAT_PING_KIND_SUCCESS","method":"GET"}]}`))
		out := captureStdout(t, func() {
			if err := monitors.ListHeartbeatPingsWithHTTPClient(context.Background(), p.client(), "test", "7", monitors.PingsOptions{}); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
		if strings.Contains(out, "next_cursor") {
			t.Errorf("expected no next_cursor, got:\n%s", out)
		}
	})

	t.Run("Empty", func(t *testing.T) {
		p := newPingsInterceptor(okPings(`{}`))
		var stdout string
		stderr := captureStderr(t, func() {
			stdout = captureStdout(t, func() {
				if err := monitors.ListHeartbeatPingsWithHTTPClient(context.Background(), p.client(), "test", "7", monitors.PingsOptions{}); err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
			})
		})
		if strings.TrimSpace(stdout) != "No pings found" {
			t.Errorf("expected 'No pings found', got %q", stdout)
		}
		if stderr != "" {
			t.Errorf("expected no stderr, got %q", stderr)
		}
	})

	t.Run("Empty JSON", func(t *testing.T) {
		output.SetJSONOutput(true)
		t.Cleanup(func() { output.SetJSONOutput(false) })

		p := newPingsInterceptor(okPings(`{}`))
		out := captureStdout(t, func() {
			if err := monitors.ListHeartbeatPingsWithHTTPClient(context.Background(), p.client(), "test", "7", monitors.PingsOptions{}); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
		if compact := strings.Join(strings.Fields(out), ""); compact != `{"pings":[]}` {
			t.Errorf(`expected {"pings":[]}, got %q`, out)
		}
	})

	t.Run("Empty quiet", func(t *testing.T) {
		output.SetQuietMode(true)
		t.Cleanup(func() { output.SetQuietMode(false) })

		p := newPingsInterceptor(okPings(`{}`))
		out := captureStdout(t, func() {
			if err := monitors.ListHeartbeatPingsWithHTTPClient(context.Background(), p.client(), "test", "7", monitors.PingsOptions{}); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
		if out != "" {
			t.Errorf("expected no output, got %q", out)
		}
	})
}

func Test_listHeartbeatPings_Errors(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name      string
		pings     pingsRoute
		get       pingsRoute
		wantKind  output.Kind
		wantInErr []string
		wantGet   int
	}{
		{
			name:      "type mismatch on an HTTP monitor points to logs",
			pings:     pingsRoute{http.StatusBadRequest, `{"code":"invalid_argument","message":"Monitor type mismatch: expected heartbeat, got http"}`},
			get:       pingsRoute{http.StatusOK, pingsHTTPMonitor},
			wantKind:  output.KindInvalidArgument,
			wantInErr: []string{"monitor 7 is not a heartbeat monitor (type: http)", "only heartbeat monitors have pings", "openstatus monitors logs 7"},
			wantGet:   1,
		},
		{
			name:      "plan rejection on a heartbeat explains the plan",
			pings:     pingsRoute{http.StatusForbidden, `{"code":"permission_denied","message":"Upgrade for response logs"}`},
			get:       pingsRoute{http.StatusOK, pingsHeartbeatMonitor},
			wantKind:  output.KindPermissionDenied,
			wantInErr: []string{"requires a plan with response logs", "openstatus monitors info 7"},
			wantGet:   1,
		},
		{
			name:      "bad cursor on a heartbeat keeps the API error",
			pings:     pingsRoute{http.StatusBadRequest, `{"code":"invalid_argument","message":"Invalid cursor"}`},
			get:       pingsRoute{http.StatusOK, pingsHeartbeatMonitor},
			wantKind:  output.KindInvalidArgument,
			wantInErr: []string{"invalid request: Invalid cursor"},
			wantGet:   1,
		},
		{
			name:      "not found",
			pings:     pingsRoute{http.StatusNotFound, `{"code":"not_found","message":"Monitor not found"}`},
			wantKind:  output.KindNotFound,
			wantInErr: []string{"monitor 7 not found", "openstatus monitors list"},
			wantGet:   0,
		},
		{
			name:      "failed lookup keeps the API error",
			pings:     pingsRoute{http.StatusForbidden, `{"code":"permission_denied","message":"Upgrade for response logs"}`},
			get:       pingsRoute{http.StatusInternalServerError, `{"code":"internal","message":"boom"}`},
			wantKind:  output.KindPermissionDenied,
			wantInErr: []string{"permission denied"},
			wantGet:   1,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			routes := map[string]pingsRoute{"ListMonitorHeartbeatPings": tc.pings}
			if tc.get.status != 0 {
				routes["GetMonitor"] = tc.get
			}
			p := newPingsInterceptor(routes)

			err := monitors.ListHeartbeatPingsWithHTTPClient(context.Background(), p.client(), "test", "7", monitors.PingsOptions{})
			if err == nil {
				t.Fatal("expected an error")
			}
			var cliErr *output.Error
			if !errors.As(err, &cliErr) || cliErr.Kind != tc.wantKind {
				t.Errorf("expected kind %q, got %v", tc.wantKind, err)
			}
			for _, want := range tc.wantInErr {
				if !strings.Contains(err.Error(), want) {
					t.Errorf("expected %q in error: %v", want, err)
				}
			}
			if got := p.count("GetMonitor"); got != tc.wantGet {
				t.Errorf("expected %d GetMonitor calls, got %d", tc.wantGet, got)
			}
		})
	}
}
