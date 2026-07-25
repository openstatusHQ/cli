package privatelocation_test

import (
	"context"
	"testing"

	"github.com/openstatusHQ/cli/internal/privatelocation"
)

const listProcedure = "/ListPrivateLocations"

func Test_ListPrivateLocations(t *testing.T) {
	t.Parallel()

	t.Run("Successfully returns private locations", func(t *testing.T) {
		body := `{"privateLocations":[{"id":"pl_1","name":"office-paris","monitorCount":4,"lastSeenAt":"2026-07-24T14:02:11Z","createdAt":"2026-01-01T00:00:00Z","updatedAt":"2026-01-01T00:00:00Z","status":"PRIVATE_LOCATION_STATUS_ACTIVE"},{"id":"pl_2","name":"k8s-prod-eu","monitorCount":2,"lastSeenAt":"","createdAt":"2026-02-01T00:00:00Z","updatedAt":"2026-02-01T00:00:00Z","status":"PRIVATE_LOCATION_STATUS_ERROR"}],"totalSize":2}`
		client := jsonResponder(map[string]string{listProcedure: body}, nil)

		err := privatelocation.ListPrivateLocationsWithHTTPClient(context.Background(), client.GetHTTPClient(), "test-token", 0)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
	})

	t.Run("Returns empty list without error", func(t *testing.T) {
		body := `{"privateLocations":[],"totalSize":0}`
		client := jsonResponder(map[string]string{listProcedure: body}, nil)

		err := privatelocation.ListPrivateLocationsWithHTTPClient(context.Background(), client.GetHTTPClient(), "test-token", 0)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
	})

	t.Run("Renders when the page is truncated", func(t *testing.T) {
		body := `{"privateLocations":[{"id":"pl_1","name":"office-paris","monitorCount":1,"status":"PRIVATE_LOCATION_STATUS_ACTIVE"}],"totalSize":63}`
		client := jsonResponder(map[string]string{listProcedure: body}, nil)

		err := privatelocation.ListPrivateLocationsWithHTTPClient(context.Background(), client.GetHTTPClient(), "test-token", 1)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
	})

	t.Run("Surfaces API failures", func(t *testing.T) {
		client := errorResponder(403, `{"code":"permission_denied","message":"no access"}`)

		err := privatelocation.ListPrivateLocationsWithHTTPClient(context.Background(), client.GetHTTPClient(), "test-token", 0)
		if err == nil {
			t.Fatal("Expected an error for a rejected call")
		}
	})
}
