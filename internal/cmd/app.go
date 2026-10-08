package cmd

import (
	"context"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/urfave/cli/v3"

	"github.com/openstatusHQ/cli/internal/api"
	"github.com/openstatusHQ/cli/internal/check"
	output "github.com/openstatusHQ/cli/internal/cli"
	"github.com/openstatusHQ/cli/internal/login"
	"github.com/openstatusHQ/cli/internal/maintenance"
	"github.com/openstatusHQ/cli/internal/monitors"
	"github.com/openstatusHQ/cli/internal/notification"
	"github.com/openstatusHQ/cli/internal/privatelocation"
	"github.com/openstatusHQ/cli/internal/run"
	"github.com/openstatusHQ/cli/internal/statuspage"
	"github.com/openstatusHQ/cli/internal/statusreport"
	"github.com/openstatusHQ/cli/internal/terraform"
	"github.com/openstatusHQ/cli/internal/version"
	"github.com/openstatusHQ/cli/internal/whoami"
)

func NewApp() *cli.Command {
	app := &cli.Command{
		Name:                  "openstatus",
		Suggest:               true,
		EnableShellCompletion: true,
		Usage:                 "Manage status pages, monitors, and incidents from the terminal",
		Description: `OpenStatus CLI lets you manage your status pages and uptime monitors
from the command line. Report and track incidents, define monitors as code,
and run on-demand checks.

Get started:
  openstatus login                Save your API token
  openstatus status-report create Report an incident
  openstatus status-report list   View active incidents
  openstatus maintenance create   Schedule a maintenance window
  openstatus maintenance list     View maintenance windows
  openstatus monitors apply       Sync monitors from config
  openstatus monitors list        List your monitors
  openstatus run                  Run synthetic tests
  openstatus pl list              List your private locations

https://docs.openstatus.dev  |  https://github.com/openstatusHQ/cli/issues/new`,
		Version: version.Version,
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:  "json",
				Usage: "Output results as JSON",
			},
			&cli.BoolFlag{
				Name:  "no-color",
				Usage: "Disable colored output",
			},
			&cli.BoolFlag{
				Name:    "quiet",
				Usage:   "Suppress non-error output",
				Aliases: []string{"q"},
			},
			&cli.BoolFlag{
				Name:  "debug",
				Usage: "Enable debug output",
			},
		},
		Before: func(ctx context.Context, cmd *cli.Command) (context.Context, error) {
			output.SetJSONOutput(cmd.Bool("json"))
			output.SetQuietMode(cmd.Bool("quiet"))
			output.SetDebugMode(cmd.Bool("debug"))
			output.InitColorSettings(cmd.Bool("no-color"))
			return ctx, nil
		},
		Commands: []*cli.Command{
			check.CheckCmd(),
			monitors.MonitorsCmd(),
			statusreport.StatusReportCmd(),
			maintenance.MaintenanceCmd(),
			statuspage.StatusPageCmd(),
			notification.NotificationCmd(),
			privatelocation.PrivateLocationsCmd(),
			run.RunCmd(),
			whoami.WhoamiCmd(),
			login.LoginCmd(),
			login.LogoutCmd(),
			terraform.TerraformCmd(),
		},
	}
	trackInvocation(app.Commands)
	return app
}

// trackInvocation records the canonical command path (aliases resolve to the
// full name) for API requests. It hooks each runnable command's Before, which
// urfave/cli runs after its parents' Before hooks and ahead of flag actions and
// the command's own Action, so requests made from any of those carry it.
func trackInvocation(cmds []*cli.Command) {
	for _, c := range cmds {
		trackInvocation(c.Commands)
		if c.Action == nil {
			continue
		}
		before := c.Before
		c.Before = func(ctx context.Context, cmd *cli.Command) (context.Context, error) {
			api.SetCommand(strings.TrimPrefix(cmd.FullName(), cmd.Root().Name+" "))
			if before == nil {
				return ctx, nil
			}
			return before(ctx, cmd)
		}
	}
}

func RunApp(app *cli.Command) error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go func() {
		<-ctx.Done()
		stop()
		// Second signal: force exit
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
		<-sigCh
		os.Exit(130)
	}()

	return app.Run(ctx, os.Args)
}
