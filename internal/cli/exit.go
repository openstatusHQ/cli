package cli

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"sync/atomic"

	clilib "github.com/urfave/cli/v3"
)

var commandPath atomic.Value

// SetCommandPath records the running command (e.g. "monitors info") so usage
// errors can point at its --help.
func SetCommandPath(path string) { commandPath.Store(path) }

func usageHint() string {
	path, _ := commandPath.Load().(string)
	if path == "" {
		return "Run 'openstatus --help' for usage."
	}
	return fmt.Sprintf("Run 'openstatus %s --help' for usage.", path)
}

// classify turns any error into an *Error, keeping the exit code of a plain
// urfave/cli ExitCoder and recognising API and network failures.
func classify(err error) *Error {
	var cliErr *Error
	if errors.As(err, &cliErr) {
		return cliErr
	}
	if errors.As(FormatError(err, "", ""), &cliErr) {
		return cliErr
	}
	var exitErr clilib.ExitCoder
	if errors.As(err, &exitErr) {
		kind := KindError
		if exitErr.ExitCode() == ExitInterrupted {
			kind = KindCancelled
		}
		return &Error{Kind: kind, err: err, exit: exitErr.ExitCode()}
	}
	return &Error{Kind: KindError, err: err}
}

// HandleExitError prints err to stderr, as JSON under --json, and exits with
// its code. It is the root command's ExitErrHandler, so every command error
// ends up here.
func HandleExitError(_ context.Context, cmd *clilib.Command, err error) {
	if err == nil {
		return
	}
	e := classify(err)
	var w io.Writer = os.Stderr
	if cmd != nil && cmd.Root().ErrWriter != nil {
		w = cmd.Root().ErrWriter
	}
	writeError(w, e)
	clilib.OsExiter(e.ExitCode())
}

type jsonError struct {
	Error jsonErrorBody `json:"error"`
}

type jsonErrorBody struct {
	Code     Kind   `json:"code"`
	Message  string `json:"message"`
	ExitCode int    `json:"exit_code"`
}

func writeError(w io.Writer, e *Error) {
	if IsJSONOutput() {
		data, _ := json.MarshalIndent(jsonError{jsonErrorBody{e.Kind, e.Error(), e.ExitCode()}}, "", "  ")
		fmt.Fprintln(w, string(data))
		return
	}
	if msg := e.Error(); msg != "" {
		fmt.Fprintln(w, msg)
	}
	hint := e.Hint
	if hint == "" && e.Kind == KindUsage {
		hint = usageHint()
	}
	if hint != "" {
		fmt.Fprintln(w, hint)
	}
}
