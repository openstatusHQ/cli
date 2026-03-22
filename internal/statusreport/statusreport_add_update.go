package statusreport

import (
	"context"
	"fmt"
	"net/http"
	"time"

	status_reportv1 "buf.build/gen/go/openstatus/api/protocolbuffers/go/openstatus/status_report/v1"
	"buf.build/gen/go/openstatus/api/connectrpc/gosimple/openstatus/status_report/v1/status_reportv1connect"
	"github.com/urfave/cli/v3"
)

func AddStatusReportUpdate(client status_reportv1connect.StatusReportServiceClient, reportId, status, message, date string, notify bool) error {
	if reportId == "" {
		return fmt.Errorf("report ID is required")
	}

	sdkStatus, err := statusToSDK(status)
	if err != nil {
		return err
	}

	req := &status_reportv1.AddStatusReportUpdateRequest{
		StatusReportId: reportId,
		Status:         sdkStatus,
		Message:        message,
	}

	if date != "" {
		req.SetDate(date)
	}

	if notify {
		req.SetNotify(true)
	}

	resp, err := client.AddStatusReportUpdate(context.Background(), req)
	if err != nil {
		return fmt.Errorf("failed to add status report update: %w", err)
	}

	report := resp.GetStatusReport()
	fmt.Printf("Status report %s updated to %s\n", report.GetId(), statusColor(statusToString(report.GetStatus())))

	if status == "resolved" {
		fmt.Println("Report resolved.")
	}

	return nil
}

func AddStatusReportUpdateWithHTTPClient(httpClient *http.Client, apiKey string, reportId, status, message, date string, notify bool) error {
	client := NewStatusReportClientWithHTTPClient(httpClient, apiKey)
	return AddStatusReportUpdate(client, reportId, status, message, date, notify)
}

func GetStatusReportAddUpdateCmd() *cli.Command {
	return &cli.Command{
		Name:      "add-update",
		Usage:     "Add an update to a status report",
		UsageText: "openstatus status-report add-update <ReportID> --status resolved --message \"Issue has been resolved\"",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "access-token",
				Usage:    "OpenStatus API Access Token",
				Aliases:  []string{"t"},
				Sources:  cli.EnvVars("OPENSTATUS_API_TOKEN"),
				Required: true,
			},
			&cli.StringFlag{
				Name:     "status",
				Usage:    "New status (investigating, identified, monitoring, resolved)",
				Required: true,
			},
			&cli.StringFlag{
				Name:     "message",
				Usage:    "Message describing what changed",
				Required: true,
			},
			&cli.StringFlag{
				Name:  "date",
				Usage: "Date for the update (RFC 3339 format, defaults to now)",
			},
			&cli.BoolFlag{
				Name:  "notify",
				Usage: "Notify subscribers about this update",
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			reportId := cmd.Args().Get(0)

			date := cmd.String("date")
			if date == "" {
				date = time.Now().UTC().Format(time.RFC3339)
			}

			client := NewStatusReportClient(cmd.String("access-token"))
			err := AddStatusReportUpdate(
				client,
				reportId,
				cmd.String("status"),
				cmd.String("message"),
				date,
				cmd.Bool("notify"),
			)
			if err != nil {
				return cli.Exit(err.Error(), 1)
			}
			return nil
		},
	}
}
