package statusreport_test

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"testing"

	"github.com/openstatusHQ/cli/internal/statusreport"
)

func Test_UpdateStatusReport(t *testing.T) {
	t.Parallel()

	t.Run("Update title only", func(t *testing.T) {
		body := `{"statusReport":{"id":"1","title":"New Title","status":"STATUS_REPORT_STATUS_INVESTIGATING","createdAt":"2026-03-20T10:00:00Z","updatedAt":"2026-03-20T11:00:00Z"}}`
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

		err := statusreport.UpdateStatusReportWithHTTPClient(
			context.Background(), interceptor.GetHTTPClient(),"test-token",
			"1", "New Title", nil, true, false,
		)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
	})

	t.Run("Update component-ids only", func(t *testing.T) {
		body := `{"statusReport":{"id":"1","title":"Title","status":"STATUS_REPORT_STATUS_INVESTIGATING","pageComponentIds":["c1","c2"],"createdAt":"2026-03-20T10:00:00Z","updatedAt":"2026-03-20T11:00:00Z"}}`
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

		err := statusreport.UpdateStatusReportWithHTTPClient(
			context.Background(), interceptor.GetHTTPClient(),"test-token",
			"1", "", []string{"c1", "c2"}, false, true,
		)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
	})

	t.Run("Update both title and component-ids", func(t *testing.T) {
		body := `{"statusReport":{"id":"1","title":"Updated Title","status":"STATUS_REPORT_STATUS_INVESTIGATING","pageComponentIds":["c3"],"createdAt":"2026-03-20T10:00:00Z","updatedAt":"2026-03-20T11:00:00Z"}}`
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

		err := statusreport.UpdateStatusReportWithHTTPClient(
			context.Background(), interceptor.GetHTTPClient(),"test-token",
			"1", "Updated Title", []string{"c3"}, true, true,
		)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
	})

	t.Run("Neither flag provided returns error", func(t *testing.T) {
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

		err := statusreport.UpdateStatusReportWithHTTPClient(
			context.Background(), interceptor.GetHTTPClient(),"test-token",
			"1", "", nil, false, false,
		)
		if err == nil {
			t.Error("Expected error, got nil")
		}
		if err.Error() != "at least one of --title or --component-ids must be provided" {
			t.Errorf("Expected validation error, got %v", err)
		}
	})

	t.Run("Empty report ID returns error", func(t *testing.T) {
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

		err := statusreport.UpdateStatusReportWithHTTPClient(
			context.Background(), interceptor.GetHTTPClient(),"test-token",
			"", "Title", nil, true, false,
		)
		if err == nil {
			t.Error("Expected error for empty report ID, got nil")
		}
	})
}
