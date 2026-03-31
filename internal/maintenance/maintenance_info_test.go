package maintenance_test

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"testing"

	"github.com/openstatusHQ/cli/internal/maintenance"
)

func Test_GetMaintenanceInfo(t *testing.T) {
	t.Parallel()

	t.Run("Successfully gets maintenance info", func(t *testing.T) {
		body := `{"maintenance":{"id":"42","title":"DB Migration","message":"Upgrading to PG16","from":"2026-04-01T10:00:00Z","to":"2026-04-01T12:00:00Z","pageId":"p1","pageComponentIds":["c1"],"createdAt":"2026-03-20T10:00:00Z","updatedAt":"2026-03-20T10:00:00Z"}}`
		r := io.NopCloser(bytes.NewReader([]byte(body)))

		interceptor := &interceptorHTTPClient{
			f: func(req *http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       r,
					Header: http.Header{
						"Content-Type": []string{"application/json"},
					},
				}, nil
			},
		}

		err := maintenance.GetMaintenanceInfoWithHTTPClient(
			context.Background(), interceptor.GetHTTPClient(), "test-token", "42",
		)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
	})

	t.Run("Missing ID returns error", func(t *testing.T) {
		interceptor := &interceptorHTTPClient{
			f: func(req *http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: http.StatusOK,
					Header: http.Header{
						"Content-Type": []string{"application/json"},
					},
				}, nil
			},
		}

		err := maintenance.GetMaintenanceInfoWithHTTPClient(
			context.Background(), interceptor.GetHTTPClient(), "test-token", "",
		)
		if err == nil {
			t.Error("Expected error for missing ID, got nil")
		}
	})
}
