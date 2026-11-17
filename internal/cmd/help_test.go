package cmd_test

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/openstatusHQ/cli/internal/cmd"
)

func Test_RootHelpGroupsCommandsByPillar(t *testing.T) {
	t.Parallel()

	app := cmd.NewApp()
	var out bytes.Buffer
	app.Writer = &out
	if err := app.Run(context.Background(), []string{"openstatus", "--help"}); err != nil {
		t.Fatal(err)
	}
	_, help, _ := strings.Cut(out.String(), "COMMANDS:")
	help, _, _ = strings.Cut(help, "GLOBAL OPTIONS:")

	groups := []struct {
		heading  string
		commands []string
	}{
		{"Monitor:", []string{"monitors, m", "check, c", "run, r", "private-locations, pl"}},
		{"Respond:", []string{"incident, inc, incidents", "notification, n"}},
		{"Communicate:", []string{"status-page, sp", "status-report, sr", "maintenance, mt"}},
		{"Infrastructure as Code:", []string{"terraform, tf"}},
		{"Account:", []string{"whoami, w", "login", "logout"}},
		{"Other:", []string{"version", "help, h"}},
	}

	last := -1
	for _, g := range groups {
		start := strings.Index(help, "   "+g.heading+"\n")
		if start <= last {
			t.Fatalf("heading %q missing or out of order in:\n%s", g.heading, help)
		}
		last = start
		for _, c := range g.commands {
			i := strings.Index(help, "     "+c+" ")
			if i <= last {
				t.Errorf("command %q not listed under %q in:\n%s", c, g.heading, help)
				continue
			}
			last = i
		}
	}
}
