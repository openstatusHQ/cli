package incident

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"unicode/utf8"

	"buf.build/gen/go/openstatus/api/connectrpc/gosimple/openstatus/incident/v1/incidentv1connect"
	incidentv1 "buf.build/gen/go/openstatus/api/protocolbuffers/go/openstatus/incident/v1"
	"github.com/urfave/cli/v3"

	output "github.com/openstatusHQ/cli/internal/cli"
)

const maxNoteLen = 10000

type statusVerb struct {
	name   string
	status string
	usage  string
	done   string
}

var (
	verbMitigate = statusVerb{name: "mitigate", status: "mitigated", usage: "Mark an incident as mitigated", done: "mitigated"}
	verbResolve  = statusVerb{name: "resolve", status: "resolved", usage: "Mark an incident as resolved", done: "resolved"}
	verbReopen   = statusVerb{name: "reopen", status: "open", usage: "Reopen a mitigated or resolved incident", done: "reopened"}
	verbCancel   = statusVerb{name: "cancel", status: "canceled", usage: "Cancel an incident (closes it)", done: "canceled"}
)

func statusDoneWord(status string) string {
	for _, v := range []statusVerb{verbMitigate, verbResolve, verbReopen, verbCancel} {
		if v.status == status {
			return v.done
		}
	}
	return status
}

func needsConfirmation(status string, autoAccept bool) bool {
	return status == "canceled" && !autoAccept
}

func validateNote(note string) error {
	if utf8.RuneCountInString(note) > maxNoteLen {
		return fmt.Errorf("note is too long: maximum %d characters", maxNoteLen)
	}
	return nil
}

func buildSetStatusRequest(id, status, note string) (*incidentv1.SetIncidentStatusRequest, error) {
	sdkStatus, err := statusToSDK(status)
	if err != nil {
		return nil, err
	}
	if err := validateNote(note); err != nil {
		return nil, err
	}
	req := &incidentv1.SetIncidentStatusRequest{Id: id, Status: sdkStatus}
	if note != "" {
		req.SetNote(note)
	}
	return req, nil
}

func SetIncidentStatus(ctx context.Context, client incidentv1connect.IncidentServiceClient, id, status, note string) (*incidentv1.Incident, error) {
	req, err := buildSetStatusRequest(id, status, note)
	if err != nil {
		return nil, err
	}
	resp, err := client.SetIncidentStatus(ctx, req)
	if err != nil {
		return nil, formatIncidentError(err, id, opSetStatus)
	}
	return resp.GetIncident(), nil
}

func statusChangedMessage(id, status string) string {
	return fmt.Sprintf("Incident %s %s", id, statusDoneWord(status))
}

func SetIncidentStatusWithHTTPClient(ctx context.Context, httpClient *http.Client, apiKey string, id, status, note string) error {
	inc, err := SetIncidentStatus(ctx, NewIncidentClientWithHTTPClient(httpClient, apiKey), id, status, note)
	if err != nil {
		return err
	}
	return renderIncidentResult(inc, statusChangedMessage(id, status))
}

func runSetStatus(ctx context.Context, cmd *cli.Command, client incidentv1connect.IncidentServiceClient, id, status string) error {
	if _, err := statusToSDK(status); err != nil {
		return err
	}
	if needsConfirmation(status, cmd.Bool("auto-accept")) {
		confirmed, err := confirm(cmd, fmt.Sprintf("You are about to cancel incident %s. This closes it and cannot be undone. Continue", id))
		if err != nil {
			return err
		}
		if !confirmed {
			return nil
		}
	}

	s := output.StartSpinner("Updating incident status...")
	inc, err := SetIncidentStatus(ctx, client, id, status, cmd.String("note"))
	output.StopSpinner(s)
	if err != nil {
		return err
	}
	return renderIncidentResult(inc, statusChangedMessage(id, status))
}

func noteFlag() cli.Flag {
	return &cli.StringFlag{
		Name:  "note",
		Usage: "Note recorded on the timeline with the change",
	}
}

func GetIncidentSetStatusCmd() *cli.Command {
	usage := "openstatus incident set-status <incident-id> --status <open|mitigated|resolved|canceled> [--note ...]"
	return &cli.Command{
		Name:  "set-status",
		Usage: "Move an incident to a new status",
		UsageText: `openstatus incident set-status <IncidentID> --status <open|mitigated|resolved|canceled> [--note "..."]
  openstatus inc set-status 42 --status mitigated --note "Rolled back the deploy"`,
		Description: `Allowed transitions: open -> mitigated, resolved or canceled; mitigated ->
resolved, open or canceled; resolved -> open. Canceled is terminal and closes
the incident. A linked status report is not updated: post the public update with
'openstatus status-report add-update'.`,
		Flags: []cli.Flag{
			accessTokenFlag(),
			&cli.StringFlag{
				Name:  "status",
				Usage: "Target status (open, mitigated, resolved, canceled)",
			},
			noteFlag(),
			autoAcceptFlag(),
		},
		Action: withClient(func(ctx context.Context, cmd *cli.Command, client incidentv1connect.IncidentServiceClient) error {
			id, err := requireID(cmd, usage)
			if err != nil {
				return err
			}
			if cmd.String("status") == "" {
				return errors.New("missing required flags: --status")
			}
			return runSetStatus(ctx, cmd, client, id, cmd.String("status"))
		}),
	}
}

func newStatusVerbCmd(v statusVerb) *cli.Command {
	usage := fmt.Sprintf("openstatus incident %s <incident-id> [--note ...]", v.name)
	flags := []cli.Flag{accessTokenFlag(), noteFlag()}
	if v.status == "canceled" {
		flags = append(flags, autoAcceptFlag())
	}
	return &cli.Command{
		Name:  v.name,
		Usage: v.usage,
		UsageText: fmt.Sprintf(`openstatus incident %s <IncidentID> [--note "..."]
  openstatus inc %s 42 --note "..."`, v.name, v.name),
		Flags: flags,
		Action: withClient(func(ctx context.Context, cmd *cli.Command, client incidentv1connect.IncidentServiceClient) error {
			id, err := requireID(cmd, usage)
			if err != nil {
				return err
			}
			return runSetStatus(ctx, cmd, client, id, v.status)
		}),
	}
}
