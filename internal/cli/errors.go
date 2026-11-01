package cli

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strings"

	"connectrpc.com/connect"
)

// listCommands maps a resource name to the command that lists it, for the
// "not found" hint. Resources without a list command get no hint.
var listCommands = map[string]string{
	"incident":         "incident",
	"maintenance":      "maintenance",
	"monitor":          "monitors",
	"notification":     "notification",
	"private location": "private-locations",
	"status-page":      "status-page",
	"status-report":    "status-report",
}

func FormatError(err error, resource string, id string) error {
	if err == nil {
		return nil
	}

	if errors.Is(err, context.Canceled) {
		return errors.New("operation cancelled")
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return errors.New("request timed out. The openstatus API did not respond in time; try again")
	}

	var connectErr *connect.Error
	if errors.As(err, &connectErr) {
		switch connectErr.Code() {
		case connect.CodeUnauthenticated:
			return fmt.Errorf("authentication failed. Check your API token via OPENSTATUS_API_TOKEN env var or --access-token flag. Verify with 'openstatus whoami'")
		case connect.CodePermissionDenied:
			return fmt.Errorf("permission denied. Check that your API token has access to this workspace")
		case connect.CodeNotFound:
			if id == "" {
				return fmt.Errorf("%s not found", resource)
			}
			if cmd, ok := listCommands[resource]; ok {
				return fmt.Errorf("%s %s not found. Run 'openstatus %s list' to see available %ss", resource, id, cmd, resource)
			}
			return fmt.Errorf("%s %s not found", resource, id)
		case connect.CodeResourceExhausted:
			return fmt.Errorf("rate limited. Wait a moment and try again")
		case connect.CodeInvalidArgument:
			return fmt.Errorf("invalid request: %s", connectErr.Message())
		case connect.CodeFailedPrecondition:
			if connectErr.Message() == "" {
				return errors.New("request conflicts with the current state of the resource")
			}
			return errors.New(connectErr.Message())
		}
	}

	var dnsErr *net.DNSError
	if errors.As(err, &dnsErr) {
		return fmt.Errorf("could not reach the openstatus API. Check your internet connection and OPENSTATUS_API_URL")
	}

	var netErr *net.OpError
	if errors.As(err, &netErr) {
		return fmt.Errorf("could not reach the openstatus API. Check your internet connection and OPENSTATUS_API_URL")
	}

	if strings.Contains(err.Error(), "connection refused") || strings.Contains(err.Error(), "no such host") {
		return fmt.Errorf("could not reach the openstatus API. Check your internet connection and OPENSTATUS_API_URL")
	}

	return err
}
