package monitors

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"

	"buf.build/gen/go/openstatus/api/connectrpc/gosimple/openstatus/monitor/v1/monitorv1connect"
	monitorv1 "buf.build/gen/go/openstatus/api/protocolbuffers/go/openstatus/monitor/v1"
	"connectrpc.com/connect"
	"github.com/fatih/color"
	"github.com/rodaine/table"
	"github.com/urfave/cli/v3"

	"github.com/openstatusHQ/cli/internal/auth"
	output "github.com/openstatusHQ/cli/internal/cli"
)

type heartbeatPingEntry struct {
	ID            string            `json:"id"`
	MonitorID     string            `json:"monitor_id"`
	Timestamp     string            `json:"timestamp"`
	Kind          string            `json:"kind"`
	Method        string            `json:"method"`
	IP            string            `json:"ip,omitempty"`
	UserAgent     string            `json:"user_agent,omitempty"`
	Headers       map[string]string `json:"headers,omitempty"`
	Body          string            `json:"body,omitempty"`
	BodyTruncated bool              `json:"body_truncated"`
}

type heartbeatPingListOutput struct {
	Pings      []heartbeatPingEntry `json:"pings"`
	NextCursor string               `json:"next_cursor,omitempty"`
}

type PingsOptions struct {
	Limit  *int32
	From   string
	To     string
	Kinds  []string
	Cursor string
}

func monitorKind(m *monitorv1.MonitorConfig) string {
	switch {
	case m.HasHttp():
		return "http"
	case m.HasTcp():
		return "tcp"
	case m.HasDns():
		return "dns"
	case m.HasIcmp():
		return "icmp"
	case m.HasGrpc():
		return "grpc"
	case m.HasHeartbeat():
		return "heartbeat"
	default:
		return "unknown"
	}
}

func heartbeatTypeMismatchError(monitorId string, kind string, reason string) error {
	return output.NewError(output.KindInvalidArgument, "monitor %s is not a heartbeat monitor (type: %s); %s", monitorId, kind, reason)
}

func heartbeatPingsPlanError(monitorId string) error {
	return output.NewError(output.KindPermissionDenied,
		"heartbeat ping history requires a plan with response logs. Run 'openstatus monitors info %s' for the last ping and totals",
		monitorId)
}

func stringToHeartbeatPingKind(s string) (monitorv1.HeartbeatPingKind, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "success":
		return monitorv1.HeartbeatPingKind_HEARTBEAT_PING_KIND_SUCCESS, nil
	case "fail":
		return monitorv1.HeartbeatPingKind_HEARTBEAT_PING_KIND_FAIL, nil
	default:
		return monitorv1.HeartbeatPingKind_HEARTBEAT_PING_KIND_UNSPECIFIED, output.UsageErrorf("invalid --kind %q: must be success or fail", s)
	}
}

func colorizePingKind(kind string) string {
	switch kind {
	case "success":
		return color.GreenString("● success")
	case "fail":
		return color.RedString("● fail")
	default:
		return "● " + kind
	}
}

// parsePingKinds drops the filter when both kinds are asked for, since that is every ping.
func parsePingKinds(values []string) ([]monitorv1.HeartbeatPingKind, error) {
	var kinds []monitorv1.HeartbeatPingKind
	seen := map[monitorv1.HeartbeatPingKind]bool{}
	for _, v := range values {
		k, err := stringToHeartbeatPingKind(v)
		if err != nil {
			return nil, err
		}
		if !seen[k] {
			seen[k] = true
			kinds = append(kinds, k)
		}
	}
	if len(kinds) == 2 {
		return nil, nil
	}
	return kinds, nil
}

func buildHeartbeatPingsRequest(monitorId string, opts PingsOptions) (*monitorv1.ListMonitorHeartbeatPingsRequest, error) {
	req := &monitorv1.ListMonitorHeartbeatPingsRequest{Id: monitorId}

	if opts.Limit != nil {
		if *opts.Limit < 1 || *opts.Limit > 100 {
			return nil, output.UsageErrorf("--limit must be between 1 and 100")
		}
		req.SetLimit(*opts.Limit)
	}

	var fromMs, toMs int64
	if opts.From != "" {
		var err error
		if fromMs, err = parseRFC3339ToUnixMillis(opts.From); err != nil {
			return nil, err
		}
		req.SetFromTimestamp(fromMs)
	}
	if opts.To != "" {
		var err error
		if toMs, err = parseRFC3339ToUnixMillis(opts.To); err != nil {
			return nil, err
		}
		req.SetToTimestamp(toMs)
	}
	if opts.From != "" && opts.To != "" && fromMs >= toMs {
		return nil, output.UsageErrorf("--from must be before --to")
	}

	kinds, err := parsePingKinds(opts.Kinds)
	if err != nil {
		return nil, err
	}
	if len(kinds) > 0 {
		req.SetKinds(kinds)
	}

	if opts.Cursor != "" {
		req.SetCursor(opts.Cursor)
	}
	return req, nil
}

// explainHeartbeatPingsError looks the monitor up only after a rejection, so the
// common path costs a single request.
func explainHeartbeatPingsError(ctx context.Context, client monitorv1connect.MonitorServiceClient, monitorId string, err error) error {
	code := connect.CodeOf(err)
	if code != connect.CodeInvalidArgument && code != connect.CodePermissionDenied {
		return err
	}
	resp, getErr := client.GetMonitor(ctx, &monitorv1.GetMonitorRequest{Id: monitorId})
	if getErr != nil {
		return err
	}
	if !resp.GetMonitor().HasHeartbeat() {
		kind := monitorKind(resp.GetMonitor())
		reason := "only heartbeat monitors have pings"
		if kind == "http" {
			reason += fmt.Sprintf(". Try 'openstatus monitors logs %s'", monitorId)
		}
		return heartbeatTypeMismatchError(monitorId, kind, reason)
	}
	if code == connect.CodePermissionDenied {
		return heartbeatPingsPlanError(monitorId)
	}
	return err
}

func toHeartbeatPingEntry(p *monitorv1.HeartbeatPing) heartbeatPingEntry {
	return heartbeatPingEntry{
		ID:            p.GetId(),
		MonitorID:     p.GetMonitorId(),
		Timestamp:     formatUnixMillis(p.GetTimestamp()),
		Kind:          heartbeatPingKindToString(p.GetKind()),
		Method:        p.GetMethod(),
		IP:            p.GetIp(),
		UserAgent:     p.GetUserAgent(),
		Headers:       p.GetHeaders(),
		Body:          p.GetBody(),
		BodyTruncated: p.GetBodyTruncated(),
	}
}

func orDash(s string) string {
	if s == "" {
		return "-"
	}
	return s
}

func ListHeartbeatPings(ctx context.Context, client monitorv1connect.MonitorServiceClient, monitorId string, opts PingsOptions, s *output.Spinner) error {
	if monitorId == "" {
		output.StopSpinner(s)
		fmt.Fprintln(os.Stderr, "Usage: openstatus monitors pings <monitor-id>")
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "Example: openstatus monitors pings 12345")
		return output.UsageErrorf("monitor ID is required")
	}

	req, err := buildHeartbeatPingsRequest(monitorId, opts)
	if err != nil {
		output.StopSpinner(s)
		return err
	}

	resp, err := client.ListMonitorHeartbeatPings(ctx, req)
	if err != nil {
		err = explainHeartbeatPingsError(ctx, client, monitorId, err)
	}
	output.StopSpinner(s)
	if err != nil {
		return output.FormatError(err, "monitor", monitorId)
	}

	pings := resp.GetPings()
	result := heartbeatPingListOutput{
		Pings:      make([]heartbeatPingEntry, 0, len(pings)),
		NextCursor: resp.GetNextCursor(),
	}
	for _, p := range pings {
		result.Pings = append(result.Pings, toHeartbeatPingEntry(p))
	}

	if output.IsJSONOutput() {
		return output.PrintJSON(result)
	}

	if len(result.Pings) == 0 {
		if !output.IsQuiet() {
			fmt.Println("No pings found")
		}
		return nil
	}

	headerFmt := color.New(color.FgGreen, color.Underline).SprintfFunc()
	columnFmt := color.New(color.FgYellow).SprintfFunc()

	tbl := table.New("Time", "Kind", "Method", "IP", "User-Agent")
	tbl.WithWriter(os.Stdout).WithHeaderFormatter(headerFmt).WithFirstColumnFormatter(columnFmt)
	for _, p := range result.Pings {
		tbl.AddRow(p.Timestamp, colorizePingKind(p.Kind), p.Method, orDash(p.IP), orDash(p.UserAgent))
	}
	tbl.Print()

	if result.NextCursor != "" {
		fmt.Fprintf(os.Stderr, "Showing %d results. Use --cursor %s to see the next page.\n", len(result.Pings), result.NextCursor)
	}

	return nil
}

func ListHeartbeatPingsWithHTTPClient(ctx context.Context, httpClient *http.Client, apiKey string, monitorId string, opts PingsOptions) error {
	client := NewMonitorClientWithHTTPClient(httpClient, apiKey)
	return ListHeartbeatPings(ctx, client, monitorId, opts, nil)
}

func GetMonitorPingsCmd() *cli.Command {
	return &cli.Command{
		Name:  "pings",
		Usage: "List the pings a heartbeat monitor received",
		UsageText: `openstatus monitors pings <MonitorID>
  openstatus monitors pings 12345
  openstatus monitors pings 12345 --kind fail
  openstatus monitors pings 12345 --from 2026-11-01T00:00:00Z --to 2026-11-02T00:00:00Z
  openstatus monitors pings 12345 --json | jq '.pings[0].body'`,
		Description: "List the pings a heartbeat monitor received, newest first. Request headers and body are only included in --json output. Requires a plan with response logs.",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "access-token",
				Usage:   "OpenStatus API Access Token",
				Aliases: []string{"t"},
				Sources: cli.EnvVars("OPENSTATUS_API_TOKEN"),
			},
			&cli.IntFlag{
				Name:  "limit",
				Usage: "Maximum number of pings to return (1-100)",
			},
			&cli.StringFlag{
				Name:  "from",
				Usage: "Start of time window (RFC 3339 format)",
			},
			&cli.StringFlag{
				Name:  "to",
				Usage: "End of time window (RFC 3339 format)",
			},
			&cli.StringSliceFlag{
				Name:  "kind",
				Usage: "Only show pings of this kind: success or fail (repeatable)",
			},
			&cli.StringFlag{
				Name:  "cursor",
				Usage: "Cursor from a previous page",
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			apiKey, err := auth.ResolveAccessToken(ctx, cmd)
			if err != nil {
				return err
			}
			monitorId := cmd.Args().Get(0)
			opts := PingsOptions{
				From:   cmd.String("from"),
				To:     cmd.String("to"),
				Kinds:  cmd.StringSlice("kind"),
				Cursor: cmd.String("cursor"),
			}
			if cmd.IsSet("limit") {
				limit := int32(cmd.Int("limit"))
				opts.Limit = &limit
			}
			s := output.StartSpinner("Fetching pings...")
			return ListHeartbeatPings(ctx, NewMonitorClient(apiKey), monitorId, opts, s)
		},
	}
}
