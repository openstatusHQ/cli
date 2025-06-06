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

func Test_getMonitorInfo(t *testing.T) {
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
		err := monitors.GetMonitorInfo(interceptor.GetHTTPClient(), "", "")
		if err == nil {
			t.Errorf("Expected log output, got nothing")
		}
	})

	t.Run("Should work", func(t *testing.T) {

		body := `{
  "id": 2260,
  "periodicity": "10m",
  "url": "https://www.openstatus.dev",
  "regions": [
    "iad",
    "hkg",
    "jnb",
    "syd",
    "gru"
  ],
  "name": "Vercel Checker Edge",
  "description": "",
  "method": "GET",
  "body": "",
  "headers": [
    {
      "key": "",
      "value": ""
    }
  ],
  "assertions": [],
  "active": false,
  "public": false,
  "degradedAfter": null,
  "timeout": 45000,
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

		var bf bytes.Buffer
		log.SetOutput(&bf)
		t.Cleanup(func() {
			log.SetOutput(os.Stdout)
		})
		err := monitors.GetMonitorInfo(interceptor.GetHTTPClient(), "test", "1")
		if err != nil {
			t.Errorf("Expected log output, got nothing")
		}
	})

}
