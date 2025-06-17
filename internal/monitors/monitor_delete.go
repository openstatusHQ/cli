package monitors

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	confirmation "github.com/openstatusHQ/cli/internal/cli"
	"github.com/urfave/cli/v3"
)

func DeleteMonitor(httpClient *http.Client, apiKey string, monitorId string) error {

	if monitorId == "" {
		return fmt.Errorf("Monitor ID is required")
	}

	url := fmt.Sprintf("https://api.openstatus.dev/v1/monitor/%s", monitorId)

	req, err := http.NewRequest(http.MethodDelete, url, nil)
	if err != nil {
		return err
	}
	req.Header.Add("x-openstatus-key", apiKey)
	res, err := httpClient.Do(req)
	if err != nil {
		return err
	}

	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("Failed to delete monitor")
	}

	defer res.Body.Close()
	body, _ := io.ReadAll(res.Body)

	var r MonitorTriggerResponse
	err = json.Unmarshal(body, &r)
	if err != nil {

		return err
	}
	fmt.Printf("Monitor deleted successfully\n")
	return nil
}

func GetMonitorDeleteCmd() *cli.Command {
	monitorsCmd := cli.Command{
		Name:      "delete",
		Usage:     "Delete a monitor",
		UsageText: "openstatus monitors delete [MonitorID] [options]",

		Flags: []cli.Flag{
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
			monitorId := cmd.Args().Get(0)

			if !cmd.Bool("auto-accept") {
				if !confirmation.AskForConfirmation(fmt.Sprintf("You are about to delete monitor: %s, do you want to continue", monitorId)) {
					return nil
				}
			}
			err := DeleteMonitor(http.DefaultClient, cmd.String("access-token"), monitorId)
			if err != nil {
				return cli.Exit("Failed to delete monitor", 1)
			}
			return nil
		},
	}
	return &monitorsCmd
}
