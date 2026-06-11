package statusreport_test

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"testing"

	status_reportv1 "buf.build/gen/go/openstatus/api/protocolbuffers/go/openstatus/status_report/v1"

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

	t.Run("Adds update with impacts sends correct request body", func(t *testing.T) {
		body := `{"statusReport":{"id":"5","title":"X","status":"STATUS_REPORT_STATUS_MONITORING","createdAt":"2026-03-20T10:00:00Z","updatedAt":"2026-03-20T11:00:00Z"}}`
		r := io.NopCloser(bytes.NewReader([]byte(body)))

		var capturedBody []byte
		interceptor := &interceptorHTTPClient{
			f: func(req *http.Request) (*http.Response, error) {
				if err := captureRequestBody(req, &capturedBody); err != nil {
					t.Fatalf("capture body: %v", err)
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

		impact := &status_reportv1.ComponentImpact{}
		impact.SetPageComponentId("comp_api")
		impact.SetImpact(status_reportv1.PageComponentImpact_PAGE_COMPONENT_IMPACT_OPERATIONAL)

		err := statusreport.AddStatusReportUpdateWithHTTPClient(
			context.Background(), interceptor.GetHTTPClient(), "test-token",
			statusreport.AddStatusReportUpdateParams{
				ReportID:         "5",
				Status:           "monitoring",
				Message:          "Recovering",
				ComponentImpacts: []*status_reportv1.ComponentImpact{impact},
			},
		)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		req := decodeBody(t, capturedBody, &status_reportv1.AddStatusReportUpdateRequest{})
		impacts := req.GetComponentImpacts()
		if len(impacts) != 1 {
			t.Fatalf("expected 1 component impact, got %d", len(impacts))
		}
		if impacts[0].GetPageComponentId() != "comp_api" {
			t.Errorf("expected comp_api, got %s", impacts[0].GetPageComponentId())
		}
		if impacts[0].GetImpact() != status_reportv1.PageComponentImpact_PAGE_COMPONENT_IMPACT_OPERATIONAL {
			t.Errorf("expected operational, got %v", impacts[0].GetImpact())
		}
	})

	t.Run("Update without impacts omits componentImpacts", func(t *testing.T) {
		body := `{"statusReport":{"id":"6","title":"X","status":"STATUS_REPORT_STATUS_IDENTIFIED","createdAt":"2026-03-20T10:00:00Z","updatedAt":"2026-03-20T11:00:00Z"}}`
		r := io.NopCloser(bytes.NewReader([]byte(body)))

		var capturedBody []byte
		interceptor := &interceptorHTTPClient{
			f: func(req *http.Request) (*http.Response, error) {
				if err := captureRequestBody(req, &capturedBody); err != nil {
					t.Fatalf("capture body: %v", err)
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

		err := statusreport.AddStatusReportUpdateWithHTTPClient(
			context.Background(), interceptor.GetHTTPClient(), "test-token",
			statusreport.AddStatusReportUpdateParams{
				ReportID: "6",
				Status:   "identified",
				Message:  "Root cause",
			},
		)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		req := decodeBody(t, capturedBody, &status_reportv1.AddStatusReportUpdateRequest{})
		if len(req.GetComponentImpacts()) != 0 {
			t.Errorf("expected no componentImpacts, got %v", req.GetComponentImpacts())
		}
	})
}
