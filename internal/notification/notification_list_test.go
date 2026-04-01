package notification_test

import (
	"bytes"
	"context"
	"io"
	"log"
	"net/http"
	"os"
	"testing"

	"github.com/openstatusHQ/cli/internal/notification"
)

func Test_ListNotifications(t *testing.T) {
	t.Parallel()

	t.Run("Successfully returns notifications", func(t *testing.T) {
		body := `{"notifications":[{"id":"1","name":"Slack Alerts","provider":"NOTIFICATION_PROVIDER_SLACK","monitorCount":3,"createdAt":"2026-01-01T00:00:00Z","updatedAt":"2026-01-01T00:00:00Z"},{"id":"2","name":"Email Alerts","provider":"NOTIFICATION_PROVIDER_EMAIL","monitorCount":0,"createdAt":"2026-02-01T00:00:00Z","updatedAt":"2026-02-01T00:00:00Z"}],"totalSize":2}`
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
		err := notification.ListNotificationsWithHTTPClient(context.Background(), interceptor.GetHTTPClient(), "test-token", 0)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
	})

	t.Run("Returns empty list", func(t *testing.T) {
		body := `{"notifications":[],"totalSize":0}`
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

		err := notification.ListNotificationsWithHTTPClient(context.Background(), interceptor.GetHTTPClient(), "test-token", 0)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
	})

	t.Run("Respects limit parameter", func(t *testing.T) {
		body := `{"notifications":[{"id":"1","name":"Slack Alerts","provider":"NOTIFICATION_PROVIDER_SLACK","monitorCount":1,"createdAt":"2026-01-01T00:00:00Z","updatedAt":"2026-01-01T00:00:00Z"}],"totalSize":1}`
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
		err := notification.ListNotificationsWithHTTPClient(context.Background(), interceptor.GetHTTPClient(), "test-token", 5)
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

		err := notification.ListNotificationsWithHTTPClient(context.Background(), interceptor.GetHTTPClient(), "test-token", 0)
		if err == nil {
			t.Error("Expected error, got nil")
		}
	})
}
