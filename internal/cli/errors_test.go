package cli_test

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"connectrpc.com/connect"

	output "github.com/openstatusHQ/cli/internal/cli"
)

func Test_FormatError(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		err      error
		resource string
		id       string
		want     string
	}{
		{
			name: "failed precondition shows server message",
			err:  connect.NewError(connect.CodeFailedPrecondition, errors.New("incident is closed")),
			want: "incident is closed",
		},
		{
			name: "failed precondition without message",
			err:  connect.NewError(connect.CodeFailedPrecondition, errors.New("")),
			want: "request conflicts with the current state of the resource",
		},
		{
			name: "cancelled request",
			err:  connect.NewError(connect.CodeCanceled, context.Canceled),
			want: "operation cancelled",
		},
		{
			name: "timed out request",
			err:  fmt.Errorf("call: %w", context.DeadlineExceeded),
			want: "request timed out",
		},
		{
			name: "unauthenticated",
			err:  connect.NewError(connect.CodeUnauthenticated, errors.New("nope")),
			want: "authentication failed",
		},
		{
			name: "permission denied",
			err:  connect.NewError(connect.CodePermissionDenied, errors.New("nope")),
			want: "permission denied",
		},
		{
			name:     "not found with id",
			err:      connect.NewError(connect.CodeNotFound, errors.New("missing")),
			resource: "incident",
			id:       "42",
			want:     "incident 42 not found. Run 'openstatus incident list' to see available incidents",
		},
		{
			name:     "not found hint uses the real list command",
			err:      connect.NewError(connect.CodeNotFound, errors.New("nope")),
			resource: "monitor",
			id:       "7",
			want:     "monitor 7 not found. Run 'openstatus monitors list' to see available monitors",
		},
		{
			name:     "not found without a list command has no hint",
			err:      connect.NewError(connect.CodeNotFound, errors.New("nope")),
			resource: "response log",
			id:       "abc",
			want:     "response log abc not found",
		},
		{
			name:     "not found without id",
			err:      connect.NewError(connect.CodeNotFound, errors.New("missing")),
			resource: "incident",
			want:     "incident not found",
		},
		{
			name: "rate limited",
			err:  connect.NewError(connect.CodeResourceExhausted, errors.New("slow down")),
			want: "rate limited",
		},
		{
			name: "invalid argument",
			err:  connect.NewError(connect.CodeInvalidArgument, errors.New("title is required")),
			want: "invalid request: title is required",
		},
		{
			name: "plain error passes through",
			err:  errors.New("boom"),
			want: "boom",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := output.FormatError(tt.err, tt.resource, tt.id)
			if got == nil || !strings.Contains(got.Error(), tt.want) {
				t.Errorf("FormatError() = %v, want it to contain %q", got, tt.want)
			}
		})
	}

	t.Run("nil error", func(t *testing.T) {
		t.Parallel()
		if err := output.FormatError(nil, "incident", ""); err != nil {
			t.Errorf("expected nil, got %v", err)
		}
	})
}
