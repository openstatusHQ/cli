package statusreport_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"testing"

	status_reportv1 "buf.build/gen/go/openstatus/api/protocolbuffers/go/openstatus/status_report/v1"

	output "github.com/openstatusHQ/cli/internal/cli"
	"github.com/openstatusHQ/cli/internal/statusreport"
)

func respondWith(body string, captured *[]byte) *interceptorHTTPClient {
	return &interceptorHTTPClient{
		f: func(req *http.Request) (*http.Response, error) {
			if err := captureRequestBody(req, captured); err != nil {
				return nil, err
			}
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(bytes.NewReader([]byte(body))),
				Header:     http.Header{"Content-Type": []string{"application/json"}},
			}, nil
		},
	}
}

func captureJSONStdout(t *testing.T, f func()) []byte {
	t.Helper()

	output.SetJSONOutput(true)
	defer output.SetJSONOutput(false)

	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to create pipe: %v", err)
	}
	original := os.Stdout
	os.Stdout = w
	defer func() { os.Stdout = original }()

	done := make(chan []byte, 1)
	go func() {
		b, _ := io.ReadAll(r)
		done <- b
	}()

	f()
	w.Close()
	return <-done
}

func Test_CreateStatusReport_IncidentID(t *testing.T) {
	t.Parallel()

	body := `{"statusReport":{"id":"44","title":"DB Down","status":"STATUS_REPORT_STATUS_INVESTIGATING","incidentId":"42"}}`

	for _, incidentID := range []string{"", "42"} {
		var captured []byte
		_, err := statusreport.CreateStatusReportWithHTTPClient(
			context.Background(), respondWith(body, &captured).GetHTTPClient(), "test-token",
			statusreport.CreateStatusReportParams{
				Title:      "DB Down",
				Status:     "investigating",
				Message:    "Looking",
				Date:       "2026-03-20T10:00:00Z",
				PageID:     "page-1",
				IncidentID: incidentID,
			},
		)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		req := decodeBody(t, captured, &status_reportv1.CreateStatusReportRequest{})
		if req.HasIncidentId() != (incidentID != "") || req.GetIncidentId() != incidentID {
			t.Errorf("incidentID=%q: unexpected request %v", incidentID, req)
		}
	}
}

func Test_StatusReport_IncidentIDInJSON(t *testing.T) {
	t.Run("list", func(t *testing.T) {
		body := `{"statusReports":[{"id":"1","title":"Linked","status":"STATUS_REPORT_STATUS_INVESTIGATING","incidentId":"42"},{"id":"2","title":"Unlinked","status":"STATUS_REPORT_STATUS_RESOLVED"}],"totalSize":2}`
		var captured []byte
		out := captureJSONStdout(t, func() {
			if err := statusreport.ListStatusReportsWithHTTPClient(context.Background(), respondWith(body, &captured).GetHTTPClient(), "test-token", "", 0); err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
		var got []map[string]string
		if err := json.Unmarshal(out, &got); err != nil {
			t.Fatalf("invalid JSON: %v\n%s", err, out)
		}
		if len(got) != 2 || got[0]["incident_id"] != "42" {
			t.Fatalf("unexpected payload %v", got)
		}
		if _, ok := got[1]["incident_id"]; ok {
			t.Error("unlinked report should omit incident_id")
		}
	})

	t.Run("info", func(t *testing.T) {
		body := `{"statusReport":{"id":"1","title":"Linked","status":"STATUS_REPORT_STATUS_INVESTIGATING","incidentId":"42"}}`
		var captured []byte
		out := captureJSONStdout(t, func() {
			if err := statusreport.GetStatusReportInfoWithHTTPClient(context.Background(), respondWith(body, &captured).GetHTTPClient(), "test-token", "1"); err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
		var got struct {
			IncidentID string `json:"incident_id"`
		}
		if err := json.Unmarshal(out, &got); err != nil {
			t.Fatalf("invalid JSON: %v\n%s", err, out)
		}
		if got.IncidentID != "42" {
			t.Errorf("incident_id = %q", got.IncidentID)
		}
	})
}

func Test_CreateStatusReport_IncidentNotFound(t *testing.T) {
	t.Parallel()

	client := &interceptorHTTPClient{
		f: func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusNotFound,
				Body:       io.NopCloser(bytes.NewReader([]byte(`{"code":"not_found","message":"incident 999 not found"}`))),
				Header:     http.Header{"Content-Type": []string{"application/json"}},
			}, nil
		},
	}
	_, err := statusreport.CreateStatusReportWithHTTPClient(
		context.Background(), client.GetHTTPClient(), "test-token",
		statusreport.CreateStatusReportParams{
			Title:      "DB Down",
			Status:     "investigating",
			Message:    "Looking",
			Date:       "2026-03-20T10:00:00Z",
			PageID:     "page-1",
			IncidentID: "999",
		},
	)
	if err == nil || err.Error() != "incident 999 not found" {
		t.Errorf("unexpected error %v", err)
	}
}

func Test_RenderCreated_JSON(t *testing.T) {
	report := &status_reportv1.StatusReport{}
	report.SetId("44")
	report.SetTitle("DB Down")
	report.SetIncidentId("42")

	var err error
	out := captureJSONStdout(t, func() {
		err = statusreport.RenderCreated(report, "42")
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var got struct {
		ID         string `json:"id"`
		IncidentID string `json:"incident_id"`
	}
	if err := json.Unmarshal(out, &got); err != nil {
		t.Fatalf("output is not JSON: %v\n%s", err, out)
	}
	if got.ID != "44" || got.IncidentID != "42" {
		t.Errorf("unexpected output %+v", got)
	}
}

func Test_RenderCreated_Quiet(t *testing.T) {
	report := &status_reportv1.StatusReport{}
	report.SetId("44")

	output.SetQuietMode(true)
	defer output.SetQuietMode(false)

	r, w, _ := os.Pipe()
	original := os.Stdout
	os.Stdout = w
	err := statusreport.RenderCreated(report, "")
	w.Close()
	os.Stdout = original
	out, _ := io.ReadAll(r)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(out) != "44\n" {
		t.Errorf("unexpected output %q", out)
	}
}
