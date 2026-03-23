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

// TriggerMonitor triggers a monitor using the SDK
func TriggerMonitor(ctx context.Context, client monitorv1connect.MonitorServiceClient, monitorId string, s *output.Spinner) error {
	if monitorId == "" {
		output.StopSpinner(s)
		fmt.Fprintln(os.Stderr, "Usage: openstatus monitors trigger <monitor-id>")
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "Example: openstatus monitors trigger 12345")
		return fmt.Errorf("monitor ID is required")
	}

	_, err := client.TriggerMonitor(ctx, &monitorv1.TriggerMonitorRequest{
		Id: monitorId,
	})
	output.StopSpinner(s)
	if err != nil {
		return output.FormatError(err, "monitor", monitorId)
	}

	if output.IsJSONOutput() {
		return output.PrintJSON(map[string]string{"status": "triggered", "monitor_id": monitorId})
	}
	fmt.Printf("Check triggered successfully\n")
	return nil
}

// TriggerMonitorWithHTTPClient is a convenience function that creates a client and triggers a monitor
func TriggerMonitorWithHTTPClient(ctx context.Context, httpClient *http.Client, apiKey string, monitorId string) error {
	client := NewMonitorClientWithHTTPClient(httpClient, apiKey)
	return TriggerMonitor(ctx, client, monitorId, nil)
}

func GetMonitorsTriggerCmd() *cli.Command {
	monitorsCmd := cli.Command{
		Name:        "trigger",
		Usage:       "Trigger a monitor execution",
		UsageText: `openstatus monitors trigger <MonitorID>
  openstatus monitors trigger 12345`,
		Description: "Trigger a monitor execution on demand. This command allows you to launch your tests on demand.",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "access-token",
				Usage:   "OpenStatus API Access Token",
				Aliases: []string{"t"},
				Sources: cli.EnvVars("OPENSTATUS_API_TOKEN"),
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			apiKey, err := auth.ResolveAccessToken(cmd)
			if err != nil {
				return cli.Exit(err.Error(), 1)
			}
			monitorId := cmd.Args().Get(0)
			s := output.StartSpinner("Triggering monitor...")
			client := NewMonitorClient(apiKey)
			err = TriggerMonitor(ctx, client, monitorId, s)
			if err != nil {
				return cli.Exit(err.Error(), 1)
			}
			return nil
		},
	}
	return &monitorsCmd
}
