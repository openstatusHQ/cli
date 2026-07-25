package privatelocation_test

import (
	"context"
	"strings"
	"testing"

	"github.com/openstatusHQ/cli/internal/privatelocation"
)

const createProcedure = "/CreatePrivateLocation"

const createBody = `{"privateLocation":{"id":"pl_new","name":"office-paris","token":"os_pl_live_9f3a2b1c8d4e6f","monitorIds":["123"],"metadata":{"env":"prod"},"createdAt":"2026-07-24T14:02:11Z","status":"PRIVATE_LOCATION_STATUS_UNSPECIFIED"}}`

// Not parallel: these subtests redirect the process-wide os.Stdout.
func Test_CreatePrivateLocation(t *testing.T) {
	t.Run("Prints the token in full", func(t *testing.T) {
		client := jsonResponder(map[string]string{createProcedure: createBody}, nil)

		out := captureStdout(t, func() {
			err := privatelocation.CreatePrivateLocationWithHTTPClient(context.Background(), client.GetHTTPClient(), "test-token", privatelocation.CreateParams{
				Name:       "office-paris",
				MonitorIDs: []string{"123"},
				Metadata:   map[string]string{"env": "prod"},
			})
			if err != nil {
				t.Errorf("Expected no error, got %v", err)
			}
		})

		// The token is only useful if it can be copied, so create never masks it.
		if !strings.Contains(out, "os_pl_live_9f3a2b1c8d4e6f") {
			t.Error("Expected the full token on create")
		}
		if !strings.Contains(out, "openstatus pl info pl_new --show-token") {
			t.Error("Expected the retrieval hint")
		}
	})

	t.Run("Sends the requested name and monitors", func(t *testing.T) {
		var calls []string
		client := jsonResponder(map[string]string{createProcedure: createBody}, &calls)

		out := captureStdout(t, func() {
			err := privatelocation.CreatePrivateLocationWithHTTPClient(context.Background(), client.GetHTTPClient(), "test-token", privatelocation.CreateParams{
				Name: "office-paris",
			})
			if err != nil {
				t.Errorf("Expected no error, got %v", err)
			}
		})
		_ = out

		if len(calls) != 1 || !strings.HasSuffix(calls[0], createProcedure) {
			t.Errorf("Expected exactly one CreatePrivateLocation call, got %v", calls)
		}
	})

	t.Run("Surfaces API failures", func(t *testing.T) {
		client := errorResponder(400, `{"code":"invalid_argument","message":"name is required"}`)

		err := privatelocation.CreatePrivateLocationWithHTTPClient(context.Background(), client.GetHTTPClient(), "test-token", privatelocation.CreateParams{Name: ""})
		if err == nil {
			t.Fatal("Expected an error for a rejected call")
		}
	})
}

func Test_CreateCmdFlags(t *testing.T) {
	t.Parallel()

	cmd := privatelocation.GetPrivateLocationCreateCmd()

	expected := map[string]bool{
		"access-token": false,
		"name":         false,
		"monitor-ids":  false,
		"metadata":     false,
	}
	for _, f := range cmd.Flags {
		for _, name := range f.Names() {
			if _, ok := expected[name]; ok {
				expected[name] = true
			}
		}
	}
	for name, found := range expected {
		if !found {
			t.Errorf("Expected flag '%s' not found", name)
		}
	}
}
