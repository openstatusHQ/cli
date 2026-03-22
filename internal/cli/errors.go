package cli

import (
	"errors"
	"fmt"
	"net"
	"strings"

	"connectrpc.com/connect"
)

func FormatError(err error, resource string, id string) error {
	if err == nil {
		return nil
	}

	var connectErr *connect.Error
	if errors.As(err, &connectErr) {
		switch connectErr.Code() {
		case connect.CodeUnauthenticated:
			return fmt.Errorf("authentication failed. Check your API token via OPENSTATUS_API_TOKEN env var or --access-token flag. Verify with 'openstatus whoami'")
		case connect.CodePermissionDenied:
			return fmt.Errorf("permission denied. Check that your API token has access to this workspace")
		case connect.CodeNotFound:
			if id != "" {
				return fmt.Errorf("%s %s not found. Run 'openstatus %s list' to see available %ss", resource, id, resource, resource)
			}
			return fmt.Errorf("%s not found", resource)
		case connect.CodeResourceExhausted:
			return fmt.Errorf("rate limited. Wait a moment and try again")
		case connect.CodeInvalidArgument:
			return fmt.Errorf("invalid request: %s", connectErr.Message())
		}
	}

	var dnsErr *net.DNSError
	if errors.As(err, &dnsErr) {
		return fmt.Errorf("could not reach api.openstatus.dev. Check your internet connection")
	}

	var netErr *net.OpError
	if errors.As(err, &netErr) {
		return fmt.Errorf("could not reach api.openstatus.dev. Check your internet connection")
	}

	if strings.Contains(err.Error(), "connection refused") || strings.Contains(err.Error(), "no such host") {
		return fmt.Errorf("could not reach api.openstatus.dev. Check your internet connection")
	}

	return err
}
