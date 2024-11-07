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
				}, nil
			},
		}

		var bf bytes.Buffer
		log.SetOutput(&bf)
		t.Cleanup(func() {
			log.SetOutput(os.Stdout)
		})
		err := monitors.MonitorTrigger(interceptor.GetHTTPClient(), "", "")
		if err == nil {
			t.Errorf("Expected log output, got nothing")
		}
	})

	t.Run("Successfully return", func(t *testing.T) {
		body := `{"resultId": 1}`
		r := io.NopCloser(bytes.NewReader([]byte(body)))

		interceptor := &interceptorHTTPClient{
			f: func(req *http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       r,
				}, nil
			},
		}

		var bf bytes.Buffer
		log.SetOutput(&bf)
		t.Cleanup(func() {
			log.SetOutput(os.Stdout)
		})
		err := monitors.MonitorTrigger(interceptor.GetHTTPClient(), "", "1")
		if err != nil {
			t.Errorf("Expected no output, got error")
		}
	})
	t.Run("No 200 throw error", func(t *testing.T) {

		interceptor := &interceptorHTTPClient{
			f: func(req *http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: http.StatusInternalServerError,
				}, nil
			},
		}

		var bf bytes.Buffer
		log.SetOutput(&bf)
		t.Cleanup(func() {
			log.SetOutput(os.Stdout)
		})
		err := monitors.MonitorTrigger(interceptor.GetHTTPClient(), "1", "1")
		if err == nil {
			t.Errorf("Expected log output, got nothing")
		}
	})
}
