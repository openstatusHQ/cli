package maintenance

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	maintenancev1 "buf.build/gen/go/openstatus/api/protocolbuffers/go/openstatus/maintenance/v1"
	"buf.build/gen/go/openstatus/api/connectrpc/gosimple/openstatus/maintenance/v1/maintenancev1connect"
	"github.com/openstatusHQ/cli/internal/auth"
	output "github.com/openstatusHQ/cli/internal/cli"
	"github.com/urfave/cli/v3"
)

func CreateMaintenance(ctx context.Context, client maintenancev1connect.MaintenanceServiceClient, title, message, from, to, pageId string, componentIds []string, notify bool) (string, error) {
	req := &maintenancev1.CreateMaintenanceRequest{
		Title:   title,
		Message: message,
		From:    from,
		To:      to,
		PageId:  pageId,
	}

	if len(componentIds) > 0 {
		req.SetPageComponentIds(componentIds)
	}

	if notify {
		req.SetNotify(true)
	}

	resp, err := client.CreateMaintenance(ctx, req)
	if err != nil {
		return "", output.FormatError(err, "maintenance", "")
	}

	return resp.GetMaintenance().GetId(), nil
}

func CreateMaintenanceWithHTTPClient(ctx context.Context, httpClient *http.Client, apiKey string, title, message, from, to, pageId string, componentIds []string, notify bool) (string, error) {
	client := NewMaintenanceClientWithHTTPClient(httpClient, apiKey)
	return CreateMaintenance(ctx, client, title, message, from, to, pageId, componentIds, notify)
}

func GetMaintenanceCreateCmd() *cli.Command {
	return &cli.Command{
		Name:      "create",
		Usage:     "Create a maintenance window",
		UsageText: "openstatus maintenance create --title \"DB Migration\" --message \"Upgrading database\" --from 2026-04-01T10:00:00Z --to 2026-04-01T12:00:00Z --page-id 123",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "access-token",
				Usage:   "OpenStatus API Access Token",
				Aliases: []string{"t"},
				Sources: cli.EnvVars("OPENSTATUS_API_TOKEN"),
			},
			&cli.StringFlag{
				Name:  "title",
				Usage: "Title of the maintenance",
			},
			&cli.StringFlag{
				Name:  "message",
				Usage: "Message describing the maintenance",
			},
			&cli.StringFlag{
				Name:  "from",
				Usage: "Start time of the maintenance window (RFC 3339 format)",
			},
			&cli.StringFlag{
				Name:  "to",
				Usage: "End time of the maintenance window (RFC 3339 format)",
			},
			&cli.StringFlag{
				Name:  "page-id",
				Usage: "Status page ID to associate with this maintenance",
			},
			&cli.StringFlag{
				Name:  "component-ids",
				Usage: "Comma-separated page component IDs",
			},
			&cli.BoolFlag{
				Name:  "notify",
				Usage: "Notify subscribers about this maintenance",
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
				Message: cmd.String("message"),
				From:    cmd.String("from"),
				To:      cmd.String("to"),
				Notify:  cmd.Bool("notify"),
			}
			if ids := cmd.String("component-ids"); ids != "" {
				inputs.ComponentIDs = strings.Split(ids, ",")
			}

			needsWizard := inputs.Title == "" || inputs.Message == "" ||
				inputs.From == "" || inputs.To == "" || inputs.PageID == ""

			if needsWizard {
				if output.IsJSONOutput() || !output.IsStdinTerminal() {
					var missing []string
					if inputs.Title == "" {
						missing = append(missing, "--title")
					}
					if inputs.Message == "" {
						missing = append(missing, "--message")
					}
					if inputs.From == "" {
						missing = append(missing, "--from")
					}
					if inputs.To == "" {
						missing = append(missing, "--to")
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

			client := NewMaintenanceClient(apiKey)
			s := output.StartSpinner("Creating maintenance...")
			id, err := CreateMaintenance(
				ctx,
				client,
				inputs.Title,
				inputs.Message,
				inputs.From,
				inputs.To,
				inputs.PageID,
				inputs.ComponentIDs,
				inputs.Notify,
			)
			output.StopSpinner(s)
			if err != nil {
				return cli.Exit(err.Error(), 1)
			}

			fmt.Printf("Maintenance created successfully (ID: %s)\n", id)
			fmt.Printf("Run 'openstatus maintenance info %s' to see details\n", id)
			return nil
		},
	}
}
