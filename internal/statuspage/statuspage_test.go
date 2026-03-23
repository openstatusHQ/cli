package statuspage_test

import (
	"net/http"
	"testing"

	"github.com/openstatusHQ/cli/internal/statuspage"
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

func Test_StatusPageCmd(t *testing.T) {
	t.Parallel()

	t.Run("Returns valid command", func(t *testing.T) {
		cmd := statuspage.StatusPageCmd()

		if cmd == nil {
			t.Fatal("Expected non-nil command")
		}

		if cmd.Name != "status-page" {
			t.Errorf("Expected command name 'status-page', got %s", cmd.Name)
		}

		if cmd.Usage != "Manage status pages" {
			t.Errorf("Expected usage 'Manage status pages', got %s", cmd.Usage)
		}
	})

	t.Run("Has expected subcommands", func(t *testing.T) {
		cmd := statuspage.StatusPageCmd()

		if len(cmd.Commands) != 2 {
			t.Errorf("Expected 2 subcommands, got %d", len(cmd.Commands))
		}

		expectedSubcommands := map[string]bool{
			"list": false,
			"info": false,
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

	t.Run("Has sp alias", func(t *testing.T) {
		cmd := statuspage.StatusPageCmd()

		found := false
		for _, alias := range cmd.Aliases {
			if alias == "sp" {
				found = true
				break
			}
		}
		if !found {
			t.Error("Expected 'sp' alias not found")
		}
	})
}
