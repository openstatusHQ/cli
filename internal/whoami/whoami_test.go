package whoami_test

import (
	"bytes"
	"io"
	"log"
	"net/http"
	"os"
	"testing"

	"github.com/openstatusHQ/cli/internal/whoami"
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

func Test_getWhoami(t *testing.T) {
	t.Parallel()

	t.Run("Successfully return", func(t *testing.T) {
		body := `{
  "name": "openstatus",
  "slug": "openstatus",
  "plan": "pro"
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
		err := whoami.GetWhoamiCmd(interceptor.GetHTTPClient(), "")
		if err != nil {
			t.Error(err)
			t.Errorf("Expected log output, got nothing")
		}
	})

	t.Run("Should return error", func(t *testing.T) {

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
		err := whoami.GetWhoamiCmd(interceptor.GetHTTPClient(), "")
		if err == nil {
			t.Errorf("Expected log output, got nothing")
		}
	})
}
