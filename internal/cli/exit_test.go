package cli_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"

	"connectrpc.com/connect"
	clilib "github.com/urfave/cli/v3"

	output "github.com/openstatusHQ/cli/internal/cli"
)

func Test_ExitCodes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		err  error
		want int
	}{
		{"usage", output.UsageErrorf("monitor ID is required"), output.ExitUsage},
		{"wrapped usage", fmt.Errorf("parse: %w", output.UsageErrorf("bad")), output.ExitUsage},
		{"unauthenticated", output.FormatError(connect.NewError(connect.CodeUnauthenticated, errors.New("x")), "", ""), output.ExitAuth},
		{"permission denied", output.FormatError(connect.NewError(connect.CodePermissionDenied, errors.New("x")), "", ""), output.ExitAuth},
		{"not found", output.FormatError(connect.NewError(connect.CodeNotFound, errors.New("x")), "monitor", "1"), output.ExitNotFound},
		{"invalid argument", output.FormatError(connect.NewError(connect.CodeInvalidArgument, errors.New("x")), "", ""), output.ExitUsage},
		{"rate limited", output.FormatError(connect.NewError(connect.CodeResourceExhausted, errors.New("x")), "", ""), output.ExitUnavailable},
		{"unavailable", output.FormatError(connect.NewError(connect.CodeUnavailable, errors.New("x")), "", ""), output.ExitUnavailable},
		{"timeout", output.FormatError(context.DeadlineExceeded, "", ""), output.ExitUnavailable},
		{"cancelled", output.FormatError(context.Canceled, "", ""), output.ExitInterrupted},
		{"checks failed", output.NewError(output.KindChecksFailed, "Some tests failed"), output.ExitError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var e *output.Error
			if !errors.As(tt.err, &e) {
				t.Fatalf("%v is not an *output.Error", tt.err)
			}
			if got := e.ExitCode(); got != tt.want {
				t.Errorf("ExitCode() = %d, want %d", got, tt.want)
			}
		})
	}
}

// handle runs HandleExitError with a captured writer and exit code.
func handle(t *testing.T, err error) (string, int) {
	t.Helper()
	var buf bytes.Buffer
	code := -1
	prev := clilib.OsExiter
	clilib.OsExiter = func(c int) { code = c }
	t.Cleanup(func() { clilib.OsExiter = prev })
	output.HandleExitError(context.Background(), &clilib.Command{ErrWriter: &buf}, err)
	return buf.String(), code
}

// Not parallel: it flips the global --json and command-path state.
func Test_HandleExitError(t *testing.T) {
	t.Cleanup(func() {
		output.SetJSONOutput(false)
		output.SetCommandPath("")
	})

	t.Run("human output with usage hint", func(t *testing.T) {
		output.SetCommandPath("monitors info")
		got, code := handle(t, output.UsageErrorf("monitor ID is required"))
		want := "monitor ID is required\nRun 'openstatus monitors info --help' for usage.\n"
		if got != want || code != output.ExitUsage {
			t.Errorf("got (%q, %d), want (%q, %d)", got, code, want, output.ExitUsage)
		}
	})

	t.Run("plain errors exit 1", func(t *testing.T) {
		got, code := handle(t, errors.New("boom"))
		if got != "boom\n" || code != output.ExitError {
			t.Errorf("got (%q, %d)", got, code)
		}
	})

	t.Run("urfave exit codes are kept", func(t *testing.T) {
		_, code := handle(t, clilib.Exit("nope", 7))
		if code != 7 {
			t.Errorf("code = %d, want 7", code)
		}
	})

	t.Run("raw API errors are classified", func(t *testing.T) {
		got, code := handle(t, connect.NewError(connect.CodeUnauthenticated, errors.New("x")))
		if !strings.HasPrefix(got, "authentication failed") || code != output.ExitAuth {
			t.Errorf("got (%q, %d)", got, code)
		}
	})

	t.Run("json output", func(t *testing.T) {
		output.SetJSONOutput(true)
		defer output.SetJSONOutput(false)
		got, code := handle(t, output.FormatError(connect.NewError(connect.CodeNotFound, errors.New("x")), "monitor", "7"))
		var body struct {
			Error struct {
				Code     string `json:"code"`
				Message  string `json:"message"`
				ExitCode int    `json:"exit_code"`
			} `json:"error"`
		}
		if err := json.Unmarshal([]byte(got), &body); err != nil {
			t.Fatalf("not JSON: %q: %v", got, err)
		}
		if body.Error.Code != "not_found" || body.Error.ExitCode != output.ExitNotFound || code != output.ExitNotFound {
			t.Errorf("got %+v, exit %d", body.Error, code)
		}
		if !strings.HasPrefix(body.Error.Message, "monitor 7 not found") {
			t.Errorf("message = %q", body.Error.Message)
		}
	})
}
