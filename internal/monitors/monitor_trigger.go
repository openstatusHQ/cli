package monitors

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/urfave/cli/v2"
)

var (
	noWait   bool
	noResult bool
)

func monitorTrigger(httpClient *http.Client, apiKey string, monitorId string) error {

	if monitorId == "" {
		return fmt.Errorf("Monitor ID is required")
	}

	url := fmt.Sprintf("https://api.openstatus.dev/v1/monitor/%s/trigger", monitorId)

	req, _ := http.NewRequest("POST", url, nil)
	req.Header.Add("x-openstatus-key", apiKey)
	res, err := httpClient.Do(req)
	if err != nil {
		return err
	}

	defer res.Body.Close()
	body, _ := io.ReadAll(res.Body)
	var monitors []Monitor
	err = json.Unmarshal(body, &monitors)
	if err != nil {
		return err
	}

	fmt.Println("Monitors")
	for _, monitor := range monitors {
		if monitor.Active || allMonitor {
			fmt.Printf("%d %s %s \n", monitor.ID, monitor.Name, monitor.URL)
		}
	}

	return nil
}

func GetMonitorsTriggerCmd() *cli.Command {
	monitorsCmd := cli.Command{
		Name:  "trigger",
		Usage: "Trigger a monitor test",
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:  "no-wait",
				Usage: "Do not wait for the result, return immediately",

				Destination: &noWait,
			},
			&cli.BoolFlag{
				Name:        "no-result",
				Usage:       "Do not return the result of the test, return the result ID",
				Destination: &noResult,
			},
		},
		Action: func(cCtx *cli.Context) error {
			r := cCtx.String("access-token")
			fmt.Println("Triggering monitor test", r)
			return nil
		},
	}
	return &monitorsCmd
}
