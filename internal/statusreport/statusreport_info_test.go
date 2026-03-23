package statusreport_test

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"testing"

	"github.com/openstatusHQ/cli/internal/statusreport"
)

func Test_GetStatusReportInfo(t *testing.T) {
	t.Parallel()

	t.Run("Successfully returns report with updates", func(t *testing.T) {
		body := `{"statusReport":{"id":"1","title":"API Outage","status":"STATUS_REPORT_STATUS_MONITORING","pageComponentIds":["c1","c2"],"updates":[{"id":"u1","status":"STATUS_REPORT_STATUS_INVESTIGATING","date":"2026-03-20T10:00:00Z","message":"Investigating the issue"},{"id":"u2","status":"STATUS_REPORT_STATUS_IDENTIFIED","date":"2026-03-20T10:30:00Z","message":"Root cause identified"}],"createdAt":"2026-03-20T10:00:00Z","updatedAt":"2026-03-20T11:00:00Z"}}`
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

		err := statusreport.GetStatusReportInfoWithHTTPClient(context.Background(), interceptor.GetHTTPClient(),"test-token", "1")
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
	})

	t.Run("Report with no updates", func(t *testing.T) {
		body := `{"statusReport":{"id":"2","title":"Scheduled Maintenance","status":"STATUS_REPORT_STATUS_INVESTIGATING","updates":[],"createdAt":"2026-03-20T10:00:00Z","updatedAt":"2026-03-20T10:00:00Z"}}`
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

		err := statusreport.GetStatusReportInfoWithHTTPClient(context.Background(), interceptor.GetHTTPClient(),"test-token", "2")
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
	})

	t.Run("Missing report ID returns error", func(t *testing.T) {
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

		err := statusreport.GetStatusReportInfoWithHTTPClient(context.Background(), interceptor.GetHTTPClient(),"test-token", "")
		if err == nil {
			t.Error("Expected error for empty report ID, got nil")
		}
		if err.Error() != "report ID is required" {
			t.Errorf("Expected 'report ID is required' error, got %v", err)
		}
	})

	t.Run("Report not found returns actionable error", func(t *testing.T) {
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

		err := statusreport.GetStatusReportInfoWithHTTPClient(context.Background(), interceptor.GetHTTPClient(),"test-token", "999")
		if err == nil {
			t.Error("Expected error, got nil")
		}
	})
}
