package statusreport_test

import (
	"net/http"
	"testing"

	"github.com/openstatusHQ/cli/internal/statusreport"
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

func Test_StatusReportCmd(t *testing.T) {
	t.Parallel()

	t.Run("Returns valid command", func(t *testing.T) {
		cmd := statusreport.StatusReportCmd()

		if cmd == nil {
			t.Fatal("Expected non-nil command")
		}

		if cmd.Name != "status-report" {
			t.Errorf("Expected command name 'status-report', got %s", cmd.Name)
		}

		if cmd.Usage != "Manage status reports" {
			t.Errorf("Expected usage 'Manage status reports', got %s", cmd.Usage)
		}
	})

	t.Run("Has expected subcommands", func(t *testing.T) {
		cmd := statusreport.StatusReportCmd()

		if len(cmd.Commands) != 6 {
			t.Errorf("Expected 6 subcommands, got %d", len(cmd.Commands))
		}

		expectedSubcommands := map[string]bool{
			"list":       false,
			"info":       false,
			"create":     false,
			"update":     false,
			"delete":     false,
			"add-update": false,
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

	t.Run("Has sr alias", func(t *testing.T) {
		cmd := statusreport.StatusReportCmd()

		found := false
		for _, alias := range cmd.Aliases {
			if alias == "sr" {
				found = true
				break
			}
		}
		if !found {
			t.Error("Expected 'sr' alias not found")
		}
	})
}
