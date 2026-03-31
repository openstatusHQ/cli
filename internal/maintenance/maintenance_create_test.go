package maintenance_test

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"testing"

	"github.com/openstatusHQ/cli/internal/maintenance"
)

func Test_CreateMaintenance(t *testing.T) {
	t.Parallel()

	t.Run("Successfully creates maintenance", func(t *testing.T) {
		body := `{"maintenance":{"id":"42","title":"DB Migration","message":"Upgrading","from":"2026-04-01T10:00:00Z","to":"2026-04-01T12:00:00Z","pageId":"p1","createdAt":"2026-03-20T10:00:00Z","updatedAt":"2026-03-20T10:00:00Z"}}`
		r := io.NopCloser(bytes.NewReader([]byte(body)))

		interceptor := &interceptorHTTPClient{
			f: func(req *http.Request) (*http.Response, error) {
				if req.Header.Get("x-openstatus-key") != "test-token" {
					t.Errorf("Expected x-openstatus-key header, got %s", req.Header.Get("x-openstatus-key"))
				}
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       r,
					Header: http.Header{
						"Content-Type": []string{"application/json"},
					},
				}, nil
			},
		}

		id, err := maintenance.CreateMaintenanceWithHTTPClient(
			context.Background(), interceptor.GetHTTPClient(), "test-token",
			"DB Migration", "Upgrading", "2026-04-01T10:00:00Z", "2026-04-01T12:00:00Z",
			"p1", nil, false,
		)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if id != "42" {
			t.Errorf("Expected ID '42', got %s", id)
		}
	})

	t.Run("Creates maintenance with optional fields", func(t *testing.T) {
		body := `{"maintenance":{"id":"43","title":"DB Migration","message":"Upgrading","from":"2026-04-01T10:00:00Z","to":"2026-04-01T12:00:00Z","pageId":"p1","pageComponentIds":["c1","c2"],"createdAt":"2026-03-20T10:00:00Z","updatedAt":"2026-03-20T10:00:00Z"}}`
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

		id, err := maintenance.CreateMaintenanceWithHTTPClient(
			context.Background(), interceptor.GetHTTPClient(), "test-token",
			"DB Migration", "Upgrading", "2026-04-01T10:00:00Z", "2026-04-01T12:00:00Z",
			"p1", []string{"c1", "c2"}, true,
		)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if id != "43" {
			t.Errorf("Expected ID '43', got %s", id)
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

		_, err := maintenance.CreateMaintenanceWithHTTPClient(
			context.Background(), interceptor.GetHTTPClient(), "test-token",
			"Title", "Message", "2026-04-01T10:00:00Z", "2026-04-01T12:00:00Z",
			"p1", nil, false,
		)
		if err == nil {
			t.Error("Expected error, got nil")
		}
	})
}
