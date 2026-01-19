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
				}, nil
			},
		}

		err := monitors.DeleteMonitor(interceptor.GetHTTPClient(), "test-api-key", "")
		if err == nil {
			t.Error("Expected error for empty monitor ID, got nil")
		}
		if err.Error() != "Monitor ID is required" {
			t.Errorf("Expected 'Monitor ID is required' error, got %v", err)
		}
	})

	t.Run("Delete monitor successfully", func(t *testing.T) {
		body := `{"resultId": 123}`
		r := io.NopCloser(bytes.NewReader([]byte(body)))

		interceptor := &interceptorHTTPClient{
			f: func(req *http.Request) (*http.Response, error) {
				if req.Method != http.MethodDelete {
					t.Errorf("Expected DELETE method, got %s", req.Method)
				}
				if req.Header.Get("x-openstatus-key") != "test-api-key" {
					t.Errorf("Expected x-openstatus-key header, got %s", req.Header.Get("x-openstatus-key"))
				}
				expectedURL := "https://api.openstatus.dev/v1/monitor/123"
				if req.URL.String() != expectedURL {
					t.Errorf("Expected URL %s, got %s", expectedURL, req.URL.String())
				}
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       r,
				}, nil
			},
		}

		err := monitors.DeleteMonitor(interceptor.GetHTTPClient(), "test-api-key", "123")
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
	})

	t.Run("Delete monitor fails with non-200 status", func(t *testing.T) {
		interceptor := &interceptorHTTPClient{
			f: func(req *http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: http.StatusNotFound,
					Body:       io.NopCloser(bytes.NewReader([]byte(`{"error": "not found"}`))),
				}, nil
			},
		}

		err := monitors.DeleteMonitor(interceptor.GetHTTPClient(), "test-api-key", "999")
		if err == nil {
			t.Error("Expected error for non-200 status, got nil")
		}
	})

	t.Run("Delete monitor with valid response body", func(t *testing.T) {
		body := `{"resultId": 456}`
		r := io.NopCloser(bytes.NewReader([]byte(body)))

		interceptor := &interceptorHTTPClient{
			f: func(req *http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       r,
				}, nil
			},
		}

		err := monitors.DeleteMonitor(interceptor.GetHTTPClient(), "test-api-key", "456")
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
	})
}
