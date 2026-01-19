package monitors_test

import (
	"net/http"
	"testing"

	"github.com/openstatusHQ/cli/internal/monitors"
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

func Test_MonitorsCmd(t *testing.T) {
	t.Parallel()

	t.Run("Returns valid command", func(t *testing.T) {
		cmd := monitors.MonitorsCmd()

		if cmd == nil {
			t.Fatal("Expected non-nil command")
		}

		if cmd.Name != "monitors" {
			t.Errorf("Expected command name 'monitors', got %s", cmd.Name)
		}

		if cmd.Usage != "Manage your monitors" {
			t.Errorf("Expected usage 'Manage your monitors', got %s", cmd.Usage)
		}
	})

	t.Run("Has expected subcommands", func(t *testing.T) {
		cmd := monitors.MonitorsCmd()

		if len(cmd.Commands) != 7 {
			t.Errorf("Expected 7 subcommands, got %d", len(cmd.Commands))
		}

		expectedSubcommands := map[string]bool{
			"apply":   false,
			"create":  false,
			"delete":  false,
			"import":  false,
			"info":    false,
			"list":    false,
			"trigger": false,
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
}
