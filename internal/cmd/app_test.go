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

		if app.Version != "v1.0.0" {
			t.Errorf("Expected version 'v1.0.0', got %s", app.Version)
		}

		if !app.Suggest {
			t.Error("Expected Suggest to be true")
		}
	})

	t.Run("Has expected commands", func(t *testing.T) {
		app := cmd.NewApp()

		if len(app.Commands) != 3 {
			t.Errorf("Expected 3 commands, got %d", len(app.Commands))
		}

		expectedCommands := map[string]bool{
			"monitors": false,
			"run":      false,
			"whoami":   false,
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

		expectedUsage := "This is OpenStatus Command Line Interface, the OpenStatus.dev CLI"
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
