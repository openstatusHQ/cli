package statusreport_test

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"testing"

	"github.com/openstatusHQ/cli/internal/statusreport"
)

func Test_AddStatusReportUpdate(t *testing.T) {
	t.Parallel()

	t.Run("Successfully adds update", func(t *testing.T) {
		body := `{"statusReport":{"id":"1","title":"API Outage","status":"STATUS_REPORT_STATUS_IDENTIFIED","updates":[{"id":"u1","status":"STATUS_REPORT_STATUS_INVESTIGATING","date":"2026-03-20T10:00:00Z","message":"Investigating"},{"id":"u2","status":"STATUS_REPORT_STATUS_IDENTIFIED","date":"2026-03-20T10:30:00Z","message":"Root cause found"}],"createdAt":"2026-03-20T10:00:00Z","updatedAt":"2026-03-20T10:30:00Z"}}`
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

		err := statusreport.AddStatusReportUpdateWithHTTPClient(
			context.Background(), interceptor.GetHTTPClient(), "test-token",
			statusreport.AddStatusReportUpdateParams{
				ReportID: "1",
				Status:   "identified",
				Message:  "Root cause found",
				Date:     "2026-03-20T10:30:00Z",
			},
		)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
	})

	t.Run("Adds update with notify", func(t *testing.T) {
		body := `{"statusReport":{"id":"1","title":"API Outage","status":"STATUS_REPORT_STATUS_RESOLVED","createdAt":"2026-03-20T10:00:00Z","updatedAt":"2026-03-20T12:00:00Z"}}`
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

		err := statusreport.AddStatusReportUpdateWithHTTPClient(
			context.Background(), interceptor.GetHTTPClient(), "test-token",
			statusreport.AddStatusReportUpdateParams{
				ReportID: "1",
				Status:   "resolved",
				Message:  "Issue resolved",
				Notify:   true,
			},
		)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
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

		err := statusreport.AddStatusReportUpdateWithHTTPClient(
			context.Background(), interceptor.GetHTTPClient(), "test-token",
			statusreport.AddStatusReportUpdateParams{
				ReportID: "1",
				Status:   "invalid",
				Message:  "Message",
			},
		)
		if err == nil {
			t.Error("Expected error for invalid status, got nil")
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

		err := statusreport.AddStatusReportUpdateWithHTTPClient(
			context.Background(), interceptor.GetHTTPClient(), "test-token",
			statusreport.AddStatusReportUpdateParams{
				Status:  "investigating",
				Message: "Message",
			},
		)
		if err == nil {
			t.Error("Expected error for empty report ID, got nil")
		}
	})

	t.Run("API error returns error", func(t *testing.T) {
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

		err := statusreport.AddStatusReportUpdateWithHTTPClient(
			context.Background(), interceptor.GetHTTPClient(), "test-token",
			statusreport.AddStatusReportUpdateParams{
				ReportID: "999",
				Status:   "investigating",
				Message:  "Message",
			},
		)
		if err == nil {
			t.Error("Expected error, got nil")
		}
	})
}
