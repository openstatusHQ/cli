package incident_test

import (
	"io"
	"net/http"
	"os"
	"strings"
	"testing"

	"github.com/fatih/color"
	"github.com/rodaine/table"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"

	output "github.com/openstatusHQ/cli/internal/cli"
)

func TestMain(m *testing.M) {
	color.NoColor = true
	os.Exit(m.Run())
}

const incidentJSON = `{
	"id": "42",
	"title": "Checkout 502",
	"severity": "INCIDENT_SEVERITY_MAJOR",
	"status": "INCIDENT_STATUS_OPEN",
	"commander": {"email": "jane@example.com", "name": "Jane"},
	"declaredAt": "2026-10-16T10:00:00Z",
	"startedAt": "2026-10-16T09:55:00Z",
	"allowedTransitions": ["INCIDENT_STATUS_MITIGATED", "INCIDENT_STATUS_RESOLVED", "INCIDENT_STATUS_CANCELED"],
	"deletable": true,
	"createdAt": "2026-10-16T10:00:00Z",
	"updatedAt": "2026-10-16T10:00:00Z"
}`

func incidentResponse() string {
	return `{"incident":` + incidentJSON + `}`
}

type fakeServer struct {
	status  int
	body    string
	calls   int
	path    string
	reqBody []byte
}

func newServer(status int, body string) *fakeServer {
	return &fakeServer{status: status, body: body}
}

func errorServer(status int, code, message string) *fakeServer {
	return newServer(status, `{"code":"`+code+`","message":"`+message+`"}`)
}

func (f *fakeServer) RoundTrip(req *http.Request) (*http.Response, error) {
	f.calls++
	f.path = req.URL.Path
	if req.Body != nil {
		body, err := io.ReadAll(req.Body)
		if err != nil {
			return nil, err
		}
		f.reqBody = body
	}
	return &http.Response{
		StatusCode: f.status,
		Body:       io.NopCloser(strings.NewReader(f.body)),
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Request:    req,
	}, nil
}

func (f *fakeServer) client() *http.Client {
	return &http.Client{Transport: f}
}

func (f *fakeServer) procedure() string {
	return f.path[strings.LastIndex(f.path, "/")+1:]
}

func decodeBody[T proto.Message](t *testing.T, body []byte, into T) T {
	t.Helper()
	if len(body) == 0 {
		t.Fatal("expected non-empty request body")
	}
	if err := protojson.Unmarshal(body, into); err != nil {
		t.Fatalf("failed to unmarshal request body: %v\nbody: %s", err, string(body))
	}
	return into
}

func captureStdout(t *testing.T, f func()) string {
	t.Helper()

	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to create pipe: %v", err)
	}
	original, originalTable := os.Stdout, table.DefaultWriter
	os.Stdout, table.DefaultWriter = w, w
	defer func() { os.Stdout, table.DefaultWriter = original, originalTable }()

	done := make(chan string, 1)
	go func() {
		b, _ := io.ReadAll(r)
		done <- string(b)
	}()

	f()
	w.Close()
	return <-done
}

func withOutputModes(t *testing.T, jsonMode, quiet bool) {
	t.Helper()
	output.SetJSONOutput(jsonMode)
	output.SetQuietMode(quiet)
	t.Cleanup(func() {
		output.SetJSONOutput(false)
		output.SetQuietMode(false)
	})
}

func strPtr(s string) *string {
	return &s
}
