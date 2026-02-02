package monitors

import (
	"context"
	"fmt"
	"net/http"

	monitorv1 "buf.build/gen/go/openstatus/api/protocolbuffers/go/openstatus/monitor/v1"
	"buf.build/gen/go/openstatus/api/connectrpc/gosimple/openstatus/monitor/v1/monitorv1connect"
	confirmation "github.com/openstatusHQ/cli/internal/cli"
	"github.com/urfave/cli/v3"
)

// DeleteMonitor deletes a monitor using the SDK
func DeleteMonitor(client monitorv1connect.MonitorServiceClient, monitorId string) error {
	if monitorId == "" {
		return fmt.Errorf("Monitor ID is required")
	}

	_, err := client.DeleteMonitor(context.Background(), &monitorv1.DeleteMonitorRequest{
		Id: monitorId,
	})
	if err != nil {
		return fmt.Errorf("failed to delete monitor: %w", err)
	}

	return nil
}

// DeleteMonitorWithHTTPClient is a convenience function that creates a client and deletes a monitor
func DeleteMonitorWithHTTPClient(httpClient *http.Client, apiKey string, monitorId string) error {
	client := NewMonitorClientWithHTTPClient(httpClient, apiKey)
	return DeleteMonitor(client, monitorId)
}

func GetMonitorDeleteCmd() *cli.Command {
	monitorsCmd := cli.Command{
		Name:            "delete",
		Usage:           "Delete a monitor",
		Hidden:          true,
		HideHelpCommand: true,
		HideHelp:        true,
		UsageText:       "openstatus monitors delete [MonitorID] [options]",

		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "access-token",
				Usage:    "OpenStatus API Access Token",
				Aliases:  []string{"t"},
				Sources:  cli.EnvVars("OPENSTATUS_API_TOKEN"),
				Required: true,
			},
			&cli.BoolFlag{
				Name:     "auto-accept",
				Usage:    "Automatically accept the prompt",
				Aliases:  []string{"y"},
				Required: false,
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			monitorId := cmd.Args().Get(0)

			if !cmd.Bool("auto-accept") {
				confirmed, err := confirmation.AskForConfirmation(fmt.Sprintf("You are about to delete monitor: %s, do you want to continue", monitorId))
				if err != nil {
					return cli.Exit(fmt.Sprintf("Failed to read input: %v", err), 1)
				}
				if !confirmed {
					return nil
				}
			}
			client := NewMonitorClient(cmd.String("access-token"))
			err := DeleteMonitor(client, monitorId)
			if err != nil {
				return cli.Exit("Failed to delete monitor", 1)
			}
			fmt.Printf("Monitor deleted successfully\n")
			return nil
		},
	}
	return &monitorsCmd
}
