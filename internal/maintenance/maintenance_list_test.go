package maintenance_test

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"testing"

	"github.com/openstatusHQ/cli/internal/maintenance"
)

func Test_ListMaintenances(t *testing.T) {
	t.Parallel()

	t.Run("Successfully lists maintenances", func(t *testing.T) {
		body := `{"maintenances":[{"id":"1","title":"M1","message":"msg1","from":"2026-04-01T10:00:00Z","to":"2026-04-01T12:00:00Z","pageId":"p1","createdAt":"2026-03-20T10:00:00Z","updatedAt":"2026-03-20T10:00:00Z"},{"id":"2","title":"M2","message":"msg2","from":"2026-04-02T10:00:00Z","to":"2026-04-02T12:00:00Z","pageId":"p1","createdAt":"2026-03-21T10:00:00Z","updatedAt":"2026-03-21T10:00:00Z"}],"totalSize":2}`
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

		err := maintenance.ListMaintenancesWithHTTPClient(
			context.Background(), interceptor.GetHTTPClient(), "test-token", "", 0,
		)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
	})

	t.Run("Empty list", func(t *testing.T) {
		body := `{"maintenances":[],"totalSize":0}`
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

		err := maintenance.ListMaintenancesWithHTTPClient(
			context.Background(), interceptor.GetHTTPClient(), "test-token", "", 0,
		)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
	})

	t.Run("With page_id filter", func(t *testing.T) {
		body := `{"maintenances":[],"totalSize":0}`
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

		err := maintenance.ListMaintenancesWithHTTPClient(
			context.Background(), interceptor.GetHTTPClient(), "test-token", "page-123", 10,
		)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
	})
}
