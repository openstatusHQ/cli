package statusreport

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"time"

	status_reportv1 "buf.build/gen/go/openstatus/api/protocolbuffers/go/openstatus/status_report/v1"
	"buf.build/gen/go/openstatus/api/connectrpc/gosimple/openstatus/status_report/v1/status_reportv1connect"
	"github.com/openstatusHQ/cli/internal/auth"
	output "github.com/openstatusHQ/cli/internal/cli"
	"github.com/urfave/cli/v3"
)

func AddStatusReportUpdate(ctx context.Context, client status_reportv1connect.StatusReportServiceClient, reportId, status, message, date string, notify bool, s *output.Spinner) error {
	if reportId == "" {
		output.StopSpinner(s)
		fmt.Fprintln(os.Stderr, "Usage: openstatus status-report add-update <report-id> --status <status> --message <message>")
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "Example: openstatus sr add-update 12345 --status resolved --message \"Fix deployed\"")
		return fmt.Errorf("report ID is required")
	}

	sdkStatus, err := statusToSDK(status)
	if err != nil {
		output.StopSpinner(s)
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

	resp, err := client.AddStatusReportUpdate(ctx, req)
	output.StopSpinner(s)
	if err != nil {
		return output.FormatError(err, "status-report", reportId)
	}

	report := resp.GetStatusReport()
	fmt.Printf("Status report %s updated to %s\n", report.GetId(), statusColor(statusToString(report.GetStatus())))

	if status == "resolved" {
		fmt.Println("Report resolved.")
	}

	return nil
}

func AddStatusReportUpdateWithHTTPClient(ctx context.Context, httpClient *http.Client, apiKey string, reportId, status, message, date string, notify bool) error {
	client := NewStatusReportClientWithHTTPClient(httpClient, apiKey)
	return AddStatusReportUpdate(ctx, client, reportId, status, message, date, notify, nil)
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
			apiKey, err := auth.ResolveAccessToken(cmd)
			if err != nil {
				return cli.Exit(err.Error(), 1)
			}
			reportId := cmd.Args().Get(0)

			date := cmd.String("date")
			if date == "" {
				date = time.Now().UTC().Format(time.RFC3339)
			}

			s := output.StartSpinner("Adding update...")
			client := NewStatusReportClient(apiKey)
			err = AddStatusReportUpdate(
				ctx,
				client,
				reportId,
				cmd.String("status"),
				cmd.String("message"),
				date,
				cmd.Bool("notify"),
				s,
			)
			if err != nil {
				return cli.Exit(err.Error(), 1)
			}
			return nil
		},
	}
}
