package run

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/fatih/color"
	"github.com/logrusorgru/aurora/v4"
	"github.com/openstatusHQ/cli/internal/api"
	"github.com/openstatusHQ/cli/internal/auth"
	output "github.com/openstatusHQ/cli/internal/cli"
	"github.com/openstatusHQ/cli/internal/config"
	"github.com/openstatusHQ/cli/internal/monitors"
	"github.com/rodaine/table"
	"github.com/urfave/cli/v3"
)

type runRegionResult struct {
	Region  string `json:"region"`
	Latency int64  `json:"latency"`
	Status  string `json:"status"`
	Error   string `json:"error,omitempty"`
}

type runMonitorResult struct {
	MonitorID string            `json:"monitor_id"`
	Results   []runRegionResult `json:"results"`
}

// MonitorTrigger triggers a monitor run and returns the results without printing.
func MonitorTrigger(ctx context.Context, httpClient *http.Client, apiKey string, monitorId string) (runMonitorResult, error) {

	if monitorId == "" {
		return runMonitorResult{}, fmt.Errorf("monitor ID is required")
	}

	url := fmt.Sprintf("%s/monitor/%s/run", api.APIBaseURL, monitorId)

	client := &http.Client{
		Timeout:   2 * time.Minute,
		Transport: httpClient.Transport,
	}

	payload := strings.NewReader("{}")

	req, err := http.NewRequestWithContext(ctx, "POST", url, payload)
	if err != nil {
		return runMonitorResult{}, err
	}
	req.Header.Add("x-openstatus-key", apiKey)
	res, err := client.Do(req)
	if err != nil {
		return runMonitorResult{}, err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return runMonitorResult{}, fmt.Errorf("failed to trigger monitor test")
	}
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return runMonitorResult{}, fmt.Errorf("failed to read response body: %w", err)
	}

	var result []json.RawMessage
	err = json.Unmarshal(body, &result)
	if err != nil {
		return runMonitorResult{}, err
	}

	var regionResults []runRegionResult
	for _, r := range result {
		rr := monitors.RunResult{}

		if err := json.Unmarshal(r, &rr); err != nil {
			return runMonitorResult{}, fmt.Errorf("unable to unmarshal: %w", err)
		}

		entry := runRegionResult{
			Region:  rr.Region,
			Latency: rr.Latency,
			Status:  "pass",
		}

		switch rr.JobType {
		case "tcp":
			var tcp monitors.TCPRunResult
			if err := json.Unmarshal(r, &tcp); err != nil {
				return runMonitorResult{}, fmt.Errorf("unable to unmarshal: %w", err)
			}
			if tcp.ErrorMessage != "" {
				entry.Status = "fail"
				entry.Error = tcp.ErrorMessage
			}
		case "http":
			var httpResult monitors.HTTPRunResult
			if err := json.Unmarshal(r, &httpResult); err != nil {
				return runMonitorResult{}, fmt.Errorf("unable to unmarshal: %w", err)
			}
			if httpResult.Error != "" {
				entry.Status = "fail"
				entry.Error = httpResult.Error
			}
		default:
			entry.Status = "unknown"
			entry.Error = fmt.Sprintf("unknown job type: %s", rr.JobType)
		}

		regionResults = append(regionResults, entry)
	}

	return runMonitorResult{
		MonitorID: monitorId,
		Results:   regionResults,
	}, nil
}

// printMonitorResult prints a single monitor's results to stdout.
func printMonitorResult(res runMonitorResult) bool {
	var inError bool
	fmt.Println(aurora.Bold(fmt.Sprintf("Monitor: %s", res.MonitorID)))
	headerFmt := color.New(color.FgGreen, color.Underline).SprintfFunc()
	columnFmt := color.New(color.FgYellow).SprintfFunc()

	tbl := table.New("Region", "Latency (ms)", "Status")
	tbl.WithHeaderFormatter(headerFmt).WithFirstColumnFormatter(columnFmt)

	for _, entry := range res.Results {
		if entry.Status == "fail" {
			tbl.AddRow(entry.Region, entry.Latency, color.RedString("fail"))
			inError = true
		} else {
			tbl.AddRow(entry.Region, entry.Latency, color.GreenString("pass"))
		}
	}
	tbl.Print()

	if inError {
		fmt.Println(color.RedString("Some regions failed"))
	} else {
		fmt.Println(color.GreenString("All regions passed"))
	}
	fmt.Println()
	return inError
}

func RunCmd() *cli.Command {
	runCmd := cli.Command{
		Name:    "run",
		Aliases: []string{"r"},
		Usage:   "Run your uptime tests",
		UsageText: `openstatus run
  openstatus run --config custom-config.yaml`,
		Description: `Run the uptime tests defined in the config.openstatus.yaml.
The config file should be in the following format:

tests:
  ids:
     - monitor-id-1
     - monitor-id-2

     `,
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

			conf, err := config.ReadConfig(path)
			if err != nil {
				return err
			}
			size := len(conf.Tests.Ids)

			if !output.IsQuiet() && !output.IsJSONOutput() {
				fmt.Print("Tests are running\n\n")
			}

			type indexedResult struct {
				index  int
				result runMonitorResult
				err    error
			}

			results := make([]indexedResult, size)
			var wg sync.WaitGroup
			var mu sync.Mutex

			for i, id := range conf.Tests.Ids {
				wg.Add(1)
				go func(idx, id int) {
					defer wg.Done()
					res, err := MonitorTrigger(ctx, http.DefaultClient, apiKey, fmt.Sprintf("%d", id))
					mu.Lock()
					results[idx] = indexedResult{index: idx, result: res, err: err}
					mu.Unlock()
				}(i, id)
			}
			wg.Wait()

			// Print results sequentially to avoid interleaved output
			var hasErrors bool

			if output.IsJSONOutput() {
				var allResults []runMonitorResult
				for _, r := range results {
					if r.err != nil {
						hasErrors = true
						continue
					}
					allResults = append(allResults, r.result)
				}
				if err := output.PrintJSON(allResults); err != nil {
					return err
				}
			} else {
				for _, r := range results {
					if r.err != nil {
						hasErrors = true
						fmt.Printf("Monitor %d: %v\n\n", conf.Tests.Ids[r.index], r.err)
						continue
					}
					if printMonitorResult(r.result) {
						hasErrors = true
					}
				}
			}

			if hasErrors {
				return cli.Exit("Some tests failed", 1)
			}
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
				Name:    "access-token",
				Usage:   "OpenStatus API Access Token",
				Aliases: []string{"t"},
				Sources: cli.EnvVars("OPENSTATUS_API_TOKEN"),
			},
		},
	}
	return &runCmd
}
