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

var allMonitor bool

func ListMonitors(httpClient *http.Client, apiKey string) error {
	url := "https://api.openstatus.dev/v1/monitor"

	req, _ := http.NewRequest(http.MethodGet, url, nil)
	req.Header.Add("x-openstatus-key", apiKey)
	res, err := httpClient.Do(req)
	if err != nil {
		return err
	}

	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("Failed to list monitors")
	}

	defer res.Body.Close()
	body, _ := io.ReadAll(res.Body)
	var monitors []Monitor
	err = json.Unmarshal(body, &monitors)
	if err != nil {
		return err
	}

	headerFmt := color.New(color.FgGreen, color.Underline).SprintfFunc()
	columnFmt := color.New(color.FgYellow).SprintfFunc()

	tbl := table.New("ID", "Name", "Url")
	tbl.WithHeaderFormatter(headerFmt).WithFirstColumnFormatter(columnFmt)

	for _, monitor := range monitors {
		if monitor.Active || allMonitor {
			tbl.AddRow(monitor.ID, monitor.Name, monitor.URL)
		}
	}
	tbl.Print()

	return nil
}

func GetMonitorsListCmd() *cli.Command {
	monitorsListCmd := cli.Command{
		Name:        "list",
		Usage:       "List all monitors",
		Description: "List all monitors. The list shows all your monitors attached to your workspace. It displays the ID, name, and URL of each monitor.",
		UsageText:   "openstatus monitors list [options]",
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:        "all",
				Usage:       "List all monitors including inactive ones",
				Destination: &allMonitor,
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
			fmt.Println("List of all monitors")
			err := ListMonitors(http.DefaultClient, cmd.String("access-token"))
			if err != nil {
				return cli.Exit("Failed to list monitors", 1)
			}
			return nil
		},
	}
	return &monitorsListCmd
}
