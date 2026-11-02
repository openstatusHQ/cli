package monitors

import (
	"context"
	"fmt"
	"net/http"
	"strconv"

	"buf.build/gen/go/openstatus/api/connectrpc/gosimple/openstatus/monitor/v1/monitorv1connect"
	monitorv1 "buf.build/gen/go/openstatus/api/protocolbuffers/go/openstatus/monitor/v1"

	output "github.com/openstatusHQ/cli/internal/cli"
	"github.com/openstatusHQ/cli/internal/config"
)

// CreateMonitor creates a monitor using the SDK, dispatching to the appropriate type
func CreateMonitor(ctx context.Context, httpClient *http.Client, apiKey string, monitor config.Monitor) (Monitor, error) {
	client := NewMonitorClientWithHTTPClient(httpClient, apiKey)

	switch monitor.Kind {
	case config.HTTP:
		return CreateHTTPMonitor(ctx, client, monitor)
	case config.TCP:
		return CreateTCPMonitor(ctx, client, monitor)
	case config.ICMP:
		return CreateICMPMonitor(ctx, client, monitor)
	default:
		return Monitor{}, output.UsageErrorf("unsupported monitor kind: %s", monitor.Kind)
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

// CreateICMPMonitor creates an ICMP monitor using the SDK
func CreateICMPMonitor(ctx context.Context, client monitorv1connect.MonitorServiceClient, monitor config.Monitor) (Monitor, error) {
	icmpMonitor, err := configToICMPMonitor(monitor)
	if err != nil {
		return Monitor{}, err
	}
	req := &monitorv1.CreateICMPMonitorRequest{
		Monitor: icmpMonitor,
	}

	resp, err := client.CreateICMPMonitor(ctx, req)
	if err != nil {
		return Monitor{}, fmt.Errorf("failed to create ICMP monitor: %w", err)
	}

	return icmpMonitorToLocal(resp.GetMonitor())
}

func httpMonitorToLocal(m *monitorv1.HTTPMonitor) (Monitor, error) {
	id, err := strconv.Atoi(m.GetId())
	if err != nil {
		return Monitor{}, output.UsageErrorf("invalid monitor ID %q: %w", m.GetId(), err)
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
		ID:                 id,
		Name:               m.GetName(),
		Description:        m.GetDescription(),
		URL:                m.GetUrl(),
		Periodicity:        periodicityToString(m.GetPeriodicity()),
		Method:             httpMethodToString(m.GetMethod()),
		Regions:            regionsToStrings(m.GetRegions()),
		PrivateLocationIDs: m.GetPrivateLocationIds(),
		Active:             m.GetActive(),
		Public:             m.GetPublic(),
		Timeout:            int(m.GetTimeout()),
		DegradedAfter:      int(m.GetDegradedAt()),
		Body:               m.GetBody(),
		Headers:            headers,
		Assertions:         assertions,
		Retry:              int(m.GetRetry()),
		JobType:            "http",
	}, nil
}

func tcpMonitorToLocal(m *monitorv1.TCPMonitor) (Monitor, error) {
	id, err := strconv.Atoi(m.GetId())
	if err != nil {
		return Monitor{}, output.UsageErrorf("invalid monitor ID %q: %w", m.GetId(), err)
	}
	return Monitor{
		ID:                 id,
		Name:               m.GetName(),
		Description:        m.GetDescription(),
		URL:                m.GetUri(),
		Periodicity:        periodicityToString(m.GetPeriodicity()),
		Regions:            regionsToStrings(m.GetRegions()),
		PrivateLocationIDs: m.GetPrivateLocationIds(),
		Active:             m.GetActive(),
		Public:             m.GetPublic(),
		Timeout:            int(m.GetTimeout()),
		DegradedAfter:      int(m.GetDegradedAt()),
		Retry:              int(m.GetRetry()),
		JobType:            "tcp",
	}, nil
}

func grpcMonitorToLocal(m *monitorv1.GRPCMonitor) (Monitor, error) {
	id, err := strconv.Atoi(m.GetId())
	if err != nil {
		return Monitor{}, output.UsageErrorf("invalid monitor ID %q: %w", m.GetId(), err)
	}
	return Monitor{
		ID:                 id,
		Name:               m.GetName(),
		Description:        m.GetDescription(),
		URL:                m.GetUri(),
		Periodicity:        periodicityToString(m.GetPeriodicity()),
		Regions:            regionsToStrings(m.GetRegions()),
		PrivateLocationIDs: m.GetPrivateLocationIds(),
		Active:             m.GetActive(),
		Public:             m.GetPublic(),
		Timeout:            int(m.GetTimeout()),
		DegradedAfter:      int(m.GetDegradedAt()),
		Retry:              int(m.GetRetry()),
		JobType:            "grpc",
	}, nil
}

// convertNumberComparator converts SDK NumberComparator to config.Compare
func convertNumberComparator(c monitorv1.NumberComparator) config.Compare {
	switch c {
	case monitorv1.NumberComparator_NUMBER_COMPARATOR_EQUAL:
		return config.Eq
	case monitorv1.NumberComparator_NUMBER_COMPARATOR_NOT_EQUAL:
		return config.NotEq
	case monitorv1.NumberComparator_NUMBER_COMPARATOR_GREATER_THAN:
		return config.Gt
	case monitorv1.NumberComparator_NUMBER_COMPARATOR_GREATER_THAN_OR_EQUAL:
		return config.Gte
	case monitorv1.NumberComparator_NUMBER_COMPARATOR_LESS_THAN:
		return config.Lt
	case monitorv1.NumberComparator_NUMBER_COMPARATOR_LESS_THAN_OR_EQUAL:
		return config.LTE
	default:
		return config.Eq
	}
}

// convertStringComparator converts SDK StringComparator to config.Compare
func convertStringComparator(c monitorv1.StringComparator) config.Compare {
	switch c {
	case monitorv1.StringComparator_STRING_COMPARATOR_EQUAL:
		return config.Eq
	case monitorv1.StringComparator_STRING_COMPARATOR_NOT_EQUAL:
		return config.NotEq
	case monitorv1.StringComparator_STRING_COMPARATOR_CONTAINS:
		return config.Contains
	case monitorv1.StringComparator_STRING_COMPARATOR_NOT_CONTAINS:
		return config.NotContains
	case monitorv1.StringComparator_STRING_COMPARATOR_EMPTY:
		return config.Empty
	case monitorv1.StringComparator_STRING_COMPARATOR_NOT_EMPTY:
		return config.NotEmpty
	case monitorv1.StringComparator_STRING_COMPARATOR_GREATER_THAN:
		return config.Gt
	case monitorv1.StringComparator_STRING_COMPARATOR_GREATER_THAN_OR_EQUAL:
		return config.Gte
	case monitorv1.StringComparator_STRING_COMPARATOR_LESS_THAN:
		return config.Lt
	case monitorv1.StringComparator_STRING_COMPARATOR_LESS_THAN_OR_EQUAL:
		return config.LTE
	default:
		return config.Eq
	}
}
