package monitors

import (
	"context"
	"fmt"
	"net/http"

	monitorv1 "buf.build/gen/go/openstatus/api/protocolbuffers/go/openstatus/monitor/v1"
	"buf.build/gen/go/openstatus/api/connectrpc/gosimple/openstatus/monitor/v1/monitorv1connect"
	"github.com/fatih/color"
	"github.com/rodaine/table"
	"github.com/urfave/cli/v3"
)

var allMonitor bool

// ListMonitors fetches and displays all monitors using the SDK
func ListMonitors(client monitorv1connect.MonitorServiceClient) error {
	resp, err := client.ListMonitors(context.Background(), &monitorv1.ListMonitorsRequest{})
	if err != nil {
		return fmt.Errorf("failed to list monitors: %w", err)
	}

	headerFmt := color.New(color.FgGreen, color.Underline).SprintfFunc()
	columnFmt := color.New(color.FgYellow).SprintfFunc()

	tbl := table.New("ID", "Name", "Url")
	tbl.WithHeaderFormatter(headerFmt).WithFirstColumnFormatter(columnFmt)

	// Add HTTP monitors
	for _, monitor := range resp.GetHttpMonitors() {
		if monitor.GetActive() || allMonitor {
			tbl.AddRow(monitor.GetId(), monitor.GetName(), monitor.GetUrl())
		}
	}

	// Add TCP monitors
	for _, monitor := range resp.GetTcpMonitors() {
		if monitor.GetActive() || allMonitor {
			tbl.AddRow(monitor.GetId(), monitor.GetName(), monitor.GetUri())
		}
	}

	// Add DNS monitors
	for _, monitor := range resp.GetDnsMonitors() {
		if monitor.GetActive() || allMonitor {
			tbl.AddRow(monitor.GetId(), monitor.GetName(), monitor.GetUri())
		}
	}

	tbl.Print()

	return nil
}

// ListMonitorsWithHTTPClient is a convenience function that creates a client and lists monitors
func ListMonitorsWithHTTPClient(httpClient *http.Client, apiKey string) error {
	client := NewMonitorClientWithHTTPClient(httpClient, apiKey)
	return ListMonitors(client)
}

func GetMonitorsListCmd() *cli.Command {
	monitorsListCmd := cli.Command{
		Name:        "list",
		Usage:       "List all monitors",
		Description: "List all monitors. The list shows all your monitors attached to your workspace. It displays the ID, name, and URL of each monitor.",
		UsageText:   "openstatus monitors list [options]",
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:        "all",
				Usage:       "List all monitors including inactive ones",
				Destination: &allMonitor,
			},
			&cli.StringFlag{
				Name:     "access-token",
				Usage:    "OpenStatus API Access Token",
				Aliases:  []string{"t"},
				Sources:  cli.EnvVars("OPENSTATUS_API_TOKEN"),
				Required: true,
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			fmt.Println("List of all monitors")
			client := NewMonitorClient(cmd.String("access-token"))
			err := ListMonitors(client)
			if err != nil {
				return cli.Exit("Failed to list monitors", 1)
			}
			return nil
		},
	}
	return &monitorsListCmd
}
