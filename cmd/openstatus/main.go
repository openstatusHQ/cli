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
		Name:    "OpenStatus",
		Usage:   "This is OpenStatus Command Line Interface",
		Version: "v0.0.2",
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
