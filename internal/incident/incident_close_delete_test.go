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

const closedResponse = `{"incident":{"id":"42","status":"INCIDENT_STATUS_RESOLVED","closedAt":"2026-10-16T12:00:00Z"}}`

func Test_CloseIncident(t *testing.T) {
	t.Run("skip postmortem only when set", func(t *testing.T) {
		for _, skip := range []bool{false, true} {
			srv := newServer(http.StatusOK, closedResponse)
			out := captureStdout(t, func() {
				if err := incident.CloseIncidentWithHTTPClient(context.Background(), srv.client(), "token", "42", skip); err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			})
			req := decodeBody(t, srv.reqBody, &incidentv1.CloseIncidentRequest{})
			if req.HasSkipPostmortem() != skip {
				t.Errorf("skip=%v: unexpected request %v", skip, req)
			}
			if strings.TrimSpace(out) != "Incident 42 closed" {
				t.Errorf("unexpected output %q", out)
			}
		}
	})

	t.Run("refused close explains requirements", func(t *testing.T) {
		srv := errorServer(http.StatusBadRequest, "failed_precondition", "postmortem is not approved")
		err := incident.CloseIncidentWithHTTPClient(context.Background(), srv.client(), "token", "42", false)
		if err == nil || !strings.Contains(err.Error(), "postmortem is not approved. Close requires a resolved incident") {
			t.Errorf("unexpected error %v", err)
		}
	})

	t.Run("permission denied explains roles", func(t *testing.T) {
		srv := errorServer(http.StatusForbidden, "permission_denied", "forbidden")
		err := incident.CloseIncidentWithHTTPClient(context.Background(), srv.client(), "token", "42", true)
		if err == nil || !strings.Contains(err.Error(), "or the incident commander") {
			t.Errorf("unexpected error %v", err)
		}
	})
}

func Test_DeleteIncident(t *testing.T) {
	t.Run("deletes", func(t *testing.T) {
		srv := newServer(http.StatusOK, `{"success":true}`)
		var err error
		out := captureStdout(t, func() {
			err = incident.DeleteIncidentWithHTTPClient(context.Background(), srv.client(), "token", "42")
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		req := decodeBody(t, srv.reqBody, &incidentv1.DeleteIncidentRequest{})
		if req.GetId() != "42" {
			t.Errorf("unexpected request %v", req)
		}
		if !strings.Contains(out, "Incident 42 deleted") {
			t.Errorf("unexpected output %q", out)
		}
	})

	t.Run("json output", func(t *testing.T) {
		withOutputModes(t, true, false)
		srv := newServer(http.StatusOK, `{"success":true}`)
		out := captureStdout(t, func() {
			if err := incident.DeleteIncidentWithHTTPClient(context.Background(), srv.client(), "token", "42"); err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
		var got struct {
			ID      string `json:"id"`
			Deleted bool   `json:"deleted"`
		}
		if err := json.Unmarshal([]byte(out), &got); err != nil {
			t.Fatalf("invalid JSON: %v\n%s", err, out)
		}
		if got.ID != "42" || !got.Deleted {
			t.Errorf("unexpected payload %+v", got)
		}
	})

	t.Run("refused delete suggests cancel", func(t *testing.T) {
		srv := errorServer(http.StatusBadRequest, "failed_precondition", "incident was mitigated")
		err := incident.DeleteIncidentWithHTTPClient(context.Background(), srv.client(), "token", "42")
		if err == nil || !strings.Contains(err.Error(), "use 'openstatus inc cancel 42' instead") {
			t.Errorf("unexpected error %v", err)
		}
	})

	t.Run("permission denied explains roles", func(t *testing.T) {
		srv := errorServer(http.StatusForbidden, "permission_denied", "forbidden")
		err := incident.DeleteIncidentWithHTTPClient(context.Background(), srv.client(), "token", "42")
		if err == nil || !strings.Contains(err.Error(), "requires a workspace owner or admin.") {
			t.Errorf("unexpected error %v", err)
		}
	})
}
