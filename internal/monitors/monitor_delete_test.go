package monitors_test

import (
	"bytes"
	"io"
	"net/http"
	"testing"

	"github.com/openstatusHQ/cli/internal/monitors"
)

func Test_DeleteMonitor(t *testing.T) {
	t.Parallel()

	t.Run("Monitor ID is required", func(t *testing.T) {
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

		err := monitors.DeleteMonitorWithHTTPClient(interceptor.GetHTTPClient(), "test-api-key", "")
		if err == nil {
			t.Error("Expected error for empty monitor ID, got nil")
		}
		if err.Error() != "Monitor ID is required" {
			t.Errorf("Expected 'Monitor ID is required' error, got %v", err)
		}
	})

	t.Run("Delete monitor successfully", func(t *testing.T) {
		body := `{"success": true}`
		r := io.NopCloser(bytes.NewReader([]byte(body)))

		interceptor := &interceptorHTTPClient{
			f: func(req *http.Request) (*http.Response, error) {
				if req.Method != http.MethodPost {
					t.Errorf("Expected POST method (Connect RPC), got %s", req.Method)
				}
				if req.Header.Get("x-openstatus-key") != "test-api-key" {
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

		err := monitors.DeleteMonitorWithHTTPClient(interceptor.GetHTTPClient(), "test-api-key", "123")
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
	})

	t.Run("Delete monitor fails with error status", func(t *testing.T) {
		body := `{"code":"not_found","message":"monitor not found"}`
		r := io.NopCloser(bytes.NewReader([]byte(body)))

		interceptor := &interceptorHTTPClient{
			f: func(req *http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: http.StatusNotFound,
					Body:       r,
					Header: http.Header{
						"Content-Type": []string{"application/json"},
					},
				}, nil
			},
		}

		err := monitors.DeleteMonitorWithHTTPClient(interceptor.GetHTTPClient(), "test-api-key", "999")
		if err == nil {
			t.Error("Expected error for non-200 status, got nil")
		}
	})
}
