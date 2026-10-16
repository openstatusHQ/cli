package incident

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"

	"buf.build/gen/go/openstatus/api/connectrpc/gosimple/openstatus/incident/v1/incidentv1connect"
	incidentv1 "buf.build/gen/go/openstatus/api/protocolbuffers/go/openstatus/incident/v1"
	"github.com/urfave/cli/v3"

	output "github.com/openstatusHQ/cli/internal/cli"
)

const noteUsage = "openstatus incident note <incident-id> --message <text|->"

func buildNoteRequest(id, message string) (*incidentv1.AddIncidentNoteRequest, error) {
	if strings.TrimSpace(message) == "" {
		return nil, errors.New("note message cannot be empty")
	}
	if err := validateNote(message); err != nil {
		return nil, err
	}
	return &incidentv1.AddIncidentNoteRequest{Id: id, Message: message}, nil
}

func AddIncidentNote(ctx context.Context, client incidentv1connect.IncidentServiceClient, id, message string) (*incidentv1.IncidentEvent, error) {
	req, err := buildNoteRequest(id, message)
	if err != nil {
		return nil, err
	}
	resp, err := client.AddIncidentNote(ctx, req)
	if err != nil {
		return nil, formatIncidentError(err, id, opGeneric)
	}
	return resp.GetEvent(), nil
}

func renderNote(id string, event *incidentv1.IncidentEvent) error {
	if output.IsJSONOutput() {
		return output.PrintJSON(toEventJSON(event))
	}
	if output.IsQuiet() {
		fmt.Println(event.GetId())
		return nil
	}
	fmt.Printf("Note added to incident %s\n", id)
	return nil
}

func AddIncidentNoteWithHTTPClient(ctx context.Context, httpClient *http.Client, apiKey string, id, message string) error {
	event, err := AddIncidentNote(ctx, NewIncidentClientWithHTTPClient(httpClient, apiKey), id, message)
	if err != nil {
		return err
	}
	return renderNote(id, event)
}

func GetIncidentNoteCmd() *cli.Command {
	return &cli.Command{
		Name:  "note",
		Usage: "Add a note to an incident timeline",
		UsageText: `openstatus incident note <IncidentID> --message "Rolled back the deploy"
  echo "multi-line markdown" | openstatus inc note 42 --message -`,
		Flags: []cli.Flag{
			accessTokenFlag(),
			&cli.StringFlag{
				Name:    "message",
				Usage:   "Note text in markdown (1-10000 characters); use - to read from stdin",
				Aliases: []string{"m"},
			},
		},
		Action: withClient(func(ctx context.Context, cmd *cli.Command, client incidentv1connect.IncidentServiceClient) error {
			id, err := requireID(cmd, noteUsage)
			if err != nil {
				return err
			}
			if cmd.String("message") == "" {
				return errors.New("missing required flags: --message")
			}
			message, err := readTextArg(cmd.String("message"), os.Stdin)
			if err != nil {
				return err
			}

			s := output.StartSpinner("Adding note...")
			event, err := AddIncidentNote(ctx, client, id, message)
			output.StopSpinner(s)
			if err != nil {
				return err
			}
			return renderNote(id, event)
		}),
	}
}
