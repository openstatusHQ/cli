package monitors_test

import (
	"bytes"
	"log"
	"os"
	"testing"

	"github.com/openstatusHQ/cli/internal/config"
	"github.com/openstatusHQ/cli/internal/monitors"
)

// setupStdinWithInput creates a pipe to simulate stdin with the given input
func setupStdinWithInput(t *testing.T, input string) func() {
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}

	oldStdin := os.Stdin
	os.Stdin = r

	go func() {
		w.WriteString(input)
		w.Close()
	}()

	return func() {
		os.Stdin = oldStdin
	}
}

func Test_CompareLockWithConfig(t *testing.T) {
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

		var bf bytes.Buffer
		log.SetOutput(&bf)
		t.Cleanup(func() {
			log.SetOutput(os.Stdout)
		})

		result, err := monitors.CompareLockWithConfig("test-api-key", false, lock, configData)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		if result != nil {
			t.Errorf("Expected nil result when no changes, got %v", result)
		}
	})

	t.Run("Detects new monitor to create and user declines", func(t *testing.T) {
		lock := config.MonitorsLock{}

		configData := config.Monitors{
			"new-monitor": {
				Name:      "New Monitor",
				Active:    true,
				Frequency: config.The5M,
				Kind:      config.HTTP,
				Regions:   []config.Region{config.Ams},
				Request: config.Request{
					URL:    "https://new.example.com",
					Method: config.Get,
				},
			},
		}

		var bf bytes.Buffer
		log.SetOutput(&bf)
		t.Cleanup(func() {
			log.SetOutput(os.Stdout)
		})

		// Setup stdin to decline the confirmation
		cleanup := setupStdinWithInput(t, "n\n")
		defer cleanup()

		// When applyChange is false, it should detect the creation needed
		// and ask for confirmation (which we decline with "n")
		result, err := monitors.CompareLockWithConfig("test-api-key", false, lock, configData)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		// Result should be nil because user declined
		if result != nil {
			t.Errorf("Expected nil result when user declines, got %v", result)
		}
	})

	t.Run("Detects monitor update needed and user declines", func(t *testing.T) {
		originalMonitor := config.Monitor{
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

		updatedMonitor := config.Monitor{
			Name:      "Test Monitor Updated",
			Active:    true,
			Frequency: config.The5M,
			Kind:      config.HTTP,
			Regions:   []config.Region{config.Iad, config.Ams},
			Request: config.Request{
				URL:    "https://example.com",
				Method: config.Get,
			},
		}

		lock := config.MonitorsLock{
			"test-monitor": {
				ID:      123,
				Monitor: originalMonitor,
			},
		}

		configData := config.Monitors{
			"test-monitor": updatedMonitor,
		}

		var bf bytes.Buffer
		log.SetOutput(&bf)
		t.Cleanup(func() {
			log.SetOutput(os.Stdout)
		})

		// Setup stdin to decline the confirmation
		cleanup := setupStdinWithInput(t, "n\n")
		defer cleanup()

		result, err := monitors.CompareLockWithConfig("test-api-key", false, lock, configData)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		// Result should be nil because user declined
		if result != nil {
			t.Errorf("Expected nil result when user declines, got %v", result)
		}
	})

	t.Run("Detects monitor deletion needed and user declines", func(t *testing.T) {
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

		configData := config.Monitors{}

		var bf bytes.Buffer
		log.SetOutput(&bf)
		t.Cleanup(func() {
			log.SetOutput(os.Stdout)
		})

		// Setup stdin to decline the confirmation
		cleanup := setupStdinWithInput(t, "n\n")
		defer cleanup()

		result, err := monitors.CompareLockWithConfig("test-api-key", false, lock, configData)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		// Result should be nil because user declined
		if result != nil {
			t.Errorf("Expected nil result when user declines, got %v", result)
		}
	})

	t.Run("Mixed changes detected and user declines", func(t *testing.T) {
		existingMonitor := config.Monitor{
			Name:      "Existing Monitor",
			Active:    true,
			Frequency: config.The10M,
			Kind:      config.HTTP,
			Regions:   []config.Region{config.Iad},
			Request: config.Request{
				URL:    "https://existing.example.com",
				Method: config.Get,
			},
		}

		toUpdateMonitor := config.Monitor{
			Name:      "To Update Monitor",
			Active:    true,
			Frequency: config.The10M,
			Kind:      config.HTTP,
			Regions:   []config.Region{config.Iad},
			Request: config.Request{
				URL:    "https://update.example.com",
				Method: config.Get,
			},
		}

		toDeleteMonitor := config.Monitor{
			Name:      "To Delete Monitor",
			Active:    true,
			Frequency: config.The10M,
			Kind:      config.HTTP,
			Regions:   []config.Region{config.Iad},
			Request: config.Request{
				URL:    "https://delete.example.com",
				Method: config.Get,
			},
		}

		lock := config.MonitorsLock{
			"existing-monitor": {
				ID:      1,
				Monitor: existingMonitor,
			},
			"to-update-monitor": {
				ID:      2,
				Monitor: toUpdateMonitor,
			},
			"to-delete-monitor": {
				ID:      3,
				Monitor: toDeleteMonitor,
			},
		}

		updatedMonitor := config.Monitor{
			Name:      "To Update Monitor - Updated",
			Active:    false,
			Frequency: config.The5M,
			Kind:      config.HTTP,
			Regions:   []config.Region{config.Iad, config.Ams},
			Request: config.Request{
				URL:    "https://update.example.com",
				Method: config.Post,
			},
		}

		newMonitor := config.Monitor{
			Name:      "New Monitor",
			Active:    true,
			Frequency: config.The1M,
			Kind:      config.HTTP,
			Regions:   []config.Region{config.Syd},
			Request: config.Request{
				URL:    "https://new.example.com",
				Method: config.Get,
			},
		}

		configData := config.Monitors{
			"existing-monitor":  existingMonitor,
			"to-update-monitor": updatedMonitor,
			"new-monitor":       newMonitor,
		}

		var bf bytes.Buffer
		log.SetOutput(&bf)
		t.Cleanup(func() {
			log.SetOutput(os.Stdout)
		})

		// Setup stdin to decline the confirmation
		cleanup := setupStdinWithInput(t, "n\n")
		defer cleanup()

		result, err := monitors.CompareLockWithConfig("test-api-key", false, lock, configData)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		// Result should be nil because user declined
		if result != nil {
			t.Errorf("Expected nil result when user declines, got %v", result)
		}
	})
}
