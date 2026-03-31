package maintenance_test

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"testing"

	"github.com/openstatusHQ/cli/internal/maintenance"
)

func Test_DeleteMaintenance(t *testing.T) {
	t.Parallel()

	t.Run("Successfully deletes maintenance", func(t *testing.T) {
		body := `{}`
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

		err := maintenance.DeleteMaintenanceWithHTTPClient(
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

		err := maintenance.DeleteMaintenanceWithHTTPClient(
			context.Background(), interceptor.GetHTTPClient(), "test-token", "",
		)
		if err == nil {
			t.Error("Expected error for missing ID, got nil")
		}
	})

	t.Run("API error returns error", func(t *testing.T) {
		body := `{"code":"internal","message":"internal error"}`
		r := io.NopCloser(bytes.NewReader([]byte(body)))

		interceptor := &interceptorHTTPClient{
			f: func(req *http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: http.StatusInternalServerError,
					Body:       r,
					Header: http.Header{
						"Content-Type": []string{"application/json"},
					},
				}, nil
			},
		}

		err := maintenance.DeleteMaintenanceWithHTTPClient(
			context.Background(), interceptor.GetHTTPClient(), "test-token", "42",
		)
		if err == nil {
			t.Error("Expected error, got nil")
		}
	})
}
