package monitors_test

import (
	"context"
	"testing"

	"github.com/openstatusHQ/cli/internal/config"
	"github.com/openstatusHQ/cli/internal/monitors"
)

func Test_ApplyChanges(t *testing.T) {
	t.Run("No changes detected", func(t *testing.T) {
		monitor := config.Monitor{
			Name:      "Test Monitor",
			Active:    true,
			Frequency: config.The10M,
			Kind:      config.HTTP,
			Regions:   []config.Region{config.Iad},
			Request: config.Request{
				URL:    "https://example.com",
				Method: config.Get,
			},
		}

		lock := config.MonitorsLock{
			"test-monitor": {
				ID:      123,
				Monitor: monitor,
			},
		}

		configData := config.Monitors{
			"test-monitor": monitor,
		}

		result, err := monitors.ApplyChanges(context.Background(), "test-api-key", lock, configData)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		if result != nil {
			t.Errorf("Expected nil result when no changes, got %v", result)
		}
	})
}
