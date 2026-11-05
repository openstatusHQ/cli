package statusreport

import (
	"context"
	"fmt"
	"net/http"
	"os"

	"buf.build/gen/go/openstatus/api/connectrpc/gosimple/openstatus/status_report/v1/status_reportv1connect"
	status_reportv1 "buf.build/gen/go/openstatus/api/protocolbuffers/go/openstatus/status_report/v1"
	"github.com/urfave/cli/v3"

	"github.com/openstatusHQ/cli/internal/auth"
	output "github.com/openstatusHQ/cli/internal/cli"
)

func DeleteStatusReport(ctx context.Context, client status_reportv1connect.StatusReportServiceClient, reportId string) error {
	if reportId == "" {
		fmt.Fprintln(os.Stderr, "Usage: openstatus status-report delete <report-id>")
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "Example: openstatus status-report delete 12345")
		return output.UsageErrorf("report ID is required")
	}

	_, err := client.DeleteStatusReport(ctx, &status_reportv1.DeleteStatusReportRequest{
		Id: reportId,
	})
	if err != nil {
		return output.FormatError(err, "status-report", reportId)
	}

	return nil
}

func DeleteStatusReportWithHTTPClient(ctx context.Context, httpClient *http.Client, apiKey string, reportId string) error {
	client := NewStatusReportClientWithHTTPClient(httpClient, apiKey)
	return DeleteStatusReport(ctx, client, reportId)
}

func GetStatusReportDeleteCmd() *cli.Command {
	return &cli.Command{
		Name:  "delete",
		Usage: "Delete a status report",
		UsageText: `openstatus status-report delete <ReportID>
  openstatus status-report delete 12345 -y`,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "access-token",
				Usage:   "OpenStatus API Access Token",
				Aliases: []string{"t"},
				Sources: cli.EnvVars("OPENSTATUS_API_TOKEN"),
			},
			&cli.BoolFlag{
				Name:    "auto-accept",
				Usage:   "Automatically accept the prompt",
				Aliases: []string{"y"},
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			apiKey, err := auth.ResolveAccessToken(ctx, cmd)
			if err != nil {
				return err
			}
			reportId := cmd.Args().Get(0)
			if reportId == "" {
				fmt.Fprintln(os.Stderr, "Usage: openstatus status-report delete <report-id>")
				return output.UsageErrorf("report ID is required")
			}

			if !cmd.Bool("auto-accept") {
				confirmed, err := output.AskForConfirmation(fmt.Sprintf("You are about to delete status report: %s, do you want to continue", reportId))
				if err != nil {
					return fmt.Errorf("failed to read input: %w", err)
				}
				if !confirmed {
					return nil
				}
			}

			client := NewStatusReportClient(apiKey)
			s := output.StartSpinner("Deleting status report...")
			err = DeleteStatusReport(ctx, client, reportId)
			output.StopSpinner(s)
			if err != nil {
				return err
			}
			return output.RenderDeleted(reportId, fmt.Sprintf("Status report %s deleted successfully", reportId), "Run 'openstatus status-report list' to see remaining reports")
		},
	}
}
