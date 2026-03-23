package monitors

import (
	"context"
	"fmt"
	"net/http"

	monitorv1 "buf.build/gen/go/openstatus/api/protocolbuffers/go/openstatus/monitor/v1"
	"buf.build/gen/go/openstatus/api/connectrpc/gosimple/openstatus/monitor/v1/monitorv1connect"
	"github.com/openstatusHQ/cli/internal/auth"
	output "github.com/openstatusHQ/cli/internal/cli"
	"github.com/fatih/color"
	"github.com/rodaine/table"
	"github.com/urfave/cli/v3"
)

type monitorListEntry struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	URL  string `json:"url"`
	Kind string `json:"kind"`
}

func ListMonitors(ctx context.Context, client monitorv1connect.MonitorServiceClient, showAll bool, s *output.Spinner) error {
	resp, err := client.ListMonitors(ctx, &monitorv1.ListMonitorsRequest{})
	output.StopSpinner(s)
	if err != nil {
		return output.FormatError(err, "monitors", "")
	}

	var entries []monitorListEntry

	for _, monitor := range resp.GetHttpMonitors() {
		if monitor.GetActive() || showAll {
			entries = append(entries, monitorListEntry{
				ID:   monitor.GetId(),
				Name: monitor.GetName(),
				URL:  monitor.GetUrl(),
				Kind: "http",
			})
		}
	}

	for _, monitor := range resp.GetTcpMonitors() {
		if monitor.GetActive() || showAll {
			entries = append(entries, monitorListEntry{
				ID:   monitor.GetId(),
				Name: monitor.GetName(),
				URL:  monitor.GetUri(),
				Kind: "tcp",
			})
		}
	}

	for _, monitor := range resp.GetDnsMonitors() {
		if monitor.GetActive() || showAll {
			entries = append(entries, monitorListEntry{
				ID:   monitor.GetId(),
				Name: monitor.GetName(),
				URL:  monitor.GetUri(),
				Kind: "dns",
			})
		}
	}

	if output.IsJSONOutput() {
		return output.PrintJSON(entries)
	}

	headerFmt := color.New(color.FgGreen, color.Underline).SprintfFunc()
	columnFmt := color.New(color.FgYellow).SprintfFunc()

	tbl := table.New("ID", "Name", "Url", "Kind")
	tbl.WithHeaderFormatter(headerFmt).WithFirstColumnFormatter(columnFmt)

	for _, e := range entries {
		tbl.AddRow(e.ID, e.Name, e.URL, e.Kind)
	}

	tbl.Print()

	return nil
}

func ListMonitorsWithHTTPClient(ctx context.Context, httpClient *http.Client, apiKey string) error {
	client := NewMonitorClientWithHTTPClient(httpClient, apiKey)
	return ListMonitors(ctx, client, false, nil)
}

func GetMonitorsListCmd() *cli.Command {
	monitorsListCmd := cli.Command{
		Name:  "list",
		Usage: "List all monitors",
		Description: `List all monitors. The list shows all your monitors attached to your workspace.
It displays the ID, name, URL, and kind of each monitor.`,
		UsageText: `openstatus monitors list
  openstatus monitors list --all`,
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:  "all",
				Usage: "List all monitors including inactive ones",
			},
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
			if !output.IsQuiet() && !output.IsJSONOutput() {
				fmt.Println("List of all monitors")
			}
			s := output.StartSpinner("Fetching monitors...")
			client := NewMonitorClient(apiKey)
			err = ListMonitors(ctx, client, cmd.Bool("all"), s)
			if err != nil {
				return cli.Exit(err.Error(), 1)
			}
			return nil
		},
	}
	return &monitorsListCmd
}
