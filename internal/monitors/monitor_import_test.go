package monitors_test

import (
	"bytes"
	"io"
	"net/http"
	"os"
	"testing"

	"github.com/openstatusHQ/cli/internal/monitors"
)

func Test_ExportMonitor(t *testing.T) {
	t.Parallel()

	t.Run("Export HTTP monitors successfully", func(t *testing.T) {
		body := `[
			{
				"id": 123,
				"name": "HTTP Monitor",
				"url": "https://example.com",
				"periodicity": "10m",
				"description": "Test monitor",
				"method": "GET",
				"regions": ["iad", "ams"],
				"active": true,
				"public": false,
				"timeout": 45000,
				"body": "",
				"headers": [{"key": "User-Agent", "value": "OpenStatus"}],
				"assertions": [{"type": "statusCode", "compare": "eq", "target": 200}],
				"retry": 3,
				"jobType": "http"
			}
		]`
		r := io.NopCloser(bytes.NewReader([]byte(body)))

		interceptor := &interceptorHTTPClient{
			f: func(req *http.Request) (*http.Response, error) {
				if req.Method != http.MethodGet {
					t.Errorf("Expected GET method, got %s", req.Method)
				}
				if req.Header.Get("x-openstatus-key") != "test-api-key" {
					t.Errorf("Expected x-openstatus-key header, got %s", req.Header.Get("x-openstatus-key"))
				}
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       r,
				}, nil
			},
		}

		outputFile, err := os.CreateTemp(".", "export*.yaml")
		if err != nil {
			t.Fatal(err)
		}
		defer os.Remove(outputFile.Name())
		defer os.Remove("openstatus.lock")
		outputFile.Close()

		err = monitors.ExportMonitor(interceptor.GetHTTPClient(), "test-api-key", outputFile.Name())
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		content, err := os.ReadFile(outputFile.Name())
		if err != nil {
			t.Fatal(err)
		}

		if len(content) == 0 {
			t.Error("Expected non-empty output file")
		}
	})

	t.Run("Export TCP monitors successfully", func(t *testing.T) {
		body := `[
			{
				"id": 456,
				"name": "TCP Monitor",
				"url": "example.com:443",
				"periodicity": "5m",
				"description": "TCP test",
				"method": "",
				"regions": ["iad"],
				"active": true,
				"public": false,
				"timeout": 10000,
				"body": "",
				"headers": [],
				"assertions": [],
				"retry": 0,
				"jobType": "tcp"
			}
		]`
		r := io.NopCloser(bytes.NewReader([]byte(body)))

		interceptor := &interceptorHTTPClient{
			f: func(req *http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       r,
				}, nil
			},
		}

		outputFile, err := os.CreateTemp(".", "export_tcp*.yaml")
		if err != nil {
			t.Fatal(err)
		}
		defer os.Remove(outputFile.Name())
		defer os.Remove("openstatus.lock")
		outputFile.Close()

		err = monitors.ExportMonitor(interceptor.GetHTTPClient(), "test-api-key", outputFile.Name())
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
	})

	t.Run("Export fails with non-200 status", func(t *testing.T) {
		interceptor := &interceptorHTTPClient{
			f: func(req *http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: http.StatusUnauthorized,
					Body:       io.NopCloser(bytes.NewReader([]byte(`{"error": "unauthorized"}`))),
				}, nil
			},
		}

		err := monitors.ExportMonitor(interceptor.GetHTTPClient(), "invalid-key", "output.yaml")
		if err == nil {
			t.Error("Expected error for non-200 status, got nil")
		}
	})

	t.Run("Export fails with unknown job type", func(t *testing.T) {
		body := `[
			{
				"id": 789,
				"name": "Unknown Monitor",
				"url": "https://example.com",
				"periodicity": "10m",
				"method": "GET",
				"regions": ["iad"],
				"active": true,
				"jobType": "unknown"
			}
		]`
		r := io.NopCloser(bytes.NewReader([]byte(body)))

		interceptor := &interceptorHTTPClient{
			f: func(req *http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       r,
				}, nil
			},
		}

		outputFile, err := os.CreateTemp(".", "export_unknown*.yaml")
		if err != nil {
			t.Fatal(err)
		}
		defer os.Remove(outputFile.Name())
		outputFile.Close()

		err = monitors.ExportMonitor(interceptor.GetHTTPClient(), "test-api-key", outputFile.Name())
		if err == nil {
			t.Error("Expected error for unknown job type, got nil")
		}
	})

	t.Run("Export handles empty headers", func(t *testing.T) {
		body := `[
			{
				"id": 111,
				"name": "No Headers Monitor",
				"url": "https://example.com",
				"periodicity": "10m",
				"method": "GET",
				"regions": ["iad"],
				"active": true,
				"public": false,
				"timeout": 45000,
				"body": "",
				"headers": [{"key": "", "value": ""}],
				"assertions": [],
				"retry": 0,
				"jobType": "http"
			}
		]`
		r := io.NopCloser(bytes.NewReader([]byte(body)))

		interceptor := &interceptorHTTPClient{
			f: func(req *http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       r,
				}, nil
			},
		}

		outputFile, err := os.CreateTemp(".", "export_noheaders*.yaml")
		if err != nil {
			t.Fatal(err)
		}
		defer os.Remove(outputFile.Name())
		defer os.Remove("openstatus.lock")
		outputFile.Close()

		err = monitors.ExportMonitor(interceptor.GetHTTPClient(), "test-api-key", outputFile.Name())
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
	})

	t.Run("Export handles status assertion type conversion", func(t *testing.T) {
		body := `[
			{
				"id": 222,
				"name": "Assertion Monitor",
				"url": "https://example.com",
				"periodicity": "10m",
				"method": "GET",
				"regions": ["iad"],
				"active": true,
				"public": false,
				"timeout": 45000,
				"body": "",
				"headers": [],
				"assertions": [{"type": "status", "compare": "eq", "target": 200}],
				"retry": 0,
				"jobType": "http"
			}
		]`
		r := io.NopCloser(bytes.NewReader([]byte(body)))

		interceptor := &interceptorHTTPClient{
			f: func(req *http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       r,
				}, nil
			},
		}

		outputFile, err := os.CreateTemp(".", "export_assertion*.yaml")
		if err != nil {
			t.Fatal(err)
		}
		defer os.Remove(outputFile.Name())
		defer os.Remove("openstatus.lock")
		outputFile.Close()

		err = monitors.ExportMonitor(interceptor.GetHTTPClient(), "test-api-key", outputFile.Name())
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
	})
}
