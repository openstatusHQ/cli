package monitors

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/urfave/cli/v2"
)

func getMonitorInfo(httpClient *http.Client, apiKey string, monitorId string) error {

	if monitorId == "" {
		return fmt.Errorf("Monitor ID is required")
	}

	url := "https://api.openstatus.dev/v1/monitor/" + monitorId

	req, _ := http.NewRequest("GET", url, nil)

	req.Header.Add("x-openstatus-key", apiKey)

	res, err := httpClient.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	body, _ := io.ReadAll(res.Body)

	var monitors Monitor
	err = json.Unmarshal(body, &monitors)
	if err != nil {
		return err
	}

	fmt.Println("Monitor")
	fmt.Printf("ID: %d\nName: %s\nURL: %s\nPeriodicity: %s\nDescription: %s\nMethod: %s\nActive: %t\nPublic: %t\nTimeout: %d\nDegradedAfter: %d\n", monitors.ID, monitors.Name, monitors.URL, monitors.Periodicity, monitors.Description, monitors.Method, monitors.Active, monitors.Public, monitors.Timeout, monitors.DegradedAfter)
	return nil
}

func GetMonitorInfoCmd() *cli.Command {
	monitorInfoCmd := cli.Command{
		Name:  "info",
		Usage: "Get monitor information",
		Action: func(cCtx *cli.Context) error {
			fmt.Println("Monitor information")
			monitorId := cCtx.Args().Get(cCtx.Args().Len() - 1)
			getMonitorInfo(http.DefaultClient, cCtx.String("access-token"), monitorId)
			return nil
		},
	}
	return &monitorInfoCmd
}
