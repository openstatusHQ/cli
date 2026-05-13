package monitors

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"time"

	"buf.build/gen/go/openstatus/api/connectrpc/gosimple/openstatus/monitor/v1/monitorv1connect"
	monitorv1 "buf.build/gen/go/openstatus/api/protocolbuffers/go/openstatus/monitor/v1"
	"github.com/fatih/color"
	"github.com/rodaine/table"
	"github.com/urfave/cli/v3"

	"github.com/openstatusHQ/cli/internal/auth"
	output "github.com/openstatusHQ/cli/internal/cli"
)

type responseLogEntry struct {
	ID            string `json:"id"`
	MonitorID     string `json:"monitor_id"`
	StatusCode    int32  `json:"status_code,omitempty"`
	Latency       int32  `json:"latency_ms"`
	Region        string `json:"region"`
	RequestStatus string `json:"request_status"`
	Trigger       string `json:"trigger"`
	Timestamp     string `json:"timestamp"`
}

type responseLogListOutput struct {
	Logs       []responseLogEntry `json:"logs"`
	Pagination *paginationOutput  `json:"pagination"`
}

type paginationOutput struct {
	Limit      int32 `json:"limit"`
	Offset     int32 `json:"offset"`
	HasMore    bool  `json:"has_more"`
	NextOffset int32 `json:"next_offset,omitempty"`
}

func requestStatusToString(s monitorv1.HTTPResponseLogRequestStatus) string {
	switch s {
	case monitorv1.HTTPResponseLogRequestStatus_HTTP_RESPONSE_LOG_REQUEST_STATUS_SUCCESS:
		return "success"
	case monitorv1.HTTPResponseLogRequestStatus_HTTP_RESPONSE_LOG_REQUEST_STATUS_ERROR:
		return "error"
	case monitorv1.HTTPResponseLogRequestStatus_HTTP_RESPONSE_LOG_REQUEST_STATUS_DEGRADED:
		return "degraded"
	default:
		return "unknown"
	}
}

func triggerToString(t monitorv1.HTTPResponseLogTrigger) string {
	switch t {
	case monitorv1.HTTPResponseLogTrigger_HTTP_RESPONSE_LOG_TRIGGER_CRON:
		return "cron"
	case monitorv1.HTTPResponseLogTrigger_HTTP_RESPONSE_LOG_TRIGGER_API:
		return "api"
	default:
		return "unknown"
	}
}

func colorizeRequestStatus(status string) string {
	switch status {
	case "success":
		return color.GreenString("● success")
	case "error":
		return color.RedString("● error")
	case "degraded":
		return color.YellowString("● degraded")
	default:
		return "● " + status
	}
}

func formatUnixMillis(ms int64) string {
	return time.UnixMilli(ms).UTC().Format(time.RFC3339)
}

func parseRFC3339ToUnixMillis(s string) (int64, error) {
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return 0, fmt.Errorf("invalid RFC 3339 timestamp %q: %w", s, err)
	}
	return t.UnixMilli(), nil
}

func ListMonitorResponseLogs(
	ctx context.Context,
	client monitorv1connect.MonitorServiceClient,
	monitorId string,
	limit int32,
	offset int32,
	from string,
	to string,
	s *output.Spinner,
) error {
	if monitorId == "" {
		output.StopSpinner(s)
		fmt.Fprintln(os.Stderr, "Usage: openstatus monitors logs <monitor-id>")
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "Example: openstatus monitors logs 12345")
		return fmt.Errorf("monitor ID is required")
	}

	req := &monitorv1.ListMonitorHTTPResponseLogsRequest{
		Id: monitorId,
	}

	if limit > 0 {
		req.SetLimit(limit)
	}
	if offset > 0 {
		req.SetOffset(offset)
	}

	var fromMs, toMs int64
	if from != "" {
		var err error
		fromMs, err = parseRFC3339ToUnixMillis(from)
		if err != nil {
			output.StopSpinner(s)
			return err
		}
		req.SetFromTimestamp(fromMs)
	}
	if to != "" {
		var err error
		toMs, err = parseRFC3339ToUnixMillis(to)
		if err != nil {
			output.StopSpinner(s)
			return err
		}
		req.SetToTimestamp(toMs)
	}
	if from != "" && to != "" && fromMs >= toMs {
		output.StopSpinner(s)
		return fmt.Errorf("--from must be before --to")
	}

	resp, err := client.ListMonitorHTTPResponseLogs(ctx, req)
	output.StopSpinner(s)
	if err != nil {
		return output.FormatError(err, "response logs", monitorId)
	}

	logs := resp.GetLogs()
	entries := make([]responseLogEntry, 0, len(logs))
	for _, l := range logs {
		entries = append(entries, responseLogEntry{
			ID:            l.GetId(),
			MonitorID:     l.GetMonitorId(),
			StatusCode:    l.GetStatusCode(),
			Latency:       l.GetLatency(),
			Region:        regionToString(l.GetRegion()),
			RequestStatus: requestStatusToString(l.GetRequestStatus()),
			Trigger:       triggerToString(l.GetTrigger()),
			Timestamp:     formatUnixMillis(l.GetTimestamp()),
		})
	}

	var pagination *paginationOutput
	if p := resp.GetPagination(); p != nil {
		pagination = &paginationOutput{
			Limit:      p.GetLimit(),
			Offset:     p.GetOffset(),
			HasMore:    p.GetHasMore(),
			NextOffset: p.GetNextOffset(),
		}
	}

	if output.IsJSONOutput() {
		return output.PrintJSON(responseLogListOutput{
			Logs:       entries,
			Pagination: pagination,
		})
	}

	if len(entries) == 0 {
		if !output.IsQuiet() {
			fmt.Println("No response logs found")
		}
		return nil
	}

	headerFmt := color.New(color.FgGreen, color.Underline).SprintfFunc()
	columnFmt := color.New(color.FgYellow).SprintfFunc()

	tbl := table.New("ID", "Status", "Code", "Latency (ms)", "Region", "Timestamp")
	tbl.WithHeaderFormatter(headerFmt).WithFirstColumnFormatter(columnFmt)

	for _, e := range entries {
		tbl.AddRow(e.ID, colorizeRequestStatus(e.RequestStatus), e.StatusCode, e.Latency, e.Region, e.Timestamp)
	}

	tbl.Print()

	if pagination != nil && pagination.HasMore {
		fmt.Fprintf(os.Stderr, "Showing %d results. Use --offset %d to see the next page.\n", len(entries), pagination.NextOffset)
	}

	return nil
}

func ListMonitorResponseLogsWithHTTPClient(ctx context.Context, httpClient *http.Client, apiKey string, monitorId string, limit int32, offset int32, from string, to string) error {
	client := NewMonitorClientWithHTTPClient(httpClient, apiKey)
	return ListMonitorResponseLogs(ctx, client, monitorId, limit, offset, from, to, nil)
}

func GetMonitorLogsCmd() *cli.Command {
	return &cli.Command{
		Name:  "logs",
		Usage: "List HTTP response logs for a monitor",
		UsageText: `openstatus monitors logs <MonitorID>
  openstatus monitors logs 12345
  openstatus monitors logs 12345 --limit 10
  openstatus monitors logs 12345 --limit 5 --offset 5
  openstatus monitors logs 12345 --from 2026-05-06T00:00:00Z --to 2026-05-07T00:00:00Z`,
		Description: "List HTTP response logs for a monitor from the 14-day retention window. Supports pagination and time filtering.",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "access-token",
				Usage:   "OpenStatus API Access Token",
				Aliases: []string{"t"},
				Sources: cli.EnvVars("OPENSTATUS_API_TOKEN"),
			},
			&cli.IntFlag{
				Name:  "limit",
				Usage: "Maximum number of logs to return (1-100)",
			},
			&cli.IntFlag{
				Name:  "offset",
				Usage: "Number of logs to skip for pagination",
			},
			&cli.StringFlag{
				Name:  "from",
				Usage: "Start of time window (RFC 3339 format)",
			},
			&cli.StringFlag{
				Name:  "to",
				Usage: "End of time window (RFC 3339 format)",
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			apiKey, err := auth.ResolveAccessToken(cmd)
			if err != nil {
				return cli.Exit(err.Error(), 1)
			}
			monitorId := cmd.Args().Get(0)
			s := output.StartSpinner("Fetching response logs...")
			err = ListMonitorResponseLogs(
				ctx,
				NewMonitorClient(apiKey),
				monitorId,
				int32(cmd.Int("limit")),
				int32(cmd.Int("offset")),
				cmd.String("from"),
				cmd.String("to"),
				s,
			)
			if err != nil {
				return cli.Exit(err.Error(), 1)
			}
			return nil
		},
	}
}
