package monitors

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"

	"github.com/google/go-cmp/cmp"
	"github.com/openstatusHQ/cli/internal/auth"
	output "github.com/openstatusHQ/cli/internal/cli"
	"github.com/openstatusHQ/cli/internal/config"
	"github.com/urfave/cli/v3"
	"sigs.k8s.io/yaml"
)

// countChanges computes the number of creates, updates, and deletes without making API calls.
func countChanges(lock config.MonitorsLock, configData config.Monitors) (created, updated, deleted int) {
	for v, configValue := range configData {
		value, exist := lock[v]
		if !exist {
			created++
		} else if !cmp.Equal(configValue, value.Monitor) {
			updated++
		}
	}
	for v := range lock {
		if _, exist := configData[v]; !exist {
			deleted++
		}
	}
	return
}

// ApplyChanges applies the changes between the lock file and the config data, making API calls.
func ApplyChanges(ctx context.Context, apiKey string, lock config.MonitorsLock, configData config.Monitors) (config.MonitorsLock, error) {
	var created, updated, deleted int

	for v, configValue := range configData {
		value, exist := lock[v]

		if !exist {
			result, err := CreateMonitor(ctx, http.DefaultClient, apiKey, configValue)
			if err != nil {
				return nil, err
			}
			lock[v] = config.Lock{
				ID:      result.ID,
				Monitor: configValue,
			}
			created++
			continue
		}
		if !cmp.Equal(configValue, value.Monitor) {
			result, err := UpdateMonitor(ctx, http.DefaultClient, apiKey, value.ID, configValue)
			if err != nil {
				return nil, err
			}
			lock[v] = config.Lock{
				ID:      result.ID,
				Monitor: configValue,
			}
			updated++
			continue
		}
	}

	for v, value := range lock {
		if _, exist := configData[v]; !exist {
			err := DeleteMonitorWithHTTPClient(ctx, http.DefaultClient, apiKey, fmt.Sprintf("%d", value.ID))
			if err != nil {
				return nil, fmt.Errorf("failed to delete monitor %d: %w", value.ID, err)
			}
			delete(lock, v)
			deleted++
		}
	}

	if created == 0 && updated == 0 && deleted == 0 {
		return nil, nil
	}

	fmt.Println("Changes applied successfully")
	if created > 0 {
		fmt.Println("  Created:", created)
	}
	if updated > 0 {
		fmt.Println("  Updated:", updated)
	}
	if deleted > 0 {
		fmt.Println("  Deleted:", deleted)
	}
	return lock, nil
}

func GetMonitorsApplyCmd() *cli.Command {
	monitorsApplyCmd := cli.Command{
		Name:  "apply",
		Usage: "Create or update monitors",
		Description: `Creates or updates monitors according to the OpenStatus configuration file.
Compares your openstatus.yaml with the current state and applies changes.`,
		UsageText: `openstatus monitors apply
  openstatus monitors apply --config custom.yaml -y
  openstatus monitors apply --dry-run`,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        "config",
				Usage:       "The configuration file containing monitor information",
				Aliases:     []string{"c"},
				DefaultText: "openstatus.yaml",
				Value:       "openstatus.yaml",
			},
			&cli.StringFlag{
				Name:    "access-token",
				Usage:   "OpenStatus API Access Token",
				Aliases: []string{"t"},
				Sources: cli.EnvVars("OPENSTATUS_API_TOKEN"),
			},
			&cli.BoolFlag{
				Name:     "auto-accept",
				Usage:    "Automatically accept the prompt",
				Aliases:  []string{"y"},
				Required: false,
			},
			&cli.BoolFlag{
				Name:    "dry-run",
				Usage:   "Show what would be changed without applying",
				Aliases: []string{"n"},
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			apiKey, err := auth.ResolveAccessToken(cmd)
			if err != nil {
				return cli.Exit(err.Error(), 1)
			}

			path := cmd.String("config")

			if path != "" {
				if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
					return cli.Exit("Config does not exist", 1)
				}
			}

			monitors, err := config.ReadOpenStatus(path)
			if err != nil {
				return cli.Exit("Unable to read config file", 1)
			}

			lock, err := config.ReadLockFile("openstatus.lock")
			if err != nil {
				return cli.Exit("Unable to read lock file", 1)
			}

			created, updated, deleted := countChanges(lock, monitors)
			if created == 0 && updated == 0 && deleted == 0 {
				fmt.Println("No changes found")
				return nil
			}

			fmt.Println("This will apply the following changes:")
			if created > 0 {
				fmt.Println("  Create:", created)
			}
			if updated > 0 {
				fmt.Println("  Update:", updated)
			}
			if deleted > 0 {
				fmt.Println("  Delete:", deleted)
			}

			if cmd.Bool("dry-run") {
				return nil
			}

			if !cmd.Bool("auto-accept") {
				confirmed, err := output.AskForConfirmation("Do you want to continue?")
				if err != nil {
					return cli.Exit(fmt.Sprintf("Failed to read input: %v", err), 1)
				}
				if !confirmed {
					return nil
				}
			}

			s := output.StartSpinner("Applying changes...")
			newLock, err := ApplyChanges(ctx, apiKey, lock, monitors)
			output.StopSpinner(s)
			if err != nil {
				return cli.Exit(fmt.Sprintf("Failed to apply changes: %v", err), 1)
			}
			if newLock == nil {
				fmt.Println("No changes found")
				return nil
			}
			y, err := yaml.Marshal(&newLock)
			if err != nil {
				return cli.Exit("Failed to marshal lock file", 1)
			}
			file, err := os.OpenFile("openstatus.lock", os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
			if err != nil {
				return cli.Exit("Failed to open lock file", 1)
			}
			defer file.Close()

			_, err = file.Write(y)
			if err != nil {
				return cli.Exit("Failed to write lock file", 1)
			}
			if err := file.Sync(); err != nil {
				return cli.Exit("Failed to sync lock file", 1)
			}
			fmt.Println("\nRun 'openstatus monitors list' to see your monitors")
			return nil
		},
	}
	return &monitorsApplyCmd
}
