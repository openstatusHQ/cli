package monitors

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/urfave/cli/v2"
)

var allMonitor bool

func listMonitors(httpClient *http.Client, apiKey string) {
	url := "https://api.openstatus.dev/v1/monitor"

	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Add("x-openstatus-key", apiKey)
	res, err := httpClient.Do(req)
	if err != nil {
		return
	}

	defer res.Body.Close()
	body, _ := io.ReadAll(res.Body)
	var monitors []Monitor
	err = json.Unmarshal(body, &monitors)
	if err != nil {
		return
	}

	fmt.Println("Monitors")
	for _, monitor := range monitors {
		if monitor.Active || allMonitor {
			fmt.Printf("%d %s %s \n", monitor.ID, monitor.Name, monitor.URL)
		}
	}
}

func GetMonitorsListCmd() *cli.Command {
	monitorsListCmd := cli.Command{
		Name:  "list",
		Usage: "List all monitors",
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:        "all",
				Usage:       "List all monitors including inactive ones",
				Destination: &allMonitor,
			},
		},
		Action: func(cCtx *cli.Context) error {
			fmt.Println("List of all monitors")
			listMonitors(http.DefaultClient, cCtx.String("access-token"))
			return nil
		},
	}
	return &monitorsListCmd
}
