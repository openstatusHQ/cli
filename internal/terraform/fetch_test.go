package terraform

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
)

type recordedCall struct {
	Procedure string
	Limit     int32
	Offset    int32
}

// fakeTransport serves a queue of bodies per procedure suffix. Procedures with no
// queue get "{}", so unrelated sections of the export come back empty.
type fakeTransport struct {
	bodies map[string][]fakeResponse
	calls  []recordedCall
}

type fakeResponse struct {
	status int
	body   string
}

func okResponse(body string) fakeResponse {
	return fakeResponse{status: http.StatusOK, body: body}
}

// connectError builds a Connect error envelope for a unary JSON call. The HTTP
// status is what maps back to a connect.Code on the client side.
func connectError(status int, code string) fakeResponse {
	return fakeResponse{status: status, body: fmt.Sprintf(`{"code":%q,"message":"denied"}`, code)}
}

func (f *fakeTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	call := recordedCall{Procedure: req.URL.Path}

	if req.Body != nil {
		raw, err := io.ReadAll(req.Body)
		if err != nil {
			return nil, err
		}
		var decoded struct {
			Limit  int32 `json:"limit"`
			Offset int32 `json:"offset"`
		}
		if len(raw) > 0 {
			_ = json.Unmarshal(raw, &decoded)
		}
		call.Limit = decoded.Limit
		call.Offset = decoded.Offset
	}
	f.calls = append(f.calls, call)

	resp := okResponse("{}")
	for suffix, queue := range f.bodies {
		if !strings.HasSuffix(req.URL.Path, suffix) {
			continue
		}
		if len(queue) == 0 {
			break
		}
		resp = queue[0]
		if len(queue) > 1 {
			f.bodies[suffix] = queue[1:]
		}
		break
	}

	return &http.Response{
		StatusCode: resp.status,
		Body:       io.NopCloser(strings.NewReader(resp.body)),
		Header:     http.Header{"Content-Type": []string{"application/json"}},
	}, nil
}

func (f *fakeTransport) callsTo(suffix string) []recordedCall {
	var out []recordedCall
	for _, c := range f.calls {
		if strings.HasSuffix(c.Procedure, suffix) {
			out = append(out, c)
		}
	}
	return out
}

func newFetchClient(bodies map[string][]fakeResponse) (*http.Client, *fakeTransport) {
	transport := &fakeTransport{bodies: bodies}
	return &http.Client{Transport: transport}, transport
}

func listBody(ids []string, totalSize int) string {
	entries := make([]string, 0, len(ids))
	for _, id := range ids {
		entries = append(entries, fmt.Sprintf(`{"id":%q,"name":%q,"monitorCount":1}`, id, id))
	}
	return fmt.Sprintf(`{"privateLocations":[%s],"totalSize":%d}`, strings.Join(entries, ","), totalSize)
}

func getBody(id string, monitorIDs []string) string {
	quoted := make([]string, 0, len(monitorIDs))
	for _, m := range monitorIDs {
		quoted = append(quoted, fmt.Sprintf("%q", m))
	}
	return fmt.Sprintf(`{"privateLocation":{"id":%q,"name":%q,"monitorIds":[%s]}}`, id, id, strings.Join(quoted, ","))
}

func manyIDs(prefix string, n int) []string {
	out := make([]string, 0, n)
	for i := range n {
		out = append(out, fmt.Sprintf("%s%d", prefix, i))
	}
	return out
}

func TestFetchPrivateLocations_SinglePage(t *testing.T) {
	client, transport := newFetchClient(map[string][]fakeResponse{
		"/ListPrivateLocations": {okResponse(listBody([]string{"pl_1", "pl_2"}, 2))},
		"/GetPrivateLocation":   {okResponse(getBody("pl_1", []string{"mon-1", "mon-2"})), okResponse(getBody("pl_2", nil))},
	})

	data, err := FetchWorkspaceDataWithHTTPClient(context.Background(), client, "test-token")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got := len(data.PrivateLocations); got != 2 {
		t.Fatalf("got %d private locations, want 2", got)
	}
	if got := data.PrivateLocations[0].GetMonitorIds(); len(got) != 2 {
		t.Errorf("got monitor_ids %v, want 2 entries", got)
	}
	if got := len(transport.callsTo("/ListPrivateLocations")); got != 1 {
		t.Errorf("got %d list calls, want 1", got)
	}
	if got := len(transport.callsTo("/GetPrivateLocation")); got != 2 {
		t.Errorf("got %d get calls, want 2", got)
	}
}

func TestFetchPrivateLocations_MultiPage(t *testing.T) {
	firstPage := manyIDs("pl_", privateLocationPageSize)
	secondPage := manyIDs("pl_second_", 50)

	client, transport := newFetchClient(map[string][]fakeResponse{
		"/ListPrivateLocations": {
			okResponse(listBody(firstPage, 150)),
			okResponse(listBody(secondPage, 150)),
		},
		"/GetPrivateLocation": {okResponse(getBody("pl_x", nil))},
	})

	data, err := FetchWorkspaceDataWithHTTPClient(context.Background(), client, "test-token")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got := len(data.PrivateLocations); got != 150 {
		t.Fatalf("got %d private locations, want 150", got)
	}

	listCalls := transport.callsTo("/ListPrivateLocations")
	if len(listCalls) != 2 {
		t.Fatalf("got %d list calls, want 2", len(listCalls))
	}
	if listCalls[0].Limit != privateLocationPageSize || listCalls[0].Offset != 0 {
		t.Errorf("first page requested limit=%d offset=%d, want limit=%d offset=0", listCalls[0].Limit, listCalls[0].Offset, privateLocationPageSize)
	}
	if listCalls[1].Offset != privateLocationPageSize {
		t.Errorf("second page requested offset=%d, want %d", listCalls[1].Offset, privateLocationPageSize)
	}
}

func TestFetchPrivateLocations_EmptyPageStopsLoop(t *testing.T) {
	client, transport := newFetchClient(map[string][]fakeResponse{
		"/ListPrivateLocations": {
			okResponse(listBody([]string{"pl_1"}, 99)),
			okResponse(listBody(nil, 99)),
		},
		"/GetPrivateLocation": {okResponse(getBody("pl_1", nil))},
	})

	data, err := FetchWorkspaceDataWithHTTPClient(context.Background(), client, "test-token")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got := len(data.PrivateLocations); got != 1 {
		t.Fatalf("got %d private locations, want 1", got)
	}
	if got := len(transport.callsTo("/ListPrivateLocations")); got != 2 {
		t.Errorf("got %d list calls, want 2 (loop must stop on an empty page)", got)
	}
}

func TestFetchPrivateLocations_EmptyWorkspace(t *testing.T) {
	client, transport := newFetchClient(map[string][]fakeResponse{
		"/ListPrivateLocations": {okResponse(listBody(nil, 0))},
	})

	data, err := FetchWorkspaceDataWithHTTPClient(context.Background(), client, "test-token")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(data.PrivateLocations) != 0 {
		t.Errorf("got %d private locations, want 0", len(data.PrivateLocations))
	}
	if got := len(transport.callsTo("/GetPrivateLocation")); got != 0 {
		t.Errorf("got %d get calls, want 0", got)
	}
}

func TestFetchPrivateLocations_ToleratesUnavailableFeature(t *testing.T) {
	tests := []struct {
		name      string
		status    int
		code      string
		procedure string
	}{
		{"permission denied on list", http.StatusForbidden, "permission_denied", "/ListPrivateLocations"},
		{"unimplemented on list", http.StatusNotFound, "unimplemented", "/ListPrivateLocations"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, _ := newFetchClient(map[string][]fakeResponse{
				tt.procedure: {connectError(tt.status, tt.code)},
			})

			var data *WorkspaceData
			var err error
			warning := captureStderr(t, func() {
				data, err = FetchWorkspaceDataWithHTTPClient(context.Background(), client, "test-token")
			})

			if err != nil {
				t.Fatalf("expected the export to continue, got error: %v", err)
			}
			if len(data.PrivateLocations) != 0 {
				t.Errorf("got %d private locations, want 0", len(data.PrivateLocations))
			}
			if !strings.Contains(warning, "warning: skipping private locations") {
				t.Errorf("expected a stderr warning, got: %q", warning)
			}
		})
	}
}

func captureStderr(t *testing.T, fn func()) string {
	t.Helper()
	orig := os.Stderr
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	os.Stderr = w
	defer func() { os.Stderr = orig }()

	done := make(chan string, 1)
	go func() {
		b, _ := io.ReadAll(r)
		done <- string(b)
	}()

	fn()
	w.Close()
	return <-done
}

func TestFetchPrivateLocations_FatalOnOtherErrors(t *testing.T) {
	client, _ := newFetchClient(map[string][]fakeResponse{
		"/ListPrivateLocations": {connectError(http.StatusUnauthorized, "unauthenticated")},
	})

	if _, err := FetchWorkspaceDataWithHTTPClient(context.Background(), client, "test-token"); err == nil {
		t.Fatal("expected an error for a non-entitlement failure, got nil")
	}
}

func TestFetchPrivateLocations_PartialResultsDiscarded(t *testing.T) {
	client, _ := newFetchClient(map[string][]fakeResponse{
		"/ListPrivateLocations": {okResponse(listBody([]string{"pl_1", "pl_2"}, 2))},
		"/GetPrivateLocation": {
			okResponse(getBody("pl_1", []string{"mon-1"})),
			connectError(http.StatusForbidden, "permission_denied"),
		},
	})

	data, err := FetchWorkspaceDataWithHTTPClient(context.Background(), client, "test-token")
	if err != nil {
		t.Fatalf("expected the export to continue, got error: %v", err)
	}
	if len(data.PrivateLocations) != 0 {
		t.Errorf("got %d private locations, want 0 — partial results must be discarded", len(data.PrivateLocations))
	}
}
