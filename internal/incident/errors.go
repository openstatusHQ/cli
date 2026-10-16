package incident

import (
	"errors"
	"fmt"
	"strings"

	"connectrpc.com/connect"

	output "github.com/openstatusHQ/cli/internal/cli"
)

type op int

const (
	opGeneric op = iota
	opSetStatus
	opDelete
	opClose
	opApprove
	opLink
	opDeclare
)

func formatIncidentError(err error, id string, o op) error {
	var connectErr *connect.Error
	if !errors.As(err, &connectErr) {
		return output.FormatError(err, "incident", id)
	}

	msg := strings.TrimSuffix(connectErr.Message(), ".")

	switch connectErr.Code() {
	case connect.CodeFailedPrecondition:
		if hint := preconditionHint(id, o); hint != "" {
			if msg == "" {
				return errors.New(hint)
			}
			return fmt.Errorf("%s. %s", msg, hint)
		}
	case connect.CodeNotFound:
		switch o {
		case opApprove:
			if strings.Contains(strings.ToLower(msg), "postmortem") {
				return fmt.Errorf("no postmortem written yet for incident %s. Write one first: openstatus inc postmortem edit %s --file postmortem.md", id, id)
			}
		case opLink, opDeclare:
			if msg != "" {
				return errors.New(msg)
			}
		}
	case connect.CodePermissionDenied:
		if hint := permissionHint(o); hint != "" {
			if msg == "" {
				return fmt.Errorf("permission denied. %s", hint)
			}
			return fmt.Errorf("permission denied: %s. %s", msg, hint)
		}
	}

	return output.FormatError(err, "incident", id)
}

func preconditionHint(id string, o op) string {
	switch o {
	case opSetStatus:
		return fmt.Sprintf("Run 'openstatus inc info %s' to see allowed transitions.", id)
	case opDelete:
		return fmt.Sprintf("Only incidents that are still open and were never mitigated can be deleted; use 'openstatus inc cancel %s' instead.", id)
	case opClose:
		return "Close requires a resolved incident and an approved postmortem (or --skip-postmortem)."
	default:
		return ""
	}
}

func permissionHint(o op) string {
	switch o {
	case opClose, opApprove:
		return "This action requires a workspace owner or admin, or the incident commander. API keys without a creator cannot perform it."
	case opDelete:
		return "This action requires a workspace owner or admin. API keys without a creator cannot perform it."
	default:
		return ""
	}
}
