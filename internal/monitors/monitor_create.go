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
	confirmation "github.com/openstatusHQ/cli/internal/cli"
	"github.com/openstatusHQ/cli/internal/config"
	"github.com/urfave/cli/v3"
)

// CreateMonitor creates a monitor using the SDK, dispatching to the appropriate type
func CreateMonitor(httpClient *http.Client, apiKey string, monitor config.Monitor) (Monitor, error) {
	client := NewMonitorClientWithHTTPClient(httpClient, apiKey)

	switch monitor.Kind {
	case config.HTTP:
		return CreateHTTPMonitor(client, monitor)
	case config.TCP:
		return CreateTCPMonitor(client, monitor)
	default:
		return Monitor{}, fmt.Errorf("unsupported monitor kind: %s", monitor.Kind)
	}
}

// CreateHTTPMonitor creates an HTTP monitor using the SDK
func CreateHTTPMonitor(client monitorv1connect.MonitorServiceClient, monitor config.Monitor) (Monitor, error) {
	req := &monitorv1.CreateHTTPMonitorRequest{
		Monitor: configToHTTPMonitor(monitor),
	}

	resp, err := client.CreateHTTPMonitor(context.Background(), req)
	if err != nil {
		return Monitor{}, fmt.Errorf("failed to create HTTP monitor: %w", err)
	}

	return httpMonitorToLocal(resp.GetMonitor()), nil
}

// CreateTCPMonitor creates a TCP monitor using the SDK
func CreateTCPMonitor(client monitorv1connect.MonitorServiceClient, monitor config.Monitor) (Monitor, error) {
	req := &monitorv1.CreateTCPMonitorRequest{
		Monitor: configToTCPMonitor(monitor),
	}

	resp, err := client.CreateTCPMonitor(context.Background(), req)
	if err != nil {
		return Monitor{}, fmt.Errorf("failed to create TCP monitor: %w", err)
	}

	return tcpMonitorToLocal(resp.GetMonitor()), nil
}

// httpMonitorToLocal converts SDK HTTPMonitor to local Monitor type
func httpMonitorToLocal(m *monitorv1.HTTPMonitor) Monitor {
	id, _ := strconv.Atoi(m.GetId())
	return Monitor{
		ID:          id,
		Name:        m.GetName(),
		Description: m.GetDescription(),
		URL:         m.GetUrl(),
		Periodicity: periodicityToString(m.GetPeriodicity()),
		Method:      httpMethodToString(m.GetMethod()),
		Regions:     regionsToStrings(m.GetRegions()),
		Active:      m.GetActive(),
		Public:      m.GetPublic(),
		Timeout:     int(m.GetTimeout()),
		Retry:       int(m.GetRetry()),
		JobType:     "http",
	}
}

// tcpMonitorToLocal converts SDK TCPMonitor to local Monitor type
func tcpMonitorToLocal(m *monitorv1.TCPMonitor) Monitor {
	id, _ := strconv.Atoi(m.GetId())
	return Monitor{
		ID:          id,
		Name:        m.GetName(),
		Description: m.GetDescription(),
		URL:         m.GetUri(),
		Periodicity: periodicityToString(m.GetPeriodicity()),
		Regions:     regionsToStrings(m.GetRegions()),
		Active:      m.GetActive(),
		Public:      m.GetPublic(),
		Timeout:     int(m.GetTimeout()),
		Retry:       int(m.GetRetry()),
		JobType:     "tcp",
	}
}

func GetMonitorCreateCmd() *cli.Command {
	monitorInfoCmd := cli.Command{
		Name:            "create",
		Usage:           "Create monitors (beta)",
		Hidden:          true,
		HideHelp:        true,
		HideHelpCommand: true,
		Description:     "Create the monitors defined in the openstatus.yaml file",
		UsageText:       "openstatus monitors create [options]",

		Action: func(ctx context.Context, cmd *cli.Command) error {

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
				confirmed, err := confirmation.AskForConfirmation(fmt.Sprintf("You are about to create %d monitors do you want to continue", len(monitors)))
				if err != nil {
					return cli.Exit(fmt.Sprintf("Failed to read input: %v", err), 1)
				}
				if !confirmed {
					return nil
				}
			}
			for _, value := range monitors {
				_, err = CreateMonitor(http.DefaultClient, cmd.String("access-token"), value)
				if err != nil {
					return cli.Exit("Unable to create monitor", 1)
				}
			}
			fmt.Printf("%d monitors created successfully\n", len(monitors))
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
				Name:     "access-token",
				Usage:    "OpenStatus API Access Token",
				Aliases:  []string{"t"},
				Sources:  cli.EnvVars("OPENSTATUS_API_TOKEN"),
				Required: true,
			},
			&cli.BoolFlag{
				Name:     "auto-accept",
				Usage:    "Automatically accept the prompt",
				Aliases:  []string{"y"},
				Required: false,
			},
		},
	}
	return &monitorInfoCmd
}
