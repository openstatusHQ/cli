package monitors_test

import (
	"bytes"
	"io"
	"net/http"
	"testing"

	"github.com/openstatusHQ/cli/internal/config"
	"github.com/openstatusHQ/cli/internal/monitors"
)

func Test_CreateMonitor(t *testing.T) {
	t.Parallel()

	t.Run("Create HTTP monitor successfully", func(t *testing.T) {
		// Connect RPC response format
		body := `{"monitor":{"id":"123","name":"Test Monitor","url":"https://example.com","periodicity":"PERIODICITY_10M","method":"HTTP_METHOD_GET","regions":["REGION_FLY_IAD","REGION_FLY_AMS"],"active":true}}`
		r := io.NopCloser(bytes.NewReader([]byte(body)))

		interceptor := &interceptorHTTPClient{
			f: func(req *http.Request) (*http.Response, error) {
				if req.Header.Get("x-openstatus-key") != "test-api-key" {
					t.Errorf("Expected x-openstatus-key header, got %s", req.Header.Get("x-openstatus-key"))
				}
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       r,
					Header:     http.Header{"Content-Type": []string{"application/json"}},
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

	t.Run("Create TCP monitor successfully", func(t *testing.T) {
		// Connect RPC response format for TCP monitor
		body := `{"monitor":{"id":"456","name":"TCP Monitor","uri":"example.com:443","periodicity":"PERIODICITY_5M","regions":["REGION_FLY_IAD"],"active":true}}`
		r := io.NopCloser(bytes.NewReader([]byte(body)))

		interceptor := &interceptorHTTPClient{
			f: func(req *http.Request) (*http.Response, error) {
				if req.Header.Get("x-openstatus-key") != "test-api-key" {
					t.Errorf("Expected x-openstatus-key header, got %s", req.Header.Get("x-openstatus-key"))
				}
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       r,
					Header:     http.Header{"Content-Type": []string{"application/json"}},
				}, nil
			},
		}

		monitor := config.Monitor{
			Name:      "TCP Monitor",
			Active:    true,
			Frequency: config.The5M,
			Kind:      config.TCP,
			Regions:   []config.Region{config.Iad},
			Request: config.Request{
				Host: "example.com",
				Port: 443,
			},
		}

		result, err := monitors.CreateMonitor(interceptor.GetHTTPClient(), "test-api-key", monitor)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		if result.ID != 456 {
			t.Errorf("Expected ID 456, got %d", result.ID)
		}
		if result.Name != "TCP Monitor" {
			t.Errorf("Expected name 'TCP Monitor', got %s", result.Name)
		}
		if result.JobType != "tcp" {
			t.Errorf("Expected jobType 'tcp', got %s", result.JobType)
		}
	})

	t.Run("Create monitor fails with non-200 status", func(t *testing.T) {
		body := `{"code":"internal","message":"internal error"}`
		r := io.NopCloser(bytes.NewReader([]byte(body)))

		interceptor := &interceptorHTTPClient{
			f: func(req *http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: http.StatusInternalServerError,
					Body:       r,
					Header:     http.Header{"Content-Type": []string{"application/json"}},
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

	t.Run("Create HTTP monitor returns correct Monitor struct", func(t *testing.T) {
		// Connect RPC response format with all fields
		body := `{"monitor":{"id":"789","name":"Full Monitor","description":"Test description","url":"https://test.example.com","periodicity":"PERIODICITY_5M","method":"HTTP_METHOD_POST","regions":["REGION_FLY_IAD","REGION_FLY_AMS","REGION_FLY_SYD"],"active":true,"public":true,"timeout":30000,"retry":2}}`
		r := io.NopCloser(bytes.NewReader([]byte(body)))

		interceptor := &interceptorHTTPClient{
			f: func(req *http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       r,
					Header:     http.Header{"Content-Type": []string{"application/json"}},
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

		if result.ID != 789 {
			t.Errorf("Expected ID 789, got %d", result.ID)
		}
		if result.Name != "Full Monitor" {
			t.Errorf("Expected name 'Full Monitor', got %s", result.Name)
		}
		if result.Description != "Test description" {
			t.Errorf("Expected description 'Test description', got %s", result.Description)
		}
		if result.URL != "https://test.example.com" {
			t.Errorf("Expected URL 'https://test.example.com', got %s", result.URL)
		}
		if result.Periodicity != "5m" {
			t.Errorf("Expected periodicity '5m', got %s", result.Periodicity)
		}
		if result.Method != "POST" {
			t.Errorf("Expected method 'POST', got %s", result.Method)
		}
		if result.Active != true {
			t.Errorf("Expected active true, got %v", result.Active)
		}
		if result.Public != true {
			t.Errorf("Expected public true, got %v", result.Public)
		}
		if result.Timeout != 30000 {
			t.Errorf("Expected timeout 30000, got %d", result.Timeout)
		}
		if result.Retry != 2 {
			t.Errorf("Expected retry 2, got %d", result.Retry)
		}
		if result.JobType != "http" {
			t.Errorf("Expected jobType 'http', got %s", result.JobType)
		}
	})

	t.Run("Unsupported monitor kind returns error", func(t *testing.T) {
		interceptor := &interceptorHTTPClient{
			f: func(req *http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(bytes.NewReader([]byte(`{}`))),
					Header:     http.Header{"Content-Type": []string{"application/json"}},
				}, nil
			},
		}

		monitor := config.Monitor{
			Name: "Test Monitor",
			Kind: "unsupported",
		}

		_, err := monitors.CreateMonitor(interceptor.GetHTTPClient(), "test-api-key", monitor)
		if err == nil {
			t.Error("Expected error for unsupported monitor kind, got nil")
		}
	})
}
