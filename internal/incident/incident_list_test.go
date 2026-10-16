package incident_test

import (
	"context"
	"encoding/json"
	"net/http"
	"slices"
	"strings"
	"testing"

	incidentv1 "buf.build/gen/go/openstatus/api/protocolbuffers/go/openstatus/incident/v1"

	"github.com/openstatusHQ/cli/internal/incident"
)

const listResponse = `{
	"incidents": [
		{
			"id": "42",
			"title": "Checkout 502",
			"severity": "INCIDENT_SEVERITY_MAJOR",
			"status": "INCIDENT_STATUS_RESOLVED",
			"commander": {"email": "jane@example.com", "name": "Jane"},
			"declaredAt": "2026-10-16T10:00:00Z",
			"startedAt": "2026-10-16T09:55:00Z",
			"closedAt": "2026-10-16T12:00:00Z",
			"statusReport": {"id": "123", "title": "Checkout issues", "status": "STATUS_REPORT_STATUS_RESOLVED"},
			"createdAt": "2026-10-16T10:00:00Z",
			"updatedAt": "2026-10-16T12:00:00Z"
		},
		{
			"id": "43",
			"title": "Login errors",
			"severity": "INCIDENT_SEVERITY_MINOR",
			"status": "INCIDENT_STATUS_OPEN",
			"declaredAt": "2026-10-16T11:00:00Z",
			"startedAt": "2026-10-16T11:00:00Z",
			"createdAt": "2026-10-16T11:00:00Z",
			"updatedAt": "2026-10-16T11:00:00Z"
		}
	],
	"totalSize": 5
}`

func Test_BuildListRequest(t *testing.T) {
	t.Parallel()

	t.Run("no filters", func(t *testing.T) {
		t.Parallel()
		req, err := incident.BuildListRequest(incident.ListParams{})
		if err != nil {
			t.Fatal(err)
		}
		if req.HasClosed() || req.HasLimit() || req.HasOffset() || len(req.GetStatuses()) != 0 {
			t.Errorf("expected empty request, got %v", req)
		}
	})

	t.Run("active sets closed false", func(t *testing.T) {
		t.Parallel()
		req, err := incident.BuildListRequest(incident.ListParams{Active: true})
		if err != nil {
			t.Fatal(err)
		}
		if !req.HasClosed() || req.GetClosed() {
			t.Errorf("expected closed=false, got %v", req)
		}
	})

	t.Run("closed sets closed true", func(t *testing.T) {
		t.Parallel()
		req, err := incident.BuildListRequest(incident.ListParams{Closed: true})
		if err != nil {
			t.Fatal(err)
		}
		if !req.HasClosed() || !req.GetClosed() {
			t.Errorf("expected closed=true, got %v", req)
		}
	})

	t.Run("active and closed are exclusive", func(t *testing.T) {
		t.Parallel()
		if _, err := incident.BuildListRequest(incident.ListParams{Active: true, Closed: true}); err == nil {
			t.Error("expected error")
		}
	})

	t.Run("statuses limit offset", func(t *testing.T) {
		t.Parallel()
		req, err := incident.BuildListRequest(incident.ListParams{Statuses: "open,mitigated", Limit: 10, Offset: 20})
		if err != nil {
			t.Fatal(err)
		}
		want := []incidentv1.IncidentStatus{
			incidentv1.IncidentStatus_INCIDENT_STATUS_OPEN,
			incidentv1.IncidentStatus_INCIDENT_STATUS_MITIGATED,
		}
		if !slices.Equal(req.GetStatuses(), want) {
			t.Errorf("statuses = %v", req.GetStatuses())
		}
		if req.GetLimit() != 10 || req.GetOffset() != 20 {
			t.Errorf("limit/offset = %d/%d", req.GetLimit(), req.GetOffset())
		}
	})

	t.Run("invalid values", func(t *testing.T) {
		t.Parallel()
		for _, p := range []incident.ListParams{{Limit: 101}, {Limit: -1}, {Offset: -1}, {Statuses: "bogus"}} {
			if _, err := incident.BuildListRequest(p); err == nil {
				t.Errorf("expected error for %+v", p)
			}
		}
	})
}

func Test_ListFooter(t *testing.T) {
	t.Parallel()

	if got := incident.ListFooter(0, 2, 2); got != "" {
		t.Errorf("expected no footer, got %q", got)
	}
	if got := incident.ListFooter(0, 0, 10); got != "" {
		t.Errorf("expected no footer for empty page, got %q", got)
	}
	if got := incident.ListFooter(0, 2, 5); got != "Showing 1-2 of 5 — use --offset 2 for more" {
		t.Errorf("got %q", got)
	}
	if got := incident.ListFooter(2, 2, 5); got != "Showing 3-4 of 5 — use --offset 4 for more" {
		t.Errorf("got %q", got)
	}
}

func Test_ListIncidents(t *testing.T) {
	t.Run("sends filters and renders table", func(t *testing.T) {
		srv := newServer(http.StatusOK, listResponse)
		var err error
		out := captureStdout(t, func() {
			err = incident.ListIncidentsWithHTTPClient(context.Background(), srv.client(), "token", incident.ListParams{Statuses: "resolved", Closed: true})
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if srv.procedure() != "ListIncidents" {
			t.Errorf("procedure = %s", srv.procedure())
		}
		req := decodeBody(t, srv.reqBody, &incidentv1.ListIncidentsRequest{})
		if !req.GetClosed() || len(req.GetStatuses()) != 1 {
			t.Errorf("unexpected request %v", req)
		}
		for _, want := range []string{"Checkout 502", "resolved (closed)", "Jane", "123", "Login errors", "unassigned", "Showing 1-2 of 5"} {
			if !strings.Contains(out, want) {
				t.Errorf("output missing %q:\n%s", want, out)
			}
		}
	})

	t.Run("json output", func(t *testing.T) {
		withOutputModes(t, true, false)
		srv := newServer(http.StatusOK, listResponse)
		out := captureStdout(t, func() {
			if err := incident.ListIncidentsWithHTTPClient(context.Background(), srv.client(), "token", incident.ListParams{}); err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
		var got struct {
			Incidents []struct {
				ID           string `json:"id"`
				Severity     string `json:"severity"`
				Status       string `json:"status"`
				ClosedAt     string `json:"closed_at"`
				StatusReport *struct {
					ID     string `json:"id"`
					Status string `json:"status"`
				} `json:"status_report"`
			} `json:"incidents"`
			TotalSize int `json:"total_size"`
		}
		if err := json.Unmarshal([]byte(out), &got); err != nil {
			t.Fatalf("invalid JSON: %v\n%s", err, out)
		}
		if got.TotalSize != 5 || len(got.Incidents) != 2 {
			t.Fatalf("unexpected payload %+v", got)
		}
		first := got.Incidents[0]
		if first.ID != "42" || first.Severity != "major" || first.Status != "resolved" || first.ClosedAt == "" {
			t.Errorf("unexpected first incident %+v", first)
		}
		if first.StatusReport == nil || first.StatusReport.ID != "123" || first.StatusReport.Status != "resolved" {
			t.Errorf("unexpected status report %+v", first.StatusReport)
		}
		if got.Incidents[1].StatusReport != nil {
			t.Error("expected no status report on second incident")
		}
	})

	t.Run("empty list", func(t *testing.T) {
		srv := newServer(http.StatusOK, `{"incidents":[],"totalSize":0}`)
		out := captureStdout(t, func() {
			if err := incident.ListIncidentsWithHTTPClient(context.Background(), srv.client(), "token", incident.ListParams{}); err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
		if !strings.Contains(out, "No incidents found") {
			t.Errorf("unexpected output %q", out)
		}
	})

	t.Run("invalid filter makes no request", func(t *testing.T) {
		srv := newServer(http.StatusOK, listResponse)
		err := incident.ListIncidentsWithHTTPClient(context.Background(), srv.client(), "token", incident.ListParams{Active: true, Closed: true})
		if err == nil {
			t.Fatal("expected error")
		}
		if srv.calls != 0 {
			t.Errorf("expected no request, got %d", srv.calls)
		}
	})

	t.Run("api error", func(t *testing.T) {
		srv := errorServer(http.StatusUnauthorized, "unauthenticated", "bad key")
		err := incident.ListIncidentsWithHTTPClient(context.Background(), srv.client(), "token", incident.ListParams{})
		if err == nil || !strings.Contains(err.Error(), "authentication failed") {
			t.Errorf("unexpected error %v", err)
		}
	})
}
