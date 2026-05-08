package monitors_test

import (
	"bytes"
	"context"
	"io"
	"log"
	"net/http"
	"os"
	"testing"

	"github.com/openstatusHQ/cli/internal/monitors"
)

func Test_listMonitorResponseLogs(t *testing.T) {
	t.Parallel()

	t.Run("Successfully returns logs", func(t *testing.T) {
		body := `{"logs":[{"id":"log-1","latency":123,"statusCode":200,"monitorId":"1","requestStatus":"HTTP_RESPONSE_LOG_REQUEST_STATUS_SUCCESS","region":"REGION_FLY_IAD","cronTimestamp":"1715000000000","trigger":"HTTP_RESPONSE_LOG_TRIGGER_CRON","timestamp":"1715000000000"}],"pagination":{"limit":25,"offset":0,"hasMore":false}}`
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
		err := monitors.ListMonitorResponseLogsWithHTTPClient(context.Background(), interceptor.GetHTTPClient(), "test-token", "1", 0, 0, "", "")
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
	})

	t.Run("Returns empty list", func(t *testing.T) {
		body := `{"logs":[],"pagination":{"limit":25,"offset":0,"hasMore":false}}`
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
		err := monitors.ListMonitorResponseLogsWithHTTPClient(context.Background(), interceptor.GetHTTPClient(), "test-token", "1", 0, 0, "", "")
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

		var bf bytes.Buffer
		log.SetOutput(&bf)
		t.Cleanup(func() {
			log.SetOutput(os.Stdout)
		})
		err := monitors.ListMonitorResponseLogsWithHTTPClient(context.Background(), interceptor.GetHTTPClient(), "test-token", "1", 0, 0, "", "")
		if err == nil {
			t.Error("Expected error, got nil")
		}
	})

	t.Run("Empty monitor ID returns error", func(t *testing.T) {
		interceptor := &interceptorHTTPClient{
			f: func(req *http.Request) (*http.Response, error) {
				return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(bytes.NewReader([]byte(`{}`)))}, nil
			},
		}

		err := monitors.ListMonitorResponseLogsWithHTTPClient(context.Background(), interceptor.GetHTTPClient(), "test-token", "", 0, 0, "", "")
		if err == nil {
			t.Error("Expected error for empty monitor ID, got nil")
		}
	})
}
