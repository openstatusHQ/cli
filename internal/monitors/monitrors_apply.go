package monitors

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"

	"github.com/openstatusHQ/cli/internal/config"
	"github.com/urfave/cli/v3"
)

func GetMonitorsApplyCmd() *cli.Command {
	monitorsApplyCmd := cli.Command{
		Name:        "apply",
		Usage:       "Apply changes to monitors",
		Description: "Apply changes to monitors. This command allows you to update the status of monitors.",
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
				Required: false,
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			path := cmd.String("config")

			if path != "" {
				if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
					return cli.Exit("Config does not exist", 1)
				}
			}

			monitors, err := config.ReadOpenStatus(path)
			if err != nil {
				return cli.Exit("Failed to list monitors", 1)
			}

			var lockFile config.LockFile

			for key, monitor := range monitors {

				n := config.Lock{
					Id:      1,
					Key:     key,
					Value:   monitor,
				}
				lockFile = append(lockFile, n)
			}

			e, err := json.Marshal(lockFile)
			if err != nil {
				fmt.Println(err)
				return err
			}
			fmt.Println(string(e))
			return nil
		},
	}
	return &monitorsApplyCmd
}
