package main

import (
	"context"
	"log"
	"os"

	"github.com/openstatusHQ/cli/internal/monitors"
	"github.com/openstatusHQ/cli/internal/run"
	"github.com/openstatusHQ/cli/internal/whoami"
	"github.com/urfave/cli/v3"
)

func main() {
	app := &cli.Command{
		Name:    "openstatus",
		Usage:   "This is OpenStatus Command Line Interface",
		Description: "OpenStatus is a command line interface for managing your monitors and triggering your synthetics tests. \n\nPlease report any issues at https://github.com/openstatusHQ/cli/issues/new",
		Version: "v0.0.4",
		Commands: []*cli.Command{
			monitors.MonitorsCmd(),
			run.RunCmd(),
			whoami.WhoamiCmd(),
		},
	}

	if err := app.Run(context.Background(), os.Args); err != nil {
		log.Fatal(err)
	}
}
