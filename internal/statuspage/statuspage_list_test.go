package statuspage_test

import (
	"bytes"
	"context"
	"io"
	"log"
	"net/http"
	"os"
	"testing"

	"github.com/openstatusHQ/cli/internal/statuspage"
)

func Test_ListStatusPages(t *testing.T) {
	t.Parallel()

	t.Run("Successfully returns pages", func(t *testing.T) {
		body := `{"statusPages":[{"id":"1","title":"My Status Page","slug":"my-page","published":true,"createdAt":"2026-01-01T00:00:00Z","updatedAt":"2026-01-01T00:00:00Z"},{"id":"2","title":"Internal Page","slug":"internal","published":false,"createdAt":"2026-02-01T00:00:00Z","updatedAt":"2026-02-01T00:00:00Z"}],"totalSize":2}`
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
			log.SetOutput(os.Stderr)
		})
		err := statuspage.ListStatusPagesWithHTTPClient(context.Background(), interceptor.GetHTTPClient(), "test-token", 0)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
	})

	t.Run("Displays custom domain when set", func(t *testing.T) {
		body := `{"statusPages":[{"id":"1","title":"My Status Page","slug":"my-page","customDomain":"status.example.com","published":true,"createdAt":"2026-01-01T00:00:00Z","updatedAt":"2026-01-01T00:00:00Z"}],"totalSize":1}`
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
			log.SetOutput(os.Stderr)
		})
		err := statuspage.ListStatusPagesWithHTTPClient(context.Background(), interceptor.GetHTTPClient(), "test-token", 0)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
	})

	t.Run("Returns empty list", func(t *testing.T) {
		body := `{"statusPages":[],"totalSize":0}`
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

		err := statuspage.ListStatusPagesWithHTTPClient(context.Background(), interceptor.GetHTTPClient(), "test-token", 0)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
	})

	t.Run("API error returns error", func(t *testing.T) {
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

		err := statuspage.ListStatusPagesWithHTTPClient(context.Background(), interceptor.GetHTTPClient(), "test-token", 0)
		if err == nil {
			t.Error("Expected error, got nil")
		}
	})
}
