package monitors

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/urfave/cli/v3"
)

var (
	noWait   bool
	noResult bool
)

func MonitorTrigger(httpClient *http.Client, apiKey string, monitorId string) error {

	if monitorId == "" {
		return fmt.Errorf("Monitor ID is required")
	}
	fmt.Println("Waiting for the result...")

	url := fmt.Sprintf("https://api.openstatus.dev/v1/monitor/%s/trigger", monitorId)

	req, err := http.NewRequest(http.MethodPost, url, nil)
	if err != nil {
		return err
	}
	req.Header.Add("x-openstatus-key", apiKey)
	res, err := httpClient.Do(req)
	if err != nil {
		return err
	}

	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("Failed to trigger monitor test")
	}

	defer res.Body.Close()
	body, _ := io.ReadAll(res.Body)

	var r MonitorTriggerResponse
	err = json.Unmarshal(body, &r)
	if err != nil {
		return err
	}
	fmt.Printf("Check triggered successfully\n")

	return nil
}

func GetMonitorsTriggerCmd() *cli.Command {
	monitorsCmd := cli.Command{
		Name:        "trigger",
		Usage:       "Trigger a monitor execution",
		UsageText:   "openstatus monitors trigger [MonitorId] [options]",
		Description: "Trigger a monitor execution on demand. This command allows you to launch your tests on demand.",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "access-token",
				Usage:    "OpenStatus API Access Token",
				Aliases:  []string{"t"},
				Sources:  cli.EnvVars("OPENSTATUS_API_TOKEN"),
				Required: true,
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			monitorId := cmd.Args().Get(0)
			err := MonitorTrigger(http.DefaultClient, cmd.String("access-token"), monitorId)
			if err != nil {
				return cli.Exit("Failed to trigger monitor", 1)
			}
			return nil
		},
	}
	return &monitorsCmd
}
