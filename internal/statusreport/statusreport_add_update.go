package statusreport

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"buf.build/gen/go/openstatus/api/connectrpc/gosimple/openstatus/status_report/v1/status_reportv1connect"
	status_reportv1 "buf.build/gen/go/openstatus/api/protocolbuffers/go/openstatus/status_report/v1"
	"github.com/urfave/cli/v3"

	"github.com/openstatusHQ/cli/internal/auth"
	output "github.com/openstatusHQ/cli/internal/cli"
)

type AddStatusReportUpdateParams struct {
	ReportID         string
	Status           string
	Message          string
	Date             string
	ComponentImpacts []*status_reportv1.ComponentImpact
	Notify           bool
}

func AddStatusReportUpdate(ctx context.Context, client status_reportv1connect.StatusReportServiceClient, p AddStatusReportUpdateParams, s *output.Spinner) error {
	if p.ReportID == "" {
		output.StopSpinner(s)
		fmt.Fprintln(os.Stderr, "Usage: openstatus status-report add-update <report-id> --status <status> --message <message>")
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "Example: openstatus sr add-update 12345 --status resolved --message \"Fix deployed\"")
		return fmt.Errorf("report ID is required")
	}

	sdkStatus, err := statusToSDK(p.Status)
	if err != nil {
		output.StopSpinner(s)
		return err
	}

	req := &status_reportv1.AddStatusReportUpdateRequest{
		StatusReportId: p.ReportID,
		Status:         sdkStatus,
		Message:        p.Message,
	}

	if p.Date != "" {
		req.SetDate(p.Date)
	}

	if len(p.ComponentImpacts) > 0 {
		req.SetComponentImpacts(p.ComponentImpacts)
	}

	if p.Notify {
		req.SetNotify(true)
	}

	resp, err := client.AddStatusReportUpdate(ctx, req)
	output.StopSpinner(s)
	if err != nil {
		return output.FormatError(err, "status-report", p.ReportID)
	}

	report := resp.GetStatusReport()
	fmt.Printf("Status report %s updated to %s\n", report.GetId(), statusColor(statusToString(report.GetStatus())))

	printImpactsDiff(report)

	if p.Status == "resolved" {
		fmt.Println("Report resolved.")
	}

	return nil
}

func printImpactsDiff(report *status_reportv1.StatusReport) {
	updates := report.GetUpdates()
	if len(updates) == 0 {
		return
	}

	now := currentImpacts(updates)
	was := currentImpacts(updates[:len(updates)-1])

	type diffRow struct {
		id   string
		from string
		to   string
	}
	var rows []diffRow
	for _, id := range affectedOrder(report) {
		nowImpact, hasNow := now[id]
		wasImpact, hasWas := was[id]
		if !hasNow {
			continue
		}
		if hasWas && wasImpact == nowImpact {
			continue
		}
		fromStr := "(none)"
		if hasWas {
			fromStr = impactToString(wasImpact)
		}
		rows = append(rows, diffRow{
			id:   id,
			from: fromStr,
			to:   impactToString(nowImpact),
		})
	}

	if len(rows) == 0 {
		return
	}

	fmt.Println("Impacts changed:")
	maxID := 0
	for _, r := range rows {
		if len(r.id) > maxID {
			maxID = len(r.id)
		}
	}
	for _, r := range rows {
		fmt.Printf("  %-*s  %s → %s\n", maxID, r.id, r.from, impactColor(r.to))
	}
}

func AddStatusReportUpdateWithHTTPClient(ctx context.Context, httpClient *http.Client, apiKey string, p AddStatusReportUpdateParams) error {
	client := NewStatusReportClientWithHTTPClient(httpClient, apiKey)
	return AddStatusReportUpdate(ctx, client, p, nil)
}

func GetStatusReportAddUpdateCmd() *cli.Command {
	return &cli.Command{
		Name:      "add-update",
		Usage:     "Add an update to a status report",
		UsageText: "openstatus status-report add-update <ReportID> --status resolved --message \"Issue has been resolved\"",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "access-token",
				Usage:   "OpenStatus API Access Token",
				Aliases: []string{"t"},
				Sources: cli.EnvVars("OPENSTATUS_API_TOKEN"),
			},
			&cli.StringFlag{
				Name:  "status",
				Usage: "New status (investigating, identified, monitoring, resolved)",
			},
			&cli.StringFlag{
				Name:  "message",
				Usage: "Message describing what changed",
			},
			&cli.StringFlag{
				Name:  "date",
				Usage: "Date for the update (RFC 3339 format, defaults to now)",
			},
			&cli.StringSliceFlag{
				Name:  "impact",
				Usage: "Per-component impact, repeatable: --impact <component_id>=<level> (level: operational, degraded, partial_outage, major_outage). Can also add new affected components.",
			},
			&cli.BoolFlag{
				Name:  "notify",
				Usage: "Notify subscribers about this update",
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			apiKey, err := auth.ResolveAccessToken(cmd)
			if err != nil {
				return cli.Exit(err.Error(), 1)
			}

			inputs := &addUpdateInputs{
				ReportID: cmd.Args().Get(0),
				Status:   cmd.String("status"),
				Message:  cmd.String("message"),
				Notify:   cmd.Bool("notify"),
			}

			impacts, err := parseImpactsFlag(cmd.StringSlice("impact"))
			if err != nil {
				return cli.Exit(err.Error(), 1)
			}
			inputs.ComponentImpacts = impactsToMap(impacts)

			needsWizard := inputs.ReportID == "" || inputs.Status == "" ||
				inputs.Message == ""

			if needsWizard {
				if output.IsJSONOutput() || !output.IsStdinTerminal() {
					var missing []string
					if inputs.ReportID == "" {
						missing = append(missing, "<report-id>")
					}
					if inputs.Status == "" {
						missing = append(missing, "--status")
					}
					if inputs.Message == "" {
						missing = append(missing, "--message")
					}
					return cli.Exit(fmt.Sprintf("missing required arguments: %s", strings.Join(missing, ", ")), 1)
				}
				inputs, err = runAddUpdateWizard(ctx, apiKey, inputs)
				if err != nil {
					return cli.Exit(err.Error(), 1)
				}
			}

			date := cmd.String("date")
			if date == "" {
				date = time.Now().UTC().Format(time.RFC3339)
			}

			componentImpacts, err := mapToImpacts(inputs.ComponentImpacts, nil)
			if err != nil {
				return cli.Exit(err.Error(), 1)
			}

			s := output.StartSpinner("Adding update...")
			client := NewStatusReportClient(apiKey)
			err = AddStatusReportUpdate(ctx, client, AddStatusReportUpdateParams{
				ReportID:         inputs.ReportID,
				Status:           inputs.Status,
				Message:          inputs.Message,
				Date:             date,
				ComponentImpacts: componentImpacts,
				Notify:           inputs.Notify,
			}, s)
			if err != nil {
				return cli.Exit(err.Error(), 1)
			}
			return nil
		},
	}
}
