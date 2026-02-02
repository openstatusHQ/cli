package monitors

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"

	monitorv1 "buf.build/gen/go/openstatus/api/protocolbuffers/go/openstatus/monitor/v1"
	"github.com/logrusorgru/aurora/v4"
	"github.com/olekukonko/tablewriter"
	"github.com/olekukonko/tablewriter/renderer"
	"github.com/olekukonko/tablewriter/tw"
	"github.com/urfave/cli/v3"
)

func GetMonitorInfo(httpClient *http.Client, apiKey string, monitorId string) error {

	if monitorId == "" {
		return fmt.Errorf("Monitor ID is required")
	}

	client := NewMonitorClientWithHTTPClient(httpClient, apiKey)

	req := &monitorv1.GetMonitorRequest{
		Id: monitorId,
	}

	resp, err := client.GetMonitor(context.Background(), req)
	if err != nil {
		return fmt.Errorf("failed to get monitor: %w", err)
	}

	monitorConfig := resp.GetMonitor()
	var monitor Monitor
	switch {
	case monitorConfig.HasHttp():
		monitor = httpMonitorToLocal(monitorConfig.GetHttp())
	case monitorConfig.HasTcp():
		monitor = tcpMonitorToLocal(monitorConfig.GetTcp())
	default:
		return fmt.Errorf("unknown monitor type")
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
				Lines: tw.Lines{ // Major internal separator lines
					ShowHeaderLine: tw.Off, // Line after header
					ShowFooterLine: tw.On,  // Line before footer (if footer exists)
				},
				Separators: tw.Separators{ // General row and column separators
					BetweenRows:    tw.Off, // Horizontal lines between data rows
					BetweenColumns: tw.On,  // Vertical lines between columns
				},
			},
		}),
		tablewriter.WithRowAlignment(tw.AlignLeft),    // Common for Markdown
		tablewriter.WithHeaderAlignment(tw.AlignLeft), //
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
	data = append(data, []string{"Locations", strings.Join(monitor.Regions, ",")})
	data = append(data, []string{"Active", fmt.Sprintf("%t", monitor.Active)})
	data = append(data, []string{"Public", fmt.Sprintf("%t", monitor.Public)})

	if monitor.Timeout > 0 {
		data = append(data, []string{"Timeout", fmt.Sprintf("%d ms", monitor.Timeout)})
	}
	if monitor.DegradedAfter > 0 {
		data = append(data, []string{"Degraded After", fmt.Sprintf("%d", monitor.DegradedAfter)})
	}

	if monitor.Body != "" {
		s := fmt.Sprintf("%s", monitor.Body)
		data = append(data, []string{"Body", s[:40]})
	}
	table.Bulk(data)
	table.Render()

	return nil
}

func GetMonitorInfoCmd() *cli.Command {
	monitorInfoCmd := cli.Command{
		Name:        "info",
		Usage:       "Get a monitor information",
		UsageText:   "openstatus monitors info [MonitorID]",
		Description: "Fetch the monitor information. The monitor information includes details such as name, description, endpoint, method, frequency, locations, active status, public status, timeout, degraded after, and body. The body is truncated to 40 characters.",
		Action: func(ctx context.Context, cmd *cli.Command) error {
			monitorId := cmd.Args().Get(0)
			err := GetMonitorInfo(http.DefaultClient, cmd.String("access-token"), monitorId)
			if err != nil {
				return cli.Exit(err.Error(), 1)
			}
			return nil
		},
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "access-token",
				Usage:    "OpenStatus API Access Token",
				Aliases:  []string{"t"},
				Sources:  cli.EnvVars("OPENSTATUS_API_TOKEN"),
				Required: true,
			}},
	}
	return &monitorInfoCmd
}
