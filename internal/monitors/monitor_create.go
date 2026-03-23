package monitors

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strconv"

	monitorv1 "buf.build/gen/go/openstatus/api/protocolbuffers/go/openstatus/monitor/v1"
	"buf.build/gen/go/openstatus/api/connectrpc/gosimple/openstatus/monitor/v1/monitorv1connect"
	"github.com/openstatusHQ/cli/internal/api"
	"github.com/openstatusHQ/cli/internal/auth"
	output "github.com/openstatusHQ/cli/internal/cli"
	"github.com/openstatusHQ/cli/internal/config"
	"github.com/urfave/cli/v3"
)

// CreateMonitor creates a monitor using the SDK, dispatching to the appropriate type
func CreateMonitor(ctx context.Context, httpClient *http.Client, apiKey string, monitor config.Monitor) (Monitor, error) {
	client := NewMonitorClientWithHTTPClient(httpClient, apiKey)

	switch monitor.Kind {
	case config.HTTP:
		return CreateHTTPMonitor(ctx, client, monitor)
	case config.TCP:
		return CreateTCPMonitor(ctx, client, monitor)
	default:
		return Monitor{}, fmt.Errorf("unsupported monitor kind: %s", monitor.Kind)
	}
}

// CreateHTTPMonitor creates an HTTP monitor using the SDK
func CreateHTTPMonitor(ctx context.Context, client monitorv1connect.MonitorServiceClient, monitor config.Monitor) (Monitor, error) {
	req := &monitorv1.CreateHTTPMonitorRequest{
		Monitor: configToHTTPMonitor(monitor),
	}

	resp, err := client.CreateHTTPMonitor(ctx, req)
	if err != nil {
		return Monitor{}, fmt.Errorf("failed to create HTTP monitor: %w", err)
	}

	return httpMonitorToLocal(resp.GetMonitor())
}

// CreateTCPMonitor creates a TCP monitor using the SDK
func CreateTCPMonitor(ctx context.Context, client monitorv1connect.MonitorServiceClient, monitor config.Monitor) (Monitor, error) {
	tcpMonitor, err := configToTCPMonitor(monitor)
	if err != nil {
		return Monitor{}, err
	}
	req := &monitorv1.CreateTCPMonitorRequest{
		Monitor: tcpMonitor,
	}

	resp, err := client.CreateTCPMonitor(ctx, req)
	if err != nil {
		return Monitor{}, fmt.Errorf("failed to create TCP monitor: %w", err)
	}

	return tcpMonitorToLocal(resp.GetMonitor())
}

func httpMonitorToLocal(m *monitorv1.HTTPMonitor) (Monitor, error) {
	id, err := strconv.Atoi(m.GetId())
	if err != nil {
		return Monitor{}, fmt.Errorf("invalid monitor ID %q: %w", m.GetId(), err)
	}

	var headers []Header
	for _, h := range m.GetHeaders() {
		headers = append(headers, Header{Key: h.GetKey(), Value: h.GetValue()})
	}

	var assertions []Assertion
	for _, a := range m.GetStatusCodeAssertions() {
		assertions = append(assertions, Assertion{
			Type:    "status_code",
			Compare: string(convertNumberComparator(a.GetComparator())),
			Target:  int(a.GetTarget()),
		})
	}
	for _, a := range m.GetBodyAssertions() {
		assertions = append(assertions, Assertion{
			Type:    "text_body",
			Compare: string(convertStringComparator(a.GetComparator())),
			Target:  a.GetTarget(),
		})
	}
	for _, a := range m.GetHeaderAssertions() {
		assertions = append(assertions, Assertion{
			Type:    "header",
			Compare: string(convertStringComparator(a.GetComparator())),
			Target:  a.GetTarget(),
			Key:     a.GetKey(),
		})
	}

	return Monitor{
		ID:            id,
		Name:          m.GetName(),
		Description:   m.GetDescription(),
		URL:           m.GetUrl(),
		Periodicity:   periodicityToString(m.GetPeriodicity()),
		Method:        httpMethodToString(m.GetMethod()),
		Regions:       regionsToStrings(m.GetRegions()),
		Active:        m.GetActive(),
		Public:        m.GetPublic(),
		Timeout:       int(m.GetTimeout()),
		DegradedAfter: int(m.GetDegradedAt()),
		Body:          m.GetBody(),
		Headers:       headers,
		Assertions:    assertions,
		Retry:         int(m.GetRetry()),
		JobType:       "http",
	}, nil
}

func tcpMonitorToLocal(m *monitorv1.TCPMonitor) (Monitor, error) {
	id, err := strconv.Atoi(m.GetId())
	if err != nil {
		return Monitor{}, fmt.Errorf("invalid monitor ID %q: %w", m.GetId(), err)
	}
	return Monitor{
		ID:            id,
		Name:          m.GetName(),
		Description:   m.GetDescription(),
		URL:           m.GetUri(),
		Periodicity:   periodicityToString(m.GetPeriodicity()),
		Regions:       regionsToStrings(m.GetRegions()),
		Active:        m.GetActive(),
		Public:        m.GetPublic(),
		Timeout:       int(m.GetTimeout()),
		DegradedAfter: int(m.GetDegradedAt()),
		Retry:         int(m.GetRetry()),
		JobType:       "tcp",
	}, nil
}

func GetMonitorCreateCmd() *cli.Command {
	monitorCreateCmd := cli.Command{
		Name:            "create",
		Usage:           "Create monitors",
		Hidden:          true,
		HideHelp:        true,
		HideHelpCommand: true,
		Description: "Create the monitors defined in the openstatus.yaml file",
		UsageText: `openstatus monitors create
  openstatus monitors create --config custom.yaml -y`,

		Action: func(ctx context.Context, cmd *cli.Command) error {
			apiKey, err := auth.ResolveAccessToken(cmd)
			if err != nil {
				return cli.Exit(err.Error(), 1)
			}

			path := cmd.String("config")

			if path != "" {
				if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
					return cli.Exit("Config does not exist", 1)
				}
			}

			accept := cmd.Bool("auto-accept")

			monitors, err := config.ReadOpenStatus(path)
			if err != nil {
				return cli.Exit("Unable to read config file", 1)
			}

			if !accept {
				confirmed, err := output.AskForConfirmation(fmt.Sprintf("You are about to create %d monitors do you want to continue", len(monitors)))
				if err != nil {
					return cli.Exit(fmt.Sprintf("Failed to read input: %v", err), 1)
				}
				if !confirmed {
					return nil
				}
			}
			s := output.StartSpinner("Creating monitors...")
			for _, value := range monitors {
				_, err = CreateMonitor(ctx, api.DefaultHTTPClient, apiKey, value)
				if err != nil {
					output.StopSpinner(s)
					return cli.Exit("Unable to create monitor", 1)
				}
			}
			output.StopSpinner(s)
			fmt.Printf("%d monitors created successfully\n", len(monitors))
			fmt.Println("Run 'openstatus monitors list' to see all monitors")
			return nil
		},
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        "config",
				Usage:       "The configuration file containing monitor information",
				Aliases:     []string{"c"},
				DefaultText: "openstatus.yaml",
				Value:       "openstatus.yaml",
			},
			&cli.StringFlag{
				Name:    "access-token",
				Usage:   "OpenStatus API Access Token",
				Aliases: []string{"t"},
				Sources: cli.EnvVars("OPENSTATUS_API_TOKEN"),
			},
			&cli.BoolFlag{
				Name:     "auto-accept",
				Usage:    "Automatically accept the prompt",
				Aliases:  []string{"y"},
				Required: false,
			},
		},
	}
	return &monitorCreateCmd
}
