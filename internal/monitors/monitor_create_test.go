package monitors_test

import (
	"bytes"
	"io"
	"net/http"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/openstatusHQ/cli/internal/config"
	"github.com/openstatusHQ/cli/internal/monitors"
)

func Test_CreateMonitor(t *testing.T) {
	t.Parallel()

	t.Run("Create monitor successfully", func(t *testing.T) {
		body := `{
			"id": 123,
			"name": "Test Monitor",
			"url": "https://example.com",
			"periodicity": "10m",
			"method": "GET",
			"regions": ["iad", "ams"],
			"active": true,
			"public": false,
			"timeout": 45000,
			"body": "",
			"retry": 3,
			"jobType": "http"
		}`
		r := io.NopCloser(bytes.NewReader([]byte(body)))

		interceptor := &interceptorHTTPClient{
			f: func(req *http.Request) (*http.Response, error) {
				if req.Method != http.MethodPost {
					t.Errorf("Expected POST method, got %s", req.Method)
				}
				if req.Header.Get("x-openstatus-key") != "test-api-key" {
					t.Errorf("Expected x-openstatus-key header, got %s", req.Header.Get("x-openstatus-key"))
				}
				if req.Header.Get("Content-Type") != "application/json" {
					t.Errorf("Expected Content-Type header, got %s", req.Header.Get("Content-Type"))
				}
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       r,
				}, nil
			},
		}

		monitor := config.Monitor{
			Name:      "Test Monitor",
			Active:    true,
			Frequency: config.The10M,
			Kind:      config.HTTP,
			Regions:   []config.Region{config.Iad, config.Ams},
			Request: config.Request{
				URL:    "https://example.com",
				Method: config.Get,
			},
			Retry: 3,
		}

		result, err := monitors.CreateMonitor(interceptor.GetHTTPClient(), "test-api-key", monitor)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		if result.ID != 123 {
			t.Errorf("Expected ID 123, got %d", result.ID)
		}
		if result.Name != "Test Monitor" {
			t.Errorf("Expected name 'Test Monitor', got %s", result.Name)
		}
	})

	t.Run("Create monitor fails with non-200 status", func(t *testing.T) {
		interceptor := &interceptorHTTPClient{
			f: func(req *http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: http.StatusBadRequest,
					Body:       io.NopCloser(bytes.NewReader([]byte(`{"error": "bad request"}`))),
				}, nil
			},
		}

		monitor := config.Monitor{
			Name: "Test Monitor",
			Kind: config.HTTP,
		}

		_, err := monitors.CreateMonitor(interceptor.GetHTTPClient(), "test-api-key", monitor)
		if err == nil {
			t.Error("Expected error for non-200 status, got nil")
		}
	})

	t.Run("Create monitor returns correct Monitor struct", func(t *testing.T) {
		body := `{
			"id": 456,
			"name": "Full Monitor",
			"url": "https://test.example.com",
			"periodicity": "5m",
			"description": "Test description",
			"method": "POST",
			"regions": ["iad", "ams", "syd"],
			"active": true,
			"public": true,
			"timeout": 30000,
			"degraded_after": 5000,
			"body": "{\"key\": \"value\"}",
			"headers": [{"key": "Content-Type", "value": "application/json"}],
			"assertions": [{"type": "statusCode", "compare": "eq", "target": 200}],
			"retry": 2,
			"jobType": "http"
		}`
		r := io.NopCloser(bytes.NewReader([]byte(body)))

		interceptor := &interceptorHTTPClient{
			f: func(req *http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       r,
				}, nil
			},
		}

		monitor := config.Monitor{
			Name: "Full Monitor",
			Kind: config.HTTP,
		}

		result, err := monitors.CreateMonitor(interceptor.GetHTTPClient(), "test-api-key", monitor)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		expected := monitors.Monitor{
			ID:            456,
			Name:          "Full Monitor",
			URL:           "https://test.example.com",
			Periodicity:   "5m",
			Description:   "Test description",
			Method:        "POST",
			Regions:       []string{"iad", "ams", "syd"},
			Active:        true,
			Public:        true,
			Timeout:       30000,
			DegradedAfter: 5000,
			Body:          "{\"key\": \"value\"}",
			Headers:       []monitors.Header{{Key: "Content-Type", Value: "application/json"}},
			Assertions:    []monitors.Assertion{{Type: "statusCode", Compare: "eq", Target: float64(200)}},
			Retry:         2,
			JobType:       "http",
		}

		if !cmp.Equal(expected, result) {
			t.Errorf("Expected %+v, got %+v", expected, result)
		}
	})
}
