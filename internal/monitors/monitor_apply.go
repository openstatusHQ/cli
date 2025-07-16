package monitors

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"

	"github.com/google/go-cmp/cmp"
	"github.com/openstatusHQ/cli/internal/config"
	"github.com/urfave/cli/v3"
	"sigs.k8s.io/yaml"
)

func CompareLockWithConfig(apiKey string, lock config.MonitorsLock, configData config.Monitors) config.MonitorsLock {

	// Create or update monitors
	for v, configValue := range configData {

		value, exist := lock[v]
		if !exist {
			result, err := CreateMonitor(http.DefaultClient, apiKey, configValue)
			if err != nil {
				fmt.Println(err)
			}
			lock[v] = config.Lock{
				ID: result.ID,
				Monitor: configValue,
			}
			continue
		}
		if !cmp.Equal(configValue, value.Monitor) {
			result, err := UpdateMonitor(http.DefaultClient,apiKey, value.ID, configValue)
			if err != nil {
				fmt.Println(err)
			}
			lock[v] = config.Lock{
				ID: result.ID,
				Monitor: configValue,
			}
		}
	}

	// Delete monitors
	for v, value := range lock {
		if _, exist := configData[v]; !exist {

			err := DeleteMonitor(http.DefaultClient,apiKey, fmt.Sprintf("%d", value.ID))
			if err != nil {
				fmt.Println(err)
			}
			delete(lock, v)
		}
	}
	return lock
}

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
				Aliases:     []string{"c"},
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

			lock, err := config.ReadLockFile("openstatus.lock")
			if err != nil {
				return cli.Exit("Unable to read lock file", 1)
			}

			newLock := CompareLockWithConfig(cmd.String("access-token"), lock, monitors)
			y, err := yaml.Marshal(&newLock)
			if err != nil {
				return err
			}
			// Write Lock file
			file, err := os.OpenFile("openstatus.lock", os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
			if err != nil {
				return err
			}
			defer file.Close()

			_, err = file.Write(y)
			if err != nil {
				return err
			}
			return nil
		},
	}
	return &monitorsListCmd
}
