package monitors

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"

	monitorv1 "buf.build/gen/go/openstatus/api/protocolbuffers/go/openstatus/monitor/v1"
	"github.com/openstatusHQ/cli/internal/api"
	"github.com/openstatusHQ/cli/internal/auth"
	output "github.com/openstatusHQ/cli/internal/cli"
	"github.com/logrusorgru/aurora/v4"
	"github.com/olekukonko/tablewriter"
	"github.com/olekukonko/tablewriter/renderer"
	"github.com/olekukonko/tablewriter/tw"
	"github.com/urfave/cli/v3"
)

func GetMonitorInfo(ctx context.Context, httpClient *http.Client, apiKey string, monitorId string, s *output.Spinner) error {

	if monitorId == "" {
		output.StopSpinner(s)
		fmt.Fprintln(os.Stderr, "Usage: openstatus monitors info <monitor-id>")
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "Example: openstatus monitors info 12345")
		return fmt.Errorf("monitor ID is required")
	}

	client := NewMonitorClientWithHTTPClient(httpClient, apiKey)

	req := &monitorv1.GetMonitorRequest{
		Id: monitorId,
	}

	resp, err := client.GetMonitor(ctx, req)
	output.StopSpinner(s)
	if err != nil {
		return output.FormatError(err, "monitor", monitorId)
	}

	monitorConfig := resp.GetMonitor()
	var monitor Monitor
	var regions []monitorv1.Region
	switch {
	case monitorConfig.HasHttp():
		monitor, err = httpMonitorToLocal(monitorConfig.GetHttp())
		if err != nil {
			return err
		}
		regions = monitorConfig.GetHttp().GetRegions()
	case monitorConfig.HasTcp():
		monitor, err = tcpMonitorToLocal(monitorConfig.GetTcp())
		if err != nil {
			return err
		}
		regions = monitorConfig.GetTcp().GetRegions()
	default:
		if monitorConfig.HasDns() {
			return fmt.Errorf("DNS monitors are not yet supported in the CLI. Monitor ID: %s", monitorId)
		}
		return fmt.Errorf("unknown monitor type for monitor ID: %s", monitorId)
	}

	if output.IsJSONOutput() {
		return output.PrintJSON(monitor)
	}

	fmt.Println(aurora.Bold("Monitor:"))
	table := tablewriter.NewTable(os.Stdout,
		tablewriter.WithRenderer(renderer.NewBlueprint()),
		tablewriter.WithRendition(tw.Rendition{
			Symbols: tw.NewSymbolCustom("custom").WithColumn("="),
			Borders: tw.Border{
				Top:    tw.Off,
				Left:   tw.Off,
				Right:  tw.Off,
				Bottom: tw.Off,
			},
			Settings: tw.Settings{
				Lines: tw.Lines{
					ShowHeaderLine: tw.Off,
					ShowFooterLine: tw.On,
				},
				Separators: tw.Separators{
					BetweenRows:    tw.Off,
					BetweenColumns: tw.On,
				},
			},
		}),
		tablewriter.WithRowAlignment(tw.AlignLeft),
		tablewriter.WithHeaderAlignment(tw.AlignLeft),
	)

	data := [][]string{
		{"ID", fmt.Sprintf("%d", monitor.ID)},
		{"Name", monitor.Name},
		{"Description", monitor.Description},
		{"Endpoint", monitor.URL},
	}
	if monitor.Method != "" {
		data = append(data, []string{"Method", monitor.Method})
	}

	data = append(data, []string{"Frequency", monitor.Periodicity})

	regionGroups := groupRegionsByProvider(regions)
	providers := []string{"Fly.io", "Koyeb", "Railway"}
	for _, provider := range providers {
		codes := regionGroups[provider]
		if len(codes) > 0 {
			data = append(data, []string{fmt.Sprintf("Locations (%s)", provider), strings.Join(codes, ", ")})
		}
	}

	data = append(data, []string{"Active", fmt.Sprintf("%t", monitor.Active)})
	data = append(data, []string{"Public", fmt.Sprintf("%t", monitor.Public)})

	if monitor.Timeout > 0 {
		data = append(data, []string{"Timeout", fmt.Sprintf("%d ms", monitor.Timeout)})
	}
	if monitor.DegradedAfter > 0 {
		data = append(data, []string{"Degraded After", fmt.Sprintf("%d", monitor.DegradedAfter)})
	}

	if monitor.Body != "" {
		s := monitor.Body
		if len(s) > 40 {
			s = s[:40]
		}
		data = append(data, []string{"Body", s})
	}
	table.Bulk(data)
	table.Render()

	return nil
}

func GetMonitorInfoCmd() *cli.Command {
	monitorInfoCmd := cli.Command{
		Name:  "info",
		Usage: "Get a monitor information",
		UsageText: `openstatus monitors info <MonitorID>
  openstatus monitors info 12345`,
		Description: "Fetch the monitor information. The monitor information includes details such as name, description, endpoint, method, frequency, locations, active status, public status, timeout, degraded after, and body.",
		Action: func(ctx context.Context, cmd *cli.Command) error {
			apiKey, err := auth.ResolveAccessToken(cmd)
			if err != nil {
				return cli.Exit(err.Error(), 1)
			}
			monitorId := cmd.Args().Get(0)
			s := output.StartSpinner("Fetching monitor details...")
			err = GetMonitorInfo(ctx, api.DefaultHTTPClient, apiKey, monitorId, s)
			if err != nil {
				return cli.Exit(err.Error(), 1)
			}
			return nil
		},
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "access-token",
				Usage:   "OpenStatus API Access Token",
				Aliases: []string{"t"},
				Sources: cli.EnvVars("OPENSTATUS_API_TOKEN"),
			}},
	}
	return &monitorInfoCmd
}
