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
		// Connect RPC response format with MonitorConfig oneof
		body := `{"monitor":{"http":{"id":"2260","name":"Vercel Checker Edge","description":"","url":"https://www.openstatus.dev","periodicity":"PERIODICITY_10M","method":"HTTP_METHOD_GET","regions":["REGION_FLY_IAD","REGION_FLY_JNB","REGION_FLY_SYD","REGION_FLY_GRU"],"active":false,"public":false,"timeout":45000}}}`
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

		var bf bytes.Buffer
		log.SetOutput(&bf)
		t.Cleanup(func() {
			log.SetOutput(os.Stdout)
		})
		err := monitors.GetMonitorInfo(interceptor.GetHTTPClient(), "test", "1")
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
	})

}
