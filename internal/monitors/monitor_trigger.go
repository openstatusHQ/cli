package monitors

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/fatih/color"
	"github.com/rodaine/table"
	"github.com/urfave/cli/v3"
)

var (
	noWait   bool
	noResult bool
)

func monitorTrigger(httpClient *http.Client, apiKey string, monitorId string) error {

	if monitorId == "" {
		return fmt.Errorf("Monitor ID is required")
	}
	fmt.Println("Waiting for the result...")

	url := fmt.Sprintf("https://api.openstatus.dev/v1/monitor/%s/run", monitorId)

	req, err := http.NewRequest("POST", url, nil)
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

	var result []RunResult
	err = json.Unmarshal(body, &result)
	if err != nil {
		return err
	}

	headerFmt := color.New(color.FgGreen, color.Underline).SprintfFunc()
	columnFmt := color.New(color.FgYellow).SprintfFunc()

	tbl := table.New("Region", "Latency (ms)", "Status")
	tbl.WithHeaderFormatter(headerFmt).WithFirstColumnFormatter(columnFmt)

	var inError bool
	for _, r := range result {
		if r.Error != "" {
			inError = true
			tbl.AddRow(r.Region, r.Latency, color.RedString("❌"))
		} else {
			tbl.AddRow(r.Region, r.Latency, color.GreenString("✔"))
		}

	}
	tbl.Print()

	if inError {
		fmt.Println(color.RedString("Some regions failed"))

		return fmt.Errorf("Some regions failed")
	} else {
		fmt.Println(color.GreenString("All regions passed"))
	}
	return nil
}

func GetMonitorsTriggerCmd() *cli.Command {
	monitorsCmd := cli.Command{
		Name:  "trigger",
		Usage: "Trigger a monitor test",
		Flags: []cli.Flag{

			&cli.BoolFlag{
				Name:        "no-result",
				Usage:       "Do not return the result of the test, return the result ID",
				Destination: &noResult,
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
			monitorId := cmd.Args().Get(0)
			err := monitorTrigger(http.DefaultClient, cmd.String("access-token"), monitorId)
			if err != nil {
				return cli.Exit("Failed to trigger monitor", 1)
			}
			return nil
		},
	}
	return &monitorsCmd
}
