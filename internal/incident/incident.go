package incident

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/mail"
	"os"
	"strings"
	"time"

	"buf.build/gen/go/openstatus/api/connectrpc/gosimple/openstatus/incident/v1/incidentv1connect"
	incidentv1 "buf.build/gen/go/openstatus/api/protocolbuffers/go/openstatus/incident/v1"
	"connectrpc.com/connect"
	"github.com/fatih/color"
	"github.com/olekukonko/tablewriter"
	"github.com/olekukonko/tablewriter/renderer"
	"github.com/olekukonko/tablewriter/tw"
	"github.com/urfave/cli/v3"

	"github.com/openstatusHQ/cli/internal/api"
	"github.com/openstatusHQ/cli/internal/auth"
	output "github.com/openstatusHQ/cli/internal/cli"
)

func NewIncidentClient(apiKey string) incidentv1connect.IncidentServiceClient {
	return NewIncidentClientWithHTTPClient(api.DefaultHTTPClient, apiKey)
}

func NewIncidentClientWithHTTPClient(httpClient *http.Client, apiKey string) incidentv1connect.IncidentServiceClient {
	return incidentv1connect.NewIncidentServiceClient(
		httpClient,
		api.ConnectBaseURL,
		connect.WithInterceptors(api.NewAuthInterceptor(apiKey)),
		connect.WithProtoJSON(),
	)
}

func severityToSDK(s string) (incidentv1.IncidentSeverity, error) {
	switch s {
	case "critical":
		return incidentv1.IncidentSeverity_INCIDENT_SEVERITY_CRITICAL, nil
	case "major":
		return incidentv1.IncidentSeverity_INCIDENT_SEVERITY_MAJOR, nil
	case "minor":
		return incidentv1.IncidentSeverity_INCIDENT_SEVERITY_MINOR, nil
	default:
		return incidentv1.IncidentSeverity_INCIDENT_SEVERITY_UNSPECIFIED,
			fmt.Errorf("invalid severity %q: must be one of critical, major, minor", s)
	}
}

func severityToString(s incidentv1.IncidentSeverity) string {
	switch s {
	case incidentv1.IncidentSeverity_INCIDENT_SEVERITY_CRITICAL:
		return "critical"
	case incidentv1.IncidentSeverity_INCIDENT_SEVERITY_MAJOR:
		return "major"
	case incidentv1.IncidentSeverity_INCIDENT_SEVERITY_MINOR:
		return "minor"
	default:
		return "unknown"
	}
}

func severityColor(s string) string {
	switch s {
	case "critical":
		return color.New(color.FgRed, color.Bold).Sprint(s)
	case "major":
		return color.MagentaString(s)
	case "minor":
		return color.YellowString(s)
	default:
		return s
	}
}

func statusToSDK(s string) (incidentv1.IncidentStatus, error) {
	switch s {
	case "open":
		return incidentv1.IncidentStatus_INCIDENT_STATUS_OPEN, nil
	case "mitigated":
		return incidentv1.IncidentStatus_INCIDENT_STATUS_MITIGATED, nil
	case "resolved":
		return incidentv1.IncidentStatus_INCIDENT_STATUS_RESOLVED, nil
	case "canceled":
		return incidentv1.IncidentStatus_INCIDENT_STATUS_CANCELED, nil
	default:
		return incidentv1.IncidentStatus_INCIDENT_STATUS_UNSPECIFIED,
			fmt.Errorf("invalid status %q: must be one of open, mitigated, resolved, canceled", s)
	}
}

func statusToString(s incidentv1.IncidentStatus) string {
	switch s {
	case incidentv1.IncidentStatus_INCIDENT_STATUS_OPEN:
		return "open"
	case incidentv1.IncidentStatus_INCIDENT_STATUS_MITIGATED:
		return "mitigated"
	case incidentv1.IncidentStatus_INCIDENT_STATUS_RESOLVED:
		return "resolved"
	case incidentv1.IncidentStatus_INCIDENT_STATUS_CANCELED:
		return "canceled"
	default:
		return "unknown"
	}
}

func statusColor(s string) string {
	switch s {
	case "open":
		return color.RedString(s)
	case "mitigated":
		return color.YellowString(s)
	case "resolved":
		return color.GreenString(s)
	case "canceled":
		return color.New(color.Faint).Sprint(s)
	default:
		return s
	}
}

func statusesToStrings(statuses []incidentv1.IncidentStatus) []string {
	out := make([]string, 0, len(statuses))
	for _, s := range statuses {
		out = append(out, statusToString(s))
	}
	return out
}

func parseStatuses(raw string) ([]incidentv1.IncidentStatus, error) {
	if strings.TrimSpace(raw) == "" {
		return nil, nil
	}
	seen := make(map[incidentv1.IncidentStatus]bool)
	var out []incidentv1.IncidentStatus
	for _, part := range strings.Split(raw, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			return nil, fmt.Errorf("invalid --status %q: empty value in list", raw)
		}
		s, err := statusToSDK(part)
		if err != nil {
			return nil, err
		}
		if seen[s] {
			continue
		}
		seen[s] = true
		out = append(out, s)
	}
	return out, nil
}

func eventTypeToString(t incidentv1.IncidentEventType) string {
	if t == incidentv1.IncidentEventType_INCIDENT_EVENT_TYPE_UNSPECIFIED {
		return "event"
	}
	name := strings.TrimPrefix(t.String(), "INCIDENT_EVENT_TYPE_")
	if name == t.String() {
		return "event"
	}
	return strings.ToLower(name)
}

func statusReportStatusToString(name string) string {
	return strings.ToLower(strings.TrimPrefix(name, "STATUS_REPORT_STATUS_"))
}

func formatUser(u *incidentv1.IncidentUser) string {
	if u == nil {
		return "unassigned"
	}
	switch {
	case u.GetName() != "" && u.GetEmail() != "":
		return fmt.Sprintf("%s <%s>", u.GetName(), u.GetEmail())
	case u.GetEmail() != "":
		return u.GetEmail()
	case u.GetName() != "":
		return u.GetName()
	default:
		return "unassigned"
	}
}

func shortUser(u *incidentv1.IncidentUser) string {
	if u == nil {
		return "unassigned"
	}
	if u.GetName() != "" {
		return u.GetName()
	}
	if u.GetEmail() != "" {
		return u.GetEmail()
	}
	return "unassigned"
}

func displayStatus(status incidentv1.IncidentStatus, closedAt string) string {
	s := statusToString(status)
	out := statusColor(s)
	if closedAt != "" && s != "canceled" {
		out += " (closed)"
	}
	return out
}

func validateEmail(flag, v string) error {
	addr, err := mail.ParseAddress(v)
	if err != nil || addr.Address != v {
		return fmt.Errorf("invalid %s %q: must be an email address", flag, v)
	}
	return nil
}

func validateTimestamp(flag, v string) error {
	if _, err := time.Parse(time.RFC3339, v); err != nil {
		return fmt.Errorf("invalid %s %q: must be RFC 3339, e.g. 2026-10-16T10:30:00Z", flag, v)
	}
	return nil
}

type userJSON struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

type statusReportRefJSON struct {
	ID     string `json:"id"`
	Title  string `json:"title"`
	Status string `json:"status"`
	PageID string `json:"page_id,omitempty"`
}

type incidentSummaryJSON struct {
	ID           string               `json:"id"`
	Title        string               `json:"title"`
	Severity     string               `json:"severity"`
	Status       string               `json:"status"`
	Commander    *userJSON            `json:"commander,omitempty"`
	DeclaredAt   string               `json:"declared_at"`
	StartedAt    string               `json:"started_at"`
	ResolvedAt   string               `json:"resolved_at,omitempty"`
	ClosedAt     string               `json:"closed_at,omitempty"`
	StatusReport *statusReportRefJSON `json:"status_report,omitempty"`
	CreatedAt    string               `json:"created_at"`
	UpdatedAt    string               `json:"updated_at"`
}

type incidentEventJSON struct {
	ID        string    `json:"id"`
	Type      string    `json:"type"`
	Message   string    `json:"message"`
	CreatedBy *userJSON `json:"created_by,omitempty"`
	CreatedAt string    `json:"created_at"`
}

type incidentDetail struct {
	ID                 string               `json:"id"`
	Title              string               `json:"title"`
	Severity           string               `json:"severity"`
	Status             string               `json:"status"`
	Summary            string               `json:"summary,omitempty"`
	Commander          *userJSON            `json:"commander,omitempty"`
	DeclaredBy         *userJSON            `json:"declared_by,omitempty"`
	ResolvedBy         *userJSON            `json:"resolved_by,omitempty"`
	DeclaredAt         string               `json:"declared_at"`
	StartedAt          string               `json:"started_at"`
	MitigatedAt        string               `json:"mitigated_at,omitempty"`
	ResolvedAt         string               `json:"resolved_at,omitempty"`
	ClosedAt           string               `json:"closed_at,omitempty"`
	StatusReport       *statusReportRefJSON `json:"status_report,omitempty"`
	SlackChannelURL    string               `json:"slack_channel_url,omitempty"`
	AllowedTransitions []string             `json:"allowed_transitions"`
	Deletable          bool                 `json:"deletable"`
	Events             []incidentEventJSON  `json:"events,omitempty"`
	CreatedAt          string               `json:"created_at"`
	UpdatedAt          string               `json:"updated_at"`
}

func toUserJSON(u *incidentv1.IncidentUser) *userJSON {
	if u == nil {
		return nil
	}
	return &userJSON{Name: u.GetName(), Email: u.GetEmail()}
}

func toStatusReportRefJSON(r *incidentv1.IncidentStatusReport) *statusReportRefJSON {
	if r == nil {
		return nil
	}
	return &statusReportRefJSON{
		ID:     r.GetId(),
		Title:  r.GetTitle(),
		Status: statusReportStatusToString(r.GetStatus().String()),
		PageID: r.GetPageId(),
	}
}

func toSummaryJSON(s *incidentv1.IncidentSummary) incidentSummaryJSON {
	return incidentSummaryJSON{
		ID:           s.GetId(),
		Title:        s.GetTitle(),
		Severity:     severityToString(s.GetSeverity()),
		Status:       statusToString(s.GetStatus()),
		Commander:    toUserJSON(s.GetCommander()),
		DeclaredAt:   s.GetDeclaredAt(),
		StartedAt:    s.GetStartedAt(),
		ResolvedAt:   s.GetResolvedAt(),
		ClosedAt:     s.GetClosedAt(),
		StatusReport: toStatusReportRefJSON(s.GetStatusReport()),
		CreatedAt:    s.GetCreatedAt(),
		UpdatedAt:    s.GetUpdatedAt(),
	}
}

func toEventJSON(e *incidentv1.IncidentEvent) incidentEventJSON {
	return incidentEventJSON{
		ID:        e.GetId(),
		Type:      eventTypeToString(e.GetType()),
		Message:   e.GetMessage(),
		CreatedBy: toUserJSON(e.GetCreatedBy()),
		CreatedAt: e.GetCreatedAt(),
	}
}

func toDetailJSON(inc *incidentv1.Incident) incidentDetail {
	d := incidentDetail{
		ID:                 inc.GetId(),
		Title:              inc.GetTitle(),
		Severity:           severityToString(inc.GetSeverity()),
		Status:             statusToString(inc.GetStatus()),
		Summary:            inc.GetSummary(),
		Commander:          toUserJSON(inc.GetCommander()),
		DeclaredBy:         toUserJSON(inc.GetDeclaredBy()),
		ResolvedBy:         toUserJSON(inc.GetResolvedBy()),
		DeclaredAt:         inc.GetDeclaredAt(),
		StartedAt:          inc.GetStartedAt(),
		MitigatedAt:        inc.GetMitigatedAt(),
		ResolvedAt:         inc.GetResolvedAt(),
		ClosedAt:           inc.GetClosedAt(),
		StatusReport:       toStatusReportRefJSON(inc.GetStatusReport()),
		SlackChannelURL:    inc.GetSlackChannelUrl(),
		AllowedTransitions: statusesToStrings(inc.GetAllowedTransitions()),
		Deletable:          inc.GetDeletable(),
		CreatedAt:          inc.GetCreatedAt(),
		UpdatedAt:          inc.GetUpdatedAt(),
	}
	for _, e := range inc.GetEvents() {
		d.Events = append(d.Events, toEventJSON(e))
	}
	return d
}

func readTextArg(value string, stdin io.Reader) (string, error) {
	if value != "-" {
		return value, nil
	}
	data, err := io.ReadAll(stdin)
	if err != nil {
		return "", fmt.Errorf("failed to read stdin: %w", err)
	}
	text := strings.TrimSuffix(strings.TrimSuffix(string(data), "\n"), "\r")
	if strings.TrimSpace(text) == "" {
		return "", errors.New("no input received on stdin")
	}
	return text, nil
}

func nextStepHint(inc *incidentv1.Incident) string {
	if inc.GetClosedAt() != "" {
		return ""
	}
	id := inc.GetId()
	switch inc.GetStatus() {
	case incidentv1.IncidentStatus_INCIDENT_STATUS_OPEN:
		return fmt.Sprintf("Next: openstatus inc mitigate %s --note \"...\" or openstatus inc resolve %s", id, id)
	case incidentv1.IncidentStatus_INCIDENT_STATUS_MITIGATED:
		return fmt.Sprintf("Next: openstatus inc resolve %s --note \"...\"", id)
	case incidentv1.IncidentStatus_INCIDENT_STATUS_RESOLVED:
		return fmt.Sprintf("Write the postmortem: openstatus inc postmortem edit %s --file postmortem.md\nOr close without one: openstatus inc close %s --skip-postmortem", id, id)
	default:
		return ""
	}
}

func renderIncidentResult(inc *incidentv1.Incident, success string) error {
	if output.IsJSONOutput() {
		return output.PrintJSON(toDetailJSON(inc))
	}
	if output.IsQuiet() {
		fmt.Println(inc.GetId())
		return nil
	}
	fmt.Println(success)
	if hint := nextStepHint(inc); hint != "" {
		fmt.Println(hint)
	}
	return nil
}

func newBlueprintTable() *tablewriter.Table {
	return tablewriter.NewTable(os.Stdout,
		tablewriter.WithRenderer(renderer.NewBlueprint()),
		tablewriter.WithRendition(tw.Rendition{
			Symbols: tw.NewSymbolCustom("custom").WithColumn("="),
			Borders: tw.Border{
				Top:    tw.Off,
				Left:   tw.Off,
				Right:  tw.Off,
				Bottom: tw.Off,
			},
			Settings: tw.Settings{
				Lines: tw.Lines{
					ShowHeaderLine: tw.Off,
					ShowFooterLine: tw.On,
				},
				Separators: tw.Separators{
					BetweenRows:    tw.Off,
					BetweenColumns: tw.On,
				},
			},
		}),
		tablewriter.WithRowAlignment(tw.AlignLeft),
		tablewriter.WithHeaderAlignment(tw.AlignLeft),
	)
}

func accessTokenFlag() cli.Flag {
	return &cli.StringFlag{
		Name:    "access-token",
		Usage:   "OpenStatus API Access Token",
		Aliases: []string{"t"},
		Sources: cli.EnvVars("OPENSTATUS_API_TOKEN"),
	}
}

func autoAcceptFlag() cli.Flag {
	return &cli.BoolFlag{
		Name:    "auto-accept",
		Usage:   "Automatically accept the prompt",
		Aliases: []string{"y"},
	}
}

func requireArg(cmd *cli.Command, index int, name, usage string) (string, error) {
	v := cmd.Args().Get(index)
	if v == "" {
		fmt.Fprintln(os.Stderr, "Usage: "+usage)
		return "", fmt.Errorf("%s is required", name)
	}
	return v, nil
}

func requireID(cmd *cli.Command, usage string) (string, error) {
	return requireArg(cmd, 0, "incident ID", usage)
}

func confirm(cmd *cli.Command, prompt string) (bool, error) {
	if cmd.Bool("auto-accept") {
		return true, nil
	}
	confirmed, err := output.AskForConfirmation(prompt)
	if err != nil {
		return false, fmt.Errorf("failed to read input: %w", err)
	}
	return confirmed, nil
}

type actionFunc func(ctx context.Context, cmd *cli.Command, client incidentv1connect.IncidentServiceClient) error

func withClient(fn actionFunc) cli.ActionFunc {
	return func(ctx context.Context, cmd *cli.Command) error {
		apiKey, err := auth.ResolveAccessToken(cmd)
		if err != nil {
			return cli.Exit(err.Error(), 1)
		}
		if err := fn(ctx, cmd, NewIncidentClient(apiKey)); err != nil {
			return cli.Exit(err.Error(), 1)
		}
		return nil
	}
}

func IncidentCmd() *cli.Command {
	return &cli.Command{
		Name:    "incident",
		Aliases: []string{"inc", "incidents"},
		Usage:   "Declare and run incidents",
		Description: `Incidents are the internal response to an outage: severity, commander,
lifecycle (open -> mitigated -> resolved, or canceled), timeline notes, a linked
public status report and the postmortem.

Use 'openstatus status-report' for the public communication on a status page.`,
		Commands: []*cli.Command{
			GetIncidentListCmd(),
			GetIncidentInfoCmd(),
			GetIncidentDeclareCmd(),
			GetIncidentUpdateCmd(),
			GetIncidentSetStatusCmd(),
			newStatusVerbCmd(verbMitigate),
			newStatusVerbCmd(verbResolve),
			newStatusVerbCmd(verbReopen),
			newStatusVerbCmd(verbCancel),
			GetIncidentNoteCmd(),
			GetIncidentLinkReportCmd(),
			GetIncidentUnlinkReportCmd(),
			GetIncidentCloseCmd(),
			GetIncidentDeleteCmd(),
		},
	}
}
