package monitors_test

import (
	"bytes"
	"context"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"testing"

	output "github.com/openstatusHQ/cli/internal/cli"
	"github.com/openstatusHQ/cli/internal/monitors"
)

// Not parallel: subtests redirect the process-wide os.Stdout.
func Test_listMonitors(t *testing.T) {
	t.Run("Successfully return", func(t *testing.T) {
		// Connect RPC response format with protobuf content
		// The response is a ListMonitorsResponse in JSON format with Connect headers
		body := `{"httpMonitors":[{"id":"1","name":"OpenStatus","url":"https://www.openstatus.dev","periodicity":"PERIODICITY_10M","active":true,"public":true}]}`
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

		var bf bytes.Buffer
		log.SetOutput(&bf)
		t.Cleanup(func() {
			log.SetOutput(os.Stdout)
		})
		err := monitors.ListMonitorsWithHTTPClient(context.Background(), interceptor.GetHTTPClient(), "test-token")
		if err != nil {
			t.Error(err)
			t.Errorf("Expected log output, got nothing")
		}
	})
	t.Run("No 200 throw error", func(t *testing.T) {
		body := `{"code":"internal","message":"internal error"}`
		r := io.NopCloser(bytes.NewReader([]byte(body)))

		interceptor := &interceptorHTTPClient{
			f: func(req *http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: http.StatusInternalServerError,
					Body:       r,
					Header: http.Header{
						"Content-Type": []string{"application/json"},
					},
				}, nil
			},
		}

		var bf bytes.Buffer
		log.SetOutput(&bf)
		t.Cleanup(func() {
			log.SetOutput(os.Stdout)
		})
		err := monitors.ListMonitorsWithHTTPClient(context.Background(), interceptor.GetHTTPClient(), "1")
		if err == nil {
			t.Errorf("Expected error, got nothing")
		}
	})
	t.Run("Includes ICMP monitors", func(t *testing.T) {
		body := `{"icmpMonitors":[{"id":"9","name":"Gateway Ping","uri":"8.8.8.8","periodicity":"PERIODICITY_5M","active":true}]}`
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

		output.SetJSONOutput(true)
		t.Cleanup(func() { output.SetJSONOutput(false) })

		out := captureStdout(t, func() {
			if err := monitors.ListMonitorsWithHTTPClient(context.Background(), interceptor.GetHTTPClient(), "test-token"); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})

		if !strings.Contains(out, `"kind": "icmp"`) {
			t.Errorf("expected an icmp entry in JSON output, got:\n%s", out)
		}
		if !strings.Contains(out, `"url": "8.8.8.8"`) {
			t.Errorf("expected ICMP uri as url in JSON output, got:\n%s", out)
		}
	})
}

func captureStdout(t *testing.T, f func()) string {
	t.Helper()

	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to create pipe: %v", err)
	}
	original := os.Stdout
	os.Stdout = w
	t.Cleanup(func() { os.Stdout = original })

	f()

	if err := w.Close(); err != nil {
		t.Fatalf("failed to close pipe: %v", err)
	}
	os.Stdout = original

	var buf bytes.Buffer
	if _, err := io.Copy(&buf, r); err != nil {
		t.Fatalf("failed to read pipe: %v", err)
	}
	return buf.String()
}
