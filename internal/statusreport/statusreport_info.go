package statusreport

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"

	status_reportv1 "buf.build/gen/go/openstatus/api/protocolbuffers/go/openstatus/status_report/v1"
	"buf.build/gen/go/openstatus/api/connectrpc/gosimple/openstatus/status_report/v1/status_reportv1connect"
	"github.com/logrusorgru/aurora/v4"
	"github.com/olekukonko/tablewriter"
	"github.com/olekukonko/tablewriter/renderer"
	"github.com/olekukonko/tablewriter/tw"
	"github.com/urfave/cli/v3"
)

func GetStatusReportInfo(client status_reportv1connect.StatusReportServiceClient, reportId string) error {
	if reportId == "" {
		return fmt.Errorf("report ID is required")
	}

	resp, err := client.GetStatusReport(context.Background(), &status_reportv1.GetStatusReportRequest{
		Id: reportId,
	})
	if err != nil {
		return fmt.Errorf("status report not found. Run 'openstatus status-report list' to see available reports")
	}

	report := resp.GetStatusReport()

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

func GetStatusReportInfoWithHTTPClient(httpClient *http.Client, apiKey string, reportId string) error {
	client := NewStatusReportClientWithHTTPClient(httpClient, apiKey)
	return GetStatusReportInfo(client, reportId)
}

func GetStatusReportInfoCmd() *cli.Command {
	return &cli.Command{
		Name:      "info",
		Usage:     "Get status report details",
		UsageText: "openstatus status-report info <ReportID>",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "access-token",
				Usage:    "OpenStatus API Access Token",
				Aliases:  []string{"t"},
				Sources:  cli.EnvVars("OPENSTATUS_API_TOKEN"),
				Required: true,
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			reportId := cmd.Args().Get(0)
			client := NewStatusReportClient(cmd.String("access-token"))
			err := GetStatusReportInfo(client, reportId)
			if err != nil {
				return cli.Exit(err.Error(), 1)
			}
			return nil
		},
	}
}
