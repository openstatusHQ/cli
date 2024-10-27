package main

import (
	"fmt"
	"log"
	"os"

	"github.com/openstatusHQ/cli/internal/monitors"
	"github.com/openstatusHQ/cli/internal/whoami"
	"github.com/urfave/cli/v2"
)

func main() {
	app := &cli.App{
		Name:    "OpenStatus",
		Usage:   "This is OpenStatus Command Line Interface",
		Version: "v0.0.1",
		Commands: []*cli.Command{
			{
				Name:    "run",
				Aliases: []string{"r"},
				Usage:   "Run your synthetics tests defined in your configuration file",
				Action: func(cCtx *cli.Context) error {
					fmt.Println("Test ran 🔥")
					return nil
				},
			},
			monitors.MonitorsCmd(),
			whoami.WhoamiCmd(),
		},
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        "config",
				Usage:       "The configuration file",
				DefaultText: "config.openstatus.yaml",
				Value:       "config.openstatus.yaml",
			},
			&cli.StringFlag{
				Name:     "access-token",
				Usage:    "OpenStatus API Access Token",
				Aliases:  []string{"t"},
				EnvVars:  []string{"OPENSTATUS_API_TOKEN"},
				Required: true,
			},
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}
