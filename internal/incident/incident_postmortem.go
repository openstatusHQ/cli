package incident

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"unicode/utf8"

	"buf.build/gen/go/openstatus/api/connectrpc/gosimple/openstatus/incident/v1/incidentv1connect"
	incidentv1 "buf.build/gen/go/openstatus/api/protocolbuffers/go/openstatus/incident/v1"
	"github.com/urfave/cli/v3"

	output "github.com/openstatusHQ/cli/internal/cli"
)

const (
	maxPostmortemLen = 100000
	pmShowUsage      = "openstatus incident postmortem show <incident-id>"
	pmEditUsage      = "openstatus incident postmortem edit <incident-id> --file <path|-> | --content <text>"
	pmApproveUsage   = "openstatus incident postmortem approve <incident-id> [--close] [-y]"
)

type postmortemJSON struct {
	IncidentID string    `json:"incident_id"`
	Status     string    `json:"status"`
	Content    string    `json:"content"`
	DraftedBy  string    `json:"drafted_by"`
	ApprovedBy *userJSON `json:"approved_by,omitempty"`
	ApprovedAt string    `json:"approved_at,omitempty"`
	CreatedAt  string    `json:"created_at"`
	UpdatedAt  string    `json:"updated_at"`
}

type approveOutput struct {
	Postmortem postmortemJSON `json:"postmortem"`
	Incident   incidentDetail `json:"incident"`
}

func postmortemStatusToString(s incidentv1.PostmortemStatus) string {
	switch s {
	case incidentv1.PostmortemStatus_POSTMORTEM_STATUS_DRAFT:
		return "draft"
	case incidentv1.PostmortemStatus_POSTMORTEM_STATUS_APPROVED:
		return "approved"
	default:
		return "unknown"
	}
}

func postmortemAuthorToString(a incidentv1.PostmortemAuthor) string {
	switch a {
	case incidentv1.PostmortemAuthor_POSTMORTEM_AUTHOR_AGENT:
		return "agent"
	case incidentv1.PostmortemAuthor_POSTMORTEM_AUTHOR_USER:
		return "user"
	default:
		return "unknown"
	}
}

func toPostmortemJSON(pm *incidentv1.Postmortem) postmortemJSON {
	return postmortemJSON{
		IncidentID: pm.GetIncidentId(),
		Status:     postmortemStatusToString(pm.GetStatus()),
		Content:    pm.GetContent(),
		DraftedBy:  postmortemAuthorToString(pm.GetDraftedBy()),
		ApprovedBy: toUserJSON(pm.GetApprovedBy()),
		ApprovedAt: pm.GetApprovedAt(),
		CreatedAt:  pm.GetCreatedAt(),
		UpdatedAt:  pm.GetUpdatedAt(),
	}
}

func GetPostmortem(ctx context.Context, client incidentv1connect.IncidentServiceClient, id string) (*incidentv1.Postmortem, error) {
	resp, err := client.GetPostmortem(ctx, &incidentv1.GetPostmortemRequest{IncidentId: id})
	if err != nil {
		return nil, formatIncidentError(err, id, opGeneric)
	}
	if !resp.HasPostmortem() {
		return nil, nil
	}
	return resp.GetPostmortem(), nil
}

func writePostmortem(w io.Writer, pm *incidentv1.Postmortem) {
	fmt.Fprintf(w, "Status:     %s\n", postmortemStatusToString(pm.GetStatus()))
	fmt.Fprintf(w, "Drafted by: %s\n", postmortemAuthorToString(pm.GetDraftedBy()))
	if pm.HasApprovedBy() {
		fmt.Fprintf(w, "Approved:   %s by %s\n", output.FormatTimestamp(pm.GetApprovedAt()), formatUser(pm.GetApprovedBy()))
	}
	fmt.Fprintf(w, "Updated:    %s\n\n", output.FormatTimestamp(pm.GetUpdatedAt()))
	fmt.Fprintln(w, pm.GetContent())
}

func renderPostmortem(id string, pm *incidentv1.Postmortem) error {
	if output.IsJSONOutput() {
		if pm == nil {
			return output.PrintJSON(nil)
		}
		return output.PrintJSON(toPostmortemJSON(pm))
	}
	if pm == nil {
		fmt.Printf("No postmortem yet for incident %s\n", id)
		fmt.Printf("Write one: openstatus inc postmortem edit %s --file postmortem.md\n", id)
		return nil
	}
	if !output.IsTerminal() {
		fmt.Println(pm.GetContent())
		return nil
	}
	output.WithPager(func(w io.Writer) {
		writePostmortem(w, pm)
	})
	return nil
}

func ShowPostmortemWithHTTPClient(ctx context.Context, httpClient *http.Client, apiKey string, id string) error {
	pm, err := GetPostmortem(ctx, NewIncidentClientWithHTTPClient(httpClient, apiKey), id)
	if err != nil {
		return err
	}
	return renderPostmortem(id, pm)
}

func resolvePostmortemContent(file, content string, fileSet, contentSet bool, stdin io.Reader) (string, error) {
	if fileSet == contentSet {
		return "", errors.New("pass exactly one of --file or --content")
	}

	text := content
	if fileSet {
		switch file {
		case "":
			return "", errors.New("--file cannot be empty")
		case "-":
			var err error
			text, err = readTextArg("-", stdin)
			if err != nil {
				return "", err
			}
		default:
			data, err := os.ReadFile(file)
			if err != nil {
				return "", fmt.Errorf("read postmortem file %q: %w", file, err)
			}
			text = trimTrailingNewline(string(data))
		}
	}

	if strings.TrimSpace(text) == "" {
		return "", errors.New("postmortem content cannot be empty")
	}
	if utf8.RuneCountInString(text) > maxPostmortemLen {
		return "", fmt.Errorf("postmortem is too long: maximum %d characters", maxPostmortemLen)
	}
	return text, nil
}

func UpdatePostmortem(ctx context.Context, client incidentv1connect.IncidentServiceClient, id, content string) (*incidentv1.Postmortem, error) {
	resp, err := client.UpdatePostmortem(ctx, &incidentv1.UpdatePostmortemRequest{IncidentId: id, Content: content})
	if err != nil {
		return nil, formatIncidentError(err, id, opGeneric)
	}
	return resp.GetPostmortem(), nil
}

func renderPostmortemSaved(id string, pm *incidentv1.Postmortem) error {
	if output.IsJSONOutput() {
		return output.PrintJSON(toPostmortemJSON(pm))
	}
	if output.IsQuiet() {
		fmt.Println(id)
		return nil
	}
	status := postmortemStatusToString(pm.GetStatus())
	fmt.Printf("Postmortem saved for incident %s (%s)\n", id, status)
	if status == "draft" {
		fmt.Printf("Approve: openstatus inc postmortem approve %s [--close]\n", id)
	}
	return nil
}

func UpdatePostmortemWithHTTPClient(ctx context.Context, httpClient *http.Client, apiKey string, id, content string) error {
	pm, err := UpdatePostmortem(ctx, NewIncidentClientWithHTTPClient(httpClient, apiKey), id, content)
	if err != nil {
		return err
	}
	return renderPostmortemSaved(id, pm)
}

func ApprovePostmortem(ctx context.Context, client incidentv1connect.IncidentServiceClient, id string, closeIncident bool) (*incidentv1.ApprovePostmortemResponse, error) {
	req := &incidentv1.ApprovePostmortemRequest{IncidentId: id}
	if closeIncident {
		req.SetClose(true)
	}
	resp, err := client.ApprovePostmortem(ctx, req)
	if err != nil {
		return nil, formatIncidentError(err, id, opApprove)
	}
	return resp, nil
}

func renderApproved(id string, resp *incidentv1.ApprovePostmortemResponse) error {
	if output.IsJSONOutput() {
		return output.PrintJSON(approveOutput{
			Postmortem: toPostmortemJSON(resp.GetPostmortem()),
			Incident:   toDetailJSON(resp.GetIncident()),
		})
	}
	if output.IsQuiet() {
		fmt.Println(id)
		return nil
	}
	fmt.Printf("Postmortem approved for incident %s\n", id)
	if resp.GetIncident().GetClosedAt() != "" {
		fmt.Printf("Incident %s closed\n", id)
		return nil
	}
	fmt.Printf("Close the incident: openstatus inc close %s\n", id)
	return nil
}

func ApprovePostmortemWithHTTPClient(ctx context.Context, httpClient *http.Client, apiKey string, id string, closeIncident bool) error {
	resp, err := ApprovePostmortem(ctx, NewIncidentClientWithHTTPClient(httpClient, apiKey), id, closeIncident)
	if err != nil {
		return err
	}
	return renderApproved(id, resp)
}

func postmortemCmd() *cli.Command {
	return &cli.Command{
		Name:    "postmortem",
		Aliases: []string{"pm"},
		Usage:   "Read, write and approve an incident's postmortem",
		Commands: []*cli.Command{
			postmortemShowCmd(),
			postmortemEditCmd(),
			postmortemApproveCmd(),
		},
	}
}

func postmortemShowCmd() *cli.Command {
	return &cli.Command{
		Name:  "show",
		Usage: "Show the postmortem of an incident",
		UsageText: `openstatus incident postmortem show <IncidentID>
  openstatus inc pm show 42 > postmortem.md`,
		Description: `In a terminal, prints the review status and the markdown body through your
pager. When piped, prints only the markdown body, so it can be edited and sent
back with 'openstatus inc pm edit <ID> --file postmortem.md'.`,
		Flags: []cli.Flag{accessTokenFlag()},
		Action: withClient(func(ctx context.Context, cmd *cli.Command, client incidentv1connect.IncidentServiceClient) error {
			id, err := requireID(cmd, pmShowUsage)
			if err != nil {
				return err
			}
			s := output.StartSpinner("Fetching postmortem...")
			pm, err := GetPostmortem(ctx, client, id)
			output.StopSpinner(s)
			if err != nil {
				return err
			}
			return renderPostmortem(id, pm)
		}),
	}
}

func postmortemEditCmd() *cli.Command {
	return &cli.Command{
		Name:  "edit",
		Usage: "Write or replace the postmortem of a resolved incident",
		UsageText: `openstatus incident postmortem edit <IncidentID> --file postmortem.md
  cat postmortem.md | openstatus inc pm edit 42 --file -
  openstatus inc pm edit 42 --content "## Summary ..."`,
		Description: "Replaces the whole markdown body. An approved postmortem stays approved.",
		Flags: []cli.Flag{
			accessTokenFlag(),
			&cli.StringFlag{
				Name:  "file",
				Usage: "Markdown file to upload; use - to read from stdin",
			},
			&cli.StringFlag{
				Name:  "content",
				Usage: "Markdown body passed inline",
			},
		},
		Action: withClient(func(ctx context.Context, cmd *cli.Command, client incidentv1connect.IncidentServiceClient) error {
			id, err := requireID(cmd, pmEditUsage)
			if err != nil {
				return err
			}
			content, err := resolvePostmortemContent(cmd.String("file"), cmd.String("content"), cmd.IsSet("file"), cmd.IsSet("content"), os.Stdin)
			if err != nil {
				return err
			}

			s := output.StartSpinner("Saving postmortem...")
			pm, err := UpdatePostmortem(ctx, client, id, content)
			output.StopSpinner(s)
			if err != nil {
				return err
			}
			return renderPostmortemSaved(id, pm)
		}),
	}
}

func postmortemApproveCmd() *cli.Command {
	return &cli.Command{
		Name:  "approve",
		Usage: "Approve the postmortem, optionally closing the incident",
		UsageText: `openstatus incident postmortem approve <IncidentID>
  openstatus inc pm approve 42 --close -y`,
		Flags: []cli.Flag{
			accessTokenFlag(),
			&cli.BoolFlag{
				Name:  "close",
				Usage: "Also close the incident",
			},
			autoAcceptFlag(),
		},
		Action: withClient(func(ctx context.Context, cmd *cli.Command, client incidentv1connect.IncidentServiceClient) error {
			id, err := requireID(cmd, pmApproveUsage)
			if err != nil {
				return err
			}
			closeIncident := cmd.Bool("close")
			if closeIncident {
				confirmed, err := confirm(cmd, fmt.Sprintf("You are about to approve the postmortem and close incident %s. It becomes read-only except for its postmortem. Continue", id))
				if err != nil {
					return err
				}
				if !confirmed {
					return nil
				}
			}

			s := output.StartSpinner("Approving postmortem...")
			resp, err := ApprovePostmortem(ctx, client, id, closeIncident)
			output.StopSpinner(s)
			if err != nil {
				return err
			}
			return renderApproved(id, resp)
		}),
	}
}
