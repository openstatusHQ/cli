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

func Test_UpdateMonitor(t *testing.T) {
	t.Parallel()

	t.Run("Update monitor successfully", func(t *testing.T) {
		body := `{
			"id": 123,
			"name": "Updated Monitor",
			"url": "https://example.com",
			"periodicity": "5m",
			"method": "GET",
			"regions": ["iad", "ams", "syd"],
			"active": true,
			"public": false,
			"timeout": 45000,
			"body": "",
			"retry": 5,
			"jobType": "http"
		}`
		r := io.NopCloser(bytes.NewReader([]byte(body)))

		interceptor := &interceptorHTTPClient{
			f: func(req *http.Request) (*http.Response, error) {
				if req.Method != http.MethodPut {
					t.Errorf("Expected PUT method, got %s", req.Method)
				}
				if req.Header.Get("x-openstatus-key") != "test-api-key" {
					t.Errorf("Expected x-openstatus-key header, got %s", req.Header.Get("x-openstatus-key"))
				}
				if req.Header.Get("Content-Type") != "application/json" {
					t.Errorf("Expected Content-Type header, got %s", req.Header.Get("Content-Type"))
				}
				expectedURL := "https://api.openstatus.dev/v1/monitor/http/123"
				if req.URL.String() != expectedURL {
					t.Errorf("Expected URL %s, got %s", expectedURL, req.URL.String())
				}
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       r,
				}, nil
			},
		}

		monitor := config.Monitor{
			Name:      "Updated Monitor",
			Active:    true,
			Frequency: config.The5M,
			Kind:      config.HTTP,
			Regions:   []config.Region{config.Iad, config.Ams, config.Syd},
			Request: config.Request{
				URL:    "https://example.com",
				Method: config.Get,
			},
			Retry: 5,
		}

		result, err := monitors.UpdateMonitor(interceptor.GetHTTPClient(), "test-api-key", 123, monitor)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		if result.ID != 123 {
			t.Errorf("Expected ID 123, got %d", result.ID)
		}
		if result.Name != "Updated Monitor" {
			t.Errorf("Expected name 'Updated Monitor', got %s", result.Name)
		}
	})

	t.Run("Update monitor fails with non-200 status", func(t *testing.T) {
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

		_, err := monitors.UpdateMonitor(interceptor.GetHTTPClient(), "test-api-key", 123, monitor)
		if err == nil {
			t.Error("Expected error for non-200 status, got nil")
		}
	})

	t.Run("Update monitor returns correct Monitor struct", func(t *testing.T) {
		body := `{
			"id": 789,
			"name": "Full Updated Monitor",
			"url": "https://updated.example.com",
			"periodicity": "30m",
			"description": "Updated description",
			"method": "PUT",
			"regions": ["lhr", "fra"],
			"active": false,
			"public": true,
			"timeout": 60000,
			"degraded_after": 10000,
			"body": "{\"updated\": true}",
			"headers": [{"key": "Authorization", "value": "Bearer token"}],
			"assertions": [{"type": "statusCode", "compare": "eq", "target": 201}],
			"retry": 1,
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
			Name: "Full Updated Monitor",
			Kind: config.HTTP,
		}

		result, err := monitors.UpdateMonitor(interceptor.GetHTTPClient(), "test-api-key", 789, monitor)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		expected := monitors.Monitor{
			ID:            789,
			Name:          "Full Updated Monitor",
			URL:           "https://updated.example.com",
			Periodicity:   "30m",
			Description:   "Updated description",
			Method:        "PUT",
			Regions:       []string{"lhr", "fra"},
			Active:        false,
			Public:        true,
			Timeout:       60000,
			DegradedAfter: 10000,
			Body:          "{\"updated\": true}",
			Headers:       []monitors.Header{{Key: "Authorization", Value: "Bearer token"}},
			Assertions:    []monitors.Assertion{{Type: "statusCode", Compare: "eq", Target: float64(201)}},
			Retry:         1,
			JobType:       "http",
		}

		if !cmp.Equal(expected, result) {
			t.Errorf("Expected %+v, got %+v", expected, result)
		}
	})

	t.Run("Update TCP monitor uses correct URL", func(t *testing.T) {
		body := `{
			"id": 100,
			"name": "TCP Monitor",
			"url": "example.com:443",
			"periodicity": "1m",
			"method": "",
			"regions": ["iad"],
			"active": true,
			"public": false,
			"timeout": 10000,
			"body": "",
			"retry": 0,
			"jobType": "tcp"
		}`
		r := io.NopCloser(bytes.NewReader([]byte(body)))

		interceptor := &interceptorHTTPClient{
			f: func(req *http.Request) (*http.Response, error) {
				expectedURL := "https://api.openstatus.dev/v1/monitor/tcp/100"
				if req.URL.String() != expectedURL {
					t.Errorf("Expected URL %s, got %s", expectedURL, req.URL.String())
				}
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       r,
				}, nil
			},
		}

		monitor := config.Monitor{
			Name: "TCP Monitor",
			Kind: config.TCP,
		}

		_, err := monitors.UpdateMonitor(interceptor.GetHTTPClient(), "test-api-key", 100, monitor)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
	})
}
