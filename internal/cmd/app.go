package cmd

import (
	"github.com/openstatusHQ/cli/internal/monitors"
	"github.com/openstatusHQ/cli/internal/run"
	"github.com/openstatusHQ/cli/internal/whoami"
	"github.com/urfave/cli/v3"
)

func NewApp() *cli.Command {
	app := &cli.Command{
		Name:        "openstatus",
		Suggest:     true,
		Usage:       "This is OpenStatus Command Line Interface, the OpenStatus.dev CLI",
		Description: "OpenStatus is a command line interface for managing your monitors and triggering your synthetics tests. \n\nPlease report any issues at https://github.com/openstatusHQ/cli/issues/new",
		Version:     "v0.0.6",
		Commands: []*cli.Command{
			monitors.MonitorsCmd(),
			run.RunCmd(),
			whoami.WhoamiCmd(),
		},
	}
	return app
}
