package cmd

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	output "github.com/openstatusHQ/cli/internal/cli"
	"github.com/openstatusHQ/cli/internal/login"
	"github.com/openstatusHQ/cli/internal/maintenance"
	"github.com/openstatusHQ/cli/internal/monitors"
	"github.com/openstatusHQ/cli/internal/run"
	"github.com/openstatusHQ/cli/internal/statuspage"
	"github.com/openstatusHQ/cli/internal/statusreport"
	"github.com/openstatusHQ/cli/internal/terraform"
	"github.com/openstatusHQ/cli/internal/whoami"
	"github.com/urfave/cli/v3"
)

func NewApp() *cli.Command {
	app := &cli.Command{
		Name:                  "openstatus",
		Suggest:               true,
		EnableShellCompletion: true,
		Usage: "Manage status pages, monitors, and incidents from the terminal",
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

https://docs.openstatus.dev  |  https://github.com/openstatusHQ/cli/issues/new`,
		Version:     "v1.0.3",
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
		Before: func(ctx context.Context, cmd *cli.Command) error {
			output.SetJSONOutput(cmd.Bool("json"))
			output.SetQuietMode(cmd.Bool("quiet"))
			output.SetDebugMode(cmd.Bool("debug"))
			output.InitColorSettings(cmd.Bool("no-color"))
			return nil
		},
		Commands: []*cli.Command{
			monitors.MonitorsCmd(),
			statusreport.StatusReportCmd(),
			maintenance.MaintenanceCmd(),
			statuspage.StatusPageCmd(),
			run.RunCmd(),
			whoami.WhoamiCmd(),
			login.LoginCmd(),
			login.LogoutCmd(),
			terraform.TerraformCmd(),
		},
	}
	return app
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
