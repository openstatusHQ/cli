package cmd_test

import (
	"testing"

	"github.com/openstatusHQ/cli/internal/cmd"
)

func Test_NewApp(t *testing.T) {
	t.Parallel()

	t.Run("Returns valid app command", func(t *testing.T) {
		app := cmd.NewApp()

		if app == nil {
			t.Fatal("Expected non-nil app")
		}

		if app.Name != "openstatus" {
			t.Errorf("Expected app name 'openstatus', got %s", app.Name)
		}

		if app.Version != "v1.0.3" {
			t.Errorf("Expected version 'v1.0.3', got %s", app.Version)
		}

		if !app.Suggest {
			t.Error("Expected Suggest to be true")
		}
	})

	t.Run("Has expected commands", func(t *testing.T) {
		app := cmd.NewApp()

		if len(app.Commands) != 10 {
			t.Errorf("Expected 10 commands, got %d", len(app.Commands))
		}

		expectedCommands := map[string]bool{
			"monitors":      false,
			"status-report": false,
			"maintenance":   false,
			"status-page":   false,
			"notification":  false,
			"run":           false,
			"whoami":        false,
			"login":         false,
			"logout":        false,
			"terraform":     false,
		}

		for _, subcmd := range app.Commands {
			if _, exists := expectedCommands[subcmd.Name]; exists {
				expectedCommands[subcmd.Name] = true
			}
		}

		for name, found := range expectedCommands {
			if !found {
				t.Errorf("Expected command '%s' not found", name)
			}
		}
	})

	t.Run("Has correct usage text", func(t *testing.T) {
		app := cmd.NewApp()

		expectedUsage := "Manage status pages, monitors, and incidents from the terminal"
		if app.Usage != expectedUsage {
			t.Errorf("Expected usage '%s', got '%s'", expectedUsage, app.Usage)
		}
	})

	t.Run("Has description", func(t *testing.T) {
		app := cmd.NewApp()

		if app.Description == "" {
			t.Error("Expected non-empty description")
		}
	})
}
