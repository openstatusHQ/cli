package maintenance

import (
	"context"
	"fmt"
	"net/http"

	maintenancev1 "buf.build/gen/go/openstatus/api/protocolbuffers/go/openstatus/maintenance/v1"
	"buf.build/gen/go/openstatus/api/connectrpc/gosimple/openstatus/maintenance/v1/maintenancev1connect"
	"github.com/fatih/color"
	"github.com/openstatusHQ/cli/internal/auth"
	output "github.com/openstatusHQ/cli/internal/cli"
	"github.com/rodaine/table"
	"github.com/urfave/cli/v3"
)

type maintenanceListEntry struct {
	ID         string   `json:"id"`
	Title      string   `json:"title"`
	Message    string   `json:"message"`
	Status     string   `json:"status"`
	From       string   `json:"from"`
	To         string   `json:"to"`
	PageID     string   `json:"page_id"`
	Components []string `json:"components,omitempty"`
	CreatedAt  string   `json:"created_at"`
	UpdatedAt  string   `json:"updated_at"`
}

func ListMaintenances(ctx context.Context, client maintenancev1connect.MaintenanceServiceClient, pageId string, limit int, s *output.Spinner) error {
	req := &maintenancev1.ListMaintenancesRequest{}

	if limit > 0 {
		l := int32(limit)
		req.SetLimit(l)
	}

	if pageId != "" {
		req.SetPageId(pageId)
	}

	resp, err := client.ListMaintenances(ctx, req)
	output.StopSpinner(s)
	if err != nil {
		return output.FormatError(err, "maintenance", "")
	}

	maintenances := resp.GetMaintenances()

	if output.IsJSONOutput() {
		entries := make([]maintenanceListEntry, 0, len(maintenances))
		for _, m := range maintenances {
			entries = append(entries, maintenanceListEntry{
				ID:         m.GetId(),
				Title:      m.GetTitle(),
				Message:    m.GetMessage(),
				Status:     timeWindowStatus(m.GetFrom(), m.GetTo()),
				From:       m.GetFrom(),
				To:         m.GetTo(),
				PageID:     m.GetPageId(),
				Components: m.GetPageComponentIds(),
				CreatedAt:  m.GetCreatedAt(),
				UpdatedAt:  m.GetUpdatedAt(),
			})
		}
		return output.PrintJSON(entries)
	}

	if len(maintenances) == 0 {
		fmt.Println("No maintenances found")
		return nil
	}

	headerFmt := color.New(color.FgGreen, color.Underline).SprintfFunc()
	columnFmt := color.New(color.FgYellow).SprintfFunc()

	tbl := table.New("ID", "Title", "Status", "From", "To")
	tbl.WithHeaderFormatter(headerFmt).WithFirstColumnFormatter(columnFmt)

	for _, m := range maintenances {
		tbl.AddRow(
			m.GetId(),
			m.GetTitle(),
			statusColor(timeWindowStatus(m.GetFrom(), m.GetTo())),
			output.FormatTimestamp(m.GetFrom()),
			output.FormatTimestamp(m.GetTo()),
		)
	}

	tbl.Print()
	return nil
}

func ListMaintenancesWithHTTPClient(ctx context.Context, httpClient *http.Client, apiKey string, pageId string, limit int) error {
	client := NewMaintenanceClientWithHTTPClient(httpClient, apiKey)
	return ListMaintenances(ctx, client, pageId, limit, nil)
}

func GetMaintenanceListCmd() *cli.Command {
	return &cli.Command{
		Name:  "list",
		Usage: "List all maintenance windows",
		UsageText: `openstatus maintenance list
  openstatus maintenance list --page-id abc --limit 10`,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "access-token",
				Usage:   "OpenStatus API Access Token",
				Aliases: []string{"t"},
				Sources: cli.EnvVars("OPENSTATUS_API_TOKEN"),
			},
			&cli.StringFlag{
				Name:  "page-id",
				Usage: "Filter by status page ID",
			},
			&cli.IntFlag{
				Name:  "limit",
				Usage: "Maximum number of maintenances to return (1-100)",
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			apiKey, err := auth.ResolveAccessToken(cmd)
			if err != nil {
				return cli.Exit(err.Error(), 1)
			}
			s := output.StartSpinner("Fetching maintenances...")
			client := NewMaintenanceClient(apiKey)
			err = ListMaintenances(ctx, client, cmd.String("page-id"), int(cmd.Int("limit")), s)
			if err != nil {
				return cli.Exit(err.Error(), 1)
			}
			return nil
		},
	}
}
