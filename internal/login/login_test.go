package login

import (
	"context"
	"errors"
	"testing"

	"github.com/urfave/cli/v3"

	output "github.com/openstatusHQ/cli/internal/cli"
)

func TestUsesTokenLogin(t *testing.T) {
	tests := []struct {
		name                              string
		withToken, noBrowser, interactive bool
		want                              bool
	}{
		{name: "terminal defaults to the browser", interactive: true, want: false},
		{name: "piped stdin is a token", want: true},
		{name: "--with-token in a terminal prompts", withToken: true, interactive: true, want: true},
		{name: "--no-browser in a terminal", noBrowser: true, interactive: true, want: false},
		{name: "--no-browser wins over a missing terminal", noBrowser: true, want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := usesTokenLogin(tt.withToken, tt.noBrowser, tt.interactive); got != tt.want {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}

func TestLogin_RejectsConflictingFlags(t *testing.T) {
	cmd := LoginCmd()
	// The default handler exits the process on a coded error.
	cmd.ExitErrHandler = func(context.Context, *cli.Command, error) {}
	err := cmd.Run(t.Context(), []string{"login", "--no-browser", "--with-token"})
	var cliErr *output.Error
	if !errors.As(err, &cliErr) || cliErr.Kind != output.KindUsage {
		t.Fatalf("err = %v", err)
	}
}
