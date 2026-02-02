package monitors

import (
	"context"
	"fmt"
	"net/http"
	"strconv"

	monitorv1 "buf.build/gen/go/openstatus/api/protocolbuffers/go/openstatus/monitor/v1"
	"buf.build/gen/go/openstatus/api/connectrpc/gosimple/openstatus/monitor/v1/monitorv1connect"
	"github.com/openstatusHQ/cli/internal/config"
)

// UpdateMonitor updates a monitor using the SDK, dispatching to the appropriate type
func UpdateMonitor(httpClient *http.Client, apiKey string, id int, monitor config.Monitor) (Monitor, error) {
	client := NewMonitorClientWithHTTPClient(httpClient, apiKey)

	switch monitor.Kind {
	case config.HTTP:
		return UpdateHTTPMonitor(client, id, monitor)
	case config.TCP:
		return UpdateTCPMonitor(client, id, monitor)
	default:
		return Monitor{}, fmt.Errorf("unsupported monitor kind: %s", monitor.Kind)
	}
}

// UpdateHTTPMonitor updates an HTTP monitor using the SDK
func UpdateHTTPMonitor(client monitorv1connect.MonitorServiceClient, id int, monitor config.Monitor) (Monitor, error) {
	httpMonitor := configToHTTPMonitor(monitor)
	httpMonitor.Id = strconv.Itoa(id)

	req := &monitorv1.UpdateHTTPMonitorRequest{
		Id:      strconv.Itoa(id),
		Monitor: httpMonitor,
	}

	resp, err := client.UpdateHTTPMonitor(context.Background(), req)
	if err != nil {
		return Monitor{}, fmt.Errorf("failed to update HTTP monitor: %w", err)
	}

	return httpMonitorToLocal(resp.GetMonitor()), nil
}

// UpdateTCPMonitor updates a TCP monitor using the SDK
func UpdateTCPMonitor(client monitorv1connect.MonitorServiceClient, id int, monitor config.Monitor) (Monitor, error) {
	tcpMonitor := configToTCPMonitor(monitor)
	tcpMonitor.Id = strconv.Itoa(id)

	req := &monitorv1.UpdateTCPMonitorRequest{
		Id:      strconv.Itoa(id),
		Monitor: tcpMonitor,
	}

	resp, err := client.UpdateTCPMonitor(context.Background(), req)
	if err != nil {
		return Monitor{}, fmt.Errorf("failed to update TCP monitor: %w", err)
	}

	return tcpMonitorToLocal(resp.GetMonitor()), nil
}
