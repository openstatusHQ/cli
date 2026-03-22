package statusreport

import (
	"context"
	"fmt"
	"net/http"

	status_reportv1 "buf.build/gen/go/openstatus/api/protocolbuffers/go/openstatus/status_report/v1"
	"buf.build/gen/go/openstatus/api/connectrpc/gosimple/openstatus/status_report/v1/status_reportv1connect"
	confirmation "github.com/openstatusHQ/cli/internal/cli"
	"github.com/urfave/cli/v3"
)

func DeleteStatusReport(client status_reportv1connect.StatusReportServiceClient, reportId string) error {
	if reportId == "" {
		return fmt.Errorf("report ID is required")
	}

	_, err := client.DeleteStatusReport(context.Background(), &status_reportv1.DeleteStatusReportRequest{
		Id: reportId,
	})
	if err != nil {
		return fmt.Errorf("failed to delete status report: %w", err)
	}

	return nil
}

func DeleteStatusReportWithHTTPClient(httpClient *http.Client, apiKey string, reportId string) error {
	client := NewStatusReportClientWithHTTPClient(httpClient, apiKey)
	return DeleteStatusReport(client, reportId)
}

func GetStatusReportDeleteCmd() *cli.Command {
	return &cli.Command{
		Name:      "delete",
		Usage:     "Delete a status report",
		UsageText: "openstatus status-report delete <ReportID>",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "access-token",
				Usage:    "OpenStatus API Access Token",
				Aliases:  []string{"t"},
				Sources:  cli.EnvVars("OPENSTATUS_API_TOKEN"),
				Required: true,
			},
			&cli.BoolFlag{
				Name:    "auto-accept",
				Usage:   "Automatically accept the prompt",
				Aliases: []string{"y"},
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			reportId := cmd.Args().Get(0)

			if !cmd.Bool("auto-accept") {
				confirmed, err := confirmation.AskForConfirmation(fmt.Sprintf("You are about to delete status report: %s, do you want to continue", reportId))
				if err != nil {
					return cli.Exit(fmt.Sprintf("Failed to read input: %v", err), 1)
				}
				if !confirmed {
					return nil
				}
			}

			client := NewStatusReportClient(cmd.String("access-token"))
			err := DeleteStatusReport(client, reportId)
			if err != nil {
				return cli.Exit(err.Error(), 1)
			}
			fmt.Printf("Status report %s deleted successfully\n", reportId)
			return nil
		},
	}
}
