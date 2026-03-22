package statusreport_test

import (
	"bytes"
	"io"
	"net/http"
	"testing"

	"github.com/openstatusHQ/cli/internal/statusreport"
)

func Test_CreateStatusReport(t *testing.T) {
	t.Parallel()

	t.Run("Successfully creates report", func(t *testing.T) {
		body := `{"statusReport":{"id":"42","title":"API Outage","status":"STATUS_REPORT_STATUS_INVESTIGATING","createdAt":"2026-03-20T10:00:00Z","updatedAt":"2026-03-20T10:00:00Z"}}`
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

		id, err := statusreport.CreateStatusReportWithHTTPClient(
			interceptor.GetHTTPClient(), "test-token",
			"API Outage", "investigating", "Investigating the issue",
			"2026-03-20T10:00:00Z", "page-1", nil, false,
		)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if id != "42" {
			t.Errorf("Expected ID '42', got %s", id)
		}
	})

	t.Run("Creates report with optional fields", func(t *testing.T) {
		body := `{"statusReport":{"id":"43","title":"DB Issue","status":"STATUS_REPORT_STATUS_INVESTIGATING","pageComponentIds":["c1","c2"],"createdAt":"2026-03-20T10:00:00Z","updatedAt":"2026-03-20T10:00:00Z"}}`
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

		id, err := statusreport.CreateStatusReportWithHTTPClient(
			interceptor.GetHTTPClient(), "test-token",
			"DB Issue", "investigating", "Looking into DB issues",
			"2026-03-20T10:00:00Z", "page-1", []string{"c1", "c2"}, true,
		)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if id != "43" {
			t.Errorf("Expected ID '43', got %s", id)
		}
	})

	t.Run("Invalid status returns error", func(t *testing.T) {
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

		_, err := statusreport.CreateStatusReportWithHTTPClient(
			interceptor.GetHTTPClient(), "test-token",
			"Title", "invalid-status", "Message",
			"2026-03-20T10:00:00Z", "page-1", nil, false,
		)
		if err == nil {
			t.Error("Expected error for invalid status, got nil")
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

		_, err := statusreport.CreateStatusReportWithHTTPClient(
			interceptor.GetHTTPClient(), "test-token",
			"Title", "investigating", "Message",
			"2026-03-20T10:00:00Z", "page-1", nil, false,
		)
		if err == nil {
			t.Error("Expected error, got nil")
		}
	})
}
