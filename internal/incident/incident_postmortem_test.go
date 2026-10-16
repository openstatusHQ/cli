package incident_test

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	incidentv1 "buf.build/gen/go/openstatus/api/protocolbuffers/go/openstatus/incident/v1"

	"github.com/openstatusHQ/cli/internal/incident"
)

const postmortemResponse = `{"postmortem":{
	"incidentId": "42",
	"status": "POSTMORTEM_STATUS_DRAFT",
	"content": "# Summary\n\nBad deploy.",
	"draftedBy": "POSTMORTEM_AUTHOR_USER",
	"createdAt": "2026-10-16T12:00:00Z",
	"updatedAt": "2026-10-16T12:30:00Z"
}}`

func Test_PostmortemCmd(t *testing.T) {
	t.Parallel()

	var pm []string
	for _, sub := range incident.IncidentCmd().Commands {
		if sub.Name != "postmortem" {
			continue
		}
		if !slices.Contains(sub.Aliases, "pm") {
			t.Error("expected alias 'pm'")
		}
		for _, c := range sub.Commands {
			pm = append(pm, c.Name)
		}
	}
	if !slices.Equal(pm, []string{"show", "edit", "approve"}) {
		t.Errorf("postmortem subcommands = %v", pm)
	}
}

func Test_ShowPostmortem(t *testing.T) {
	t.Run("piped output is the raw body", func(t *testing.T) {
		srv := newServer(http.StatusOK, postmortemResponse)
		var err error
		out := captureStdout(t, func() {
			err = incident.ShowPostmortemWithHTTPClient(context.Background(), srv.client(), "token", "42")
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		req := decodeBody(t, srv.reqBody, &incidentv1.GetPostmortemRequest{})
		if req.GetIncidentId() != "42" {
			t.Errorf("unexpected request %v", req)
		}
		if out != "# Summary\n\nBad deploy.\n" {
			t.Errorf("unexpected output %q", out)
		}
	})

	t.Run("none yet", func(t *testing.T) {
		srv := newServer(http.StatusOK, `{}`)
		out := captureStdout(t, func() {
			if err := incident.ShowPostmortemWithHTTPClient(context.Background(), srv.client(), "token", "42"); err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
		if !strings.Contains(out, "No postmortem yet for incident 42") || !strings.Contains(out, "postmortem edit 42") {
			t.Errorf("unexpected output %q", out)
		}
	})

	t.Run("none yet json is null", func(t *testing.T) {
		withOutputModes(t, true, false)
		srv := newServer(http.StatusOK, `{}`)
		out := captureStdout(t, func() {
			if err := incident.ShowPostmortemWithHTTPClient(context.Background(), srv.client(), "token", "42"); err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
		if strings.TrimSpace(out) != "null" {
			t.Errorf("unexpected output %q", out)
		}
	})

	t.Run("json", func(t *testing.T) {
		withOutputModes(t, true, false)
		srv := newServer(http.StatusOK, postmortemResponse)
		out := captureStdout(t, func() {
			if err := incident.ShowPostmortemWithHTTPClient(context.Background(), srv.client(), "token", "42"); err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
		var got struct {
			IncidentID string `json:"incident_id"`
			Status     string `json:"status"`
			DraftedBy  string `json:"drafted_by"`
			Content    string `json:"content"`
		}
		if err := json.Unmarshal([]byte(out), &got); err != nil {
			t.Fatalf("invalid JSON: %v\n%s", err, out)
		}
		if got.IncidentID != "42" || got.Status != "draft" || got.DraftedBy != "user" || !strings.HasPrefix(got.Content, "# Summary") {
			t.Errorf("unexpected payload %+v", got)
		}
	})
}

func Test_ResolvePostmortemContent(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "pm.md")
	if err := os.WriteFile(path, []byte("# From file\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name       string
		file       string
		content    string
		fileSet    bool
		contentSet bool
		stdin      string
		want       string
		wantErr    string
	}{
		{name: "file", file: path, fileSet: true, want: "# From file\n"},
		{name: "stdin", file: "-", fileSet: true, stdin: "# From stdin\n", want: "# From stdin"},
		{name: "content", content: "# Inline", contentSet: true, want: "# Inline"},
		{name: "neither", wantErr: "exactly one of --file or --content"},
		{name: "both", file: path, fileSet: true, content: "x", contentSet: true, wantErr: "exactly one of --file or --content"},
		{name: "missing file", file: filepath.Join(dir, "nope.md"), fileSet: true, wantErr: "read postmortem file"},
		{name: "empty content", content: "  ", contentSet: true, wantErr: "cannot be empty"},
		{name: "empty file flag", file: "", fileSet: true, wantErr: "--file cannot be empty"},
		{name: "too long", content: strings.Repeat("a", 100001), contentSet: true, wantErr: "too long"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := incident.ResolvePostmortemContent(tt.file, tt.content, tt.fileSet, tt.contentSet, strings.NewReader(tt.stdin))
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Errorf("got %v, want error containing %q", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func Test_UpdatePostmortem(t *testing.T) {
	srv := newServer(http.StatusOK, postmortemResponse)
	var err error
	out := captureStdout(t, func() {
		err = incident.UpdatePostmortemWithHTTPClient(context.Background(), srv.client(), "token", "42", "# Summary")
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	req := decodeBody(t, srv.reqBody, &incidentv1.UpdatePostmortemRequest{})
	if req.GetIncidentId() != "42" || req.GetContent() != "# Summary" {
		t.Errorf("unexpected request %v", req)
	}
	if !strings.Contains(out, "Postmortem saved for incident 42 (draft)") || !strings.Contains(out, "postmortem approve 42") {
		t.Errorf("unexpected output %q", out)
	}
}

func Test_ApprovePostmortem(t *testing.T) {
	approved := `{"postmortem":{"incidentId":"42","status":"POSTMORTEM_STATUS_APPROVED","draftedBy":"POSTMORTEM_AUTHOR_AGENT","approvedBy":{"email":"jane@example.com","name":"Jane"}},"incident":` + closedResponse[len(`{"incident":`):]

	t.Run("close sent only when requested", func(t *testing.T) {
		for _, closeIncident := range []bool{false, true} {
			srv := newServer(http.StatusOK, approved)
			out := captureStdout(t, func() {
				if err := incident.ApprovePostmortemWithHTTPClient(context.Background(), srv.client(), "token", "42", closeIncident); err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			})
			req := decodeBody(t, srv.reqBody, &incidentv1.ApprovePostmortemRequest{})
			if req.HasClose() != closeIncident || req.GetIncidentId() != "42" {
				t.Errorf("close=%v: unexpected request %v", closeIncident, req)
			}
			if !strings.Contains(out, "Postmortem approved for incident 42") || !strings.Contains(out, "Incident 42 closed") {
				t.Errorf("unexpected output %q", out)
			}
		}
	})

	t.Run("json has postmortem and incident", func(t *testing.T) {
		withOutputModes(t, true, false)
		srv := newServer(http.StatusOK, approved)
		out := captureStdout(t, func() {
			if err := incident.ApprovePostmortemWithHTTPClient(context.Background(), srv.client(), "token", "42", true); err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
		var got struct {
			Postmortem struct {
				Status     string `json:"status"`
				DraftedBy  string `json:"drafted_by"`
				ApprovedBy struct {
					Email string `json:"email"`
				} `json:"approved_by"`
			} `json:"postmortem"`
			Incident struct {
				ClosedAt string `json:"closed_at"`
			} `json:"incident"`
		}
		if err := json.Unmarshal([]byte(out), &got); err != nil {
			t.Fatalf("invalid JSON: %v\n%s", err, out)
		}
		if got.Postmortem.Status != "approved" || got.Postmortem.DraftedBy != "agent" || got.Postmortem.ApprovedBy.Email != "jane@example.com" || got.Incident.ClosedAt == "" {
			t.Errorf("unexpected payload %+v", got)
		}
	})

	t.Run("not found means no postmortem written", func(t *testing.T) {
		srv := errorServer(http.StatusNotFound, "not_found", "postmortem not found")
		err := incident.ApprovePostmortemWithHTTPClient(context.Background(), srv.client(), "token", "42", false)
		if err == nil || !strings.Contains(err.Error(), "no postmortem written yet for incident 42") {
			t.Errorf("unexpected error %v", err)
		}
	})

	t.Run("permission denied explains roles", func(t *testing.T) {
		srv := errorServer(http.StatusForbidden, "permission_denied", "forbidden")
		err := incident.ApprovePostmortemWithHTTPClient(context.Background(), srv.client(), "token", "42", false)
		if err == nil || !strings.Contains(err.Error(), "or the incident commander") {
			t.Errorf("unexpected error %v", err)
		}
	})
}

func Test_WritePostmortemHeader(t *testing.T) {
	t.Parallel()

	approvedAt := "2026-10-16T13:00:00Z"
	pm := incidentv1.Postmortem_builder{
		IncidentId: "42",
		Status:     incidentv1.PostmortemStatus_POSTMORTEM_STATUS_APPROVED,
		Content:    "# Summary",
		DraftedBy:  incidentv1.PostmortemAuthor_POSTMORTEM_AUTHOR_AGENT,
		ApprovedBy: incidentv1.IncidentUser_builder{Name: "Jane", Email: "jane@example.com"}.Build(),
		ApprovedAt: &approvedAt,
		UpdatedAt:  "2026-10-16T13:00:00Z",
	}.Build()

	var buf strings.Builder
	incident.WritePostmortem(&buf, pm)
	out := buf.String()
	for _, want := range []string{"Status:     approved", "Drafted by: agent", "Approved:   2026-10-16 13:00 UTC by Jane <jane@example.com>", "\n\n# Summary\n"} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q:\n%s", want, out)
		}
	}
}
