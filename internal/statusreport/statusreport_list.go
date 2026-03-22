package statusreport

import (
	"context"
	"fmt"
	"net/http"

	status_reportv1 "buf.build/gen/go/openstatus/api/protocolbuffers/go/openstatus/status_report/v1"
	"buf.build/gen/go/openstatus/api/connectrpc/gosimple/openstatus/status_report/v1/status_reportv1connect"
	"github.com/fatih/color"
	"github.com/rodaine/table"
	"github.com/urfave/cli/v3"
)

func ListStatusReports(client status_reportv1connect.StatusReportServiceClient, statusFilter string, limit int) error {
	req := &status_reportv1.ListStatusReportsRequest{}

	if limit > 0 {
		l := int32(limit)
		req.SetLimit(l)
	}

	if statusFilter != "" {
		s, err := statusToSDK(statusFilter)
		if err != nil {
			return err
		}
		req.SetStatuses([]status_reportv1.StatusReportStatus{s})
	}

	resp, err := client.ListStatusReports(context.Background(), req)
	if err != nil {
		return fmt.Errorf("failed to list status reports: %w", err)
	}

	reports := resp.GetStatusReports()
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
			r.GetCreatedAt(),
			r.GetUpdatedAt(),
		)
	}

	tbl.Print()
	return nil
}

func ListStatusReportsWithHTTPClient(httpClient *http.Client, apiKey string, statusFilter string, limit int) error {
	client := NewStatusReportClientWithHTTPClient(httpClient, apiKey)
	return ListStatusReports(client, statusFilter, limit)
}

func GetStatusReportListCmd() *cli.Command {
	return &cli.Command{
		Name:      "list",
		Usage:     "List all status reports",
		UsageText: "openstatus status-report list [options]",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "access-token",
				Usage:    "OpenStatus API Access Token",
				Aliases:  []string{"t"},
				Sources:  cli.EnvVars("OPENSTATUS_API_TOKEN"),
				Required: true,
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
			client := NewStatusReportClient(cmd.String("access-token"))
			err := ListStatusReports(client, cmd.String("status"), int(cmd.Int("limit")))
			if err != nil {
				return cli.Exit(err.Error(), 1)
			}
			return nil
		},
	}
}
