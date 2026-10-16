package incident_test

import (
	"context"
	"net/http"
	"strings"
	"testing"

	incidentv1 "buf.build/gen/go/openstatus/api/protocolbuffers/go/openstatus/incident/v1"

	"github.com/openstatusHQ/cli/internal/incident"
)

func Test_BuildSetStatusRequest(t *testing.T) {
	t.Parallel()

	tests := []struct {
		status string
		want   incidentv1.IncidentStatus
	}{
		{"open", incidentv1.IncidentStatus_INCIDENT_STATUS_OPEN},
		{"mitigated", incidentv1.IncidentStatus_INCIDENT_STATUS_MITIGATED},
		{"resolved", incidentv1.IncidentStatus_INCIDENT_STATUS_RESOLVED},
		{"canceled", incidentv1.IncidentStatus_INCIDENT_STATUS_CANCELED},
	}
	for _, tt := range tests {
		req, err := incident.BuildSetStatusRequest("42", tt.status, "")
		if err != nil {
			t.Fatalf("%s: %v", tt.status, err)
		}
		if req.GetStatus() != tt.want || req.HasNote() {
			t.Errorf("%s: unexpected request %v", tt.status, req)
		}
	}

	req, err := incident.BuildSetStatusRequest("42", "resolved", "Rolled back")
	if err != nil {
		t.Fatal(err)
	}
	if req.GetNote() != "Rolled back" {
		t.Errorf("note = %q", req.GetNote())
	}

	if _, err := incident.BuildSetStatusRequest("42", "closed", ""); err == nil {
		t.Error("expected error for invalid status")
	}
	if _, err := incident.BuildSetStatusRequest("42", "resolved", strings.Repeat("a", 10001)); err == nil {
		t.Error("expected error for long note")
	}
}

func Test_NeedsConfirmation(t *testing.T) {
	t.Parallel()

	if !incident.NeedsConfirmation("canceled", false) {
		t.Error("cancel without -y needs confirmation")
	}
	if incident.NeedsConfirmation("canceled", true) {
		t.Error("cancel with -y does not need confirmation")
	}
	for _, s := range []string{"open", "mitigated", "resolved"} {
		if incident.NeedsConfirmation(s, false) {
			t.Errorf("%s should not need confirmation", s)
		}
	}
}

func Test_StatusVerbs(t *testing.T) {
	t.Parallel()

	cmd := incident.IncidentCmd()
	verbs := map[string]bool{"mitigate": false, "resolve": false, "reopen": false, "cancel": true}
	for _, sub := range cmd.Commands {
		wantAutoAccept, ok := verbs[sub.Name]
		if !ok {
			continue
		}
		hasAutoAccept := false
		for _, f := range sub.Flags {
			for _, name := range f.Names() {
				if name == "auto-accept" {
					hasAutoAccept = true
				}
			}
		}
		if hasAutoAccept != wantAutoAccept {
			t.Errorf("%s: auto-accept flag = %v, want %v", sub.Name, hasAutoAccept, wantAutoAccept)
		}
	}

	for status, word := range map[string]string{"open": "reopened", "mitigated": "mitigated", "resolved": "resolved", "canceled": "canceled"} {
		if got := incident.StatusDoneWord(status); got != word {
			t.Errorf("StatusDoneWord(%q) = %q, want %q", status, got, word)
		}
	}
}

func Test_SetIncidentStatus(t *testing.T) {
	t.Run("sends status and note", func(t *testing.T) {
		srv := newServer(http.StatusOK, `{"incident":{"id":"42","status":"INCIDENT_STATUS_RESOLVED"}}`)
		var err error
		out := captureStdout(t, func() {
			err = incident.SetIncidentStatusWithHTTPClient(context.Background(), srv.client(), "token", "42", "resolved", "Fixed")
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if srv.procedure() != "SetIncidentStatus" {
			t.Errorf("procedure = %s", srv.procedure())
		}
		req := decodeBody(t, srv.reqBody, &incidentv1.SetIncidentStatusRequest{})
		if req.GetStatus() != incidentv1.IncidentStatus_INCIDENT_STATUS_RESOLVED || req.GetNote() != "Fixed" {
			t.Errorf("unexpected request %v", req)
		}
		if !strings.Contains(out, "Incident 42 resolved") || !strings.Contains(out, "postmortem edit 42") {
			t.Errorf("unexpected output %q", out)
		}
	})

	t.Run("refused transition hints at info", func(t *testing.T) {
		srv := errorServer(http.StatusBadRequest, "failed_precondition", "cannot move from resolved to mitigated.")
		err := incident.SetIncidentStatusWithHTTPClient(context.Background(), srv.client(), "token", "42", "mitigated", "")
		want := "cannot move from resolved to mitigated. Run 'openstatus inc info 42' to see allowed transitions."
		if err == nil || err.Error() != want {
			t.Errorf("got %v, want %q", err, want)
		}
	})
}
