package config_test

import (
	"os"
	"testing"

	"github.com/openstatusHQ/cli/internal/config"
)

var openstatusConfig = `
"test-monitor":
  active: true
  assertions:
    - compare: eq
      kind: statusCode
      target: 200
  description: Test monitor description
  frequency: 10m
  kind: http
  name: Test Monitor
  regions:
    - iad
    - ams
  request:
    headers:
      User-Agent: OpenStatus
    method: GET
    url: https://example.com
  retry: 3
`

func Test_ReadOpenStatus(t *testing.T) {
	t.Run("Read valid openstatus config", func(t *testing.T) {
		f, err := os.CreateTemp(".", "openstatus*.yaml")
		if err != nil {
			t.Fatal(err)
		}
		defer os.Remove(f.Name())

		if _, err := f.Write([]byte(openstatusConfig)); err != nil {
			t.Fatal(err)
		}
		if err := f.Close(); err != nil {
			t.Fatal(err)
		}

		out, err := config.ReadOpenStatus(f.Name())
		if err != nil {
			t.Fatal(err)
		}

		// Check that the monitor was read correctly
		// Note: We check for the specific monitor because the global koanf instance
		// may have accumulated state from previous tests
		monitor, exists := out["test-monitor"]
		if !exists {
			t.Fatal("Expected 'test-monitor' to exist in output")
		}

		if monitor.Name != "Test Monitor" {
			t.Errorf("Expected name 'Test Monitor', got %s", monitor.Name)
		}
		if monitor.Description != "Test monitor description" {
			t.Errorf("Expected description 'Test monitor description', got %s", monitor.Description)
		}
		if monitor.Frequency != config.The10M {
			t.Errorf("Expected frequency '10m', got %s", monitor.Frequency)
		}
		if monitor.Kind != config.HTTP {
			t.Errorf("Expected kind 'http', got %s", monitor.Kind)
		}
		if !monitor.Active {
			t.Error("Expected monitor to be active")
		}
		if monitor.Retry != 3 {
			t.Errorf("Expected retry 3, got %d", monitor.Retry)
		}
		if len(monitor.Regions) != 2 {
			t.Errorf("Expected 2 regions, got %d", len(monitor.Regions))
		}
		if monitor.Request.URL != "https://example.com" {
			t.Errorf("Expected URL 'https://example.com', got %s", monitor.Request.URL)
		}
		if monitor.Request.Method != config.Get {
			t.Errorf("Expected method 'GET', got %s", monitor.Request.Method)
		}
		if len(monitor.Assertions) != 1 {
			t.Errorf("Expected 1 assertion, got %d", len(monitor.Assertions))
		}
	})

	t.Run("File does not exist", func(t *testing.T) {
		_, err := config.ReadOpenStatus("nonexistent.yaml")
		if err == nil {
			t.Error("Expected error for non-existent file, got nil")
		}
	})
}

func Test_ParseConfigMonitorsToMonitor(t *testing.T) {
	t.Run("Parse monitors map to slice", func(t *testing.T) {
		monitors := config.Monitors{
			"monitor-1": {
				Name:      "Monitor 1",
				Active:    true,
				Frequency: config.The5M,
				Kind:      config.HTTP,
				Regions:   []config.Region{config.Iad},
				Request: config.Request{
					URL:    "https://example1.com",
					Method: config.Get,
				},
			},
			"monitor-2": {
				Name:      "Monitor 2",
				Active:    false,
				Frequency: config.The10M,
				Kind:      config.HTTP,
				Regions:   []config.Region{config.Ams},
				Request: config.Request{
					URL:    "https://example2.com",
					Method: config.Post,
				},
			},
		}

		result := config.ParseConfigMonitorsToMonitor(monitors)

		if len(result) != 2 {
			t.Errorf("Expected 2 monitors, got %d", len(result))
		}
	})

	t.Run("Empty monitors map", func(t *testing.T) {
		monitors := config.Monitors{}
		result := config.ParseConfigMonitorsToMonitor(monitors)

		if len(result) != 0 {
			t.Errorf("Expected 0 monitors, got %d", len(result))
		}
	})

	t.Run("Assertions are converted", func(t *testing.T) {
		monitors := config.Monitors{
			"monitor-1": {
				Name:      "Monitor 1",
				Active:    true,
				Frequency: config.The5M,
				Kind:      config.HTTP,
				Regions:   []config.Region{config.Iad},
				Request: config.Request{
					URL:    "https://example.com",
					Method: config.Get,
				},
				Assertions: []config.Assertion{
					{
						Kind:    config.StatusCode,
						Compare: config.Eq,
						Target:  float64(200),
					},
				},
			},
		}

		result := config.ParseConfigMonitorsToMonitor(monitors)

		if len(result) != 1 {
			t.Fatalf("Expected 1 monitor, got %d", len(result))
		}

		if result[0].Assertions[0].Target != 200 {
			t.Errorf("Expected target to be converted to int 200, got %v", result[0].Assertions[0].Target)
		}
	})
}
