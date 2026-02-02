package monitors_test

import (
	"bytes"
	"io"
	"log"
	"net/http"
	"os"
	"testing"

	"github.com/openstatusHQ/cli/internal/monitors"
)

func Test_getMonitorTrigger(t *testing.T) {
	t.Parallel()

	t.Run("Monitor ID is required", func(t *testing.T) {
		interceptor := &interceptorHTTPClient{
			f: func(req *http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: http.StatusOK,
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
		err := monitors.TriggerMonitorWithHTTPClient(interceptor.GetHTTPClient(), "", "")
		if err == nil {
			t.Errorf("Expected error for empty monitor ID, got nil")
		}
	})

	t.Run("Successfully return", func(t *testing.T) {
		body := `{"success": true}`
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
		err := monitors.TriggerMonitorWithHTTPClient(interceptor.GetHTTPClient(), "test-token", "1")
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
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
		err := monitors.TriggerMonitorWithHTTPClient(interceptor.GetHTTPClient(), "test-token", "1")
		if err == nil {
			t.Errorf("Expected error for non-200 status, got nil")
		}
	})
}
