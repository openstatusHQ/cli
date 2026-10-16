package incident_test

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/openstatusHQ/cli/internal/incident"
)

func Test_MutationOutputModes(t *testing.T) {
	t.Run("quiet prints only the id", func(t *testing.T) {
		withOutputModes(t, false, true)
		srv := newServer(http.StatusOK, incidentResponse())
		out := captureStdout(t, func() {
			if err := incident.DeclareIncidentWithHTTPClient(context.Background(), srv.client(), "token", incident.DeclareParams{Title: "x", Severity: "major"}); err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
		if out != "42\n" {
			t.Errorf("unexpected output %q", out)
		}
	})

	t.Run("json prints the incident without events", func(t *testing.T) {
		withOutputModes(t, true, false)
		srv := newServer(http.StatusOK, incidentResponse())
		out := captureStdout(t, func() {
			if err := incident.SetIncidentStatusWithHTTPClient(context.Background(), srv.client(), "token", "42", "mitigated", ""); err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
		var got map[string]json.RawMessage
		if err := json.Unmarshal([]byte(out), &got); err != nil {
			t.Fatalf("invalid JSON: %v\n%s", err, out)
		}
		for _, key := range []string{"id", "title", "severity", "status", "commander", "allowed_transitions", "deletable"} {
			if _, ok := got[key]; !ok {
				t.Errorf("missing key %q in %s", key, out)
			}
		}
		if _, ok := got["events"]; ok {
			t.Error("mutation output should not include events")
		}
		if strings.Contains(out, "Next:") {
			t.Error("json output should not include hints")
		}
	})
}
