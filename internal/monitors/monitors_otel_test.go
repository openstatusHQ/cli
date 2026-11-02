package monitors_test

import (
	"testing"

	"github.com/openstatusHQ/cli/internal/config"
	"github.com/openstatusHQ/cli/internal/monitors"
)

func Test_ConfigToMonitor_OpenTelemetry(t *testing.T) {
	t.Parallel()

	otel := config.OpenTelemetryConfig{
		Endpoint: "https://otel.example.com/v1/metrics",
		Headers:  map[string]string{"b-key": "2", "a-key": "1"},
	}

	t.Run("HTTP monitor forwards endpoint and sorted headers", func(t *testing.T) {
		m := monitors.ConfigToHTTPMonitor(config.Monitor{
			Name:          "http",
			Request:       config.Request{URL: "https://example.com"},
			OpenTelemetry: otel,
		})
		ot := m.GetOpenTelemetry()
		if ot.GetEndpoint() != otel.Endpoint {
			t.Errorf("Expected endpoint %q, got %q", otel.Endpoint, ot.GetEndpoint())
		}
		headers := ot.GetHeaders()
		if len(headers) != 2 || headers[0].GetKey() != "a-key" || headers[1].GetValue() != "2" {
			t.Errorf("Unexpected headers: %v", headers)
		}
	})

	t.Run("TCP monitor forwards endpoint", func(t *testing.T) {
		m, err := monitors.ConfigToTCPMonitor(config.Monitor{
			Name:          "tcp",
			Request:       config.Request{Host: "example.com", Port: 443},
			OpenTelemetry: otel,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if m.GetOpenTelemetry().GetEndpoint() != otel.Endpoint {
			t.Errorf("Expected OTel endpoint on TCP monitor, got %v", m.GetOpenTelemetry())
		}
	})

	t.Run("No config sends no OTel block", func(t *testing.T) {
		m := monitors.ConfigToHTTPMonitor(config.Monitor{Name: "plain", Request: config.Request{URL: "https://example.com"}})
		if m.GetOpenTelemetry() != nil {
			t.Errorf("Expected nil OpenTelemetry, got %v", m.GetOpenTelemetry())
		}
	})
}
