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

const deleteUsage = "openstatus incident delete <incident-id> [-y]"

type deleteOutput struct {
	ID      string `json:"id"`
	Deleted bool   `json:"deleted"`
}

func DeleteIncident(ctx context.Context, client incidentv1connect.IncidentServiceClient, id string) error {
	if _, err := client.DeleteIncident(ctx, &incidentv1.DeleteIncidentRequest{Id: id}); err != nil {
		return formatIncidentError(err, id, opDelete)
	}
	return nil
}

func renderDeleted(id string) error {
	if output.IsJSONOutput() {
		return output.PrintJSON(deleteOutput{ID: id, Deleted: true})
	}
	if output.IsQuiet() {
		fmt.Println(id)
		return nil
	}
	fmt.Printf("Incident %s deleted\n", id)
	fmt.Println("Run 'openstatus inc list' to see remaining incidents")
	return nil
}

func DeleteIncidentWithHTTPClient(ctx context.Context, httpClient *http.Client, apiKey string, id string) error {
	if err := DeleteIncident(ctx, NewIncidentClientWithHTTPClient(httpClient, apiKey), id); err != nil {
		return err
	}
	return renderDeleted(id)
}

func GetIncidentDeleteCmd() *cli.Command {
	return &cli.Command{
		Name:  "delete",
		Usage: "Delete an incident declared by mistake",
		UsageText: `openstatus incident delete <IncidentID>
  openstatus inc delete 42 -y`,
		Description: `Delete an incident declared by mistake. Only allowed while the incident is
open and was never mitigated, resolved or closed; cancel it otherwise.`,
		Flags: []cli.Flag{accessTokenFlag(), autoAcceptFlag()},
		Action: withClient(func(ctx context.Context, cmd *cli.Command, client incidentv1connect.IncidentServiceClient) error {
			id, err := requireID(cmd, deleteUsage)
			if err != nil {
				return err
			}
			confirmed, err := confirm(cmd, fmt.Sprintf("You are about to delete incident %s, do you want to continue", id))
			if err != nil {
				return err
			}
			if !confirmed {
				return nil
			}

			s := output.StartSpinner("Deleting incident...")
			err = DeleteIncident(ctx, client, id)
			output.StopSpinner(s)
			if err != nil {
				return err
			}
			return renderDeleted(id)
		}),
	}
}
