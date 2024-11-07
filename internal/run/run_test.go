package run_test

import (
	"bytes"
	"io"
	"log"
	"net/http"
	"os"
	"testing"

	"github.com/openstatusHQ/cli/internal/run"
)

type interceptorHTTPClient struct {
	f func(req *http.Request) (*http.Response, error)
}

func (i *interceptorHTTPClient) RoundTrip(req *http.Request) (*http.Response, error) {
	return i.f(req)
}

func (i *interceptorHTTPClient) GetHTTPClient() *http.Client {
	return &http.Client{
		Transport: i,
	}
}

func Test_run(t *testing.T) {
	t.Parallel()

	t.Run("No Id should return error", func(t *testing.T) {
		// 		body := `{
		//   "name": "openstatus",
		//   "slug": "openstatus",
		//   "plan": "pro"
		// }`
		// 		r := io.NopCloser(bytes.NewReader([]byte(body)))

		interceptor := &interceptorHTTPClient{
			f: func(req *http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: http.StatusOK,
					// Body:       r,
				}, nil
			},
		}

		var bf bytes.Buffer
		log.SetOutput(&bf)
		t.Cleanup(func() {
			log.SetOutput(os.Stdout)
		})
		err := run.MonitorTrigger(interceptor.GetHTTPClient(), "", "")
		if err == nil {
			t.Error(err)
			t.Errorf("Monitor Trigger should return error")
		}
	})
	t.Run("Successfully run http reponse", func(t *testing.T) {
		body := `[
  {
  "jobType": "http",
    "status": 200,
    "latency": 318,
    "region": "iad",
    "timestamp": 1730296465106,
    "timing": {
      "dnsStart": 1730296465106,
      "dnsDone": 1730296465109,
      "connectStart": 1730296465109,
      "connectDone": 1730296465110,
      "tlsHandshakeStart": 1730296465110,
      "tlsHandshakeDone": 1730296465116,
      "firstByteStart": 1730296465117,
      "firstByteDone": 1730296465425,
      "transferStart": 1730296465425,
      "transferDone": 1730296465425
    },
    "body": "{\"ping\":\"pong\"}"
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
		err := run.MonitorTrigger(interceptor.GetHTTPClient(), "", "1")
		if err != nil {
			t.Error(err)
			t.Errorf("Monitor Trigger should return error")
		}
	})
	t.Run("Successfully run tcp reponse", func(t *testing.T) {
		body := `[
		{
    "jobType": "tcp",
    "latency": 3,
    "region": "ams",
    "timestamp": 1730990324626,
    "timing": {
      "tcpStart": 1730990324626,
      "tcpDone": 1730990324629
    },
    "errorMessage": ""
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
		err := run.MonitorTrigger(interceptor.GetHTTPClient(), "", "1")
		if err != nil {
			t.Error(err)
			t.Errorf("Monitor Trigger should return error")
		}
	})
}
