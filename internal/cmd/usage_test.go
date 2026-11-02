package cmd

import (
	"bytes"
	"context"
	"testing"

	"github.com/urfave/cli/v3"
)

func Test_suggest(t *testing.T) {
	t.Parallel()

	commands := map[string]string{
		"monitors": "monitors", "m": "monitors",
		"status-report": "status-report", "sr": "status-report",
		"version": "version", "run": "run", "r": "run",
	}
	tests := []struct{ in, want string }{
		{"monitr", "monitors"},
		{"monitor", "monitors"},
		{"statusreport", "status-report"},
		{"versoin", "version"},
		{"stat", "status-report"},
		{"x", ""},
		{"deploy", ""},
	}
	for _, tt := range tests {
		if got := suggest(tt.in, commands); got != tt.want {
			t.Errorf("suggest(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func Test_usageErrors(t *testing.T) {
	var gotCode int
	prev := cli.OsExiter
	cli.OsExiter = func(c int) { gotCode = c }
	t.Cleanup(func() { cli.OsExiter = prev })

	run := func(args ...string) (string, int, error) {
		gotCode = 0
		app := NewApp()
		var stderr bytes.Buffer
		app.ErrWriter = &stderr
		app.Writer = &bytes.Buffer{}
		err := app.Run(context.Background(), append([]string{"openstatus"}, args...))
		return stderr.String(), gotCode, err
	}

	t.Run("unknown command", func(t *testing.T) {
		got, code, _ := run("monitr")
		want := "unknown command \"monitr\" for \"openstatus\". Did you mean \"monitors\"?\nRun 'openstatus --help' for a list of commands.\n"
		if got != want || code != 2 {
			t.Errorf("got (%q, %d), want (%q, 2)", got, code, want)
		}
	})

	t.Run("unknown subcommand", func(t *testing.T) {
		got, code, _ := run("monitors", "lst")
		want := "unknown command \"lst\" for \"openstatus monitors\". Did you mean \"list\"?\nRun 'openstatus monitors --help' for a list of commands.\n"
		if got != want || code != 2 {
			t.Errorf("got (%q, %d), want (%q, 2)", got, code, want)
		}
	})

	t.Run("unknown flag", func(t *testing.T) {
		got, code, _ := run("monitors", "list", "--al")
		want := "unknown flag --al. Did you mean --all?\nRun 'openstatus monitors list --help' for usage.\n"
		if got != want || code != 2 {
			t.Errorf("got (%q, %d), want (%q, 2)", got, code, want)
		}
	})
}
