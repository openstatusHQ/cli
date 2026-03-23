package statusreport

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"

	status_reportv1 "buf.build/gen/go/openstatus/api/protocolbuffers/go/openstatus/status_report/v1"
	"buf.build/gen/go/openstatus/api/connectrpc/gosimple/openstatus/status_report/v1/status_reportv1connect"
	"github.com/openstatusHQ/cli/internal/auth"
	output "github.com/openstatusHQ/cli/internal/cli"
	"github.com/urfave/cli/v3"
)

func UpdateStatusReport(ctx context.Context, client status_reportv1connect.StatusReportServiceClient, reportId string, title string, componentIds []string, hasTitle bool, hasComponents bool) error {
	if reportId == "" {
		fmt.Fprintln(os.Stderr, "Usage: openstatus status-report update <report-id> [--title ...] [--component-ids ...]")
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "Example: openstatus status-report update 12345 --title \"Updated title\"")
		return fmt.Errorf("report ID is required")
	}

	if !hasTitle && !hasComponents {
		return fmt.Errorf("at least one of --title or --component-ids must be provided")
	}

	req := &status_reportv1.UpdateStatusReportRequest{
		Id: reportId,
	}

	if hasTitle {
		req.SetTitle(title)
	}

	if hasComponents {
		req.SetPageComponentIds(componentIds)
	}

	_, err := client.UpdateStatusReport(ctx, req)
	if err != nil {
		return output.FormatError(err, "status-report", reportId)
	}

	return nil
}

func UpdateStatusReportWithHTTPClient(ctx context.Context, httpClient *http.Client, apiKey string, reportId string, title string, componentIds []string, hasTitle bool, hasComponents bool) error {
	client := NewStatusReportClientWithHTTPClient(httpClient, apiKey)
	return UpdateStatusReport(ctx, client, reportId, title, componentIds, hasTitle, hasComponents)
}

func GetStatusReportUpdateCmd() *cli.Command {
	return &cli.Command{
		Name:      "update",
		Usage:     "Update status report metadata",
		UsageText: "openstatus status-report update <ReportID> [--title \"New title\"] [--component-ids id1,id2]",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "access-token",
				Usage:   "OpenStatus API Access Token",
				Aliases: []string{"t"},
				Sources: cli.EnvVars("OPENSTATUS_API_TOKEN"),
			},
			&cli.StringFlag{
				Name:  "title",
				Usage: "New title for the report",
			},
			&cli.StringFlag{
				Name:  "component-ids",
				Usage: "Comma-separated page component IDs (replaces existing list)",
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			apiKey, err := auth.ResolveAccessToken(cmd)
			if err != nil {
				return cli.Exit(err.Error(), 1)
			}
			reportId := cmd.Args().Get(0)

			hasTitle := cmd.IsSet("title")
			hasComponents := cmd.IsSet("component-ids") && cmd.String("component-ids") != ""

			var componentIds []string
			if ids := cmd.String("component-ids"); ids != "" {
				componentIds = strings.Split(ids, ",")
			}

			client := NewStatusReportClient(apiKey)
			s := output.StartSpinner("Updating status report...")
			err = UpdateStatusReport(ctx, client, reportId, cmd.String("title"), componentIds, hasTitle, hasComponents)
			output.StopSpinner(s)
			if err != nil {
				return cli.Exit(err.Error(), 1)
			}
			fmt.Printf("Status report %s updated successfully\n", reportId)
			fmt.Println("Run 'openstatus status-report info " + reportId + "' to see the report")
			return nil
		},
	}
}
