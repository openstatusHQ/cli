package monitors

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/openstatusHQ/cli/internal/config"
	"github.com/urfave/cli/v3"
	"sigs.k8s.io/yaml"
)

func ExportMonitor(httpClient *http.Client, apiKey string, path string) error {
	url := APIBaseURL + "/monitor"

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Add("x-openstatus-key", apiKey)
	res, err := httpClient.Do(req)
	if err != nil {
		return err
	}

	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to get all monitors")
	}

	defer res.Body.Close()
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}
	var monitors []Monitor
	err = json.Unmarshal(body, &monitors)
	if err != nil {
		return err
	}

	t := map[string]config.Monitor{}
	lock := make(map[string]config.Lock, len(monitors))

	file, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return err
	}
	defer file.Close()

	for _, monitor := range monitors {

		var regions []config.Region

		for _, region := range monitor.Regions {
			regions = append(regions, config.Region(region))
		}
		var request config.Request
		var assertions []config.Assertion
		switch monitor.JobType {
		case "http":
			var headers = make(map[string]string)
			for _, header := range monitor.Headers {
				// Our API should not allow empty headers
				if header.Key == "" {
					continue
				}
				headers[header.Key] = header.Value
			}

			for _, assertion := range monitor.Assertions {

				kind := config.AssertionKind(assertion.Type)
				// Our api return status instead of Status Code
				if kind == "status" {
					kind = config.StatusCode
				}

				assertions = append(assertions, config.Assertion{
					Kind:    kind,
					Target:  assertion.Target,
					Compare: config.Compare(assertion.Compare),
					Key:     assertion.Key,
				})
			}

			request = config.Request{
				URL:     monitor.URL,
				Method:  config.Method(monitor.Method),
				Body:    monitor.Body,
				Headers: headers,
			}
		case "tcp":
			uri := strings.Split(monitor.URL, ":")

			port, _ := strconv.Atoi(uri[1])
			request = config.Request{
				Host: uri[0],
				Port: int64(port),
			}

		default:
			return fmt.Errorf("unknown job type: %s", monitor.JobType)
		}

		t[fmt.Sprint(monitor.ID)] = config.Monitor{
			Name:          monitor.Name,
			Active:        monitor.Active,
			Public:        monitor.Public,
			Description:   monitor.Description,
			DegradedAfter: int64(monitor.DegradedAfter),
			Frequency:     config.Frequency(monitor.Periodicity),
			Request:       request,
			Kind:          config.CoordinateKind(monitor.JobType),
			Retry:         int64(monitor.Retry),
			Regions:       regions,
			Assertions:    assertions,
		}
	}
	y, err := yaml.Marshal(&t)
	if err != nil {
		return err
	}

	// file.WriteString("# yaml-language-server: $schema=https://raw.githubusercontent.com/openstatusHQ/json-schema/refs/heads/improve-schema/1.0.1.json\n\n")
	_, err = file.WriteString("# yaml-language-server: $schema=https://www.openstatus.dev/schema.json\n\n")
	if err != nil {
		return err
	}
	_, err = file.Write(y)
	if err != nil {
		return err
	}

	//
	for id, monitor := range t {
		i, _ := strconv.Atoi(id)
		lock[id] = config.Lock{
			ID:      i,
			Monitor: monitor,
		}
	}

	lockFile, err := os.OpenFile("openstatus.lock", os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return cli.Exit("Failed to apply change", 1)

	}
	defer lockFile.Close()

	y, err = yaml.Marshal(&lock)
	if err != nil {
		return cli.Exit("Failed to apply change", 1)
	}

	_, err = lockFile.Write(y)
	if err != nil {
		return cli.Exit("Failed to apply change", 1)
	}

	return nil
}

func GetMonitorImportCmd() *cli.Command {
	monitorInfoCmd := cli.Command{
		Name:        "import",
		Usage:       "Import all your monitors",
		UsageText:   "openstatus monitors import [options]",
		Description: "Import all your monitors from your workspace to a YAML file; it will also create a lock file to manage your monitors with 'apply'.",
		Action: func(ctx context.Context, cmd *cli.Command) error {
			// monitorId := cmd.Args().Get(0)
			err := ExportMonitor(http.DefaultClient, cmd.String("access-token"), cmd.String("output"))
			if err != nil {
				return cli.Exit(err.Error(), 1)
			}
			fmt.Printf("Monitors successfully imported to: %s", cmd.String("output"))
			return nil
		},
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "access-token",
				Usage:    "OpenStatus API Access Token",
				Aliases:  []string{"t"},
				Sources:  cli.EnvVars("OPENSTATUS_API_TOKEN"),
				Required: true,
			},
			&cli.StringFlag{
				Name:        "output",
				Usage:       "The output file name ",
				DefaultText: "openstatus.yaml",
				Value:       "openstatus.yaml",
				Aliases:     []string{"o"},
			},
		},
	}
	return &monitorInfoCmd
}
