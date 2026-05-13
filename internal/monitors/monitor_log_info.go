package monitors

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"sort"
	"strings"

	"buf.build/gen/go/openstatus/api/connectrpc/gosimple/openstatus/monitor/v1/monitorv1connect"
	monitorv1 "buf.build/gen/go/openstatus/api/protocolbuffers/go/openstatus/monitor/v1"
	"github.com/fatih/color"
	"github.com/logrusorgru/aurora/v4"
	"github.com/openstatusHQ/cli/internal/auth"
	output "github.com/openstatusHQ/cli/internal/cli"
	"github.com/urfave/cli/v3"
)

type responseLogDetailOutput struct {
	ID            string            `json:"id"`
	MonitorID     string            `json:"monitor_id"`
	URL           string            `json:"url"`
	StatusCode    int32             `json:"status_code,omitempty"`
	Latency       int32             `json:"latency_ms"`
	Region        string            `json:"region"`
	RequestStatus string            `json:"request_status"`
	Trigger       string            `json:"trigger"`
	Timestamp     string            `json:"timestamp"`
	Error         bool              `json:"error"`
	Message       string            `json:"message,omitempty"`
	Headers       map[string]string `json:"headers,omitempty"`
	Assertions    string            `json:"assertions,omitempty"`
	Timing        *timingOutput     `json:"timing,omitempty"`
}

type timingOutput struct {
	DNS      int32 `json:"dns_ms"`
	Connect  int32 `json:"connect_ms"`
	TLS      int32 `json:"tls_ms"`
	TTFB     int32 `json:"ttfb_ms"`
	Transfer int32 `json:"transfer_ms"`
}

func GetMonitorResponseLogInfo(
	ctx context.Context,
	client monitorv1connect.MonitorServiceClient,
	monitorId string,
	logId string,
	s *output.Spinner,
) error {
	if monitorId == "" {
		output.StopSpinner(s)
		fmt.Fprintln(os.Stderr, "Usage: openstatus monitors log-info <monitor-id> <log-id>")
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "Example: openstatus monitors log-info 12345 abc-def")
		return fmt.Errorf("monitor ID is required")
	}
	if logId == "" {
		output.StopSpinner(s)
		fmt.Fprintln(os.Stderr, "Usage: openstatus monitors log-info <monitor-id> <log-id>")
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "Example: openstatus monitors log-info 12345 abc-def")
		return fmt.Errorf("log ID is required")
	}

	resp, err := client.GetMonitorHTTPResponseLog(ctx, &monitorv1.GetMonitorHTTPResponseLogRequest{
		Id:    monitorId,
		LogId: logId,
	})
	output.StopSpinner(s)
	if err != nil {
		return output.FormatError(err, "response log", logId)
	}

	detail := resp.GetLog()
	logItem := detail.GetLog()

	var timing *timingOutput
	if logItem.HasTiming() {
		t := logItem.GetTiming()
		timing = &timingOutput{
			DNS:      t.GetDns(),
			Connect:  t.GetConnect(),
			TLS:      t.GetTls(),
			TTFB:     t.GetTtfb(),
			Transfer: t.GetTransfer(),
		}
	}

	detailOut := responseLogDetailOutput{
		ID:            logItem.GetId(),
		MonitorID:     logItem.GetMonitorId(),
		URL:           detail.GetUrl(),
		StatusCode:    logItem.GetStatusCode(),
		Latency:       logItem.GetLatency(),
		Region:        regionToString(logItem.GetRegion()),
		RequestStatus: requestStatusToString(logItem.GetRequestStatus()),
		Trigger:       triggerToString(logItem.GetTrigger()),
		Timestamp:     formatUnixMillis(logItem.GetTimestamp()),
		Error:         detail.GetError(),
		Message:       detail.GetMessage(),
		Headers:       detail.GetHeaders(),
		Assertions:    detail.GetAssertions(),
		Timing:        timing,
	}

	if output.IsJSONOutput() {
		return output.PrintJSON(detailOut)
	}

	// Section 1: Response Log
	fmt.Println(aurora.Bold("Response Log:"))
	tbl := newBlueprintTable()
	data := [][]string{
		{"ID", detailOut.ID},
		{"Monitor ID", detailOut.MonitorID},
		{"URL", detailOut.URL},
		{"Status", colorizeRequestStatus(detailOut.RequestStatus)},
		{"Status Code", fmt.Sprintf("%d", detailOut.StatusCode)},
		{"Latency", fmt.Sprintf("%d ms", detailOut.Latency)},
		{"Region", detailOut.Region},
		{"Trigger", detailOut.Trigger},
		{"Timestamp", detailOut.Timestamp},
	}
	if detailOut.Error {
		msg := detailOut.Message
		if msg == "" {
			msg = "yes"
		}
		data = append(data, []string{"Error", color.RedString(msg)})
	}
	tbl.Bulk(data)
	tbl.Render()

	// Section 2: Timing waterfall
	if timing != nil {
		fmt.Println()
		fmt.Println(aurora.Bold("Timing:"))
		renderTimingWaterfall(timing)
	}

	// Section 3: Response Headers
	if len(detailOut.Headers) > 0 {
		fmt.Println()
		fmt.Println(aurora.Bold("Response Headers:"))
		headerTable := newBlueprintTable()
		keys := make([]string, 0, len(detailOut.Headers))
		for k := range detailOut.Headers {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		var headerData [][]string
		for _, k := range keys {
			headerData = append(headerData, []string{k, detailOut.Headers[k]})
		}
		headerTable.Bulk(headerData)
		headerTable.Render()
	}

	// Section 4: Assertions
	if detailOut.Assertions != "" {
		fmt.Println()
		fmt.Println(aurora.Bold("Assertions:"))
		renderAssertions(detailOut.Assertions)
	}

	return nil
}

func renderTimingWaterfall(t *timingOutput) {
	phases := []struct {
		name string
		ms   int32
	}{
		{"DNS", t.DNS},
		{"Connect", t.Connect},
		{"TLS", t.TLS},
		{"TTFB", t.TTFB},
		{"Transfer", t.Transfer},
	}

	total := t.DNS + t.Connect + t.TLS + t.TTFB + t.Transfer
	const maxBarWidth = 30

	tbl := newBlueprintTable()
	var data [][]string
	for _, p := range phases {
		bar := ""
		if total > 0 && p.ms > 0 {
			width := int(float64(p.ms) / float64(total) * maxBarWidth)
			if width < 1 {
				width = 1
			}
			bar = color.CyanString(strings.Repeat("█", width))
		}
		data = append(data, []string{p.name, fmt.Sprintf("%d ms", p.ms), bar})
	}
	data = append(data, []string{"─────", "─────────", ""})
	data = append(data, []string{"Total", fmt.Sprintf("%d ms", total), ""})
	tbl.Bulk(data)
	tbl.Render()
}

func renderAssertions(raw string) {
	var assertions []Assertion
	if err := json.Unmarshal([]byte(raw), &assertions); err == nil && len(assertions) > 0 {
		tbl := newBlueprintTable()
		var data [][]string
		for _, a := range assertions {
			row := []string{a.Type, a.Compare, fmt.Sprintf("%v", a.Target)}
			if a.Key != "" {
				row = append(row, a.Key)
			}
			data = append(data, row)
		}
		tbl.Bulk(data)
		tbl.Render()
		return
	}

	var buf json.RawMessage
	if err := json.Unmarshal([]byte(raw), &buf); err == nil {
		indented, err := json.MarshalIndent(buf, "", "  ")
		if err == nil {
			fmt.Println(string(indented))
			return
		}
	}

	fmt.Println(raw)
}

func GetMonitorResponseLogInfoWithHTTPClient(ctx context.Context, httpClient *http.Client, apiKey string, monitorId string, logId string) error {
	client := NewMonitorClientWithHTTPClient(httpClient, apiKey)
	return GetMonitorResponseLogInfo(ctx, client, monitorId, logId, nil)
}

func GetMonitorLogInfoCmd() *cli.Command {
	return &cli.Command{
		Name:  "log-info",
		Usage: "Get detailed HTTP response log for a monitor",
		UsageText: `openstatus monitors log-info <MonitorID> <LogID>
  openstatus monitors log-info 12345 abc-def-ghi`,
		Description: "Fetch a single HTTP response log with full details including timing phases, response headers, and assertion results.",
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
			logId := cmd.Args().Get(1)
			s := output.StartSpinner("Fetching response log details...")
			err = GetMonitorResponseLogInfo(
				ctx,
				NewMonitorClient(apiKey),
				monitorId,
				logId,
				s,
			)
			if err != nil {
				return cli.Exit(err.Error(), 1)
			}
			return nil
		},
	}
}
