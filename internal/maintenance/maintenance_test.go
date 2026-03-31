package maintenance_test

import (
	"net/http"
	"testing"

	"github.com/openstatusHQ/cli/internal/maintenance"
)

type interceptorHTTPClient struct {
	f func(req *http.Request) (*http.Response, error)
}

func (i *interceptorHTTPClient) RoundTrip(req *http.Request) (*http.Response, error) {
	return i.f(req)
}

func (i *interceptorHTTPClient) GetHTTPClient() *http.Client {
	return &http.Client{
		Transport: i,
	}
}

func Test_MaintenanceCmd(t *testing.T) {
	t.Parallel()

	t.Run("Returns valid command", func(t *testing.T) {
		cmd := maintenance.MaintenanceCmd()

		if cmd == nil {
			t.Fatal("Expected non-nil command")
		}

		if cmd.Name != "maintenance" {
			t.Errorf("Expected command name 'maintenance', got %s", cmd.Name)
		}

		if cmd.Usage != "Manage maintenance windows" {
			t.Errorf("Expected usage 'Manage maintenance windows', got %s", cmd.Usage)
		}
	})

	t.Run("Has expected subcommands", func(t *testing.T) {
		cmd := maintenance.MaintenanceCmd()

		if len(cmd.Commands) != 5 {
			t.Errorf("Expected 5 subcommands, got %d", len(cmd.Commands))
		}

		expectedSubcommands := map[string]bool{
			"list":   false,
			"info":   false,
			"create": false,
			"update": false,
			"delete": false,
		}

		for _, subcmd := range cmd.Commands {
			if _, exists := expectedSubcommands[subcmd.Name]; exists {
				expectedSubcommands[subcmd.Name] = true
			}
		}

		for name, found := range expectedSubcommands {
			if !found {
				t.Errorf("Expected subcommand '%s' not found", name)
			}
		}
	})

	t.Run("Has mt alias", func(t *testing.T) {
		cmd := maintenance.MaintenanceCmd()

		found := false
		for _, alias := range cmd.Aliases {
			if alias == "mt" {
				found = true
				break
			}
		}
		if !found {
			t.Error("Expected 'mt' alias not found")
		}
	})
}
