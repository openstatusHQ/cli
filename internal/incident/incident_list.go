package incident

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"buf.build/gen/go/openstatus/api/connectrpc/gosimple/openstatus/incident/v1/incidentv1connect"
	incidentv1 "buf.build/gen/go/openstatus/api/protocolbuffers/go/openstatus/incident/v1"
	"github.com/fatih/color"
	"github.com/rodaine/table"
	"github.com/urfave/cli/v3"

	output "github.com/openstatusHQ/cli/internal/cli"
)

type ListParams struct {
	Statuses string
	Active   bool
	Closed   bool
	Limit    int
	Offset   int
}

type listOutput struct {
	Incidents []incidentSummaryJSON `json:"incidents"`
	TotalSize int32                 `json:"total_size"`
}

func buildListRequest(p ListParams) (*incidentv1.ListIncidentsRequest, error) {
	if p.Active && p.Closed {
		return nil, errors.New("--active and --closed are mutually exclusive")
	}
	if p.Limit < 0 || p.Limit > 100 {
		return nil, fmt.Errorf("invalid --limit %d: must be between 1 and 100", p.Limit)
	}
	if p.Offset < 0 {
		return nil, fmt.Errorf("invalid --offset %d: must be 0 or more", p.Offset)
	}

	statuses, err := parseStatuses(p.Statuses)
	if err != nil {
		return nil, err
	}

	req := &incidentv1.ListIncidentsRequest{}
	if len(statuses) > 0 {
		req.SetStatuses(statuses)
	}
	if p.Active {
		req.SetClosed(false)
	}
	if p.Closed {
		req.SetClosed(true)
	}
	if p.Limit > 0 {
		req.SetLimit(int32(p.Limit))
	}
	if p.Offset > 0 {
		req.SetOffset(int32(p.Offset))
	}
	return req, nil
}

func listFooter(offset, shown int, total int32) string {
	if shown == 0 || int(total) <= offset+shown {
		return ""
	}
	return fmt.Sprintf("Showing %d-%d of %d — use --offset %d for more", offset+1, offset+shown, total, offset+shown)
}

func ListIncidents(ctx context.Context, client incidentv1connect.IncidentServiceClient, p ListParams, s *output.Spinner) error {
	req, err := buildListRequest(p)
	if err != nil {
		output.StopSpinner(s)
		return err
	}

	resp, err := client.ListIncidents(ctx, req)
	output.StopSpinner(s)
	if err != nil {
		return formatIncidentError(err, "", opGeneric)
	}

	incidents := resp.GetIncidents()

	if output.IsJSONOutput() {
		out := listOutput{
			Incidents: make([]incidentSummaryJSON, 0, len(incidents)),
			TotalSize: resp.GetTotalSize(),
		}
		for _, inc := range incidents {
			out.Incidents = append(out.Incidents, toSummaryJSON(inc))
		}
		return output.PrintJSON(out)
	}

	if len(incidents) == 0 {
		fmt.Println("No incidents found")
		return nil
	}

	headerFmt := color.New(color.FgGreen, color.Underline).SprintfFunc()
	columnFmt := color.New(color.FgYellow).SprintfFunc()

	tbl := table.New("ID", "Title", "Severity", "Status", "Commander", "Declared", "Report")
	tbl.WithHeaderFormatter(headerFmt).WithFirstColumnFormatter(columnFmt).WithWidthFunc(visibleWidth)

	for _, inc := range incidents {
		report := "—"
		if inc.HasStatusReport() {
			report = inc.GetStatusReport().GetId()
		}
		tbl.AddRow(
			inc.GetId(),
			inc.GetTitle(),
			severityColor(severityToString(inc.GetSeverity())),
			displayStatus(inc.GetStatus(), inc.GetClosedAt()),
			shortUser(inc.GetCommander()),
			output.FormatTimestamp(inc.GetDeclaredAt()),
			report,
		)
	}

	tbl.Print()

	if footer := listFooter(p.Offset, len(incidents), resp.GetTotalSize()); footer != "" {
		fmt.Println(footer)
	}
	return nil
}

func ListIncidentsWithHTTPClient(ctx context.Context, httpClient *http.Client, apiKey string, p ListParams) error {
	return ListIncidents(ctx, NewIncidentClientWithHTTPClient(httpClient, apiKey), p, nil)
}

func GetIncidentListCmd() *cli.Command {
	return &cli.Command{
		Name:  "list",
		Usage: "List incidents",
		UsageText: `openstatus incident list
  openstatus inc list --active
  openstatus inc list --status open,mitigated --limit 10`,
		Flags: []cli.Flag{
			accessTokenFlag(),
			&cli.StringFlag{
				Name:  "status",
				Usage: "Filter by status, comma-separated (open, mitigated, resolved, canceled)",
			},
			&cli.BoolFlag{
				Name:  "active",
				Usage: "Only incidents that are not closed",
			},
			&cli.BoolFlag{
				Name:  "closed",
				Usage: "Only closed incidents",
			},
			&cli.IntFlag{
				Name:  "limit",
				Usage: "Maximum number of incidents to return (1-100, defaults to 50)",
			},
			&cli.IntFlag{
				Name:  "offset",
				Usage: "Number of incidents to skip",
			},
		},
		Action: withClient(func(ctx context.Context, cmd *cli.Command, client incidentv1connect.IncidentServiceClient) error {
			p := ListParams{
				Statuses: cmd.String("status"),
				Active:   cmd.Bool("active"),
				Closed:   cmd.Bool("closed"),
				Limit:    int(cmd.Int("limit")),
				Offset:   int(cmd.Int("offset")),
			}
			if cmd.IsSet("limit") && p.Limit < 1 {
				return fmt.Errorf("invalid --limit %d: must be between 1 and 100", p.Limit)
			}
			s := output.StartSpinner("Fetching incidents...")
			return ListIncidents(ctx, client, p, s)
		}),
	}
}
