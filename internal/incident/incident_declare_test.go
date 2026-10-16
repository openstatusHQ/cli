package incident_test

import (
	"context"
	"net/http"
	"slices"
	"strings"
	"testing"

	incidentv1 "buf.build/gen/go/openstatus/api/protocolbuffers/go/openstatus/incident/v1"

	"github.com/openstatusHQ/cli/internal/incident"
)

func Test_BuildDeclareRequest(t *testing.T) {
	t.Parallel()

	t.Run("minimal request leaves optional fields unset", func(t *testing.T) {
		t.Parallel()
		req, err := incident.BuildDeclareRequest(incident.DeclareParams{Title: "Checkout 502", Severity: "major"})
		if err != nil {
			t.Fatal(err)
		}
		if req.GetTitle() != "Checkout 502" || req.GetSeverity() != incidentv1.IncidentSeverity_INCIDENT_SEVERITY_MAJOR {
			t.Errorf("unexpected request %v", req)
		}
		if req.HasSummary() || req.HasCommanderEmail() || req.HasStartedAt() || req.HasStatusReportId() || req.HasOpenSlackChannel() {
			t.Errorf("expected optional fields unset, got %v", req)
		}
	})

	t.Run("all fields", func(t *testing.T) {
		t.Parallel()
		req, err := incident.BuildDeclareRequest(incident.DeclareParams{
			Title:          "Database down",
			Severity:       "critical",
			Summary:        "Primary unreachable",
			Commander:      "jane@example.com",
			StartedAt:      "2026-10-16T09:55:00Z",
			StatusReportID: "123",
			Slack:          true,
		})
		if err != nil {
			t.Fatal(err)
		}
		if req.GetSummary() != "Primary unreachable" || req.GetCommanderEmail() != "jane@example.com" ||
			req.GetStartedAt() != "2026-10-16T09:55:00Z" || req.GetStatusReportId() != "123" || !req.GetOpenSlackChannel() {
			t.Errorf("unexpected request %v", req)
		}
	})

	t.Run("validation errors", func(t *testing.T) {
		t.Parallel()
		tests := []struct {
			name string
			p    incident.DeclareParams
			want string
		}{
			{name: "bad severity", p: incident.DeclareParams{Title: "x", Severity: "high"}, want: "invalid severity"},
			{name: "bad commander", p: incident.DeclareParams{Title: "x", Severity: "minor", Commander: "jane"}, want: "invalid --commander"},
			{name: "named commander", p: incident.DeclareParams{Title: "x", Severity: "minor", Commander: "Jane <jane@example.com>"}, want: "invalid --commander"},
			{name: "bad started-at", p: incident.DeclareParams{Title: "x", Severity: "minor", StartedAt: "yesterday"}, want: "invalid --started-at"},
			{name: "blank title", p: incident.DeclareParams{Title: "   ", Severity: "minor"}, want: "title cannot be empty"},
			{name: "long title", p: incident.DeclareParams{Title: strings.Repeat("a", 257), Severity: "minor"}, want: "title is too long"},
			{name: "long summary", p: incident.DeclareParams{Title: "x", Severity: "minor", Summary: strings.Repeat("a", 4001)}, want: "summary is too long"},
		}
		for _, tt := range tests {
			if _, err := incident.BuildDeclareRequest(tt.p); err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Errorf("%s: got %v, want error containing %q", tt.name, err, tt.want)
			}
		}
	})
}

func Test_MissingDeclareFlags(t *testing.T) {
	t.Parallel()

	if got := incident.MissingDeclareFlags(incident.DeclareParams{}); !slices.Equal(got, []string{"--title", "--severity"}) {
		t.Errorf("got %v", got)
	}
	if got := incident.MissingDeclareFlags(incident.DeclareParams{Title: "x"}); !slices.Equal(got, []string{"--severity"}) {
		t.Errorf("got %v", got)
	}
	if got := incident.MissingDeclareFlags(incident.DeclareParams{Title: "x", Severity: "minor"}); len(got) != 0 {
		t.Errorf("got %v", got)
	}
}

func Test_ValidateDeclareFlags(t *testing.T) {
	t.Parallel()

	if err := incident.ValidateDeclareFlags(incident.DeclareParams{}); err != nil {
		t.Errorf("empty params should be valid before the wizard, got %v", err)
	}
	for _, p := range []incident.DeclareParams{
		{Severity: "foo"},
		{Commander: "not-an-email"},
		{StartedAt: "yesterday"},
		{Title: "   "},
	} {
		if err := incident.ValidateDeclareFlags(p); err == nil {
			t.Errorf("expected error for %+v", p)
		}
	}
}

func Test_DeclareIncident(t *testing.T) {
	t.Run("sends request and prints hint", func(t *testing.T) {
		srv := newServer(http.StatusOK, incidentResponse())
		var err error
		out := captureStdout(t, func() {
			err = incident.DeclareIncidentWithHTTPClient(context.Background(), srv.client(), "token", incident.DeclareParams{Title: "Checkout 502", Severity: "major", Slack: true})
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if srv.procedure() != "DeclareIncident" {
			t.Errorf("procedure = %s", srv.procedure())
		}
		req := decodeBody(t, srv.reqBody, &incidentv1.DeclareIncidentRequest{})
		if req.GetTitle() != "Checkout 502" || !req.GetOpenSlackChannel() {
			t.Errorf("unexpected request %v", req)
		}
		if !strings.Contains(out, "Incident declared (ID: 42)") || !strings.Contains(out, "openstatus inc mitigate 42") {
			t.Errorf("unexpected output %q", out)
		}
	})

	t.Run("validation error makes no request", func(t *testing.T) {
		srv := newServer(http.StatusOK, incidentResponse())
		err := incident.DeclareIncidentWithHTTPClient(context.Background(), srv.client(), "token", incident.DeclareParams{Title: "x", Severity: "minor", StartedAt: "now"})
		if err == nil {
			t.Fatal("expected error")
		}
		if srv.calls != 0 {
			t.Errorf("expected no request, got %d", srv.calls)
		}
	})

	t.Run("missing status report shows server message", func(t *testing.T) {
		srv := errorServer(http.StatusNotFound, "not_found", "status report 999 not found")
		err := incident.DeclareIncidentWithHTTPClient(context.Background(), srv.client(), "token", incident.DeclareParams{Title: "x", Severity: "minor", StatusReportID: "999"})
		if err == nil || err.Error() != "status report 999 not found" {
			t.Errorf("unexpected error %v", err)
		}
	})
}
