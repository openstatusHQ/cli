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

const noteResponse = `{"event":{"id":"e9","type":"INCIDENT_EVENT_TYPE_NOTE","message":"Rolled back","createdAt":"2026-10-16T10:10:00Z"}}`

func Test_BuildNoteRequest(t *testing.T) {
	t.Parallel()

	req, err := incident.BuildNoteRequest("42", "line one\nline two")
	if err != nil {
		t.Fatal(err)
	}
	if req.GetId() != "42" || req.GetMessage() != "line one\nline two" {
		t.Errorf("unexpected request %v", req)
	}
	if _, err := incident.BuildNoteRequest("42", "  "); err == nil {
		t.Error("expected error for blank note")
	}
	if _, err := incident.BuildNoteRequest("42", strings.Repeat("é", 10001)); err == nil {
		t.Error("expected error for long note")
	}
	if _, err := incident.BuildNoteRequest("42", strings.Repeat("é", 10000)); err != nil {
		t.Errorf("10000 characters should be accepted: %v", err)
	}
}

func Test_AddIncidentNote(t *testing.T) {
	t.Run("text output", func(t *testing.T) {
		srv := newServer(http.StatusOK, noteResponse)
		var err error
		out := captureStdout(t, func() {
			err = incident.AddIncidentNoteWithHTTPClient(context.Background(), srv.client(), "token", "42", "Rolled back")
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		req := decodeBody(t, srv.reqBody, &incidentv1.AddIncidentNoteRequest{})
		if req.GetMessage() != "Rolled back" {
			t.Errorf("message = %q", req.GetMessage())
		}
		if strings.TrimSpace(out) != "Note added to incident 42" {
			t.Errorf("unexpected output %q", out)
		}
	})

	t.Run("json output is the event", func(t *testing.T) {
		withOutputModes(t, true, false)
		srv := newServer(http.StatusOK, noteResponse)
		out := captureStdout(t, func() {
			if err := incident.AddIncidentNoteWithHTTPClient(context.Background(), srv.client(), "token", "42", "Rolled back"); err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
		var got struct {
			ID   string `json:"id"`
			Type string `json:"type"`
		}
		if err := json.Unmarshal([]byte(out), &got); err != nil {
			t.Fatalf("invalid JSON: %v\n%s", err, out)
		}
		if got.ID != "e9" || got.Type != "note" {
			t.Errorf("unexpected payload %+v", got)
		}
	})

	t.Run("quiet prints event id", func(t *testing.T) {
		withOutputModes(t, false, true)
		srv := newServer(http.StatusOK, noteResponse)
		out := captureStdout(t, func() {
			if err := incident.AddIncidentNoteWithHTTPClient(context.Background(), srv.client(), "token", "42", "Rolled back"); err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
		if strings.TrimSpace(out) != "e9" {
			t.Errorf("unexpected output %q", out)
		}
	})
}
