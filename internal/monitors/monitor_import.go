package monitors

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"

	monitorv1 "buf.build/gen/go/openstatus/api/protocolbuffers/go/openstatus/monitor/v1"
	"buf.build/gen/go/openstatus/api/connectrpc/gosimple/openstatus/monitor/v1/monitorv1connect"
	"github.com/openstatusHQ/cli/internal/config"
	"github.com/urfave/cli/v3"
	"sigs.k8s.io/yaml"
)

// ExportMonitor exports all monitors to a YAML file using the SDK
func ExportMonitor(client monitorv1connect.MonitorServiceClient, path string) error {
	resp, err := client.ListMonitors(context.Background(), &monitorv1.ListMonitorsRequest{})
	if err != nil {
		return fmt.Errorf("failed to list monitors: %w", err)
	}

	t := map[string]config.Monitor{}
	lock := make(map[string]config.Lock)

	file, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return err
	}
	defer file.Close()

	// Process HTTP monitors
	for _, monitor := range resp.GetHttpMonitors() {
		configMonitor := convertHTTPMonitorToConfig(monitor)
		id := monitor.GetId()
		t[id] = configMonitor
	}

	// Process TCP monitors
	for _, monitor := range resp.GetTcpMonitors() {
		configMonitor := convertTCPMonitorToConfig(monitor)
		id := monitor.GetId()
		t[id] = configMonitor
	}

	// Process DNS monitors (skip for now as config.Monitor doesn't support DNS)
	// DNS monitors would need config.Kind = "dns" support

	y, err := yaml.Marshal(&t)
	if err != nil {
		return err
	}

	_, err = file.WriteString("# yaml-language-server: $schema=https://www.openstatus.dev/schema.json\n\n")
	if err != nil {
		return err
	}
	_, err = file.Write(y)
	if err != nil {
		return err
	}

	// Build lock file
	for id, monitor := range t {
		i, _ := strconv.Atoi(id)
		lock[id] = config.Lock{
			ID:      i,
			Monitor: monitor,
		}
	}

	lockFile, err := os.OpenFile("openstatus.lock", os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return cli.Exit("Failed to apply change", 1)
	}
	defer lockFile.Close()

	y, err = yaml.Marshal(&lock)
	if err != nil {
		return cli.Exit("Failed to apply change", 1)
	}

	_, err = lockFile.Write(y)
	if err != nil {
		return cli.Exit("Failed to apply change", 1)
	}

	return nil
}

// convertHTTPMonitorToConfig converts an SDK HTTPMonitor to config.Monitor
func convertHTTPMonitorToConfig(m *monitorv1.HTTPMonitor) config.Monitor {
	regions := make([]config.Region, len(m.GetRegions()))
	for i, r := range m.GetRegions() {
		regions[i] = config.Region(regionToString(r))
	}

	headers := make(map[string]string)
	for _, h := range m.GetHeaders() {
		if h.GetKey() != "" {
			headers[h.GetKey()] = h.GetValue()
		}
	}

	var assertions []config.Assertion
	for _, a := range m.GetStatusCodeAssertions() {
		assertions = append(assertions, config.Assertion{
			Kind:    config.StatusCode,
			Target:  int(a.GetTarget()),
			Compare: convertNumberComparator(a.GetComparator()),
		})
	}
	for _, a := range m.GetBodyAssertions() {
		assertions = append(assertions, config.Assertion{
			Kind:    config.TextBody,
			Target:  a.GetTarget(),
			Compare: convertStringComparator(a.GetComparator()),
		})
	}
	for _, a := range m.GetHeaderAssertions() {
		assertions = append(assertions, config.Assertion{
			Kind:    config.Header,
			Target:  a.GetTarget(),
			Compare: convertStringComparator(a.GetComparator()),
			Key:     a.GetKey(),
		})
	}

	return config.Monitor{
		Name:          m.GetName(),
		Description:   m.GetDescription(),
		Active:        m.GetActive(),
		Public:        m.GetPublic(),
		Frequency:     convertPeriodicity(m.GetPeriodicity()),
		DegradedAfter: m.GetDegradedAt(),
		Timeout:       m.GetTimeout(),
		Retry:         m.GetRetry(),
		Kind:          config.HTTP,
		Regions:       regions,
		Assertions:    assertions,
		Request: config.Request{
			URL:     m.GetUrl(),
			Method:  convertHTTPMethod(m.GetMethod()),
			Body:    m.GetBody(),
			Headers: headers,
		},
	}
}

// convertTCPMonitorToConfig converts an SDK TCPMonitor to config.Monitor
func convertTCPMonitorToConfig(m *monitorv1.TCPMonitor) config.Monitor {
	regions := make([]config.Region, len(m.GetRegions()))
	for i, r := range m.GetRegions() {
		regions[i] = config.Region(regionToString(r))
	}

	// Parse host:port from URI
	uri := m.GetUri()
	parts := strings.Split(uri, ":")
	host := parts[0]
	var port int64
	if len(parts) > 1 {
		p, _ := strconv.Atoi(parts[1])
		port = int64(p)
	}

	return config.Monitor{
		Name:          m.GetName(),
		Description:   m.GetDescription(),
		Active:        m.GetActive(),
		Public:        m.GetPublic(),
		Frequency:     convertPeriodicity(m.GetPeriodicity()),
		DegradedAfter: m.GetDegradedAt(),
		Timeout:       m.GetTimeout(),
		Retry:         m.GetRetry(),
		Kind:          config.TCP,
		Regions:       regions,
		Request: config.Request{
			Host: host,
			Port: port,
		},
	}
}

// convertPeriodicity converts SDK Periodicity enum to config.Frequency string
func convertPeriodicity(p monitorv1.Periodicity) config.Frequency {
	switch p {
	case monitorv1.Periodicity_PERIODICITY_30S:
		return config.The30S
	case monitorv1.Periodicity_PERIODICITY_1M:
		return config.The1M
	case monitorv1.Periodicity_PERIODICITY_5M:
		return config.The5M
	case monitorv1.Periodicity_PERIODICITY_10M:
		return config.The10M
	case monitorv1.Periodicity_PERIODICITY_30M:
		return config.The30M
	case monitorv1.Periodicity_PERIODICITY_1H:
		return config.The1H
	default:
		return config.The10M
	}
}

// convertHTTPMethod converts SDK HTTPMethod enum to config.Method string
func convertHTTPMethod(m monitorv1.HTTPMethod) config.Method {
	switch m {
	case monitorv1.HTTPMethod_HTTP_METHOD_GET:
		return config.Get
	case monitorv1.HTTPMethod_HTTP_METHOD_POST:
		return config.Post
	case monitorv1.HTTPMethod_HTTP_METHOD_PUT:
		return config.Put
	case monitorv1.HTTPMethod_HTTP_METHOD_PATCH:
		return config.Patch
	case monitorv1.HTTPMethod_HTTP_METHOD_DELETE:
		return config.Delete
	case monitorv1.HTTPMethod_HTTP_METHOD_HEAD:
		return config.Head
	case monitorv1.HTTPMethod_HTTP_METHOD_OPTIONS:
		return config.Options
	default:
		return config.Get
	}
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

// ExportMonitorWithHTTPClient is a convenience function that creates a client and exports monitors
func ExportMonitorWithHTTPClient(httpClient *http.Client, apiKey string, path string) error {
	client := NewMonitorClientWithHTTPClient(httpClient, apiKey)
	return ExportMonitor(client, path)
}

func GetMonitorImportCmd() *cli.Command {
	monitorInfoCmd := cli.Command{
		Name:        "import",
		Usage:       "Import all your monitors",
		UsageText:   "openstatus monitors import [options]",
		Description: "Import all your monitors from your workspace to a YAML file; it will also create a lock file to manage your monitors with 'apply'.",
		Action: func(ctx context.Context, cmd *cli.Command) error {
			client := NewMonitorClient(cmd.String("access-token"))
			err := ExportMonitor(client, cmd.String("output"))
			if err != nil {
				return cli.Exit(err.Error(), 1)
			}
			fmt.Printf("Monitors successfully imported to: %s", cmd.String("output"))
			return nil
		},
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "access-token",
				Usage:    "OpenStatus API Access Token",
				Aliases:  []string{"t"},
				Sources:  cli.EnvVars("OPENSTATUS_API_TOKEN"),
				Required: true,
			},
			&cli.StringFlag{
				Name:        "output",
				Usage:       "The output file name ",
				DefaultText: "openstatus.yaml",
				Value:       "openstatus.yaml",
				Aliases:     []string{"o"},
			},
		},
	}
	return &monitorInfoCmd
}
