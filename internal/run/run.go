package run

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"

	"github.com/fatih/color"
	"github.com/openstatusHQ/cli/internal/config"
	"github.com/openstatusHQ/cli/internal/monitors"
	"github.com/rodaine/table"
	"github.com/urfave/cli/v3"
)

func MonitorTrigger(httpClient *http.Client, apiKey string, monitorId string) error {

	if monitorId == "" {
		return fmt.Errorf("Monitor ID is required")
	}

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

	var result []json.RawMessage
	err = json.Unmarshal(body, &result)
	if err != nil {
		return err
	}
	fmt.Println("Results for monitor:", monitorId)
	headerFmt := color.New(color.FgGreen, color.Underline).SprintfFunc()
	columnFmt := color.New(color.FgYellow).SprintfFunc()

	tbl := table.New("Region", "Latency (ms)", "Status")
	tbl.WithHeaderFormatter(headerFmt).WithFirstColumnFormatter(columnFmt)

	var inError bool
	for _, r := range result {
		result := monitors.RunResult{}

		if err := json.Unmarshal(r, &result); err != nil {

			return fmt.Errorf("unable to unmarshal : %w", err)
		}
		switch result.JobType {
		case "tcp":
			{
				var tcp monitors.TCPRunResult
				if err := json.Unmarshal(r, &result); err != nil {
					return fmt.Errorf("unable to unmarshal : %w", err)
				}
				if tcp.ErrorMessage != "" {
					inError = true
					tbl.AddRow(result.Region, result.Latency, color.RedString("❌"))
					continue
				}

			}
		case "http":
			{
				var http monitors.HTTPRunResult
				if err := json.Unmarshal(r, &http); err != nil {
					fmt.Println("Error", err)
					return fmt.Errorf("unable to unmarshal : %w", err)
				}
				if http.Error != "" {
					inError = true
					tbl.AddRow(result.Region, result.Latency, color.RedString("❌"))
					continue
				}
			}
		default:
			return fmt.Errorf("Unknown job type")
		}

		tbl.AddRow(result.Region, result.Latency, color.GreenString("✔"))

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

func RunCmd() *cli.Command {
	runCmd := cli.Command{
		Name:    "run",
		Aliases: []string{"r"},
		Usage:   "Run your synthetics tests defined in your configuration file",
		Action: func(ctx context.Context, cmd *cli.Command) error {

			path := cmd.String("config")
			conf, err := config.ReadConfig(path)
			if err != nil {
				return err
			}
			size := len(conf.Tests.Ids)
			ch := make(chan error, size)

			fmt.Println("Tests are running")

			var wg sync.WaitGroup

			for _, id := range conf.Tests.Ids {
				wg.Add(1)
				go func(id int) {
					defer wg.Done()
					if err := MonitorTrigger(http.DefaultClient, cmd.String("access-token"), fmt.Sprintf("%d", id)); err != nil {
						ch <- err
					}

				}(id)
			}
			wg.Wait()
			close(ch) // Close the channel when all workers have finished

			if len(ch) > 0 {
				return cli.Exit("Some tests failed", 1)
			}
			fmt.Println("Test ran succesfully🔥")
			return nil
		},
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        "config",
				Usage:       "The configuration file",
				DefaultText: "config.openstatus.yaml",
				Value:       "config.openstatus.yaml",
			},
			&cli.StringFlag{
				Name:     "access-token",
				Usage:    "OpenStatus API Access Token",
				Aliases:  []string{"t"},
				Sources:  cli.EnvVars("OPENSTATUS_API_TOKEN"),
				Required: true,
			},
		},
	}
	return &runCmd
}
