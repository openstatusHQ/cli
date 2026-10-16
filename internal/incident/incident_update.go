package incident

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"buf.build/gen/go/openstatus/api/connectrpc/gosimple/openstatus/incident/v1/incidentv1connect"
	incidentv1 "buf.build/gen/go/openstatus/api/protocolbuffers/go/openstatus/incident/v1"
	"github.com/urfave/cli/v3"

	output "github.com/openstatusHQ/cli/internal/cli"
)

const updateUsage = "openstatus incident update <incident-id> [--title ...] [--severity ...] [--summary ... | --clear-summary] [--commander ... | --clear-commander] [--started-at ...]"

type UpdateParams struct {
	Title          *string
	Severity       *string
	Summary        *string
	Commander      *string
	StartedAt      *string
	ClearSummary   bool
	ClearCommander bool
}

func (p UpdateParams) empty() bool {
	return p.Title == nil && p.Severity == nil && p.Summary == nil && p.Commander == nil &&
		p.StartedAt == nil && !p.ClearSummary && !p.ClearCommander
}

func buildUpdateRequest(id string, p UpdateParams) (*incidentv1.UpdateIncidentRequest, error) {
	if p.empty() {
		return nil, errors.New("nothing to update: pass at least one of --title, --severity, --summary, --clear-summary, --commander, --clear-commander, --started-at")
	}
	if p.Summary != nil && p.ClearSummary {
		return nil, errors.New("--summary and --clear-summary are mutually exclusive")
	}
	if p.Commander != nil && p.ClearCommander {
		return nil, errors.New("--commander and --clear-commander are mutually exclusive")
	}

	req := &incidentv1.UpdateIncidentRequest{Id: id}
	if p.Title != nil {
		if err := validateTitle(*p.Title); err != nil {
			return nil, err
		}
		req.SetTitle(*p.Title)
	}
	if p.Severity != nil {
		severity, err := severityToSDK(*p.Severity)
		if err != nil {
			return nil, err
		}
		req.SetSeverity(severity)
	}
	if p.Summary != nil {
		if *p.Summary == "" {
			return nil, errors.New("--summary cannot be empty; use --clear-summary to remove it")
		}
		if err := validateSummary(*p.Summary); err != nil {
			return nil, err
		}
		req.Set_Summary(*p.Summary)
	}
	if p.ClearSummary {
		req.SetClearSummary(true)
	}
	if p.Commander != nil {
		if err := validateEmail("--commander", *p.Commander); err != nil {
			return nil, err
		}
		req.SetCommanderEmail(*p.Commander)
	}
	if p.ClearCommander {
		req.SetClearCommander(true)
	}
	if p.StartedAt != nil {
		if err := validateTimestamp("--started-at", *p.StartedAt); err != nil {
			return nil, err
		}
		req.SetStartedAt(*p.StartedAt)
	}
	return req, nil
}

func UpdateIncident(ctx context.Context, client incidentv1connect.IncidentServiceClient, id string, p UpdateParams) (*incidentv1.Incident, error) {
	req, err := buildUpdateRequest(id, p)
	if err != nil {
		return nil, err
	}
	resp, err := client.UpdateIncident(ctx, req)
	if err != nil {
		return nil, formatIncidentError(err, id, opGeneric)
	}
	return resp.GetIncident(), nil
}

func UpdateIncidentWithHTTPClient(ctx context.Context, httpClient *http.Client, apiKey string, id string, p UpdateParams) error {
	inc, err := UpdateIncident(ctx, NewIncidentClientWithHTTPClient(httpClient, apiKey), id, p)
	if err != nil {
		return err
	}
	return renderIncidentResult(inc, fmt.Sprintf("Incident %s updated", id))
}

func optionalString(cmd *cli.Command, name string) *string {
	if !cmd.IsSet(name) {
		return nil
	}
	v := cmd.String(name)
	return &v
}

func GetIncidentUpdateCmd() *cli.Command {
	return &cli.Command{
		Name:  "update",
		Usage: "Edit an incident's title, severity, summary, commander or start time",
		UsageText: `openstatus incident update <IncidentID> [flags]
  openstatus inc update 42 --severity critical
  openstatus inc update 42 --commander jane@example.com
  openstatus inc update 42 --clear-summary`,
		Flags: []cli.Flag{
			accessTokenFlag(),
			&cli.StringFlag{
				Name:  "title",
				Usage: "New title (1-256 characters)",
			},
			&cli.StringFlag{
				Name:  "severity",
				Usage: "New severity (critical, major, minor)",
			},
			&cli.StringFlag{
				Name:  "summary",
				Usage: "New summary",
			},
			&cli.BoolFlag{
				Name:  "clear-summary",
				Usage: "Remove the summary",
			},
			&cli.StringFlag{
				Name:  "commander",
				Usage: "Email of the new commander",
			},
			&cli.BoolFlag{
				Name:  "clear-commander",
				Usage: "Unassign the commander",
			},
			&cli.StringFlag{
				Name:  "started-at",
				Usage: "New start of impact (RFC 3339)",
			},
		},
		Action: withClient(func(ctx context.Context, cmd *cli.Command, client incidentv1connect.IncidentServiceClient) error {
			id, err := requireID(cmd, updateUsage)
			if err != nil {
				return err
			}
			p := UpdateParams{
				Title:          optionalString(cmd, "title"),
				Severity:       optionalString(cmd, "severity"),
				Summary:        optionalString(cmd, "summary"),
				Commander:      optionalString(cmd, "commander"),
				StartedAt:      optionalString(cmd, "started-at"),
				ClearSummary:   cmd.Bool("clear-summary"),
				ClearCommander: cmd.Bool("clear-commander"),
			}

			s := output.StartSpinner("Updating incident...")
			inc, err := UpdateIncident(ctx, client, id, p)
			output.StopSpinner(s)
			if err != nil {
				return err
			}
			return renderIncidentResult(inc, fmt.Sprintf("Incident %s updated", id))
		}),
	}
}
