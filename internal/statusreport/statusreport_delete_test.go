package statusreport_test

import (
	"bytes"
	"io"
	"net/http"
	"testing"

	"github.com/openstatusHQ/cli/internal/statusreport"
)

func Test_DeleteStatusReport(t *testing.T) {
	t.Parallel()

	t.Run("Report ID is required", func(t *testing.T) {
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

		err := statusreport.DeleteStatusReportWithHTTPClient(interceptor.GetHTTPClient(), "test-token", "")
		if err == nil {
			t.Error("Expected error for empty report ID, got nil")
		}
		if err.Error() != "report ID is required" {
			t.Errorf("Expected 'report ID is required' error, got %v", err)
		}
	})

	t.Run("Delete report successfully", func(t *testing.T) {
		body := `{"success":true}`
		r := io.NopCloser(bytes.NewReader([]byte(body)))

		interceptor := &interceptorHTTPClient{
			f: func(req *http.Request) (*http.Response, error) {
				if req.Method != http.MethodPost {
					t.Errorf("Expected POST method (Connect RPC), got %s", req.Method)
				}
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

		err := statusreport.DeleteStatusReportWithHTTPClient(interceptor.GetHTTPClient(), "test-token", "123")
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
	})

	t.Run("Delete report fails with error status", func(t *testing.T) {
		body := `{"code":"not_found","message":"status report not found"}`
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

		err := statusreport.DeleteStatusReportWithHTTPClient(interceptor.GetHTTPClient(), "test-token", "999")
		if err == nil {
			t.Error("Expected error for not found, got nil")
		}
	})
}
