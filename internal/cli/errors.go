package cli

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strings"

	"connectrpc.com/connect"
)

// Exit codes. Scripts depend on these, so keep the README in sync.
const (
	ExitOK          = 0
	ExitError       = 1 // generic failure, including failed checks in `run`
	ExitUsage       = 2 // bad flags, arguments or config
	ExitAuth        = 3 // missing, invalid or insufficient API token
	ExitNotFound    = 4
	ExitUnavailable = 5 // API unreachable, timed out or rate limited
	ExitInterrupted = 130
)

// Kind classifies an error. It is the "code" field of a JSON error.
type Kind string

const (
	KindError              Kind = "error"
	KindUsage              Kind = "usage"
	KindUnauthenticated    Kind = "unauthenticated"
	KindPermissionDenied   Kind = "permission_denied"
	KindNotFound           Kind = "not_found"
	KindInvalidArgument    Kind = "invalid_argument"
	KindFailedPrecondition Kind = "failed_precondition"
	KindRateLimited        Kind = "rate_limited"
	KindUnavailable        Kind = "unavailable"
	KindTimeout            Kind = "timeout"
	KindCancelled          Kind = "cancelled"
	KindChecksFailed       Kind = "checks_failed"
)

var kindExitCodes = map[Kind]int{
	KindError:              ExitError,
	KindUsage:              ExitUsage,
	KindUnauthenticated:    ExitAuth,
	KindPermissionDenied:   ExitAuth,
	KindNotFound:           ExitNotFound,
	KindInvalidArgument:    ExitUsage,
	KindFailedPrecondition: ExitError,
	KindRateLimited:        ExitUnavailable,
	KindUnavailable:        ExitUnavailable,
	KindTimeout:            ExitUnavailable,
	KindCancelled:          ExitInterrupted,
	KindChecksFailed:       ExitError,
}

// Error is a classified CLI error. It implements urfave/cli's ExitCoder, and
// HandleExitError turns it into a message (or JSON) on stderr and an exit code.
type Error struct {
	Kind Kind
	// Hint is printed after the message in human output, e.g. where to find usage.
	Hint string
	err  error
	exit int
}

func (e *Error) Error() string { return e.err.Error() }
func (e *Error) Unwrap() error { return e.err }

func (e *Error) ExitCode() int {
	if e.exit != 0 {
		return e.exit
	}
	if code, ok := kindExitCodes[e.Kind]; ok {
		return code
	}
	return ExitError
}

// NewError returns an error of the given kind. The format follows fmt.Errorf,
// so %w wrapping works.
func NewError(kind Kind, format string, args ...any) error {
	return &Error{Kind: kind, err: fmt.Errorf(format, args...)}
}

// UsageErrorf reports invalid input: a missing argument, a bad flag value or
// an invalid config. It exits with ExitUsage.
func UsageErrorf(format string, args ...any) error {
	return NewError(KindUsage, format, args...)
}

// UsageErrorWithHint is a usage error whose hint replaces the default pointer
// to the running command's --help.
func UsageErrorWithHint(msg, hint string) error {
	return &Error{Kind: KindUsage, Hint: hint, err: errors.New(msg)}
}

func newError(kind Kind, msg string) error {
	return &Error{Kind: kind, err: errors.New(msg)}
}

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

const unreachableMsg = "could not reach the openstatus API. Check your internet connection and OPENSTATUS_API_URL"

func FormatError(err error, resource string, id string) error {
	if err == nil {
		return nil
	}

	var cliErr *Error
	if errors.As(err, &cliErr) {
		return err
	}

	if errors.Is(err, context.Canceled) {
		return newError(KindCancelled, "operation cancelled")
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return newError(KindTimeout, "request timed out. The openstatus API did not respond in time; try again")
	}

	var connectErr *connect.Error
	if errors.As(err, &connectErr) {
		switch connectErr.Code() {
		case connect.CodeUnauthenticated:
			return newError(KindUnauthenticated, "authentication failed. Check your API token via OPENSTATUS_API_TOKEN env var or --access-token flag. Verify with 'openstatus whoami'")
		case connect.CodePermissionDenied:
			return newError(KindPermissionDenied, "permission denied. Check that your API token has access to this workspace")
		case connect.CodeNotFound:
			if resource == "" {
				return newError(KindNotFound, "not found")
			}
			if id == "" {
				return NewError(KindNotFound, "%s not found", resource)
			}
			if cmd, ok := listCommands[resource]; ok {
				return NewError(KindNotFound, "%s %s not found. Run 'openstatus %s list' to see available %ss", resource, id, cmd, resource)
			}
			return NewError(KindNotFound, "%s %s not found", resource, id)
		case connect.CodeResourceExhausted:
			return newError(KindRateLimited, "rate limited. Wait a moment and try again")
		case connect.CodeInvalidArgument:
			return NewError(KindInvalidArgument, "invalid request: %s", connectErr.Message())
		case connect.CodeFailedPrecondition:
			if connectErr.Message() == "" {
				return newError(KindFailedPrecondition, "request conflicts with the current state of the resource")
			}
			return newError(KindFailedPrecondition, connectErr.Message())
		case connect.CodeUnavailable:
			return newError(KindUnavailable, unreachableMsg)
		}
	}

	var dnsErr *net.DNSError
	if errors.As(err, &dnsErr) {
		return newError(KindUnavailable, unreachableMsg)
	}

	var netErr *net.OpError
	if errors.As(err, &netErr) {
		return newError(KindUnavailable, unreachableMsg)
	}

	if strings.Contains(err.Error(), "connection refused") || strings.Contains(err.Error(), "no such host") {
		return newError(KindUnavailable, unreachableMsg)
	}

	return err
}
