package monitors

import (
	"context"
	"fmt"
	"net/http"
	"os"

	monitorv1 "buf.build/gen/go/openstatus/api/protocolbuffers/go/openstatus/monitor/v1"
	"buf.build/gen/go/openstatus/api/connectrpc/gosimple/openstatus/monitor/v1/monitorv1connect"
	"github.com/openstatusHQ/cli/internal/auth"
	output "github.com/openstatusHQ/cli/internal/cli"
	"github.com/urfave/cli/v3"
)

// DeleteMonitor deletes a monitor using the SDK
func DeleteMonitor(ctx context.Context, client monitorv1connect.MonitorServiceClient, monitorId string) error {
	if monitorId == "" {
		fmt.Fprintln(os.Stderr, "Usage: openstatus monitors delete <monitor-id>")
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "Example: openstatus monitors delete 12345")
		return fmt.Errorf("monitor ID is required")
	}

	_, err := client.DeleteMonitor(ctx, &monitorv1.DeleteMonitorRequest{
		Id: monitorId,
	})
	if err != nil {
		return output.FormatError(err, "monitor", monitorId)
	}

	return nil
}

// DeleteMonitorWithHTTPClient is a convenience function that creates a client and deletes a monitor
func DeleteMonitorWithHTTPClient(ctx context.Context, httpClient *http.Client, apiKey string, monitorId string) error {
	client := NewMonitorClientWithHTTPClient(httpClient, apiKey)
	return DeleteMonitor(ctx, client, monitorId)
}

func GetMonitorDeleteCmd() *cli.Command {
	monitorsCmd := cli.Command{
		Name:            "delete",
		Usage:           "Delete a monitor",
		Hidden:          true,
		HideHelpCommand: true,
		HideHelp:        true,
		UsageText: `openstatus monitors delete <MonitorID>
  openstatus monitors delete 12345 -y`,

		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "access-token",
				Usage:   "OpenStatus API Access Token",
				Aliases: []string{"t"},
				Sources: cli.EnvVars("OPENSTATUS_API_TOKEN"),
			},
			&cli.BoolFlag{
				Name:     "auto-accept",
				Usage:    "Automatically accept the prompt",
				Aliases:  []string{"y"},
				Required: false,
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			apiKey, err := auth.ResolveAccessToken(cmd)
			if err != nil {
				return cli.Exit(err.Error(), 1)
			}
			monitorId := cmd.Args().Get(0)

			if !cmd.Bool("auto-accept") {
				confirmed, err := output.AskForConfirmation(fmt.Sprintf("You are about to delete monitor: %s, do you want to continue", monitorId))
				if err != nil {
					return cli.Exit(fmt.Sprintf("Failed to read input: %v", err), 1)
				}
				if !confirmed {
					return nil
				}
			}
			client := NewMonitorClient(apiKey)
			s := output.StartSpinner("Deleting monitor...")
			err = DeleteMonitor(ctx, client, monitorId)
			output.StopSpinner(s)
			if err != nil {
				return cli.Exit(err.Error(), 1)
			}
			fmt.Printf("Monitor deleted successfully\n")
			fmt.Println("Run 'openstatus monitors list' to see remaining monitors")
			return nil
		},
	}
	return &monitorsCmd
}
