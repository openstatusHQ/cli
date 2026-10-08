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
	var gotCommand string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotCommand = r.Header.Get(api.HeaderCLICommand)
	}))
	defer srv.Close()
	t.Setenv("DO_NOT_TRACK", "")
	t.Setenv("OPENSTATUS_NO_TELEMETRY", "")
	t.Cleanup(func() { api.SetCommand("") })

	app := &cli.Command{
		Name:    "openstatus",
		Version: "v9.9.9",
		Commands: []*cli.Command{{
			Name:    "private-locations",
			Aliases: []string{"pl"},
			Commands: []*cli.Command{{
				Name: "list",
				Action: func(ctx context.Context, cmd *cli.Command) error {
					resp, err := (&http.Client{Transport: api.NewTransport(nil)}).Get(srv.URL)
					if err != nil {
						return err
					}
					return resp.Body.Close()
				},
			}},
		}},
	}
	trackInvocation(app.Commands)

	if err := app.Run(context.Background(), []string{"openstatus", "pl", "list"}); err != nil {
		t.Fatal(err)
	}
	if gotCommand != "private-locations list" {
		t.Errorf("command header = %q, want %q", gotCommand, "private-locations list")
	}
}
