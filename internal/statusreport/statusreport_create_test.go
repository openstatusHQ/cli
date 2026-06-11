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
			context.Background(), interceptor.GetHTTPClient(), "test-token",
			statusreport.CreateStatusReportParams{
				Title:   "API Outage",
				Status:  "investigating",
				Message: "Investigating the issue",
				Date:    "2026-03-20T10:00:00Z",
				PageID:  "page-1",
			},
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
			context.Background(), interceptor.GetHTTPClient(), "test-token",
			statusreport.CreateStatusReportParams{
				Title:        "DB Issue",
				Status:       "investigating",
				Message:      "Looking into DB issues",
				Date:         "2026-03-20T10:00:00Z",
				PageID:       "page-1",
				ComponentIDs: []string{"c1", "c2"},
				Notify:       true,
			},
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
			context.Background(), interceptor.GetHTTPClient(), "test-token",
			statusreport.CreateStatusReportParams{
				Title:   "Title",
				Status:  "invalid-status",
				Message: "Message",
				Date:    "2026-03-20T10:00:00Z",
				PageID:  "page-1",
			},
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
			context.Background(), interceptor.GetHTTPClient(), "test-token",
			statusreport.CreateStatusReportParams{
				Title:   "Title",
				Status:  "investigating",
				Message: "Message",
				Date:    "2026-03-20T10:00:00Z",
				PageID:  "page-1",
			},
		)
		if err == nil {
			t.Error("Expected error, got nil")
		}
	})

	t.Run("Creates report with component impacts sends correct request body", func(t *testing.T) {
		body := `{"statusReport":{"id":"44","title":"DB Down","status":"STATUS_REPORT_STATUS_INVESTIGATING","createdAt":"2026-03-20T10:00:00Z","updatedAt":"2026-03-20T10:00:00Z"}}`
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

		impactAPI := &status_reportv1.ComponentImpact{}
		impactAPI.SetPageComponentId("comp_api")
		impactAPI.SetImpact(status_reportv1.PageComponentImpact_PAGE_COMPONENT_IMPACT_MAJOR_OUTAGE)
		impactDB := &status_reportv1.ComponentImpact{}
		impactDB.SetPageComponentId("comp_db")
		impactDB.SetImpact(status_reportv1.PageComponentImpact_PAGE_COMPONENT_IMPACT_DEGRADED_PERFORMANCE)

		_, err := statusreport.CreateStatusReportWithHTTPClient(
			context.Background(), interceptor.GetHTTPClient(), "test-token",
			statusreport.CreateStatusReportParams{
				Title:            "DB Down",
				Status:           "investigating",
				Message:          "Looking",
				Date:             "2026-03-20T10:00:00Z",
				PageID:           "page-1",
				ComponentImpacts: []*status_reportv1.ComponentImpact{impactAPI, impactDB},
			},
		)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		req := decodeBody(t, capturedBody, &status_reportv1.CreateStatusReportRequest{})
		impacts := req.GetComponentImpacts()
		if len(impacts) != 2 {
			t.Fatalf("expected 2 component impacts, got %d", len(impacts))
		}
		if impacts[0].GetPageComponentId() != "comp_api" {
			t.Errorf("expected comp_api first, got %s", impacts[0].GetPageComponentId())
		}
		if impacts[0].GetImpact() != status_reportv1.PageComponentImpact_PAGE_COMPONENT_IMPACT_MAJOR_OUTAGE {
			t.Errorf("expected major_outage for comp_api, got %v", impacts[0].GetImpact())
		}
		if impacts[1].GetPageComponentId() != "comp_db" {
			t.Errorf("expected comp_db second, got %s", impacts[1].GetPageComponentId())
		}
		if impacts[1].GetImpact() != status_reportv1.PageComponentImpact_PAGE_COMPONENT_IMPACT_DEGRADED_PERFORMANCE {
			t.Errorf("expected degraded for comp_db, got %v", impacts[1].GetImpact())
		}
	})

	t.Run("Creates report without impacts omits componentImpacts", func(t *testing.T) {
		body := `{"statusReport":{"id":"45","title":"X","status":"STATUS_REPORT_STATUS_INVESTIGATING","createdAt":"2026-03-20T10:00:00Z","updatedAt":"2026-03-20T10:00:00Z"}}`
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

		_, err := statusreport.CreateStatusReportWithHTTPClient(
			context.Background(), interceptor.GetHTTPClient(), "test-token",
			statusreport.CreateStatusReportParams{
				Title:   "X",
				Status:  "investigating",
				Message: "M",
				Date:    "2026-03-20T10:00:00Z",
				PageID:  "page-1",
			},
		)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		req := decodeBody(t, capturedBody, &status_reportv1.CreateStatusReportRequest{})
		if len(req.GetComponentImpacts()) != 0 {
			t.Errorf("expected no componentImpacts, got %v", req.GetComponentImpacts())
		}
	})
}
