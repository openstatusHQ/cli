package statuspage_test

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"testing"

	"github.com/openstatusHQ/cli/internal/statuspage"
)

func Test_GetStatusPageInfo(t *testing.T) {
	t.Parallel()

	t.Run("Successfully returns page with components", func(t *testing.T) {
		body := `{"statusPage":{"id":"1","title":"My Page","description":"Main status page","slug":"my-page","published":true,"accessType":"PAGE_ACCESS_TYPE_PUBLIC","theme":"PAGE_THEME_SYSTEM","createdAt":"2026-01-01T00:00:00Z","updatedAt":"2026-01-01T00:00:00Z"},"components":[{"id":"c1","pageId":"1","name":"API","type":"PAGE_COMPONENT_TYPE_MONITOR","order":1},{"id":"c2","pageId":"1","name":"CDN","type":"PAGE_COMPONENT_TYPE_STATIC","groupId":"g1","order":1,"groupOrder":1}],"groups":[{"id":"g1","pageId":"1","name":"Infrastructure"}]}`
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

		err := statuspage.GetStatusPageInfoWithHTTPClient(context.Background(), interceptor.GetHTTPClient(), "test-token", "1")
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
	})

	t.Run("Successfully returns page without components", func(t *testing.T) {
		body := `{"statusPage":{"id":"2","title":"Empty Page","slug":"empty","published":false,"accessType":"PAGE_ACCESS_TYPE_PUBLIC","theme":"PAGE_THEME_LIGHT","createdAt":"2026-01-01T00:00:00Z","updatedAt":"2026-01-01T00:00:00Z"},"components":[],"groups":[]}`
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

		err := statuspage.GetStatusPageInfoWithHTTPClient(context.Background(), interceptor.GetHTTPClient(), "test-token", "2")
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
	})

	t.Run("Missing page ID returns error", func(t *testing.T) {
		interceptor := &interceptorHTTPClient{
			f: func(req *http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: http.StatusOK,
					Header: http.Header{
						"Content-Type": []string{"application/json"},
					},
				}, nil
			},
		}

		err := statuspage.GetStatusPageInfoWithHTTPClient(context.Background(), interceptor.GetHTTPClient(), "test-token", "")
		if err == nil {
			t.Error("Expected error for empty page ID, got nil")
		}
		if err.Error() != "page ID is required" {
			t.Errorf("Expected 'page ID is required' error, got %v", err)
		}
	})

	t.Run("Uses custom domain as URL", func(t *testing.T) {
		body := `{"statusPage":{"id":"3","title":"Custom Page","slug":"custom","published":true,"customDomain":"status.example.com","accessType":"PAGE_ACCESS_TYPE_PUBLIC","theme":"PAGE_THEME_SYSTEM","createdAt":"2026-01-01T00:00:00Z","updatedAt":"2026-01-01T00:00:00Z"},"components":[],"groups":[]}`
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

		err := statuspage.GetStatusPageInfoWithHTTPClient(context.Background(), interceptor.GetHTTPClient(), "test-token", "3")
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
	})

	t.Run("Strips scheme from custom domain", func(t *testing.T) {
		body := `{"statusPage":{"id":"4","title":"Scheme Page","slug":"scheme","published":true,"customDomain":"https://status.example.com","accessType":"PAGE_ACCESS_TYPE_PUBLIC","theme":"PAGE_THEME_SYSTEM","createdAt":"2026-01-01T00:00:00Z","updatedAt":"2026-01-01T00:00:00Z"},"components":[],"groups":[]}`
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

		err := statuspage.GetStatusPageInfoWithHTTPClient(context.Background(), interceptor.GetHTTPClient(), "test-token", "4")
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
	})

	t.Run("Page not found returns actionable error", func(t *testing.T) {
		body := `{"code":"not_found","message":"status page not found"}`
		r := io.NopCloser(bytes.NewReader([]byte(body)))

		interceptor := &interceptorHTTPClient{
			f: func(req *http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: http.StatusNotFound,
					Body:       r,
					Header: http.Header{
						"Content-Type": []string{"application/json"},
					},
				}, nil
			},
		}

		err := statuspage.GetStatusPageInfoWithHTTPClient(context.Background(), interceptor.GetHTTPClient(), "test-token", "999")
		if err == nil {
			t.Error("Expected error, got nil")
		}
	})
}
