package notification_test

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"testing"

	"github.com/openstatusHQ/cli/internal/notification"
)

func Test_GetNotificationInfo(t *testing.T) {
	t.Parallel()

	t.Run("Successfully returns notification with slack data", func(t *testing.T) {
		body := `{"notification":{"id":"1","name":"Slack Alerts","provider":"NOTIFICATION_PROVIDER_SLACK","data":{"slack":{"webhookUrl":"https://hooks.slack.com/services/T00/B00/xxx"}},"monitorIds":["m1","m2"],"createdAt":"2026-01-01T00:00:00Z","updatedAt":"2026-01-01T00:00:00Z"}}`
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

		err := notification.GetNotificationInfoWithHTTPClient(context.Background(), interceptor.GetHTTPClient(), "test-token", "1")
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
	})

	t.Run("Successfully returns notification with opsgenie data", func(t *testing.T) {
		body := `{"notification":{"id":"2","name":"OpsGenie Alerts","provider":"NOTIFICATION_PROVIDER_OPSGENIE","data":{"opsgenie":{"apiKey":"test-key","region":"OPSGENIE_REGION_EU"}},"monitorIds":[],"createdAt":"2026-01-01T00:00:00Z","updatedAt":"2026-01-01T00:00:00Z"}}`
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

		err := notification.GetNotificationInfoWithHTTPClient(context.Background(), interceptor.GetHTTPClient(), "test-token", "2")
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
	})

	t.Run("Missing notification ID returns error", func(t *testing.T) {
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

		err := notification.GetNotificationInfoWithHTTPClient(context.Background(), interceptor.GetHTTPClient(), "test-token", "")
		if err == nil {
			t.Error("Expected error for empty notification ID, got nil")
		}
		if err.Error() != "notification ID is required" {
			t.Errorf("Expected 'notification ID is required' error, got %v", err)
		}
	})

	t.Run("Notification not found returns error", func(t *testing.T) {
		body := `{"code":"not_found","message":"notification not found"}`
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

		err := notification.GetNotificationInfoWithHTTPClient(context.Background(), interceptor.GetHTTPClient(), "test-token", "999")
		if err == nil {
			t.Error("Expected error, got nil")
		}
	})
}
