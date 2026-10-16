package incident

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"buf.build/gen/go/openstatus/api/connectrpc/gosimple/openstatus/incident/v1/incidentv1connect"
	incidentv1 "buf.build/gen/go/openstatus/api/protocolbuffers/go/openstatus/incident/v1"
	"github.com/logrusorgru/aurora/v4"
	"github.com/urfave/cli/v3"

	output "github.com/openstatusHQ/cli/internal/cli"
)

const infoUsage = "openstatus incident info <incident-id>"

func GetIncidentInfo(ctx context.Context, client incidentv1connect.IncidentServiceClient, id string, s *output.Spinner) error {
	resp, err := client.GetIncident(ctx, &incidentv1.GetIncidentRequest{Id: id})
	output.StopSpinner(s)
	if err != nil {
		return formatIncidentError(err, id, opGeneric)
	}

	inc := resp.GetIncident()

	if output.IsJSONOutput() {
		return output.PrintJSON(toDetailJSON(inc))
	}

	fmt.Println(aurora.Bold("Incident:"))
	tbl := newBlueprintTable()
	tbl.Bulk(infoRows(inc))
	tbl.Render()

	printTimeline(inc.GetEvents())
	return nil
}

func infoRows(inc *incidentv1.Incident) [][]string {
	rows := [][]string{
		{"ID", inc.GetId()},
		{"Title", inc.GetTitle()},
		{"Severity", severityColor(severityToString(inc.GetSeverity()))},
		{"Status", displayStatus(inc.GetStatus(), inc.GetClosedAt())},
	}
	if inc.HasSummary() {
		rows = append(rows, []string{"Summary", inc.GetSummary()})
	}
	rows = append(rows, []string{"Commander", formatUser(inc.GetCommander())})
	if inc.HasDeclaredBy() {
		rows = append(rows, []string{"Declared by", formatUser(inc.GetDeclaredBy())})
	}
	rows = append(rows,
		[]string{"Declared", output.FormatTimestamp(inc.GetDeclaredAt())},
		[]string{"Started", output.FormatTimestamp(inc.GetStartedAt())},
	)
	if inc.HasMitigatedAt() {
		rows = append(rows, []string{"Mitigated", output.FormatTimestamp(inc.GetMitigatedAt())})
	}
	if inc.HasResolvedAt() {
		resolved := output.FormatTimestamp(inc.GetResolvedAt())
		if inc.HasResolvedBy() {
			resolved += " by " + formatUser(inc.GetResolvedBy())
		}
		rows = append(rows, []string{"Resolved", resolved})
	}
	if inc.HasClosedAt() {
		rows = append(rows, []string{"Closed", output.FormatTimestamp(inc.GetClosedAt())})
	}
	if inc.HasStatusReport() {
		r := inc.GetStatusReport()
		rows = append(rows, []string{"Status report", fmt.Sprintf("%s %q [%s]", r.GetId(), r.GetTitle(), statusReportStatusToString(r.GetStatus().String()))})
	}
	if inc.HasSlackChannelUrl() {
		rows = append(rows, []string{"Slack channel", inc.GetSlackChannelUrl()})
	}
	transitions := "none"
	if t := statusesToStrings(inc.GetAllowedTransitions()); len(t) > 0 {
		transitions = strings.Join(t, ", ")
	}
	deletable := "no"
	if inc.GetDeletable() {
		deletable = "yes"
	}
	rows = append(rows,
		[]string{"Allowed transitions", transitions},
		[]string{"Deletable", deletable},
		[]string{"Created", output.FormatTimestamp(inc.GetCreatedAt())},
		[]string{"Updated", output.FormatTimestamp(inc.GetUpdatedAt())},
	)
	return rows
}

func printTimeline(events []*incidentv1.IncidentEvent) {
	if len(events) == 0 {
		fmt.Println("\nNo timeline events")
		return
	}

	fmt.Println(aurora.Bold("\nTimeline:"))
	for _, e := range events {
		lines := strings.Split(e.GetMessage(), "\n")
		line := fmt.Sprintf("  %s  %s  %s", output.FormatTimestamp(e.GetCreatedAt()), eventTypeToString(e.GetType()), lines[0])
		if e.HasCreatedBy() {
			line += "  — " + shortUser(e.GetCreatedBy())
		}
		fmt.Println(line)
		for _, l := range lines[1:] {
			fmt.Println("    " + l)
		}
	}
}

func GetIncidentInfoWithHTTPClient(ctx context.Context, httpClient *http.Client, apiKey string, id string) error {
	return GetIncidentInfo(ctx, NewIncidentClientWithHTTPClient(httpClient, apiKey), id, nil)
}

func GetIncidentInfoCmd() *cli.Command {
	return &cli.Command{
		Name:  "info",
		Usage: "Get incident details and timeline",
		UsageText: `openstatus incident info <IncidentID>
  openstatus inc info 42`,
		Flags: []cli.Flag{accessTokenFlag()},
		Action: withClient(func(ctx context.Context, cmd *cli.Command, client incidentv1connect.IncidentServiceClient) error {
			id, err := requireID(cmd, infoUsage)
			if err != nil {
				return err
			}
			s := output.StartSpinner("Fetching incident...")
			return GetIncidentInfo(ctx, client, id, s)
		}),
	}
}
