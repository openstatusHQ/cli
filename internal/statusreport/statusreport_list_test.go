package statusreport_test

import (
	"bytes"
	"io"
	"log"
	"net/http"
	"os"
	"testing"

	"github.com/openstatusHQ/cli/internal/statusreport"
)

func Test_ListStatusReports(t *testing.T) {
	t.Parallel()

	t.Run("Successfully returns reports", func(t *testing.T) {
		body := `{"statusReports":[{"id":"1","title":"API Outage","status":"STATUS_REPORT_STATUS_INVESTIGATING","createdAt":"2026-03-20T10:00:00Z","updatedAt":"2026-03-20T10:30:00Z"},{"id":"2","title":"Database Maintenance","status":"STATUS_REPORT_STATUS_RESOLVED","createdAt":"2026-03-19T08:00:00Z","updatedAt":"2026-03-19T12:00:00Z"}],"totalSize":2}`
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

		var bf bytes.Buffer
		log.SetOutput(&bf)
		t.Cleanup(func() {
			log.SetOutput(os.Stdout)
		})
		err := statusreport.ListStatusReportsWithHTTPClient(interceptor.GetHTTPClient(), "test-token", "", 0)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
	})

	t.Run("Returns empty list", func(t *testing.T) {
		body := `{"statusReports":[],"totalSize":0}`
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

		err := statusreport.ListStatusReportsWithHTTPClient(interceptor.GetHTTPClient(), "test-token", "", 0)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
	})

	t.Run("Filters by status", func(t *testing.T) {
		body := `{"statusReports":[{"id":"1","title":"API Outage","status":"STATUS_REPORT_STATUS_INVESTIGATING","createdAt":"2026-03-20T10:00:00Z","updatedAt":"2026-03-20T10:30:00Z"}],"totalSize":1}`
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

		err := statusreport.ListStatusReportsWithHTTPClient(interceptor.GetHTTPClient(), "test-token", "investigating", 0)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
	})

	t.Run("Invalid status filter returns error", func(t *testing.T) {
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

		err := statusreport.ListStatusReportsWithHTTPClient(interceptor.GetHTTPClient(), "test-token", "invalid", 0)
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

		err := statusreport.ListStatusReportsWithHTTPClient(interceptor.GetHTTPClient(), "test-token", "", 0)
		if err == nil {
			t.Error("Expected error, got nil")
		}
	})
}
