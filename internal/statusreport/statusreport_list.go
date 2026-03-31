package statusreport

import (
	"context"
	"fmt"
	"net/http"

	status_reportv1 "buf.build/gen/go/openstatus/api/protocolbuffers/go/openstatus/status_report/v1"
	"buf.build/gen/go/openstatus/api/connectrpc/gosimple/openstatus/status_report/v1/status_reportv1connect"
	"github.com/openstatusHQ/cli/internal/auth"
	output "github.com/openstatusHQ/cli/internal/cli"
	"github.com/fatih/color"
	"github.com/rodaine/table"
	"github.com/urfave/cli/v3"
)

type statusReportListEntry struct {
	ID        string `json:"id"`
	Title     string `json:"title"`
	Status    string `json:"status"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

func ListStatusReports(ctx context.Context, client status_reportv1connect.StatusReportServiceClient, statusFilter string, limit int, s *output.Spinner) error {
	req := &status_reportv1.ListStatusReportsRequest{}

	if limit > 0 {
		l := int32(limit)
		req.SetLimit(l)
	}

	if statusFilter != "" {
		sdkStatus, err := statusToSDK(statusFilter)
		if err != nil {
			output.StopSpinner(s)
			return err
		}
		req.SetStatuses([]status_reportv1.StatusReportStatus{sdkStatus})
	}

	resp, err := client.ListStatusReports(ctx, req)
	output.StopSpinner(s)
	if err != nil {
		return output.FormatError(err, "status-report", "")
	}

	reports := resp.GetStatusReports()

	if output.IsJSONOutput() {
		entries := make([]statusReportListEntry, 0, len(reports))
		for _, r := range reports {
			entries = append(entries, statusReportListEntry{
				ID:        r.GetId(),
				Title:     r.GetTitle(),
				Status:    statusToString(r.GetStatus()),
				CreatedAt: r.GetCreatedAt(),
				UpdatedAt: r.GetUpdatedAt(),
			})
		}
		return output.PrintJSON(entries)
	}

	if len(reports) == 0 {
		fmt.Println("No status reports found")
		return nil
	}

	headerFmt := color.New(color.FgGreen, color.Underline).SprintfFunc()
	columnFmt := color.New(color.FgYellow).SprintfFunc()

	tbl := table.New("ID", "Title", "Status", "Created", "Updated")
	tbl.WithHeaderFormatter(headerFmt).WithFirstColumnFormatter(columnFmt)

	for _, r := range reports {
		tbl.AddRow(
			r.GetId(),
			r.GetTitle(),
			statusColor(statusToString(r.GetStatus())),
			output.FormatTimestamp(r.GetCreatedAt()),
			output.FormatTimestamp(r.GetUpdatedAt()),
		)
	}

	tbl.Print()
	return nil
}

func ListStatusReportsWithHTTPClient(ctx context.Context, httpClient *http.Client, apiKey string, statusFilter string, limit int) error {
	client := NewStatusReportClientWithHTTPClient(httpClient, apiKey)
	return ListStatusReports(ctx, client, statusFilter, limit, nil)
}

func GetStatusReportListCmd() *cli.Command {
	return &cli.Command{
		Name:  "list",
		Usage: "List all status reports",
		UsageText: `openstatus status-report list
  openstatus status-report list --status investigating --limit 10`,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "access-token",
				Usage:   "OpenStatus API Access Token",
				Aliases: []string{"t"},
				Sources: cli.EnvVars("OPENSTATUS_API_TOKEN"),
			},
			&cli.StringFlag{
				Name:  "status",
				Usage: "Filter by status (investigating, identified, monitoring, resolved)",
			},
			&cli.IntFlag{
				Name:  "limit",
				Usage: "Maximum number of reports to return (1-100)",
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			apiKey, err := auth.ResolveAccessToken(cmd)
			if err != nil {
				return cli.Exit(err.Error(), 1)
			}
			s := output.StartSpinner("Fetching status reports...")
			client := NewStatusReportClient(apiKey)
			err = ListStatusReports(ctx, client, cmd.String("status"), int(cmd.Int("limit")), s)
			if err != nil {
				return cli.Exit(err.Error(), 1)
			}
			return nil
		},
	}
}
