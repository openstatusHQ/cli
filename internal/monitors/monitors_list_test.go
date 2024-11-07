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

func Test_listMonitors(t *testing.T) {
	t.Parallel()

	t.Run("Successfully return", func(t *testing.T) {
		body := `[
  {
    "id": 1,
    "periodicity": "10m",
    "url": "https://www.openstatus.dev",
    "regions": [
      "ams",
      "scl"
    ],
    "name": "OpenStatus ",
    "description": "Our website 🌐",
    "method": "GET",
    "body": "",
    "headers": [
      {
        "key": "",
        "value": ""
      }
    ],
    "assertions": [],
    "active": true,
    "public": true,
    "degradedAfter": null,
    "timeout": 45000,
    "jobType": "http"
  }]`
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
		err := monitors.ListMonitors(interceptor.GetHTTPClient(), "")
		if err != nil {
			t.Error(err)
			t.Errorf("Expected log output, got nothing")
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
		err := monitors.ListMonitors(interceptor.GetHTTPClient(), "1")
		if err == nil {
			t.Errorf("Expected log output, got nothing")
		}
	})
}
