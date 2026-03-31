package maintenance

import (
	"context"
	"fmt"
	"net/http"
	"os"

	maintenancev1 "buf.build/gen/go/openstatus/api/protocolbuffers/go/openstatus/maintenance/v1"
	"buf.build/gen/go/openstatus/api/connectrpc/gosimple/openstatus/maintenance/v1/maintenancev1connect"
	"github.com/openstatusHQ/cli/internal/auth"
	output "github.com/openstatusHQ/cli/internal/cli"
	"github.com/urfave/cli/v3"
)

func DeleteMaintenance(ctx context.Context, client maintenancev1connect.MaintenanceServiceClient, maintenanceId string) error {
	if maintenanceId == "" {
		fmt.Fprintln(os.Stderr, "Usage: openstatus maintenance delete <maintenance-id>")
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "Example: openstatus maintenance delete 12345")
		return fmt.Errorf("maintenance ID is required")
	}

	_, err := client.DeleteMaintenance(ctx, &maintenancev1.DeleteMaintenanceRequest{
		Id: maintenanceId,
	})
	if err != nil {
		return output.FormatError(err, "maintenance", maintenanceId)
	}

	return nil
}

func DeleteMaintenanceWithHTTPClient(ctx context.Context, httpClient *http.Client, apiKey string, maintenanceId string) error {
	client := NewMaintenanceClientWithHTTPClient(httpClient, apiKey)
	return DeleteMaintenance(ctx, client, maintenanceId)
}

func GetMaintenanceDeleteCmd() *cli.Command {
	return &cli.Command{
		Name:      "delete",
		Usage:     "Delete a maintenance window",
		UsageText: `openstatus maintenance delete <MaintenanceID>
  openstatus maintenance delete 12345 -y`,
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
			apiKey, err := auth.ResolveAccessToken(cmd)
			if err != nil {
				return cli.Exit(err.Error(), 1)
			}
			maintenanceId := cmd.Args().Get(0)
			if maintenanceId == "" {
				fmt.Fprintln(os.Stderr, "Usage: openstatus maintenance delete <maintenance-id>")
				return cli.Exit("maintenance ID is required", 1)
			}

			if !cmd.Bool("auto-accept") {
				confirmed, err := output.AskForConfirmation(fmt.Sprintf("You are about to delete maintenance: %s, do you want to continue", maintenanceId))
				if err != nil {
					return cli.Exit(fmt.Sprintf("Failed to read input: %v", err), 1)
				}
				if !confirmed {
					return nil
				}
			}

			client := NewMaintenanceClient(apiKey)
			s := output.StartSpinner("Deleting maintenance...")
			err = DeleteMaintenance(ctx, client, maintenanceId)
			output.StopSpinner(s)
			if err != nil {
				return cli.Exit(err.Error(), 1)
			}
			fmt.Printf("Maintenance %s deleted successfully\n", maintenanceId)
			fmt.Println("Run 'openstatus maintenance list' to see remaining maintenances")
			return nil
		},
	}
}
