package monitors

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/openstatusHQ/cli/internal/config"
	"github.com/urfave/cli/v3"
)


func GetMonitorsApplyCmd() *cli.Command {
	monitorsListCmd := cli.Command{
		Name:        "apply",
		Usage:       "Create or update monitors",
		Description: "Creates or updates monitors according to the OpenStatus configuration file",
		UsageText:   "openstatus monitors apply [options]",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        "config",
				Usage:       "The configuration file containing monitor information",
				DefaultText: "openstatus.yaml",
				Value:       "openstatus.yaml",
			},
			&cli.StringFlag{
				Name:     "access-token",
				Usage:    "OpenStatus API Access Token",
				Aliases:  []string{"t"},
				Sources:  cli.EnvVars("OPENSTATUS_API_TOKEN"),
				Required: true,
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {

			path := cmd.String("config")

			if path != "" {
				if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
					return cli.Exit("Config does not exist", 1)
				}
			}

			// Read Config file
			//
			monitors, err := config.ReadOpenStatus(path)
			if err != nil {
				return cli.Exit("Unable to read config file", 1)
			}

			lock, err := config.ReadLockFile()
			if err != nil {
				return cli.Exit("Unable to read lock file", 1)
			}

			fmt.Println(monitors,lock)

			// Read Lock file
			// Compare Config file with Lock file
			// If changes detected, apply changes
			return nil
		},
	}
	return &monitorsListCmd
}
