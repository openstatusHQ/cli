package monitors_test

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/rodaine/table"

	"github.com/openstatusHQ/cli/internal/monitors"
	"github.com/openstatusHQ/cli/internal/privatelocation"
)

func bodyResponder(body string) *interceptorHTTPClient {
	return &interceptorHTTPClient{
		f: func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(body)),
				Header:     http.Header{"Content-Type": []string{"application/json"}},
			}, nil
		},
	}
}

// captureTable collects table output. rodaine/table binds DefaultWriter to os.Stdout at package
// init, so redirecting os.Stdout after the fact would capture nothing.
func captureTable(t *testing.T, f func()) string {
	t.Helper()

	var buf bytes.Buffer
	original := table.DefaultWriter
	table.DefaultWriter = &buf
	t.Cleanup(func() { table.DefaultWriter = original })

	f()

	return buf.String()
}

const monitorsWithoutPrivateLocations = `{"httpMonitors":[{"id":"123","name":"api-prod","url":"https://api.example.com","active":true}],"tcpMonitors":[],"dnsMonitors":[]}`

const monitorsWithPrivateLocations = `{"httpMonitors":[{"id":"123","name":"api-prod","url":"https://api.example.com","active":true,"privateLocationIds":["pl_1"]},{"id":"124","name":"marketing","url":"https://example.com","active":true}],"tcpMonitors":[],"dnsMonitors":[]}`

// Not parallel: these subtests swap the package-level table.DefaultWriter.
func Test_ListMonitors_PrivateLocations(t *testing.T) {
	t.Run("Output is unchanged when nothing is attached", func(t *testing.T) {
		client := bodyResponder(monitorsWithoutPrivateLocations)
		monitorClient := monitors.NewMonitorClientWithHTTPClient(client.GetHTTPClient(), "test-token")

		resolverCalls := 0
		resolver := func(ctx context.Context, ids []string) (map[string]privatelocation.Ref, error) {
			resolverCalls++
			return nil, nil
		}

		out := captureTable(t, func() {
			if err := monitors.ListMonitors(context.Background(), monitorClient, resolver, false, nil); err != nil {
				t.Errorf("Expected no error, got %v", err)
			}
		})

		if resolverCalls != 0 {
			t.Errorf("Expected zero private-location lookups, got %d", resolverCalls)
		}
		if strings.Contains(out, "Private Locations") {
			t.Error("Expected no private-location column when nothing is attached")
		}
		for _, want := range []string{"ID", "Name", "Url", "Kind", "api-prod"} {
			if !strings.Contains(out, want) {
				t.Errorf("Expected the original output to contain %q", want)
			}
		}
	})

	t.Run("Renders resolved names when attached", func(t *testing.T) {
		client := bodyResponder(monitorsWithPrivateLocations)
		monitorClient := monitors.NewMonitorClientWithHTTPClient(client.GetHTTPClient(), "test-token")

		resolver := func(ctx context.Context, ids []string) (map[string]privatelocation.Ref, error) {
			return map[string]privatelocation.Ref{
				"pl_1": {ID: "pl_1", Name: "office-paris", Status: "active"},
			}, nil
		}

		out := captureTable(t, func() {
			if err := monitors.ListMonitors(context.Background(), monitorClient, resolver, false, nil); err != nil {
				t.Errorf("Expected no error, got %v", err)
			}
		})

		if !strings.Contains(out, "Private Locations") {
			t.Error("Expected the private-location column")
		}
		if !strings.Contains(out, "office-paris") {
			t.Error("Expected the resolved location name")
		}
	})

	t.Run("Falls back to raw IDs when the lookup fails", func(t *testing.T) {
		client := bodyResponder(monitorsWithPrivateLocations)
		monitorClient := monitors.NewMonitorClientWithHTTPClient(client.GetHTTPClient(), "test-token")

		resolver := func(ctx context.Context, ids []string) (map[string]privatelocation.Ref, error) {
			return nil, errors.New("permission denied")
		}

		out := captureTable(t, func() {
			if err := monitors.ListMonitors(context.Background(), monitorClient, resolver, false, nil); err != nil {
				t.Errorf("Expected the command to succeed despite the lookup failure, got %v", err)
			}
		})

		if !strings.Contains(out, "pl_1") {
			t.Error("Expected the raw private-location ID as a fallback")
		}
	})

	t.Run("Tolerates a nil resolver", func(t *testing.T) {
		client := bodyResponder(monitorsWithPrivateLocations)
		monitorClient := monitors.NewMonitorClientWithHTTPClient(client.GetHTTPClient(), "test-token")

		out := captureTable(t, func() {
			if err := monitors.ListMonitors(context.Background(), monitorClient, nil, false, nil); err != nil {
				t.Errorf("Expected no error, got %v", err)
			}
		})

		if !strings.Contains(out, "pl_1") {
			t.Error("Expected the raw private-location ID when no resolver is supplied")
		}
	})
}
