package statusreport

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"buf.build/gen/go/openstatus/api/connectrpc/gosimple/openstatus/status_report/v1/status_reportv1connect"
	status_reportv1 "buf.build/gen/go/openstatus/api/protocolbuffers/go/openstatus/status_report/v1"
	"connectrpc.com/connect"
	"github.com/urfave/cli/v3"

	"github.com/openstatusHQ/cli/internal/auth"
	output "github.com/openstatusHQ/cli/internal/cli"
)

type CreateStatusReportParams struct {
	Title            string
	Status           string
	Message          string
	Date             string
	PageID           string
	ComponentIDs     []string
	ComponentImpacts []*status_reportv1.ComponentImpact
	Notify           bool
	IncidentID       string
}

func CreateStatusReport(ctx context.Context, client status_reportv1connect.StatusReportServiceClient, p CreateStatusReportParams) (string, error) {
	report, err := createStatusReport(ctx, client, p)
	if err != nil {
		return "", err
	}
	return report.GetId(), nil
}

func createStatusReport(ctx context.Context, client status_reportv1connect.StatusReportServiceClient, p CreateStatusReportParams) (*status_reportv1.StatusReport, error) {
	sdkStatus, err := statusToSDK(p.Status)
	if err != nil {
		return nil, err
	}

	req := &status_reportv1.CreateStatusReportRequest{
		Title:   p.Title,
		Status:  sdkStatus,
		Message: p.Message,
		Date:    p.Date,
		PageId:  p.PageID,
	}

	if len(p.ComponentIDs) > 0 {
		req.SetPageComponentIds(p.ComponentIDs)
	}

	if len(p.ComponentImpacts) > 0 {
		req.SetComponentImpacts(p.ComponentImpacts)
	}

	if p.Notify {
		req.SetNotify(true)
	}

	if p.IncidentID != "" {
		req.SetIncidentId(p.IncidentID)
	}

	resp, err := client.CreateStatusReport(ctx, req)
	if err != nil {
		return nil, formatCreateError(err, p.IncidentID)
	}

	return resp.GetStatusReport(), nil
}

// formatCreateError keeps the server message on NotFound when an incident is
// linked, since it names the missing resource (page or incident); the generic
// mapping would always blame the status report.
func formatCreateError(err error, incidentID string) error {
	var connectErr *connect.Error
	if incidentID != "" && errors.As(err, &connectErr) && connectErr.Code() == connect.CodeNotFound {
		if msg := strings.TrimSuffix(connectErr.Message(), "."); msg != "" {
			return errors.New(msg)
		}
		return fmt.Errorf("incident %s not found. Run 'openstatus incident list' to see available incidents", incidentID)
	}
	return output.FormatError(err, "status-report", "")
}

func CreateStatusReportWithHTTPClient(ctx context.Context, httpClient *http.Client, apiKey string, p CreateStatusReportParams) (string, error) {
	client := NewStatusReportClientWithHTTPClient(httpClient, apiKey)
	return CreateStatusReport(ctx, client, p)
}

func GetStatusReportCreateCmd() *cli.Command {
	return &cli.Command{
		Name:      "create",
		Usage:     "Create a status report",
		UsageText: "openstatus status-report create --title \"API Degradation\" --status investigating --message \"Investigating increased latency\" --page-id 123",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "access-token",
				Usage:   "OpenStatus API Access Token",
				Aliases: []string{"t"},
				Sources: cli.EnvVars("OPENSTATUS_API_TOKEN"),
			},
			&cli.StringFlag{
				Name:  "title",
				Usage: "Title of the status report",
			},
			&cli.StringFlag{
				Name:  "status",
				Usage: "Initial status (investigating, identified, monitoring, resolved)",
			},
			&cli.StringFlag{
				Name:  "message",
				Usage: "Initial message describing the incident",
			},
			&cli.StringFlag{
				Name:  "page-id",
				Usage: "Status page ID to associate with this report",
			},
			&cli.StringFlag{
				Name:  "component-ids",
				Usage: "Comma-separated page component IDs (legacy; mutually exclusive with --impact)",
			},
			&cli.StringSliceFlag{
				Name:  "impact",
				Usage: "Per-component impact, repeatable: --impact <component_id>=<level> (level: operational, degraded, partial_outage, major_outage). Mutually exclusive with --component-ids.",
			},
			&cli.BoolFlag{
				Name:  "notify",
				Usage: "Notify subscribers about this status report",
			},
			&cli.StringFlag{
				Name:  "date",
				Usage: "Date when the event occurred (RFC 3339 format, defaults to now)",
			},
			&cli.StringFlag{
				Name:  "incident-id",
				Usage: "ID of an incident to link the report to",
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			apiKey, err := auth.ResolveAccessToken(cmd)
			if err != nil {
				return cli.Exit(err.Error(), 1)
			}

			inputs := &createInputs{
				PageID:  cmd.String("page-id"),
				Title:   cmd.String("title"),
				Status:  cmd.String("status"),
				Message: cmd.String("message"),
				Notify:  cmd.Bool("notify"),
			}
			if ids := cmd.String("component-ids"); ids != "" {
				inputs.ComponentIDs = strings.Split(ids, ",")
			}

			impacts, err := parseImpactsFlag(cmd.StringSlice("impact"))
			if err != nil {
				return cli.Exit(err.Error(), 1)
			}
			if len(impacts) > 0 && len(inputs.ComponentIDs) > 0 {
				return cli.Exit("--component-ids and --impact are mutually exclusive; pick one", 1)
			}
			inputs.ComponentImpacts = impactsToMap(impacts)

			needsWizard := inputs.Title == "" || inputs.Status == "" ||
				inputs.Message == "" || inputs.PageID == ""

			if needsWizard {
				if output.IsJSONOutput() || !output.IsStdinTerminal() {
					var missing []string
					if inputs.Title == "" {
						missing = append(missing, "--title")
					}
					if inputs.Status == "" {
						missing = append(missing, "--status")
					}
					if inputs.Message == "" {
						missing = append(missing, "--message")
					}
					if inputs.PageID == "" {
						missing = append(missing, "--page-id")
					}
					return cli.Exit(fmt.Sprintf("missing required flags: %s", strings.Join(missing, ", ")), 1)
				}
				inputs, err = runCreateWizard(ctx, apiKey, inputs)
				if err != nil {
					return cli.Exit(err.Error(), 1)
				}
			}

			date := cmd.String("date")
			if date == "" {
				date = time.Now().UTC().Format(time.RFC3339)
			}

			componentImpacts, err := mapToImpacts(inputs.ComponentImpacts, inputs.ComponentIDs)
			if err != nil {
				return cli.Exit(err.Error(), 1)
			}

			componentIDs := inputs.ComponentIDs
			if len(componentImpacts) > 0 {
				componentIDs = nil
			}

			client := NewStatusReportClient(apiKey)
			s := output.StartSpinner("Creating status report...")
			report, err := createStatusReport(ctx, client, CreateStatusReportParams{
				Title:            inputs.Title,
				Status:           inputs.Status,
				Message:          inputs.Message,
				Date:             date,
				PageID:           inputs.PageID,
				ComponentIDs:     componentIDs,
				ComponentImpacts: componentImpacts,
				Notify:           inputs.Notify,
				IncidentID:       cmd.String("incident-id"),
			})
			output.StopSpinner(s)
			if err != nil {
				return cli.Exit(err.Error(), 1)
			}

			return renderCreated(report, cmd.String("incident-id"))
		},
	}
}

// renderCreated prints the created report. requestedIncidentID is the
// --incident-id the user passed, so a silently dropped link can be flagged.
func renderCreated(report *status_reportv1.StatusReport, requestedIncidentID string) error {
	id := report.GetId()
	if requestedIncidentID != "" && !report.HasIncidentId() {
		fmt.Fprintf(os.Stderr, "Warning: the report was not linked to incident %s. Link it with: openstatus inc link-report %s %s\n", requestedIncidentID, requestedIncidentID, id)
	}
	if output.IsJSONOutput() {
		return output.PrintJSON(toStatusReportDetail(report, affectedOrder(report), currentImpacts(report.GetUpdates())))
	}
	if output.IsQuiet() {
		fmt.Println(id)
		return nil
	}
	fmt.Printf("Status report created successfully (ID: %s)\n", id)
	if report.HasIncidentId() {
		fmt.Printf("Linked to incident %s\n", report.GetIncidentId())
	}
	fmt.Printf("To add updates, run: openstatus status-report add-update %s --status identified --message '...'\n", id)
	fmt.Println("Tip: pass --impact <component>=<level> to update per-component impact.")
	return nil
}
