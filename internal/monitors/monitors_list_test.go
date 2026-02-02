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
		err := monitors.ListMonitorsWithHTTPClient(interceptor.GetHTTPClient(), "test-token")
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
		err := monitors.ListMonitorsWithHTTPClient(interceptor.GetHTTPClient(), "1")
		if err == nil {
			t.Errorf("Expected error, got nothing")
		}
	})
}
