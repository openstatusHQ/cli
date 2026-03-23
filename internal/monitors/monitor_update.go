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
func UpdateMonitor(ctx context.Context, httpClient *http.Client, apiKey string, id int, monitor config.Monitor) (Monitor, error) {
	client := NewMonitorClientWithHTTPClient(httpClient, apiKey)

	switch monitor.Kind {
	case config.HTTP:
		return UpdateHTTPMonitor(ctx, client, id, monitor)
	case config.TCP:
		return UpdateTCPMonitor(ctx, client, id, monitor)
	default:
		return Monitor{}, fmt.Errorf("unsupported monitor kind: %s", monitor.Kind)
	}
}

// UpdateHTTPMonitor updates an HTTP monitor using the SDK
func UpdateHTTPMonitor(ctx context.Context, client monitorv1connect.MonitorServiceClient, id int, monitor config.Monitor) (Monitor, error) {
	httpMonitor := configToHTTPMonitor(monitor)
	httpMonitor.Id = strconv.Itoa(id)

	req := &monitorv1.UpdateHTTPMonitorRequest{
		Id:      strconv.Itoa(id),
		Monitor: httpMonitor,
	}

	resp, err := client.UpdateHTTPMonitor(ctx, req)
	if err != nil {
		return Monitor{}, fmt.Errorf("failed to update HTTP monitor: %w", err)
	}

	return httpMonitorToLocal(resp.GetMonitor())
}

// UpdateTCPMonitor updates a TCP monitor using the SDK
func UpdateTCPMonitor(ctx context.Context, client monitorv1connect.MonitorServiceClient, id int, monitor config.Monitor) (Monitor, error) {
	tcpMonitor, err := configToTCPMonitor(monitor)
	if err != nil {
		return Monitor{}, err
	}
	tcpMonitor.Id = strconv.Itoa(id)

	req := &monitorv1.UpdateTCPMonitorRequest{
		Id:      strconv.Itoa(id),
		Monitor: tcpMonitor,
	}

	resp, err := client.UpdateTCPMonitor(ctx, req)
	if err != nil {
		return Monitor{}, fmt.Errorf("failed to update TCP monitor: %w", err)
	}

	return tcpMonitorToLocal(resp.GetMonitor())
}
