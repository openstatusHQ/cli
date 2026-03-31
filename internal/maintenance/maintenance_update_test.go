package maintenance_test

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"testing"

	"github.com/openstatusHQ/cli/internal/maintenance"
)

func Test_UpdateMaintenance(t *testing.T) {
	t.Parallel()

	t.Run("Successfully updates maintenance", func(t *testing.T) {
		body := `{"maintenance":{"id":"42","title":"Updated","message":"msg","from":"2026-04-01T10:00:00Z","to":"2026-04-01T12:00:00Z","pageId":"p1","createdAt":"2026-03-20T10:00:00Z","updatedAt":"2026-03-20T11:00:00Z"}}`
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

		err := maintenance.UpdateMaintenanceWithHTTPClient(
			context.Background(), interceptor.GetHTTPClient(), "test-token",
			"42", "Updated", "", "", "", nil,
			true, false, false, false, false,
		)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
	})

	t.Run("Partial update with title only", func(t *testing.T) {
		body := `{"maintenance":{"id":"42","title":"New Title","message":"msg","from":"2026-04-01T10:00:00Z","to":"2026-04-01T12:00:00Z","pageId":"p1","createdAt":"2026-03-20T10:00:00Z","updatedAt":"2026-03-20T11:00:00Z"}}`
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

		err := maintenance.UpdateMaintenanceWithHTTPClient(
			context.Background(), interceptor.GetHTTPClient(), "test-token",
			"42", "New Title", "", "", "", nil,
			true, false, false, false, false,
		)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
	})

	t.Run("No flags provided returns error", func(t *testing.T) {
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

		err := maintenance.UpdateMaintenanceWithHTTPClient(
			context.Background(), interceptor.GetHTTPClient(), "test-token",
			"42", "", "", "", "", nil,
			false, false, false, false, false,
		)
		if err == nil {
			t.Error("Expected error for no flags, got nil")
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

		err := maintenance.UpdateMaintenanceWithHTTPClient(
			context.Background(), interceptor.GetHTTPClient(), "test-token",
			"", "Title", "", "", "", nil,
			true, false, false, false, false,
		)
		if err == nil {
			t.Error("Expected error for missing ID, got nil")
		}
	})
}
