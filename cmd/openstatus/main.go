package main

import (
	"context"
	"fmt"
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
		Suggest: true,
		Usage:   "This is OpenStatus Command Line Interface",
  		UsageText: "openstatus [command] [flags]",
		Description: "OpenStatus is a command line interface for managing your monitors and triggering your synthetics tests.",
		Version: "v0.0.4",
		Commands: []*cli.Command{
			monitors.MonitorsCmd(),
			run.RunCmd(),
			whoami.WhoamiCmd(),
		},
		EnableShellCompletion: true,

		Action: func(ctx context.Context, cmd *cli.Command) error {
			cli.ShowAppHelp(cmd)
			fmt.Println("\n\nPlease report any issues at https://github.com/openstatusHQ/cli/issues/new")
			return nil
		},
	}

	if err := app.Run(context.Background(), os.Args); err != nil {
		log.Fatal(err)
	}
}
