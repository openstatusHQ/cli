package whoami_test

import (
	"bytes"
	"context"
	"errors"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
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
		err := whoami.GetWhoamiCmd(context.Background(), interceptor.GetHTTPClient(), "", nil)
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
		err := whoami.GetWhoamiCmd(context.Background(), interceptor.GetHTTPClient(), "", nil)
		if err == nil {
			t.Errorf("Expected log output, got nothing")
		}
	})
}

func Test_Fetch(t *testing.T) {
	t.Parallel()

	respond := func(status int, body string) *http.Client {
		return (&interceptorHTTPClient{
			f: func(req *http.Request) (*http.Response, error) {
				return &http.Response{StatusCode: status, Body: io.NopCloser(strings.NewReader(body))}, nil
			},
		}).GetHTTPClient()
	}

	t.Run("returns workspace", func(t *testing.T) {
		got, err := whoami.Fetch(context.Background(), respond(http.StatusOK, `{"name":"acme","slug":"acme-slug","plan":"pro"}`), "key")
		if err != nil {
			t.Fatal(err)
		}
		if got.Name != "acme" || got.Slug != "acme-slug" || got.Plan != "pro" {
			t.Errorf("unexpected workspace: %+v", got)
		}
	})

	t.Run("rejected token is ErrUnauthorized", func(t *testing.T) {
		_, err := whoami.Fetch(context.Background(), respond(http.StatusUnauthorized, ""), "key")
		if !errors.Is(err, whoami.ErrUnauthorized) {
			t.Errorf("expected ErrUnauthorized, got %v", err)
		}
	})

	t.Run("server error is not ErrUnauthorized", func(t *testing.T) {
		_, err := whoami.Fetch(context.Background(), respond(http.StatusBadGateway, ""), "key")
		if err == nil || errors.Is(err, whoami.ErrUnauthorized) {
			t.Errorf("expected a non-auth error, got %v", err)
		}
	})
}
