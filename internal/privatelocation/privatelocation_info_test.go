package privatelocation_test

import (
	"bytes"
	"context"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/openstatusHQ/cli/internal/privatelocation"
)

const (
	getProcedure         = "/GetPrivateLocation"
	listMonitorProcedure = "/ListMonitors"
)

// captureStdout runs f with stdout redirected and returns everything it printed.
func captureStdout(t *testing.T, f func()) string {
	t.Helper()

	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to create pipe: %v", err)
	}
	original := os.Stdout
	os.Stdout = w
	t.Cleanup(func() { os.Stdout = original })

	f()

	if err := w.Close(); err != nil {
		t.Fatalf("failed to close pipe: %v", err)
	}
	os.Stdout = original

	var buf bytes.Buffer
	if _, err := io.Copy(&buf, r); err != nil {
		t.Fatalf("failed to read pipe: %v", err)
	}
	return buf.String()
}

const infoBody = `{"privateLocation":{"id":"pl_1","name":"office-paris","token":"os_pl_live_9f3a2b1c8d4e6f","monitorIds":["123","999"],"lastSeenAt":"2026-07-24T14:02:11Z","createdAt":"2026-01-01T00:00:00Z","updatedAt":"2026-01-01T00:00:00Z","metadata":{"env":"prod"},"status":"PRIVATE_LOCATION_STATUS_ACTIVE"}}`

const monitorsBody = `{"httpMonitors":[{"id":"123","name":"api-prod","url":"https://api.example.com","active":true}],"tcpMonitors":[],"dnsMonitors":[]}`

// Not parallel: these subtests redirect the process-wide os.Stdout.
func Test_GetPrivateLocationInfo(t *testing.T) {
	t.Run("Masks the token by default", func(t *testing.T) {
		client := jsonResponder(map[string]string{
			getProcedure:         infoBody,
			listMonitorProcedure: monitorsBody,
		}, nil)

		out := captureStdout(t, func() {
			if err := privatelocation.GetPrivateLocationInfoWithHTTPClient(context.Background(), client.GetHTTPClient(), "test-token", "pl_1", false); err != nil {
				t.Errorf("Expected no error, got %v", err)
			}
		})

		if strings.Contains(out, "os_pl_live_9f3a2b1c8d4e6f") {
			t.Error("Expected the token to be masked by default")
		}
		if !strings.Contains(out, "4e6f") {
			t.Error("Expected the masked token to keep the last four characters")
		}
	})

	t.Run("Reveals the token with showToken", func(t *testing.T) {
		client := jsonResponder(map[string]string{
			getProcedure:         infoBody,
			listMonitorProcedure: monitorsBody,
		}, nil)

		out := captureStdout(t, func() {
			if err := privatelocation.GetPrivateLocationInfoWithHTTPClient(context.Background(), client.GetHTTPClient(), "test-token", "pl_1", true); err != nil {
				t.Errorf("Expected no error, got %v", err)
			}
		})

		if !strings.Contains(out, "os_pl_live_9f3a2b1c8d4e6f") {
			t.Error("Expected the full token when showToken is set")
		}
	})

	t.Run("Resolves monitor names and falls back to IDs", func(t *testing.T) {
		client := jsonResponder(map[string]string{
			getProcedure:         infoBody,
			listMonitorProcedure: monitorsBody,
		}, nil)

		out := captureStdout(t, func() {
			if err := privatelocation.GetPrivateLocationInfoWithHTTPClient(context.Background(), client.GetHTTPClient(), "test-token", "pl_1", false); err != nil {
				t.Errorf("Expected no error, got %v", err)
			}
		})

		if !strings.Contains(out, "api-prod") {
			t.Error("Expected the resolved monitor name")
		}
		// 999 has no matching monitor, so it should appear as a bare ID.
		if !strings.Contains(out, "999") {
			t.Error("Expected the unresolved monitor ID to still be listed")
		}
	})

	t.Run("Requires an ID", func(t *testing.T) {
		client := jsonResponder(map[string]string{}, nil)

		err := privatelocation.GetPrivateLocationInfoWithHTTPClient(context.Background(), client.GetHTTPClient(), "test-token", "", false)
		if err == nil {
			t.Fatal("Expected an error when no ID is given")
		}
	})

	t.Run("Gives a pl-specific hint when not found", func(t *testing.T) {
		client := errorResponder(404, `{"code":"not_found","message":"missing"}`)

		err := privatelocation.GetPrivateLocationInfoWithHTTPClient(context.Background(), client.GetHTTPClient(), "test-token", "pl_bogus", false)
		if err == nil {
			t.Fatal("Expected a not-found error")
		}
		if !strings.Contains(err.Error(), "openstatus pl list") {
			t.Errorf("Expected the hint to reference 'openstatus pl list', got %q", err.Error())
		}
		if !strings.Contains(err.Error(), "pl_bogus") {
			t.Errorf("Expected the offending ID in the message, got %q", err.Error())
		}
	})
}
