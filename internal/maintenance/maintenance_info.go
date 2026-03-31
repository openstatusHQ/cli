package maintenance

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"

	maintenancev1 "buf.build/gen/go/openstatus/api/protocolbuffers/go/openstatus/maintenance/v1"
	"buf.build/gen/go/openstatus/api/connectrpc/gosimple/openstatus/maintenance/v1/maintenancev1connect"
	"github.com/openstatusHQ/cli/internal/auth"
	output "github.com/openstatusHQ/cli/internal/cli"
	"github.com/logrusorgru/aurora/v4"
	"github.com/olekukonko/tablewriter"
	"github.com/olekukonko/tablewriter/renderer"
	"github.com/olekukonko/tablewriter/tw"
	"github.com/urfave/cli/v3"
)

type maintenanceDetail struct {
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

func GetMaintenanceInfo(ctx context.Context, client maintenancev1connect.MaintenanceServiceClient, maintenanceId string, s *output.Spinner) error {
	if maintenanceId == "" {
		output.StopSpinner(s)
		fmt.Fprintln(os.Stderr, "Usage: openstatus maintenance info <maintenance-id>")
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "Example: openstatus maintenance info 12345")
		return fmt.Errorf("maintenance ID is required")
	}

	resp, err := client.GetMaintenance(ctx, &maintenancev1.GetMaintenanceRequest{
		Id: maintenanceId,
	})
	output.StopSpinner(s)
	if err != nil {
		return output.FormatError(err, "maintenance", maintenanceId)
	}

	m := resp.GetMaintenance()

	if output.IsJSONOutput() {
		detail := maintenanceDetail{
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
		}
		return output.PrintJSON(detail)
	}

	fmt.Println(aurora.Bold("Maintenance:"))
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
		{"ID", m.GetId()},
		{"Title", m.GetTitle()},
		{"Message", m.GetMessage()},
		{"Status", statusColor(timeWindowStatus(m.GetFrom(), m.GetTo()))},
		{"From", output.FormatTimestamp(m.GetFrom())},
		{"To", output.FormatTimestamp(m.GetTo())},
	}

	if len(m.GetPageComponentIds()) > 0 {
		data = append(data, []string{"Components", strings.Join(m.GetPageComponentIds(), ", ")})
	}

	data = append(data, []string{"Created", output.FormatTimestamp(m.GetCreatedAt())})
	data = append(data, []string{"Updated", output.FormatTimestamp(m.GetUpdatedAt())})

	table.Bulk(data)
	table.Render()

	return nil
}

func GetMaintenanceInfoWithHTTPClient(ctx context.Context, httpClient *http.Client, apiKey string, maintenanceId string) error {
	client := NewMaintenanceClientWithHTTPClient(httpClient, apiKey)
	return GetMaintenanceInfo(ctx, client, maintenanceId, nil)
}

func GetMaintenanceInfoCmd() *cli.Command {
	return &cli.Command{
		Name:  "info",
		Usage: "Get maintenance window details",
		UsageText: `openstatus maintenance info <MaintenanceID>
  openstatus maintenance info 12345`,
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
			maintenanceId := cmd.Args().Get(0)
			s := output.StartSpinner("Fetching maintenance...")
			client := NewMaintenanceClient(apiKey)
			err = GetMaintenanceInfo(ctx, client, maintenanceId, s)
			if err != nil {
				return cli.Exit(err.Error(), 1)
			}
			return nil
		},
	}
}
