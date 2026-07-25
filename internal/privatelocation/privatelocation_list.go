package privatelocation

import (
	"context"
	"fmt"
	"net/http"

	"buf.build/gen/go/openstatus/api/connectrpc/gosimple/openstatus/private_location/v1/private_locationv1connect"
	private_locationv1 "buf.build/gen/go/openstatus/api/protocolbuffers/go/openstatus/private_location/v1"
	"github.com/fatih/color"
	"github.com/rodaine/table"
	"github.com/urfave/cli/v3"

	"github.com/openstatusHQ/cli/internal/auth"
	output "github.com/openstatusHQ/cli/internal/cli"
)

type listEntry struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	Status       string `json:"status"`
	MonitorCount int32  `json:"monitor_count"`
	LastSeenAt   string `json:"last_seen_at"`
	CreatedAt    string `json:"created_at"`
	UpdatedAt    string `json:"updated_at"`
}

func ListPrivateLocations(ctx context.Context, client private_locationv1connect.PrivateLocationServiceClient, limit int, s *output.Spinner) error {
	req := &private_locationv1.ListPrivateLocationsRequest{}
	if limit > 0 {
		req.SetLimit(int32(limit))
	}

	resp, err := client.ListPrivateLocations(ctx, req)
	output.StopSpinner(s)
	if err != nil {
		return output.FormatError(err, "private location", "")
	}

	locations := resp.GetPrivateLocations()

	if output.IsJSONOutput() {
		entries := make([]listEntry, 0, len(locations))
		for _, l := range locations {
			entries = append(entries, listEntry{
				ID:           l.GetId(),
				Name:         l.GetName(),
				Status:       statusToString(l.GetStatus()),
				MonitorCount: l.GetMonitorCount(),
				LastSeenAt:   l.GetLastSeenAt(),
				CreatedAt:    l.GetCreatedAt(),
				UpdatedAt:    l.GetUpdatedAt(),
			})
		}
		return output.PrintJSON(entries)
	}

	if len(locations) == 0 {
		if !output.IsQuiet() {
			fmt.Println("No private locations found")
		}
		return nil
	}

	headerFmt := color.New(color.FgGreen, color.Underline).SprintfFunc()
	columnFmt := color.New(color.FgYellow).SprintfFunc()

	tbl := table.New("ID", "Name", "Status", "Monitors", "Last Seen")
	tbl.WithHeaderFormatter(headerFmt).WithFirstColumnFormatter(columnFmt)

	for _, l := range locations {
		tbl.AddRow(
			l.GetId(),
			l.GetName(),
			colorizeStatus(statusToString(l.GetStatus())),
			l.GetMonitorCount(),
			formatLastSeen(l.GetLastSeenAt()),
		)
	}

	tbl.Print()

	if total := int(resp.GetTotalSize()); total > len(locations) && !output.IsQuiet() {
		fmt.Printf("\nShowing %d of %d. Use --limit to see more.\n", len(locations), total)
	}

	return nil
}

func ListPrivateLocationsWithHTTPClient(ctx context.Context, httpClient *http.Client, apiKey string, limit int) error {
	client := NewPrivateLocationClientWithHTTPClient(httpClient, apiKey)
	return ListPrivateLocations(ctx, client, limit, nil)
}

func GetPrivateLocationListCmd() *cli.Command {
	return &cli.Command{
		Name:  "list",
		Usage: "List all private locations",
		UsageText: `openstatus private-locations list
  openstatus pl list
  openstatus pl list --limit 10`,
		Description: `List the private locations in your workspace, with the health of each agent
and the number of monitors it runs.`,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "access-token",
				Usage:   "OpenStatus API Access Token",
				Aliases: []string{"t"},
				Sources: cli.EnvVars("OPENSTATUS_API_TOKEN"),
			},
			&cli.IntFlag{
				Name:  "limit",
				Usage: "Maximum number of private locations to return (1-100)",
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			apiKey, err := auth.ResolveAccessToken(cmd)
			if err != nil {
				return cli.Exit(err.Error(), 1)
			}
			s := output.StartSpinner("Fetching private locations...")
			client := NewPrivateLocationClient(apiKey)
			if err := ListPrivateLocations(ctx, client, int(cmd.Int("limit")), s); err != nil {
				return cli.Exit(err.Error(), 1)
			}
			return nil
		},
	}
}
