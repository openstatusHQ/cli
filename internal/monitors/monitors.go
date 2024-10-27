package monitors

import (
	"github.com/urfave/cli/v2"
)

type Monitor struct {
	ID            int    `json:"id"`
	Name          string `json:"name"`
	URL           string `json:"url"`
	Periodicity   string `json:"periodicity"`
	Description   string `json:"description"`
	Method        string `json:"method"`
	Active        bool   `json:"active"`
	Public        bool   `json:"public"`
	Timeout       int    `json:"timeout"`
	DegradedAfter int    `json:"degraded_after,omitempty"`
}

func MonitorsCmd() *cli.Command {
	monitorsCmd := cli.Command{
		Name:  "monitors",
		Usage: "Manage your monitors",

		Subcommands: []*cli.Command{
			GetMonitorInfoCmd(),
			GetMonitorsListCmd(),
			GetMonitorsTriggerCmd(),
		},
	}
	return &monitorsCmd
}
