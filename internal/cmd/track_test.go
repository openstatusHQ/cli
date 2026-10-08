package cmd

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/urfave/cli/v3"

	"github.com/openstatusHQ/cli/internal/api"
)

func Test_trackInvocation(t *testing.T) {
	var gotCommands []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotCommands = append(gotCommands, r.Header.Get(api.HeaderCLICommand))
	}))
	defer srv.Close()
	t.Setenv("DO_NOT_TRACK", "")
	t.Setenv("OPENSTATUS_NO_TELEMETRY", "")
	prevBase := api.BaseURL
	api.BaseURL = srv.URL
	t.Cleanup(func() {
		api.BaseURL = prevBase
		api.SetCommand("")
	})

	call := func() error {
		resp, err := (&http.Client{Transport: api.NewTransport(nil)}).Get(srv.URL)
		if err != nil {
			return err
		}
		return resp.Body.Close()
	}
	app := &cli.Command{
		Name: "openstatus",
		Commands: []*cli.Command{{
			Name:    "private-locations",
			Aliases: []string{"pl"},
			Commands: []*cli.Command{{
				Name: "list",
				// A request from the command's own Before hook must be tagged too.
				Before: func(ctx context.Context, cmd *cli.Command) (context.Context, error) {
					return ctx, call()
				},
				Action: func(ctx context.Context, cmd *cli.Command) error {
					return call()
				},
			}},
		}},
	}
	trackInvocation(app.Commands)

	if err := app.Run(context.Background(), []string{"openstatus", "pl", "list"}); err != nil {
		t.Fatal(err)
	}
	want := []string{"private-locations list", "private-locations list"}
	if len(gotCommands) != len(want) || gotCommands[0] != want[0] || gotCommands[1] != want[1] {
		t.Errorf("command headers = %q, want %q", gotCommands, want)
	}
}
