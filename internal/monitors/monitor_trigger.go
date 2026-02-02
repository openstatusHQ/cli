package monitors

import (
	"context"
	"fmt"
	"net/http"

	monitorv1 "buf.build/gen/go/openstatus/api/protocolbuffers/go/openstatus/monitor/v1"
	"buf.build/gen/go/openstatus/api/connectrpc/gosimple/openstatus/monitor/v1/monitorv1connect"
	"github.com/urfave/cli/v3"
)

// TriggerMonitor triggers a monitor using the SDK
func TriggerMonitor(client monitorv1connect.MonitorServiceClient, monitorId string) error {
	if monitorId == "" {
		return fmt.Errorf("Monitor ID is required")
	}
	fmt.Println("Waiting for the result...")

	_, err := client.TriggerMonitor(context.Background(), &monitorv1.TriggerMonitorRequest{
		Id: monitorId,
	})
	if err != nil {
		return fmt.Errorf("failed to trigger monitor: %w", err)
	}

	fmt.Printf("Check triggered successfully\n")
	return nil
}

// TriggerMonitorWithHTTPClient is a convenience function that creates a client and triggers a monitor
func TriggerMonitorWithHTTPClient(httpClient *http.Client, apiKey string, monitorId string) error {
	client := NewMonitorClientWithHTTPClient(httpClient, apiKey)
	return TriggerMonitor(client, monitorId)
}

func GetMonitorsTriggerCmd() *cli.Command {
	monitorsCmd := cli.Command{
		Name:        "trigger",
		Usage:       "Trigger a monitor execution",
		UsageText:   "openstatus monitors trigger [MonitorId] [options]",
		Description: "Trigger a monitor execution on demand. This command allows you to launch your tests on demand.",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "access-token",
				Usage:    "OpenStatus API Access Token",
				Aliases:  []string{"t"},
				Sources:  cli.EnvVars("OPENSTATUS_API_TOKEN"),
				Required: true,
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			monitorId := cmd.Args().Get(0)
			client := NewMonitorClient(cmd.String("access-token"))
			err := TriggerMonitor(client, monitorId)
			if err != nil {
				return cli.Exit("Failed to trigger monitor", 1)
			}
			return nil
		},
	}
	return &monitorsCmd
}
