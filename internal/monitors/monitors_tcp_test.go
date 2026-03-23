package monitors_test

import (
	"strings"
	"testing"

	"github.com/openstatusHQ/cli/internal/config"
	"github.com/openstatusHQ/cli/internal/monitors"
)

func Test_ConfigToTCPMonitor_ZeroPort(t *testing.T) {
	m := config.Monitor{
		Name:    "test",
		Kind:    config.TCP,
		Request: config.Request{Host: "example.com", Port: 0},
	}
	_, err := monitors.ConfigToTCPMonitor(m)
	if err == nil {
		t.Error("expected error for port 0")
	}
}

func Test_ConfigToTCPMonitor_NegativePort(t *testing.T) {
	m := config.Monitor{
		Name:    "test",
		Kind:    config.TCP,
		Request: config.Request{Host: "example.com", Port: -1},
	}
	_, err := monitors.ConfigToTCPMonitor(m)
	if err == nil {
		t.Error("expected error for negative port")
	}
}

func Test_ConfigToTCPMonitor_ExceedsMax(t *testing.T) {
	m := config.Monitor{
		Name:    "test",
		Kind:    config.TCP,
		Request: config.Request{Host: "example.com", Port: 65536},
	}
	_, err := monitors.ConfigToTCPMonitor(m)
	if err == nil {
		t.Error("expected error for port > 65535")
	}
}

func Test_ConfigToTCPMonitor_EmptyHost(t *testing.T) {
	m := config.Monitor{
		Name:    "test",
		Kind:    config.TCP,
		Request: config.Request{Host: "", Port: 80},
	}
	_, err := monitors.ConfigToTCPMonitor(m)
	if err == nil {
		t.Error("expected error for empty host")
	}
}

func Test_ConfigToTCPMonitor_ValidInputs(t *testing.T) {
	m := config.Monitor{
		Name:      "test",
		Kind:      config.TCP,
		Frequency: config.The10M,
		Request:   config.Request{Host: "example.com", Port: 80},
	}
	result, err := monitors.ConfigToTCPMonitor(m)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(result.GetUri(), "example.com:80") {
		t.Errorf("expected URI to contain 'example.com:80', got %q", result.GetUri())
	}
}

func Test_ConfigToTCPMonitor_MaxPort(t *testing.T) {
	m := config.Monitor{
		Name:      "test",
		Kind:      config.TCP,
		Frequency: config.The10M,
		Request:   config.Request{Host: "db.internal", Port: 65535},
	}
	_, err := monitors.ConfigToTCPMonitor(m)
	if err != nil {
		t.Fatalf("unexpected error for port 65535: %v", err)
	}
}
