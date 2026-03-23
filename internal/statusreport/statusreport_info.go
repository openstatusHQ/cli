package statusreport

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"

	status_reportv1 "buf.build/gen/go/openstatus/api/protocolbuffers/go/openstatus/status_report/v1"
	"buf.build/gen/go/openstatus/api/connectrpc/gosimple/openstatus/status_report/v1/status_reportv1connect"
	"github.com/openstatusHQ/cli/internal/auth"
	output "github.com/openstatusHQ/cli/internal/cli"
	"github.com/logrusorgru/aurora/v4"
	"github.com/olekukonko/tablewriter"
	"github.com/olekukonko/tablewriter/renderer"
	"github.com/olekukonko/tablewriter/tw"
	"github.com/urfave/cli/v3"
)

type statusReportDetail struct {
	ID         string               `json:"id"`
	Title      string               `json:"title"`
	Status     string               `json:"status"`
	Components []string             `json:"components,omitempty"`
	CreatedAt  string               `json:"created_at"`
	UpdatedAt  string               `json:"updated_at"`
	Updates    []statusReportUpdate `json:"updates,omitempty"`
}

type statusReportUpdate struct {
	Date    string `json:"date"`
	Status  string `json:"status"`
	Message string `json:"message"`
}

func GetStatusReportInfo(ctx context.Context, client status_reportv1connect.StatusReportServiceClient, reportId string, s *output.Spinner) error {
	if reportId == "" {
		output.StopSpinner(s)
		fmt.Fprintln(os.Stderr, "Usage: openstatus status-report info <report-id>")
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "Example: openstatus status-report info 12345")
		return fmt.Errorf("report ID is required")
	}

	resp, err := client.GetStatusReport(ctx, &status_reportv1.GetStatusReportRequest{
		Id: reportId,
	})
	output.StopSpinner(s)
	if err != nil {
		return output.FormatError(err, "status-report", reportId)
	}

	report := resp.GetStatusReport()

	if output.IsJSONOutput() {
		detail := statusReportDetail{
			ID:         report.GetId(),
			Title:      report.GetTitle(),
			Status:     statusToString(report.GetStatus()),
			Components: report.GetPageComponentIds(),
			CreatedAt:  report.GetCreatedAt(),
			UpdatedAt:  report.GetUpdatedAt(),
		}
		for _, u := range report.GetUpdates() {
			detail.Updates = append(detail.Updates, statusReportUpdate{
				Date:    u.GetDate(),
				Status:  statusToString(u.GetStatus()),
				Message: u.GetMessage(),
			})
		}
		return output.PrintJSON(detail)
	}

	fmt.Println(aurora.Bold("Status Report:"))
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
		{"ID", report.GetId()},
		{"Title", report.GetTitle()},
		{"Status", statusColor(statusToString(report.GetStatus()))},
	}

	if len(report.GetPageComponentIds()) > 0 {
		data = append(data, []string{"Components", strings.Join(report.GetPageComponentIds(), ", ")})
	}

	data = append(data, []string{"Created", report.GetCreatedAt()})
	data = append(data, []string{"Updated", report.GetUpdatedAt()})

	table.Bulk(data)
	table.Render()

	updates := report.GetUpdates()
	if len(updates) == 0 {
		fmt.Println("\nNo updates yet")
		return nil
	}

	fmt.Println(aurora.Bold("\nUpdate Timeline:"))
	for _, u := range updates {
		fmt.Printf("  %s  [%s]  %s\n",
			u.GetDate(),
			statusColor(statusToString(u.GetStatus())),
			u.GetMessage(),
		)
	}

	return nil
}

func GetStatusReportInfoWithHTTPClient(ctx context.Context, httpClient *http.Client, apiKey string, reportId string) error {
	client := NewStatusReportClientWithHTTPClient(httpClient, apiKey)
	return GetStatusReportInfo(ctx, client, reportId, nil)
}

func GetStatusReportInfoCmd() *cli.Command {
	return &cli.Command{
		Name:  "info",
		Usage: "Get status report details",
		UsageText: `openstatus status-report info <ReportID>
  openstatus status-report info 12345`,
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
			reportId := cmd.Args().Get(0)
			s := output.StartSpinner("Fetching status report...")
			client := NewStatusReportClient(apiKey)
			err = GetStatusReportInfo(ctx, client, reportId, s)
			if err != nil {
				return cli.Exit(err.Error(), 1)
			}
			return nil
		},
	}
}
