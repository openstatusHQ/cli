package monitors

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"

	confirmation "github.com/openstatusHQ/cli/internal/cli"
	"github.com/openstatusHQ/cli/internal/config"
	"github.com/urfave/cli/v3"
)

func CreateMonitor(httpClient *http.Client, apiKey string, monitor config.Monitor) error {

	url := fmt.Sprintf("https://api.openstatus.dev/v1/monitor/%s", monitor.Kind)

	payloadBuf := new(bytes.Buffer)
	json.NewEncoder(payloadBuf).Encode(monitor)
	req, _ := http.NewRequest(http.MethodPost, url, payloadBuf)

	req.Header.Add("x-openstatus-key", apiKey)
	req.Header.Add("Content-Type", "application/json")

	res, err := httpClient.Do(req)
	if err != nil {
		return err
	}

	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("Failed to create monitor")
	}
	defer res.Body.Close()
	body, _ := io.ReadAll(res.Body)

	var monitors Monitor
	err = json.Unmarshal(body, &monitors)
	if err != nil {
		return err
	}

	return nil
}

func GetMonitorCreateCmd() *cli.Command {
	monitorInfoCmd := cli.Command{
		Name:        "create",
		Usage:       "Create monitors (beta)",
		Description: "Create the monitors defined in the openstatus.yaml file",
		UsageText:   "openstatus monitors create [options]",

		Action: func(ctx context.Context, cmd *cli.Command) error {

			path := cmd.String("config")

			if path != "" {
				if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
					return cli.Exit("Config does not exist", 1)
				}
			}

			accept := cmd.Bool("auto-accept")

			out, err := config.ReadOpenStatus(path)
			if err != nil {
				return cli.Exit("Unable to read config file", 1)
			}

			monitors := config.TranslateMonitors(out)

			if !accept {
				if !confirmation.AskForConfirmation(fmt.Sprintf("You are about to create %d monitors do you want to continue", len(monitors))) {
					return nil
				}
			}

			for _, value := range monitors {
				err = CreateMonitor(http.DefaultClient, cmd.String("access-token"), value)
				if err != nil {
					return cli.Exit("Unable to create monitor", 1)
				}
			}
			fmt.Printf("%d monitors created successfully\n", len(monitors))
			return nil
		},
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
			&cli.BoolFlag{
				Name:     "auto-accept",
				Usage:    "Automatically accept the prompt",
				Aliases:  []string{"y"},
				Required: false,
			},
		},
	}
	return &monitorInfoCmd
}
