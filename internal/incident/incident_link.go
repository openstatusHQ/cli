package incident

import (
	"context"
	"fmt"
	"net/http"

	"buf.build/gen/go/openstatus/api/connectrpc/gosimple/openstatus/incident/v1/incidentv1connect"
	incidentv1 "buf.build/gen/go/openstatus/api/protocolbuffers/go/openstatus/incident/v1"
	"github.com/urfave/cli/v3"

	output "github.com/openstatusHQ/cli/internal/cli"
)

const (
	linkUsage   = "openstatus incident link-report <incident-id> <status-report-id>"
	unlinkUsage = "openstatus incident unlink-report <incident-id>"
)

func LinkStatusReport(ctx context.Context, client incidentv1connect.IncidentServiceClient, id, reportID string) (*incidentv1.Incident, error) {
	resp, err := client.LinkStatusReport(ctx, &incidentv1.LinkStatusReportRequest{Id: id, StatusReportId: reportID})
	if err != nil {
		return nil, formatIncidentError(err, id, opLink)
	}
	return resp.GetIncident(), nil
}

func UnlinkStatusReport(ctx context.Context, client incidentv1connect.IncidentServiceClient, id string) (*incidentv1.Incident, error) {
	resp, err := client.UnlinkStatusReport(ctx, &incidentv1.UnlinkStatusReportRequest{Id: id})
	if err != nil {
		return nil, formatIncidentError(err, id, opGeneric)
	}
	return resp.GetIncident(), nil
}

func LinkStatusReportWithHTTPClient(ctx context.Context, httpClient *http.Client, apiKey string, id, reportID string) error {
	inc, err := LinkStatusReport(ctx, NewIncidentClientWithHTTPClient(httpClient, apiKey), id, reportID)
	if err != nil {
		return err
	}
	return renderIncidentResult(inc, fmt.Sprintf("Status report %s linked to incident %s", reportID, id))
}

func UnlinkStatusReportWithHTTPClient(ctx context.Context, httpClient *http.Client, apiKey string, id string) error {
	inc, err := UnlinkStatusReport(ctx, NewIncidentClientWithHTTPClient(httpClient, apiKey), id)
	if err != nil {
		return err
	}
	return renderIncidentResult(inc, fmt.Sprintf("Status report unlinked from incident %s", id))
}

func GetIncidentLinkReportCmd() *cli.Command {
	return &cli.Command{
		Name:  "link-report",
		Usage: "Link a public status report to an incident",
		UsageText: `openstatus incident link-report <IncidentID> <StatusReportID>
  openstatus inc link-report 42 123`,
		Flags: []cli.Flag{accessTokenFlag()},
		Action: withClient(func(ctx context.Context, cmd *cli.Command, client incidentv1connect.IncidentServiceClient) error {
			id, err := requireID(cmd, linkUsage)
			if err != nil {
				return err
			}
			reportID, err := requireArg(cmd, 1, "status report ID", linkUsage)
			if err != nil {
				return err
			}

			s := output.StartSpinner("Linking status report...")
			inc, err := LinkStatusReport(ctx, client, id, reportID)
			output.StopSpinner(s)
			if err != nil {
				return err
			}
			return renderIncidentResult(inc, fmt.Sprintf("Status report %s linked to incident %s", reportID, id))
		}),
	}
}

func GetIncidentUnlinkReportCmd() *cli.Command {
	return &cli.Command{
		Name:  "unlink-report",
		Usage: "Remove the link to an incident's status report",
		UsageText: `openstatus incident unlink-report <IncidentID>
  openstatus inc unlink-report 42`,
		Flags: []cli.Flag{accessTokenFlag()},
		Action: withClient(func(ctx context.Context, cmd *cli.Command, client incidentv1connect.IncidentServiceClient) error {
			id, err := requireID(cmd, unlinkUsage)
			if err != nil {
				return err
			}

			s := output.StartSpinner("Unlinking status report...")
			inc, err := UnlinkStatusReport(ctx, client, id)
			output.StopSpinner(s)
			if err != nil {
				return err
			}
			return renderIncidentResult(inc, fmt.Sprintf("Status report unlinked from incident %s", id))
		}),
	}
}
