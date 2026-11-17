package monitors

import (
	"context"
	"fmt"
	"net/http"
	"os"

	"buf.build/gen/go/openstatus/api/connectrpc/gosimple/openstatus/monitor/v1/monitorv1connect"
	monitorv1 "buf.build/gen/go/openstatus/api/protocolbuffers/go/openstatus/monitor/v1"
	"github.com/urfave/cli/v3"

	"github.com/openstatusHQ/cli/internal/auth"
	output "github.com/openstatusHQ/cli/internal/cli"
)

type regenerateTokenOutput struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	PingURL string `json:"ping_url"`
}

// ConfirmFunc asks the user to approve an action. A nil ConfirmFunc skips the prompt.
type ConfirmFunc func(prompt string) (bool, error)

func RegenerateHeartbeatToken(ctx context.Context, client monitorv1connect.MonitorServiceClient, monitorId string, confirm ConfirmFunc, s *output.Spinner) error {
	if monitorId == "" {
		output.StopSpinner(s)
		fmt.Fprintln(os.Stderr, "Usage: openstatus monitors regenerate-token <monitor-id>")
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "Example: openstatus monitors regenerate-token 12345")
		return output.UsageErrorf("monitor ID is required")
	}

	resp, err := client.GetMonitor(ctx, &monitorv1.GetMonitorRequest{Id: monitorId})
	output.StopSpinner(s)
	if err != nil {
		return output.FormatError(err, "monitor", monitorId)
	}
	if !resp.GetMonitor().HasHeartbeat() {
		return heartbeatTypeMismatchError(monitorId, monitorKind(resp.GetMonitor()), "only heartbeat monitors have a ping URL")
	}

	if confirm != nil {
		name := resp.GetMonitor().GetHeartbeat().GetName()
		confirmed, err := confirm(fmt.Sprintf("Regenerate the ping URL of '%s' (%s)? The current URL stops working immediately", name, monitorId))
		if err != nil {
			return fmt.Errorf("failed to read input: %w", err)
		}
		if !confirmed {
			return nil
		}
	}

	s = output.StartSpinner("Regenerating ping URL...")
	regenerated, err := client.RegenerateHeartbeatToken(ctx, &monitorv1.RegenerateHeartbeatTokenRequest{Id: monitorId})
	output.StopSpinner(s)
	if err != nil {
		return output.FormatError(err, "monitor", monitorId)
	}

	m := regenerated.GetMonitor()
	if m.GetPingUrl() == "" {
		return output.NewError(output.KindError, "the API did not return a new ping URL for monitor %s. Run 'openstatus monitors info %s --show-ping-url' to check the current one", monitorId, monitorId)
	}
	return renderRegeneratedToken(m)
}

func renderRegeneratedToken(m *monitorv1.HeartbeatMonitor) error {
	if output.IsJSONOutput() {
		return output.PrintJSON(regenerateTokenOutput{
			ID:      m.GetId(),
			Name:    m.GetName(),
			PingURL: m.GetPingUrl(),
		})
	}

	if output.IsQuiet() {
		fmt.Println(m.GetPingUrl())
		return nil
	}

	fmt.Println(output.Bold("Ping URL regenerated:"))
	tbl := newBlueprintTable()
	tbl.Bulk([][]string{
		{"ID", m.GetId()},
		{"Name", m.GetName()},
		{"Ping URL", m.GetPingUrl()},
	})
	tbl.Render()

	fmt.Println("\nThe previous URL no longer works. Update your job to use the new one.")
	return nil
}

func RegenerateHeartbeatTokenWithHTTPClient(ctx context.Context, httpClient *http.Client, apiKey string, monitorId string, confirm ConfirmFunc) error {
	client := NewMonitorClientWithHTTPClient(httpClient, apiKey)
	return RegenerateHeartbeatToken(ctx, client, monitorId, confirm, nil)
}

func GetMonitorRegenerateTokenCmd() *cli.Command {
	return &cli.Command{
		Name:  "regenerate-token",
		Usage: "Replace the ping URL of a heartbeat monitor",
		UsageText: `openstatus monitors regenerate-token <MonitorID>
  openstatus monitors regenerate-token 12345
  openstatus monitors regenerate-token 12345 -y
  openstatus monitors regenerate-token 12345 -y -q | pbcopy`,
		Description: "Replace the ping URL of a heartbeat monitor, for example after it leaked. The current URL stops working immediately, so update your job with the new one.",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "access-token",
				Usage:   "OpenStatus API Access Token",
				Aliases: []string{"t"},
				Sources: cli.EnvVars("OPENSTATUS_API_TOKEN"),
			},
			&cli.BoolFlag{
				Name:    "auto-accept",
				Usage:   "Automatically accept the prompt",
				Aliases: []string{"y"},
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			apiKey, err := auth.ResolveAccessToken(ctx, cmd)
			if err != nil {
				return err
			}
			var confirm ConfirmFunc = output.AskForConfirmation
			if cmd.Bool("auto-accept") {
				confirm = nil
			}
			s := output.StartSpinner("Fetching monitor...")
			return RegenerateHeartbeatToken(ctx, NewMonitorClient(apiKey), cmd.Args().Get(0), confirm, s)
		},
	}
}
