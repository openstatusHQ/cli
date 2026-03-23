package monitors

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync"

	monitorv1 "buf.build/gen/go/openstatus/api/protocolbuffers/go/openstatus/monitor/v1"
	"github.com/fatih/color"
	"github.com/logrusorgru/aurora/v4"
	"github.com/olekukonko/tablewriter"
	"github.com/olekukonko/tablewriter/renderer"
	"github.com/olekukonko/tablewriter/tw"
	"github.com/openstatusHQ/cli/internal/api"
	"github.com/openstatusHQ/cli/internal/auth"
	output "github.com/openstatusHQ/cli/internal/cli"
	"github.com/urfave/cli/v3"
)

type MonitorInfoOutput struct {
	Monitor Monitor              `json:"monitor"`
	Status  string               `json:"status,omitempty"`
	Regions []RegionStatusOutput `json:"regions,omitempty"`
	Summary *SummaryOutput       `json:"summary,omitempty"`
}

type RegionStatusOutput struct {
	Region   string `json:"region"`
	Provider string `json:"provider"`
	Status   string `json:"status"`
}

type SummaryOutput struct {
	TotalSuccessful int64  `json:"total_successful"`
	TotalDegraded   int64  `json:"total_degraded"`
	TotalFailed     int64  `json:"total_failed"`
	P50             int64  `json:"p50"`
	P75             int64  `json:"p75"`
	P90             int64  `json:"p90"`
	P95             int64  `json:"p95"`
	P99             int64  `json:"p99"`
	TimeRange       string `json:"time_range"`
	LastPingAt      string `json:"last_ping_at,omitempty"`
}

func monitorStatusToString(s monitorv1.MonitorStatus) string {
	switch s {
	case monitorv1.MonitorStatus_MONITOR_STATUS_ACTIVE:
		return "active"
	case monitorv1.MonitorStatus_MONITOR_STATUS_DEGRADED:
		return "degraded"
	case monitorv1.MonitorStatus_MONITOR_STATUS_ERROR:
		return "error"
	default:
		return "unknown"
	}
}

func colorizeStatus(status string) string {
	switch status {
	case "active":
		return color.GreenString("● active")
	case "degraded":
		return color.YellowString("● degraded")
	case "error":
		return color.RedString("● error")
	case "unknown":
		return "● unknown"
	default:
		return status
	}
}

func parseTimeRange(s string) (monitorv1.TimeRange, error) {
	switch s {
	case "1d":
		return monitorv1.TimeRange_TIME_RANGE_1D, nil
	case "7d":
		return monitorv1.TimeRange_TIME_RANGE_7D, nil
	case "14d":
		return monitorv1.TimeRange_TIME_RANGE_14D, nil
	default:
		return monitorv1.TimeRange_TIME_RANGE_UNSPECIFIED, fmt.Errorf("invalid time range %q: must be 1d, 7d, or 14d", s)
	}
}

func regionProviderLabel(r monitorv1.Region) string {
	code := regionToString(r)
	enumStr := r.String()
	switch {
	case strings.HasPrefix(enumStr, "REGION_FLY_"):
		return fmt.Sprintf("%s (Fly.io)", code)
	case strings.HasPrefix(enumStr, "REGION_KOYEB_"):
		return fmt.Sprintf("%s (Koyeb)", code)
	case strings.HasPrefix(enumStr, "REGION_RAILWAY_"):
		return fmt.Sprintf("%s (Railway)", code)
	default:
		return code
	}
}

func regionProvider(r monitorv1.Region) string {
	enumStr := r.String()
	switch {
	case strings.HasPrefix(enumStr, "REGION_FLY_"):
		return "Fly.io"
	case strings.HasPrefix(enumStr, "REGION_KOYEB_"):
		return "Koyeb"
	case strings.HasPrefix(enumStr, "REGION_RAILWAY_"):
		return "Railway"
	default:
		return "Unknown"
	}
}

func deriveGlobalStatus(regions []*monitorv1.RegionStatus) string {
	if len(regions) == 0 {
		return "unknown"
	}
	hasError := false
	hasDegraded := false
	for _, r := range regions {
		switch r.GetStatus() {
		case monitorv1.MonitorStatus_MONITOR_STATUS_ERROR:
			hasError = true
		case monitorv1.MonitorStatus_MONITOR_STATUS_DEGRADED:
			hasDegraded = true
		}
	}
	if hasError {
		return "error"
	}
	if hasDegraded {
		return "degraded"
	}
	return "active"
}

func newBlueprintTable() *tablewriter.Table {
	return tablewriter.NewTable(os.Stdout,
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
}

func GetMonitorInfo(ctx context.Context, httpClient *http.Client, apiKey string, monitorId string, timeRange monitorv1.TimeRange, timeRangeStr string, s *output.Spinner) error {

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

	var (
		statusResp  *monitorv1.GetMonitorStatusResponse
		summaryResp *monitorv1.GetMonitorSummaryResponse
		statusErr   error
		summaryErr  error
		wg          sync.WaitGroup
	)
	wg.Add(2)
	go func() {
		defer wg.Done()
		statusResp, statusErr = client.GetMonitorStatus(ctx, &monitorv1.GetMonitorStatusRequest{Id: monitorId})
	}()
	go func() {
		defer wg.Done()
		summaryResp, summaryErr = client.GetMonitorSummary(ctx, &monitorv1.GetMonitorSummaryRequest{
			Id:        monitorId,
			TimeRange: timeRange,
		})
	}()
	wg.Wait()

	if statusErr != nil {
		fmt.Fprintln(os.Stderr, "Warning: could not fetch monitor status:", statusErr)
		statusResp = nil
	}
	if summaryErr != nil {
		fmt.Fprintln(os.Stderr, "Warning: could not fetch monitor summary:", summaryErr)
		summaryResp = nil
	}

	var globalStatus string
	if statusResp != nil {
		globalStatus = deriveGlobalStatus(statusResp.GetRegions())
	}

	if output.IsJSONOutput() {
		infoOutput := MonitorInfoOutput{
			Monitor: monitor,
		}
		if statusResp != nil {
			infoOutput.Status = globalStatus
			for _, rs := range statusResp.GetRegions() {
				infoOutput.Regions = append(infoOutput.Regions, RegionStatusOutput{
					Region:   regionToString(rs.GetRegion()),
					Provider: regionProvider(rs.GetRegion()),
					Status:   monitorStatusToString(rs.GetStatus()),
				})
			}
		}
		if summaryResp != nil {
			infoOutput.Summary = &SummaryOutput{
				TotalSuccessful: summaryResp.GetTotalSuccessful(),
				TotalDegraded:   summaryResp.GetTotalDegraded(),
				TotalFailed:     summaryResp.GetTotalFailed(),
				P50:             summaryResp.GetP50(),
				P75:             summaryResp.GetP75(),
				P90:             summaryResp.GetP90(),
				P95:             summaryResp.GetP95(),
				P99:             summaryResp.GetP99(),
				TimeRange:       timeRangeStr,
				LastPingAt:      summaryResp.GetLastPingAt(),
			}
		}
		return output.PrintJSON(infoOutput)
	}

	// Section 1: Monitor config table
	fmt.Println(aurora.Bold("Monitor:"))
	table := newBlueprintTable()

	data := [][]string{
		{"ID", fmt.Sprintf("%d", monitor.ID)},
	}
	if statusResp != nil {
		data = append(data, []string{"Status", colorizeStatus(globalStatus)})
	}
	data = append(data, []string{"Name", monitor.Name})
	data = append(data, []string{"Description", monitor.Description})
	data = append(data, []string{"Endpoint", monitor.URL})
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
		data = append(data, []string{"Degraded After", fmt.Sprintf("%d ms", monitor.DegradedAfter)})
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

	// Section 2: Assertions table
	if len(monitor.Assertions) > 0 {
		fmt.Println()
		fmt.Println(aurora.Bold("Assertions:"))
		assertionTable := newBlueprintTable()
		var assertionData [][]string
		for _, a := range monitor.Assertions {
			row := []string{a.Type, a.Compare, fmt.Sprintf("%v", a.Target)}
			if a.Key != "" {
				row = append(row, a.Key)
			}
			assertionData = append(assertionData, row)
		}
		assertionTable.Bulk(assertionData)
		assertionTable.Render()
	}

	// Section 3: Region status table
	if statusResp != nil && len(statusResp.GetRegions()) > 0 {
		fmt.Println()
		fmt.Println(aurora.Bold("Region Status:"))
		regionTable := newBlueprintTable()
		var regionData [][]string
		for _, rs := range statusResp.GetRegions() {
			regionData = append(regionData, []string{
				regionProviderLabel(rs.GetRegion()),
				colorizeStatus(monitorStatusToString(rs.GetStatus())),
			})
		}
		regionTable.Bulk(regionData)
		regionTable.Render()
	}

	// Section 3: Summary table
	if summaryResp != nil {
		fmt.Println()
		fmt.Println(aurora.Bold(fmt.Sprintf("Summary (%s):", timeRangeStr)))
		summaryTable := newBlueprintTable()
		summaryData := [][]string{
			{"Total Successful", fmt.Sprintf("%d", summaryResp.GetTotalSuccessful())},
			{"Total Degraded", fmt.Sprintf("%d", summaryResp.GetTotalDegraded())},
			{"Total Failed", fmt.Sprintf("%d", summaryResp.GetTotalFailed())},
			{"P50 Latency", fmt.Sprintf("%d ms", summaryResp.GetP50())},
			{"P75 Latency", fmt.Sprintf("%d ms", summaryResp.GetP75())},
			{"P90 Latency", fmt.Sprintf("%d ms", summaryResp.GetP90())},
			{"P95 Latency", fmt.Sprintf("%d ms", summaryResp.GetP95())},
			{"P99 Latency", fmt.Sprintf("%d ms", summaryResp.GetP99())},
		}
		if summaryResp.GetLastPingAt() != "" {
			summaryData = append(summaryData, []string{"Last Ping", summaryResp.GetLastPingAt()})
		}
		summaryTable.Bulk(summaryData)
		summaryTable.Render()
	}

	return nil
}

func GetMonitorInfoCmd() *cli.Command {
	monitorInfoCmd := cli.Command{
		Name:  "info",
		Usage: "Get a monitor information",
		UsageText: `openstatus monitors info <MonitorID>
  openstatus monitors info 12345
  openstatus monitors info 12345 --time-range 7d`,
		Description: "Fetch the monitor information including configuration, live status per region, and summary metrics.",
		Action: func(ctx context.Context, cmd *cli.Command) error {
			apiKey, err := auth.ResolveAccessToken(cmd)
			if err != nil {
				return cli.Exit(err.Error(), 1)
			}
			monitorId := cmd.Args().Get(0)
			timeRangeStr := cmd.String("time-range")
			timeRange, err := parseTimeRange(timeRangeStr)
			if err != nil {
				return cli.Exit(err.Error(), 1)
			}
			s := output.StartSpinner("Fetching monitor details...")
			err = GetMonitorInfo(ctx, api.DefaultHTTPClient, apiKey, monitorId, timeRange, timeRangeStr, s)
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
			},
			&cli.StringFlag{
				Name:  "time-range",
				Usage: "Time range for summary metrics (1d, 7d, 14d)",
				Value: "1d",
			},
		},
	}
	return &monitorInfoCmd
}
