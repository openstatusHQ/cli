package monitors

import (
	"bytes"
	"log"
	"net/http"
	"os"
	"testing"
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

func Test_getMonitorInfo(t *testing.T) {
	t.Parallel()

	t.Run("Monitor ID is required", func(t *testing.T) {

		interceptor := &interceptorHTTPClient{
			f: func(req *http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: http.StatusAccepted,
				}, nil
			},
		}

		var bf bytes.Buffer
		log.SetOutput(&bf)
		t.Cleanup(func() {
			log.SetOutput(os.Stdout)
		})
		err := getMonitorInfo(interceptor.GetHTTPClient(), "", "")
		if err == nil {
			t.Errorf("Expected log output, got nothing")
		}
	})

	t.Run("Monitor ID is required", func(t *testing.T) {

		interceptor := &interceptorHTTPClient{
			f: func(req *http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: http.StatusAccepted,
				}, nil
			},
		}

		var bf bytes.Buffer
		log.SetOutput(&bf)
		t.Cleanup(func() {
			log.SetOutput(os.Stdout)
		})
		err := getMonitorInfo(interceptor.GetHTTPClient(), "", "")
		if err == nil {
			t.Errorf("Expected log output, got nothing")
		}
	})

}
