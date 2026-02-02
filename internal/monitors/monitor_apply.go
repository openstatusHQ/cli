package monitors

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"

	"github.com/google/go-cmp/cmp"
	confirmation "github.com/openstatusHQ/cli/internal/cli"
	"github.com/openstatusHQ/cli/internal/config"
	"github.com/urfave/cli/v3"
	"sigs.k8s.io/yaml"
)

func CompareLockWithConfig(apiKey string, applyChange bool, lock config.MonitorsLock, configData config.Monitors) (config.MonitorsLock, error) {

	var created, updated, deleted int
	// Create or update monitors
	for v, configValue := range configData {
		value, exist := lock[v]

		if !exist {

			if applyChange {

				result, err := CreateMonitor(http.DefaultClient, apiKey, configValue)
				if err != nil {
					return nil, err
				}
				lock[v] = config.Lock{
					ID:      result.ID,
					Monitor: configValue,
				}
			}

			created++

			continue
		}
		if !cmp.Equal(configValue, value.Monitor) {
			if applyChange {

				result, err := UpdateMonitor(http.DefaultClient, apiKey, value.ID, configValue)
				if err != nil {
					return nil, err
				}
				lock[v] = config.Lock{
					ID:      result.ID,
					Monitor: configValue,
				}
			}
			updated++
			continue
		}
	}

	// Delete monitors
	for v, value := range lock {
		if _, exist := configData[v]; !exist {
			if applyChange {

				err := DeleteMonitorWithHTTPClient(http.DefaultClient, apiKey, fmt.Sprintf("%d", value.ID))
				if err != nil {
					fmt.Println(err)
				}
				delete(lock, v)
			}
			deleted++
		}
	}

	if created == 0 && updated == 0 && deleted == 0 {
		fmt.Println("No change founded")
		return nil, nil
	}

	if applyChange {
		fmt.Println("Successfully apply")
		// if created > 0 {
		// fmt.Println("Monitor Created:", created)
		// }
		// if updated > 0 {
		// 	fmt.Println("Monitor Updated:", updated)
		// }
		// if deleted > 0 {
		// 	fmt.Println("Monitor Deleted:", deleted)
		// }

		return lock, nil
	}
	fmt.Println("This will apply the following change:")
	if created > 0 {
		fmt.Println("Monitor Create:", created)
	}
	if updated > 0 {
		fmt.Println("Monitor Update:", updated)
	}
	if deleted > 0 {
		fmt.Println("Monitor Delete:", deleted)
	}

	confirmed, err := confirmation.AskForConfirmation("Do you want to continue?")
	if err != nil {
		return nil, fmt.Errorf("failed to read user input: %w", err)
	}
	if !confirmed {
		return nil, nil
	}
	return lock, nil
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
			&cli.BoolFlag{
				Name:     "auto-accept",
				Usage:    "Automatically accept the prompt",
				Aliases:  []string{"y"},
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

			accept := cmd.Bool("auto-accept")
			if !accept {
				r, err := CompareLockWithConfig(cmd.String("access-token"), false, lock, monitors)
				if err != nil {
					return cli.Exit("Failed to apply change", 1)

				}
				if r == nil {
					return nil
				}
			}

			newLock, err := CompareLockWithConfig(cmd.String("access-token"), true, lock, monitors)
			if err != nil {
				return cli.Exit("Failed to apply change", 1)
			}
			if newLock == nil {
				fmt.Println("No change founded")
				return nil
			}
			y, err := yaml.Marshal(&newLock)
			if err != nil {
				return cli.Exit("Failed to apply change", 1)
			}
			// Write Lock file
			file, err := os.OpenFile("openstatus.lock", os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
			if err != nil {
				return cli.Exit("Failed to apply change", 1)

			}
			defer file.Close()

			_, err = file.Write(y)
			if err != nil {
				return cli.Exit("Failed to apply change", 1)
			}
			return nil
		},
	}
	return &monitorsListCmd
}
