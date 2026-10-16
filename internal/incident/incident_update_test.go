package incident_test

import (
	"context"
	"net/http"
	"strings"
	"testing"

	incidentv1 "buf.build/gen/go/openstatus/api/protocolbuffers/go/openstatus/incident/v1"

	"github.com/openstatusHQ/cli/internal/incident"
)

func Test_BuildUpdateRequest(t *testing.T) {
	t.Parallel()

	t.Run("only set fields", func(t *testing.T) {
		t.Parallel()
		req, err := incident.BuildUpdateRequest("42", incident.UpdateParams{Severity: strPtr("critical")})
		if err != nil {
			t.Fatal(err)
		}
		if req.GetId() != "42" || req.GetSeverity() != incidentv1.IncidentSeverity_INCIDENT_SEVERITY_CRITICAL {
			t.Errorf("unexpected request %v", req)
		}
		if req.HasTitle() || req.Has_Summary() || req.HasClearSummary() || req.HasCommanderEmail() || req.HasClearCommander() || req.HasStartedAt() {
			t.Errorf("expected only severity set, got %v", req)
		}
	})

	t.Run("all fields", func(t *testing.T) {
		t.Parallel()
		req, err := incident.BuildUpdateRequest("42", incident.UpdateParams{
			Title:     strPtr("New title"),
			Summary:   strPtr("New summary"),
			Commander: strPtr("jane@example.com"),
			StartedAt: strPtr("2026-10-16T09:00:00+02:00"),
		})
		if err != nil {
			t.Fatal(err)
		}
		if req.GetTitle() != "New title" || req.Get_Summary() != "New summary" || req.GetCommanderEmail() != "jane@example.com" || req.GetStartedAt() != "2026-10-16T09:00:00+02:00" {
			t.Errorf("unexpected request %v", req)
		}
	})

	t.Run("clear flags", func(t *testing.T) {
		t.Parallel()
		req, err := incident.BuildUpdateRequest("42", incident.UpdateParams{ClearSummary: true, ClearCommander: true})
		if err != nil {
			t.Fatal(err)
		}
		if !req.GetClearSummary() || !req.GetClearCommander() || req.Has_Summary() || req.HasCommanderEmail() {
			t.Errorf("unexpected request %v", req)
		}
	})

	t.Run("errors", func(t *testing.T) {
		t.Parallel()
		tests := []struct {
			name string
			p    incident.UpdateParams
			want string
		}{
			{name: "nothing", p: incident.UpdateParams{}, want: "nothing to update"},
			{name: "summary and clear", p: incident.UpdateParams{Summary: strPtr("x"), ClearSummary: true}, want: "--summary and --clear-summary"},
			{name: "commander and clear", p: incident.UpdateParams{Commander: strPtr("a@b.co"), ClearCommander: true}, want: "--commander and --clear-commander"},
			{name: "empty summary", p: incident.UpdateParams{Summary: strPtr("")}, want: "use --clear-summary"},
			{name: "bad severity", p: incident.UpdateParams{Severity: strPtr("low")}, want: "invalid severity"},
			{name: "bad commander", p: incident.UpdateParams{Commander: strPtr("nope")}, want: "invalid --commander"},
			{name: "bad started-at", p: incident.UpdateParams{StartedAt: strPtr("2026-10-16")}, want: "invalid --started-at"},
			{name: "empty title", p: incident.UpdateParams{Title: strPtr("")}, want: "title cannot be empty"},
		}
		for _, tt := range tests {
			if _, err := incident.BuildUpdateRequest("42", tt.p); err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Errorf("%s: got %v, want error containing %q", tt.name, err, tt.want)
			}
		}
	})
}

func Test_UpdateIncident(t *testing.T) {
	t.Run("sends request", func(t *testing.T) {
		srv := newServer(http.StatusOK, incidentResponse())
		var err error
		out := captureStdout(t, func() {
			err = incident.UpdateIncidentWithHTTPClient(context.Background(), srv.client(), "token", "42", incident.UpdateParams{Title: strPtr("Renamed")})
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		req := decodeBody(t, srv.reqBody, &incidentv1.UpdateIncidentRequest{})
		if req.GetId() != "42" || req.GetTitle() != "Renamed" {
			t.Errorf("unexpected request %v", req)
		}
		if !strings.Contains(out, "Incident 42 updated") {
			t.Errorf("unexpected output %q", out)
		}
	})

	t.Run("validation error makes no request", func(t *testing.T) {
		srv := newServer(http.StatusOK, incidentResponse())
		if err := incident.UpdateIncidentWithHTTPClient(context.Background(), srv.client(), "token", "42", incident.UpdateParams{}); err == nil {
			t.Fatal("expected error")
		}
		if srv.calls != 0 {
			t.Errorf("expected no request, got %d", srv.calls)
		}
	})
}
