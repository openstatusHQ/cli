package maintenance

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"

	maintenancev1 "buf.build/gen/go/openstatus/api/protocolbuffers/go/openstatus/maintenance/v1"
	"buf.build/gen/go/openstatus/api/connectrpc/gosimple/openstatus/maintenance/v1/maintenancev1connect"
	"github.com/openstatusHQ/cli/internal/auth"
	output "github.com/openstatusHQ/cli/internal/cli"
	"github.com/urfave/cli/v3"
)

func UpdateMaintenance(ctx context.Context, client maintenancev1connect.MaintenanceServiceClient, id, title, message, from, to string, componentIds []string, hasTitle, hasMessage, hasFrom, hasTo, hasComponents bool) error {
	if id == "" {
		fmt.Fprintln(os.Stderr, "Usage: openstatus maintenance update <maintenance-id> [--title ...] [--message ...] [--from ...] [--to ...]")
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "Example: openstatus maintenance update 12345 --title \"Updated title\"")
		return fmt.Errorf("maintenance ID is required")
	}

	if !hasTitle && !hasMessage && !hasFrom && !hasTo && !hasComponents {
		return fmt.Errorf("at least one of --title, --message, --from, --to, or --component-ids must be provided")
	}

	req := &maintenancev1.UpdateMaintenanceRequest{
		Id: id,
	}

	if hasTitle {
		req.SetTitle(title)
	}
	if hasMessage {
		req.SetMessage(message)
	}
	if hasFrom {
		req.SetFrom(from)
	}
	if hasTo {
		req.SetTo(to)
	}
	if hasComponents {
		req.SetPageComponentIds(componentIds)
	}

	_, err := client.UpdateMaintenance(ctx, req)
	if err != nil {
		return output.FormatError(err, "maintenance", id)
	}

	return nil
}

func UpdateMaintenanceWithHTTPClient(ctx context.Context, httpClient *http.Client, apiKey string, id, title, message, from, to string, componentIds []string, hasTitle, hasMessage, hasFrom, hasTo, hasComponents bool) error {
	client := NewMaintenanceClientWithHTTPClient(httpClient, apiKey)
	return UpdateMaintenance(ctx, client, id, title, message, from, to, componentIds, hasTitle, hasMessage, hasFrom, hasTo, hasComponents)
}

func GetMaintenanceUpdateCmd() *cli.Command {
	return &cli.Command{
		Name:      "update",
		Usage:     "Update a maintenance window",
		UsageText: "openstatus maintenance update <MaintenanceID> [--title \"New title\"] [--message \"New message\"] [--from ...] [--to ...]",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "access-token",
				Usage:   "OpenStatus API Access Token",
				Aliases: []string{"t"},
				Sources: cli.EnvVars("OPENSTATUS_API_TOKEN"),
			},
			&cli.StringFlag{
				Name:  "title",
				Usage: "New title for the maintenance",
			},
			&cli.StringFlag{
				Name:  "message",
				Usage: "New message for the maintenance",
			},
			&cli.StringFlag{
				Name:  "from",
				Usage: "New start time (RFC 3339 format)",
			},
			&cli.StringFlag{
				Name:  "to",
				Usage: "New end time (RFC 3339 format)",
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
			maintenanceId := cmd.Args().Get(0)

			hasTitle := cmd.IsSet("title")
			hasMessage := cmd.IsSet("message")
			hasFrom := cmd.IsSet("from")
			hasTo := cmd.IsSet("to")
			hasComponents := cmd.IsSet("component-ids") && cmd.String("component-ids") != ""

			var componentIds []string
			if ids := cmd.String("component-ids"); ids != "" {
				componentIds = strings.Split(ids, ",")
			}

			client := NewMaintenanceClient(apiKey)
			s := output.StartSpinner("Updating maintenance...")
			err = UpdateMaintenance(ctx, client, maintenanceId, cmd.String("title"), cmd.String("message"), cmd.String("from"), cmd.String("to"), componentIds, hasTitle, hasMessage, hasFrom, hasTo, hasComponents)
			output.StopSpinner(s)
			if err != nil {
				return cli.Exit(err.Error(), 1)
			}
			fmt.Printf("Maintenance %s updated successfully\n", maintenanceId)
			fmt.Println("Run 'openstatus maintenance info " + maintenanceId + "' to see the maintenance")
			return nil
		},
	}
}
