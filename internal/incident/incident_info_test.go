package incident_test

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	incidentv1 "buf.build/gen/go/openstatus/api/protocolbuffers/go/openstatus/incident/v1"

	"github.com/openstatusHQ/cli/internal/incident"
)

const fullIncidentResponse = `{"incident": {
	"id": "42",
	"title": "Checkout 502",
	"severity": "INCIDENT_SEVERITY_CRITICAL",
	"status": "INCIDENT_STATUS_RESOLVED",
	"summary": "Payments failing",
	"commander": {"email": "jane@example.com", "name": "Jane"},
	"declaredBy": {"email": "bob@example.com", "name": "Bob"},
	"resolvedBy": {"email": "jane@example.com", "name": "Jane"},
	"declaredAt": "2026-10-16T10:00:00Z",
	"startedAt": "2026-10-16T09:55:00Z",
	"mitigatedAt": "2026-10-16T10:30:00Z",
	"resolvedAt": "2026-10-16T11:00:00Z",
	"statusReport": {"id": "123", "title": "Checkout issues", "status": "STATUS_REPORT_STATUS_MONITORING", "pageId": "7"},
	"slackChannelUrl": "https://slack.com/app_redirect?channel=C1",
	"allowedTransitions": ["INCIDENT_STATUS_OPEN"],
	"deletable": false,
	"events": [
		{"id": "e3", "type": "INCIDENT_EVENT_TYPE_NOTE", "message": "Root cause found\nBad deploy", "createdBy": {"email": "jane@example.com", "name": "Jane"}, "createdAt": "2026-10-16T11:05:00Z"},
		{"id": "e2", "type": "INCIDENT_EVENT_TYPE_RESOLVED", "message": "Resolved", "createdAt": "2026-10-16T11:00:00Z"},
		{"id": "e1", "type": "INCIDENT_EVENT_TYPE_DECLARED", "message": "Declared", "createdAt": "2026-10-16T10:00:00Z"}
	],
	"createdAt": "2026-10-16T10:00:00Z",
	"updatedAt": "2026-10-16T11:05:00Z"
}}`

func Test_GetIncidentInfo(t *testing.T) {
	t.Run("renders details and timeline", func(t *testing.T) {
		srv := newServer(http.StatusOK, fullIncidentResponse)
		var err error
		out := captureStdout(t, func() {
			err = incident.GetIncidentInfoWithHTTPClient(context.Background(), srv.client(), "token", "42")
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		req := decodeBody(t, srv.reqBody, &incidentv1.GetIncidentRequest{})
		if req.GetId() != "42" {
			t.Errorf("id = %s", req.GetId())
		}
		for _, want := range []string{
			"Checkout 502", "critical", "Payments failing", "Jane <jane@example.com>", "Bob <bob@example.com>",
			"2026-10-16 11:00 UTC by Jane", `123 "Checkout issues" [monitoring]`, "slack.com",
			"Allowed transitions", "open", "Timeline:", "note  Root cause found  — Jane", "    Bad deploy", "declared  Declared",
		} {
			if !strings.Contains(out, want) {
				t.Errorf("output missing %q:\n%s", want, out)
			}
		}
		if strings.Index(out, "Root cause found") > strings.Index(out, "declared  Declared") {
			t.Error("expected newest event first")
		}
	})

	t.Run("json includes events", func(t *testing.T) {
		withOutputModes(t, true, false)
		srv := newServer(http.StatusOK, fullIncidentResponse)
		out := captureStdout(t, func() {
			if err := incident.GetIncidentInfoWithHTTPClient(context.Background(), srv.client(), "token", "42"); err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
		var got struct {
			ID                 string   `json:"id"`
			Severity           string   `json:"severity"`
			AllowedTransitions []string `json:"allowed_transitions"`
			Deletable          bool     `json:"deletable"`
			StatusReport       struct {
				PageID string `json:"page_id"`
			} `json:"status_report"`
			Events []struct {
				Type      string `json:"type"`
				CreatedBy *struct {
					Email string `json:"email"`
				} `json:"created_by"`
			} `json:"events"`
		}
		if err := json.Unmarshal([]byte(out), &got); err != nil {
			t.Fatalf("invalid JSON: %v\n%s", err, out)
		}
		if got.ID != "42" || got.Severity != "critical" || got.Deletable || got.StatusReport.PageID != "7" {
			t.Errorf("unexpected payload %+v", got)
		}
		if len(got.AllowedTransitions) != 1 || got.AllowedTransitions[0] != "open" {
			t.Errorf("allowed transitions = %v", got.AllowedTransitions)
		}
		if len(got.Events) != 3 || got.Events[0].Type != "note" || got.Events[0].CreatedBy == nil || got.Events[1].CreatedBy != nil {
			t.Errorf("unexpected events %+v", got.Events)
		}
	})

	t.Run("minimal incident", func(t *testing.T) {
		srv := newServer(http.StatusOK, `{"incident":{"id":"43","title":"Login","severity":"INCIDENT_SEVERITY_MINOR","status":"INCIDENT_STATUS_OPEN","declaredAt":"2026-10-16T10:00:00Z","startedAt":"2026-10-16T10:00:00Z","deletable":true}}`)
		out := captureStdout(t, func() {
			if err := incident.GetIncidentInfoWithHTTPClient(context.Background(), srv.client(), "token", "43"); err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
		for _, want := range []string{"unassigned", "No timeline events", "yes"} {
			if !strings.Contains(out, want) {
				t.Errorf("output missing %q:\n%s", want, out)
			}
		}
		for _, notWant := range []string{"Summary", "Status report", "Slack channel", "Mitigated", "Closed"} {
			if strings.Contains(out, notWant) {
				t.Errorf("output should not contain %q:\n%s", notWant, out)
			}
		}
	})

	t.Run("not found", func(t *testing.T) {
		srv := errorServer(http.StatusNotFound, "not_found", "incident not found")
		err := incident.GetIncidentInfoWithHTTPClient(context.Background(), srv.client(), "token", "99")
		if err == nil || !strings.Contains(err.Error(), "Run 'openstatus incident list'") {
			t.Errorf("unexpected error %v", err)
		}
	})
}

func Test_InfoRows(t *testing.T) {
	t.Parallel()

	inc := incidentv1.Incident_builder{
		Id:       "42",
		Status:   incidentv1.IncidentStatus_INCIDENT_STATUS_RESOLVED,
		ClosedAt: strPtr("2026-10-16T12:00:00Z"),
	}.Build()

	var status, closed, transitions string
	for _, row := range incident.InfoRows(inc) {
		switch row[0] {
		case "Status":
			status = row[1]
		case "Closed":
			closed = row[1]
		case "Allowed transitions":
			transitions = row[1]
		}
	}
	if status != "resolved (closed)" || closed != "2026-10-16 12:00 UTC" || transitions != "none" {
		t.Errorf("status=%q closed=%q transitions=%q", status, closed, transitions)
	}
}
