package incident_test

import (
	"context"
	"net/http"
	"strings"
	"testing"

	incidentv1 "buf.build/gen/go/openstatus/api/protocolbuffers/go/openstatus/incident/v1"

	"github.com/openstatusHQ/cli/internal/incident"
)

func Test_LinkStatusReport(t *testing.T) {
	t.Run("link sends both ids", func(t *testing.T) {
		srv := newServer(http.StatusOK, incidentResponse())
		var err error
		out := captureStdout(t, func() {
			err = incident.LinkStatusReportWithHTTPClient(context.Background(), srv.client(), "token", "42", "123")
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		req := decodeBody(t, srv.reqBody, &incidentv1.LinkStatusReportRequest{})
		if req.GetId() != "42" || req.GetStatusReportId() != "123" {
			t.Errorf("unexpected request %v", req)
		}
		if !strings.Contains(out, "Status report 123 linked to incident 42") {
			t.Errorf("unexpected output %q", out)
		}
	})

	t.Run("link not found shows server message", func(t *testing.T) {
		srv := errorServer(http.StatusNotFound, "not_found", "status report 123 not found")
		err := incident.LinkStatusReportWithHTTPClient(context.Background(), srv.client(), "token", "42", "123")
		if err == nil || err.Error() != "status report 123 not found" {
			t.Errorf("unexpected error %v", err)
		}
	})

	t.Run("already linked", func(t *testing.T) {
		srv := errorServer(http.StatusBadRequest, "failed_precondition", "status report is already linked to another incident")
		err := incident.LinkStatusReportWithHTTPClient(context.Background(), srv.client(), "token", "42", "123")
		if err == nil || err.Error() != "status report is already linked to another incident" {
			t.Errorf("unexpected error %v", err)
		}
	})

	t.Run("unlink", func(t *testing.T) {
		srv := newServer(http.StatusOK, incidentResponse())
		var err error
		out := captureStdout(t, func() {
			err = incident.UnlinkStatusReportWithHTTPClient(context.Background(), srv.client(), "token", "42")
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if srv.procedure() != "UnlinkStatusReport" {
			t.Errorf("procedure = %s", srv.procedure())
		}
		req := decodeBody(t, srv.reqBody, &incidentv1.UnlinkStatusReportRequest{})
		if req.GetId() != "42" {
			t.Errorf("unexpected request %v", req)
		}
		if !strings.Contains(out, "Status report unlinked from incident 42") {
			t.Errorf("unexpected output %q", out)
		}
	})
}
