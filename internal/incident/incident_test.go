package incident_test

import (
	"slices"
	"strings"
	"testing"

	incidentv1 "buf.build/gen/go/openstatus/api/protocolbuffers/go/openstatus/incident/v1"

	"github.com/openstatusHQ/cli/internal/incident"
)

func Test_IncidentCmd(t *testing.T) {
	t.Parallel()

	cmd := incident.IncidentCmd()

	if cmd.Name != "incident" {
		t.Errorf("expected name 'incident', got %s", cmd.Name)
	}
	if cmd.Usage != "Declare and run incidents" {
		t.Errorf("unexpected usage %q", cmd.Usage)
	}
	for _, alias := range []string{"inc", "incidents"} {
		if !slices.Contains(cmd.Aliases, alias) {
			t.Errorf("expected alias %q", alias)
		}
	}

	expected := []string{
		"list", "info", "declare", "update", "set-status",
		"mitigate", "resolve", "reopen", "cancel",
		"note", "link-report", "unlink-report", "close", "delete", "postmortem",
	}
	if len(cmd.Commands) != len(expected) {
		t.Errorf("expected %d subcommands, got %d", len(expected), len(cmd.Commands))
	}
	names := make([]string, 0, len(cmd.Commands))
	for _, sub := range cmd.Commands {
		names = append(names, sub.Name)
		if sub.Name == "declare" && !slices.Contains(sub.Aliases, "create") {
			t.Error("expected declare to have alias 'create'")
		}
	}
	for _, name := range expected {
		if !slices.Contains(names, name) {
			t.Errorf("expected subcommand %q", name)
		}
	}
}

func Test_SeverityConversions(t *testing.T) {
	t.Parallel()

	for _, s := range []string{"critical", "major", "minor"} {
		sdk, err := incident.SeverityToSDK(s)
		if err != nil {
			t.Fatalf("SeverityToSDK(%q) error: %v", s, err)
		}
		if got := incident.SeverityToString(sdk); got != s {
			t.Errorf("round trip %q -> %q", s, got)
		}
	}
	if _, err := incident.SeverityToSDK("high"); err == nil || !strings.Contains(err.Error(), "critical, major, minor") {
		t.Errorf("expected invalid severity error, got %v", err)
	}
	if got := incident.SeverityToString(incidentv1.IncidentSeverity_INCIDENT_SEVERITY_UNSPECIFIED); got != "unknown" {
		t.Errorf("expected unknown, got %q", got)
	}
}

func Test_StatusConversions(t *testing.T) {
	t.Parallel()

	for _, s := range []string{"open", "mitigated", "resolved", "canceled"} {
		sdk, err := incident.StatusToSDK(s)
		if err != nil {
			t.Fatalf("StatusToSDK(%q) error: %v", s, err)
		}
		if got := incident.StatusToString(sdk); got != s {
			t.Errorf("round trip %q -> %q", s, got)
		}
	}
	if _, err := incident.StatusToSDK("closed"); err == nil {
		t.Error("expected error for 'closed'")
	}
	if got := incident.StatusToString(incidentv1.IncidentStatus_INCIDENT_STATUS_UNSPECIFIED); got != "unknown" {
		t.Errorf("expected unknown, got %q", got)
	}
}

func Test_ParseStatuses(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		raw     string
		want    []incidentv1.IncidentStatus
		wantErr bool
	}{
		{name: "empty", raw: ""},
		{name: "single", raw: "open", want: []incidentv1.IncidentStatus{incidentv1.IncidentStatus_INCIDENT_STATUS_OPEN}},
		{
			name: "list with spaces and duplicates",
			raw:  "open, mitigated,open",
			want: []incidentv1.IncidentStatus{
				incidentv1.IncidentStatus_INCIDENT_STATUS_OPEN,
				incidentv1.IncidentStatus_INCIDENT_STATUS_MITIGATED,
			},
		},
		{name: "unknown", raw: "open,closed", wantErr: true},
		{name: "empty item", raw: "open,,resolved", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := incident.ParseStatuses(tt.raw)
			if (err != nil) != tt.wantErr {
				t.Fatalf("error = %v, wantErr %v", err, tt.wantErr)
			}
			if !slices.Equal(got, tt.want) {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_EventTypeToString(t *testing.T) {
	t.Parallel()

	for value, name := range incidentv1.IncidentEventType_name {
		got := incident.EventTypeToString(incidentv1.IncidentEventType(value))
		if value == 0 {
			if got != "event" {
				t.Errorf("unspecified: got %q", got)
			}
			continue
		}
		want := strings.ToLower(strings.TrimPrefix(name, "INCIDENT_EVENT_TYPE_"))
		if got != want {
			t.Errorf("%s: got %q, want %q", name, got, want)
		}
	}
	if got := incident.EventTypeToString(incidentv1.IncidentEventType(999)); got != "event" {
		t.Errorf("unknown value: got %q", got)
	}
}

func Test_FormatUser(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		user      *incidentv1.IncidentUser
		full      string
		shortName string
	}{
		{name: "nil", user: nil, full: "unassigned", shortName: "unassigned"},
		{name: "name and email", user: incidentv1.IncidentUser_builder{Name: "Jane", Email: "jane@example.com"}.Build(), full: "Jane <jane@example.com>", shortName: "Jane"},
		{name: "email only", user: incidentv1.IncidentUser_builder{Email: "jane@example.com"}.Build(), full: "jane@example.com", shortName: "jane@example.com"},
		{name: "deleted user", user: incidentv1.IncidentUser_builder{Name: "Deleted user"}.Build(), full: "Deleted user", shortName: "Deleted user"},
		{name: "empty", user: &incidentv1.IncidentUser{}, full: "unassigned", shortName: "unassigned"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := incident.FormatUser(tt.user); got != tt.full {
				t.Errorf("FormatUser = %q, want %q", got, tt.full)
			}
			if got := incident.ShortUser(tt.user); got != tt.shortName {
				t.Errorf("ShortUser = %q, want %q", got, tt.shortName)
			}
		})
	}
}

func Test_DisplayStatus(t *testing.T) {
	t.Parallel()

	if got := incident.DisplayStatus(incidentv1.IncidentStatus_INCIDENT_STATUS_RESOLVED, ""); got != "resolved" {
		t.Errorf("got %q", got)
	}
	if got := incident.DisplayStatus(incidentv1.IncidentStatus_INCIDENT_STATUS_RESOLVED, "2026-10-16T12:00:00Z"); got != "resolved (closed)" {
		t.Errorf("got %q", got)
	}
	if got := incident.DisplayStatus(incidentv1.IncidentStatus_INCIDENT_STATUS_CANCELED, "2026-10-16T12:00:00Z"); got != "canceled" {
		t.Errorf("got %q", got)
	}
}

func Test_ReadTextArg(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		value   string
		stdin   string
		want    string
		wantErr bool
	}{
		{name: "literal", value: "hello", want: "hello"},
		{name: "stdin", value: "-", stdin: "line one\nline two\n", want: "line one\nline two"},
		{name: "stdin crlf", value: "-", stdin: "hello\r\n", want: "hello"},
		{name: "empty stdin", value: "-", stdin: "  \n", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := incident.ReadTextArg(tt.value, strings.NewReader(tt.stdin))
			if (err != nil) != tt.wantErr {
				t.Fatalf("error = %v, wantErr %v", err, tt.wantErr)
			}
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func Test_NextStepHint(t *testing.T) {
	t.Parallel()

	build := func(status incidentv1.IncidentStatus, closedAt *string) *incidentv1.Incident {
		return incidentv1.Incident_builder{Id: "42", Status: status, ClosedAt: closedAt}.Build()
	}

	tests := []struct {
		name string
		inc  *incidentv1.Incident
		want string
	}{
		{name: "open", inc: build(incidentv1.IncidentStatus_INCIDENT_STATUS_OPEN, nil), want: "openstatus inc mitigate 42"},
		{name: "mitigated", inc: build(incidentv1.IncidentStatus_INCIDENT_STATUS_MITIGATED, nil), want: "openstatus inc resolve 42"},
		{name: "resolved", inc: build(incidentv1.IncidentStatus_INCIDENT_STATUS_RESOLVED, nil), want: "openstatus inc postmortem edit 42"},
		{name: "closed", inc: build(incidentv1.IncidentStatus_INCIDENT_STATUS_RESOLVED, strPtr("2026-10-16T12:00:00Z")), want: ""},
		{name: "canceled", inc: build(incidentv1.IncidentStatus_INCIDENT_STATUS_CANCELED, strPtr("2026-10-16T12:00:00Z")), want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := incident.NextStepHint(tt.inc)
			if tt.want == "" {
				if got != "" {
					t.Errorf("expected no hint, got %q", got)
				}
				return
			}
			if !strings.Contains(got, tt.want) {
				t.Errorf("hint %q does not contain %q", got, tt.want)
			}
		})
	}
}
