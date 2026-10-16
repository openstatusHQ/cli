package incident

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"unicode/utf8"

	"buf.build/gen/go/openstatus/api/connectrpc/gosimple/openstatus/incident/v1/incidentv1connect"
	incidentv1 "buf.build/gen/go/openstatus/api/protocolbuffers/go/openstatus/incident/v1"
	"github.com/urfave/cli/v3"

	output "github.com/openstatusHQ/cli/internal/cli"
)

const (
	maxTitleLen   = 256
	maxSummaryLen = 4000
)

type DeclareParams struct {
	Title          string
	Severity       string
	Summary        string
	Commander      string
	StartedAt      string
	StatusReportID string
	Slack          bool
}

func validateTitle(title string) error {
	if strings.TrimSpace(title) == "" {
		return errors.New("title cannot be empty")
	}
	if utf8.RuneCountInString(title) > maxTitleLen {
		return fmt.Errorf("title is too long: maximum %d characters", maxTitleLen)
	}
	return nil
}

func validateSummary(summary string) error {
	if utf8.RuneCountInString(summary) > maxSummaryLen {
		return fmt.Errorf("summary is too long: maximum %d characters", maxSummaryLen)
	}
	return nil
}

func missingDeclareFlags(p DeclareParams) []string {
	var missing []string
	if p.Title == "" {
		missing = append(missing, "--title")
	}
	if p.Severity == "" {
		missing = append(missing, "--severity")
	}
	return missing
}

// validateProvidedDeclareFlags checks the flags that were passed so the wizard
// does not prompt for the rest before rejecting a bad value.
func validateProvidedDeclareFlags(p DeclareParams) error {
	if p.Title != "" {
		if err := validateTitle(p.Title); err != nil {
			return err
		}
	}
	if p.Severity != "" {
		if _, err := severityToSDK(p.Severity); err != nil {
			return err
		}
	}
	if err := validateSummary(p.Summary); err != nil {
		return err
	}
	if p.Commander != "" {
		if err := validateEmail("--commander", p.Commander); err != nil {
			return err
		}
	}
	if p.StartedAt != "" {
		if err := validateTimestamp("--started-at", p.StartedAt); err != nil {
			return err
		}
	}
	return nil
}

func buildDeclareRequest(p DeclareParams) (*incidentv1.DeclareIncidentRequest, error) {
	if err := validateTitle(p.Title); err != nil {
		return nil, err
	}
	severity, err := severityToSDK(p.Severity)
	if err != nil {
		return nil, err
	}
	if err := validateSummary(p.Summary); err != nil {
		return nil, err
	}

	req := &incidentv1.DeclareIncidentRequest{
		Title:    p.Title,
		Severity: severity,
	}
	if p.Summary != "" {
		req.SetSummary(p.Summary)
	}
	if p.Commander != "" {
		if err := validateEmail("--commander", p.Commander); err != nil {
			return nil, err
		}
		req.SetCommanderEmail(p.Commander)
	}
	if p.StartedAt != "" {
		if err := validateTimestamp("--started-at", p.StartedAt); err != nil {
			return nil, err
		}
		req.SetStartedAt(p.StartedAt)
	}
	if p.StatusReportID != "" {
		req.SetStatusReportId(p.StatusReportID)
	}
	if p.Slack {
		req.SetOpenSlackChannel(true)
	}
	return req, nil
}

func DeclareIncident(ctx context.Context, client incidentv1connect.IncidentServiceClient, p DeclareParams) (*incidentv1.Incident, error) {
	req, err := buildDeclareRequest(p)
	if err != nil {
		return nil, err
	}
	resp, err := client.DeclareIncident(ctx, req)
	if err != nil {
		return nil, formatIncidentError(err, "", opDeclare)
	}
	return resp.GetIncident(), nil
}

func declaredMessage(inc *incidentv1.Incident) string {
	return fmt.Sprintf("Incident declared (ID: %s)", inc.GetId())
}

func DeclareIncidentWithHTTPClient(ctx context.Context, httpClient *http.Client, apiKey string, p DeclareParams) error {
	inc, err := DeclareIncident(ctx, NewIncidentClientWithHTTPClient(httpClient, apiKey), p)
	if err != nil {
		return err
	}
	return renderIncidentResult(inc, declaredMessage(inc))
}

func GetIncidentDeclareCmd() *cli.Command {
	return &cli.Command{
		Name:    "declare",
		Aliases: []string{"create"},
		Usage:   "Declare a new incident",
		UsageText: `openstatus incident declare --title "Checkout API returns 502" --severity major
  openstatus inc declare --title "Database down" --severity critical --commander jane@example.com --slack
  ID=$(openstatus inc declare --title "Login errors" --severity minor -q)`,
		Description: `Declare a new incident in the open status.

Run without --title or --severity in an interactive terminal to be prompted.
In scripts, pass every required flag.`,
		Flags: []cli.Flag{
			accessTokenFlag(),
			&cli.StringFlag{
				Name:  "title",
				Usage: "Title of the incident (1-256 characters)",
			},
			&cli.StringFlag{
				Name:  "severity",
				Usage: "Severity of the incident (critical, major, minor)",
			},
			&cli.StringFlag{
				Name:  "summary",
				Usage: "Short human summary",
			},
			&cli.StringFlag{
				Name:  "commander",
				Usage: "Email of the workspace member leading the response",
			},
			&cli.StringFlag{
				Name:  "started-at",
				Usage: "When the impact began (RFC 3339, defaults to now)",
			},
			&cli.StringFlag{
				Name:  "status-report-id",
				Usage: "ID of an existing status report to link",
			},
			&cli.BoolFlag{
				Name:  "slack",
				Usage: "Open a dedicated Slack channel when Slack is connected",
			},
		},
		Action: withClient(func(ctx context.Context, cmd *cli.Command, client incidentv1connect.IncidentServiceClient) error {
			p := DeclareParams{
				Title:          cmd.String("title"),
				Severity:       cmd.String("severity"),
				Summary:        cmd.String("summary"),
				Commander:      cmd.String("commander"),
				StartedAt:      cmd.String("started-at"),
				StatusReportID: cmd.String("status-report-id"),
				Slack:          cmd.Bool("slack"),
			}

			if missing := missingDeclareFlags(p); len(missing) > 0 {
				if output.IsJSONOutput() || !output.IsStdinTerminal() {
					return fmt.Errorf("missing required flags: %s", strings.Join(missing, ", "))
				}
				if err := validateProvidedDeclareFlags(p); err != nil {
					return err
				}
				var err error
				p, err = runDeclareWizard(p)
				if err != nil {
					return err
				}
			}

			s := output.StartSpinner("Declaring incident...")
			inc, err := DeclareIncident(ctx, client, p)
			output.StopSpinner(s)
			if err != nil {
				return err
			}
			return renderIncidentResult(inc, declaredMessage(inc))
		}),
	}
}
