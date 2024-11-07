package monitors

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/urfave/cli/v3"
)

func GetMonitorInfo(httpClient *http.Client, apiKey string, monitorId string) error {

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
	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("Failed to get monitor information")
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
		Action: func(ctx context.Context, cmd *cli.Command) error {
			fmt.Println("Monitor information")
			monitorId := cmd.Args().Get(0)
			err := GetMonitorInfo(http.DefaultClient, cmd.String("access-token"), monitorId)
			if err != nil {
				return cli.Exit("Failed to get monitor information", 1)
			}
			return nil
		},
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "access-token",
				Usage:    "OpenStatus API Access Token",
				Aliases:  []string{"t"},
				Sources:  cli.EnvVars("OPENSTATUS_API_TOKEN"),
				Required: true,
			}},
	}
	return &monitorInfoCmd
}
