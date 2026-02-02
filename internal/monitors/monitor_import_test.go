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
		// Connect RPC response format with ListMonitorsResponse
		body := `{"httpMonitors":[{"id":"123","name":"HTTP Monitor","url":"https://example.com","periodicity":"PERIODICITY_10M","active":true,"public":false}]}`
		r := io.NopCloser(bytes.NewReader([]byte(body)))

		interceptor := &interceptorHTTPClient{
			f: func(req *http.Request) (*http.Response, error) {
				if req.Header.Get("x-openstatus-key") != "test-api-key" {
					t.Errorf("Expected x-openstatus-key header, got %s", req.Header.Get("x-openstatus-key"))
				}
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       r,
					Header: http.Header{
						"Content-Type": []string{"application/json"},
					},
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

		err = monitors.ExportMonitorWithHTTPClient(interceptor.GetHTTPClient(), "test-api-key", outputFile.Name())
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
		body := `{"tcpMonitors":[{"id":"456","name":"TCP Monitor","uri":"example.com:443","periodicity":"PERIODICITY_5M","active":true}]}`
		r := io.NopCloser(bytes.NewReader([]byte(body)))

		interceptor := &interceptorHTTPClient{
			f: func(req *http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       r,
					Header: http.Header{
						"Content-Type": []string{"application/json"},
					},
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

		err = monitors.ExportMonitorWithHTTPClient(interceptor.GetHTTPClient(), "test-api-key", outputFile.Name())
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
	})

	t.Run("Export fails with error status", func(t *testing.T) {
		body := `{"code":"permission_denied","message":"unauthorized"}`
		r := io.NopCloser(bytes.NewReader([]byte(body)))

		interceptor := &interceptorHTTPClient{
			f: func(req *http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: http.StatusUnauthorized,
					Body:       r,
					Header: http.Header{
						"Content-Type": []string{"application/json"},
					},
				}, nil
			},
		}

		err := monitors.ExportMonitorWithHTTPClient(interceptor.GetHTTPClient(), "invalid-key", "output.yaml")
		if err == nil {
			t.Error("Expected error for non-200 status, got nil")
		}
	})

	t.Run("Export handles empty monitors", func(t *testing.T) {
		body := `{"httpMonitors":[],"tcpMonitors":[],"dnsMonitors":[]}`
		r := io.NopCloser(bytes.NewReader([]byte(body)))

		interceptor := &interceptorHTTPClient{
			f: func(req *http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       r,
					Header: http.Header{
						"Content-Type": []string{"application/json"},
					},
				}, nil
			},
		}

		outputFile, err := os.CreateTemp(".", "export_empty*.yaml")
		if err != nil {
			t.Fatal(err)
		}
		defer os.Remove(outputFile.Name())
		defer os.Remove("openstatus.lock")
		outputFile.Close()

		err = monitors.ExportMonitorWithHTTPClient(interceptor.GetHTTPClient(), "test-api-key", outputFile.Name())
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
	})
}
