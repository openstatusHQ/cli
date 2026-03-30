package statusreport

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	status_reportv1 "buf.build/gen/go/openstatus/api/protocolbuffers/go/openstatus/status_report/v1"
	"buf.build/gen/go/openstatus/api/connectrpc/gosimple/openstatus/status_report/v1/status_reportv1connect"
	"github.com/openstatusHQ/cli/internal/auth"
	output "github.com/openstatusHQ/cli/internal/cli"
	"github.com/urfave/cli/v3"
)

func CreateStatusReport(ctx context.Context, client status_reportv1connect.StatusReportServiceClient, title, status, message, date, pageId string, componentIds []string, notify bool) (string, error) {
	sdkStatus, err := statusToSDK(status)
	if err != nil {
		return "", err
	}

	req := &status_reportv1.CreateStatusReportRequest{
		Title:   title,
		Status:  sdkStatus,
		Message: message,
		Date:    date,
		PageId:  pageId,
	}

	if len(componentIds) > 0 {
		req.SetPageComponentIds(componentIds)
	}

	if notify {
		req.SetNotify(true)
	}

	resp, err := client.CreateStatusReport(ctx, req)
	if err != nil {
		return "", output.FormatError(err, "status-report", "")
	}

	return resp.GetStatusReport().GetId(), nil
}

func CreateStatusReportWithHTTPClient(ctx context.Context, httpClient *http.Client, apiKey string, title, status, message, date, pageId string, componentIds []string, notify bool) (string, error) {
	client := NewStatusReportClientWithHTTPClient(httpClient, apiKey)
	return CreateStatusReport(ctx, client, title, status, message, date, pageId, componentIds, notify)
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
				Usage: "Comma-separated page component IDs",
			},
			&cli.BoolFlag{
				Name:  "notify",
				Usage: "Notify subscribers about this status report",
			},
			&cli.StringFlag{
				Name:  "date",
				Usage: "Date when the event occurred (RFC 3339 format, defaults to now)",
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

			client := NewStatusReportClient(apiKey)
			s := output.StartSpinner("Creating status report...")
			id, err := CreateStatusReport(
				ctx,
				client,
				inputs.Title,
				inputs.Status,
				inputs.Message,
				date,
				inputs.PageID,
				inputs.ComponentIDs,
				inputs.Notify,
			)
			output.StopSpinner(s)
			if err != nil {
				return cli.Exit(err.Error(), 1)
			}

			fmt.Printf("Status report created successfully (ID: %s)\n", id)
			fmt.Printf("To add updates, run: openstatus status-report add-update %s --status identified --message '...'\n", id)
			return nil
		},
	}
}
