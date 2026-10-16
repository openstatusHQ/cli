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

const closeUsage = "openstatus incident close <incident-id> [--skip-postmortem] [-y]"

func CloseIncident(ctx context.Context, client incidentv1connect.IncidentServiceClient, id string, skipPostmortem bool) (*incidentv1.Incident, error) {
	req := &incidentv1.CloseIncidentRequest{Id: id}
	if skipPostmortem {
		req.SetSkipPostmortem(true)
	}
	resp, err := client.CloseIncident(ctx, req)
	if err != nil {
		return nil, formatIncidentError(err, id, opClose)
	}
	return resp.GetIncident(), nil
}

func CloseIncidentWithHTTPClient(ctx context.Context, httpClient *http.Client, apiKey string, id string, skipPostmortem bool) error {
	inc, err := CloseIncident(ctx, NewIncidentClientWithHTTPClient(httpClient, apiKey), id, skipPostmortem)
	if err != nil {
		return err
	}
	return renderIncidentResult(inc, fmt.Sprintf("Incident %s closed", id))
}

func GetIncidentCloseCmd() *cli.Command {
	return &cli.Command{
		Name:  "close",
		Usage: "Close a resolved incident",
		UsageText: `openstatus incident close <IncidentID>
  openstatus inc close 42 --skip-postmortem -y`,
		Description: `Close a resolved incident. A closed incident is read-only except for its
postmortem. Requires an approved postmortem, or --skip-postmortem.`,
		Flags: []cli.Flag{
			accessTokenFlag(),
			&cli.BoolFlag{
				Name:  "skip-postmortem",
				Usage: "Close without an approved postmortem",
			},
			autoAcceptFlag(),
		},
		Action: withClient(func(ctx context.Context, cmd *cli.Command, client incidentv1connect.IncidentServiceClient) error {
			id, err := requireID(cmd, closeUsage)
			if err != nil {
				return err
			}
			confirmed, err := confirm(cmd, fmt.Sprintf("You are about to close incident %s. It becomes read-only except for its postmortem. Continue", id))
			if err != nil {
				return err
			}
			if !confirmed {
				return nil
			}

			s := output.StartSpinner("Closing incident...")
			inc, err := CloseIncident(ctx, client, id, cmd.Bool("skip-postmortem"))
			output.StopSpinner(s)
			if err != nil {
				return err
			}
			return renderIncidentResult(inc, fmt.Sprintf("Incident %s closed", id))
		}),
	}
}
