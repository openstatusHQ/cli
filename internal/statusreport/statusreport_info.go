package statusreport

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"

	"buf.build/gen/go/openstatus/api/connectrpc/gosimple/openstatus/status_report/v1/status_reportv1connect"
	status_reportv1 "buf.build/gen/go/openstatus/api/protocolbuffers/go/openstatus/status_report/v1"
	"github.com/logrusorgru/aurora/v4"
	"github.com/olekukonko/tablewriter"
	"github.com/olekukonko/tablewriter/renderer"
	"github.com/olekukonko/tablewriter/tw"
	"github.com/urfave/cli/v3"

	"github.com/openstatusHQ/cli/internal/auth"
	output "github.com/openstatusHQ/cli/internal/cli"
)

type statusReportDetail struct {
	ID             string                `json:"id"`
	Title          string                `json:"title"`
	Status         string                `json:"status"`
	Components     []string              `json:"components,omitempty"`
	CurrentImpacts []componentImpactJSON `json:"current_impacts,omitempty"`
	CreatedAt      string                `json:"created_at"`
	UpdatedAt      string                `json:"updated_at"`
	Updates        []statusReportUpdate  `json:"updates,omitempty"`
}

type statusReportUpdate struct {
	Date    string                `json:"date"`
	Status  string                `json:"status"`
	Message string                `json:"message"`
	Impacts []componentImpactJSON `json:"impacts,omitempty"`
}

type componentImpactJSON struct {
	ComponentID string `json:"component_id"`
	Impact      string `json:"impact"`
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

	affected := affectedOrder(report)
	current := currentImpacts(report.GetUpdates())

	if output.IsJSONOutput() {
		detail := statusReportDetail{
			ID:         report.GetId(),
			Title:      report.GetTitle(),
			Status:     statusToString(report.GetStatus()),
			Components: report.GetPageComponentIds(),
			CreatedAt:  report.GetCreatedAt(),
			UpdatedAt:  report.GetUpdatedAt(),
		}
		for _, id := range affected {
			detail.CurrentImpacts = append(detail.CurrentImpacts, componentImpactJSON{
				ComponentID: id,
				Impact:      impactToString(current[id]),
			})
		}
		for _, u := range report.GetUpdates() {
			ue := statusReportUpdate{
				Date:    u.GetDate(),
				Status:  statusToString(u.GetStatus()),
				Message: u.GetMessage(),
			}
			for _, ci := range u.GetComponentImpacts() {
				ue.Impacts = append(ue.Impacts, componentImpactJSON{
					ComponentID: ci.GetPageComponentId(),
					Impact:      impactToString(ci.GetImpact()),
				})
			}
			detail.Updates = append(detail.Updates, ue)
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
		{"Created", output.FormatTimestamp(report.GetCreatedAt())},
		{"Updated", output.FormatTimestamp(report.GetUpdatedAt())},
	}

	table.Bulk(data)
	table.Render()

	if len(affected) > 0 {
		printAffectedSection(report, affected, current)
	}

	updates := report.GetUpdates()
	if len(updates) == 0 {
		fmt.Println("\nNo updates yet")
		return nil
	}

	fmt.Println(aurora.Bold("\nUpdate Timeline:"))
	for _, u := range updates {
		fmt.Printf("  %s  [%s]  %s\n",
			output.FormatTimestamp(u.GetDate()),
			statusColor(statusToString(u.GetStatus())),
			u.GetMessage(),
		)
		if len(u.GetComponentImpacts()) > 0 {
			parts := make([]string, 0, len(u.GetComponentImpacts()))
			for _, ci := range u.GetComponentImpacts() {
				parts = append(parts, ci.GetPageComponentId()+"="+impactColor(impactToString(ci.GetImpact())))
			}
			fmt.Printf("    → %s\n", strings.Join(parts, ", "))
		}
	}

	return nil
}

func printAffectedSection(report *status_reportv1.StatusReport, affected []string, current map[string]status_reportv1.PageComponentImpact) {
	fmt.Println(aurora.Bold("\nAffected:"))

	priorPerComponent := make(map[string]status_reportv1.PageComponentImpact)
	finalPerComponent := make(map[string]status_reportv1.PageComponentImpact)
	for _, u := range report.GetUpdates() {
		for _, ci := range u.GetComponentImpacts() {
			id := ci.GetPageComponentId()
			if prev, ok := finalPerComponent[id]; ok && prev != ci.GetImpact() {
				priorPerComponent[id] = prev
			}
			finalPerComponent[id] = ci.GetImpact()
		}
	}

	maxID := 0
	for _, id := range affected {
		if len(id) > maxID {
			maxID = len(id)
		}
	}

	for _, id := range affected {
		impact := current[id]
		token := impactToString(impact)
		display := "—"
		if token != "" {
			display = impactColor(token)
		}
		line := fmt.Sprintf("  %-*s  %s", maxID, id, display)
		if prior, ok := priorPerComponent[id]; ok {
			line += "  ← was " + impactToString(prior)
		}
		fmt.Println(line)
	}
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
