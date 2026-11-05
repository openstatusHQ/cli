package monitors_test

import (
	"bytes"
	"context"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"testing"

	"github.com/openstatusHQ/cli/internal/monitors"
)

func Test_listMonitorResponseLogs_Heartbeat(t *testing.T) {
	t.Parallel()

	const heartbeat = `{"monitor":{"heartbeat":{"id":"7","name":"Nightly backup","period":"86400","grace":"1800"}}}`
	const httpMonitor = `{"monitor":{"http":{"id":"7","name":"API","url":"https://example.com"}}}`

	respond := func(status int, body string) *http.Response {
		return &http.Response{
			StatusCode: status,
			Body:       io.NopCloser(strings.NewReader(body)),
			Header:     http.Header{"Content-Type": []string{"application/json"}},
		}
	}

	for _, tc := range []struct {
		name          string
		logsStatus    int
		logsBody      string
		getStatus     int
		getBody       string
		wantHeartbeat bool
		wantInError   string
	}{
		{
			name:          "type mismatch on a heartbeat is explained",
			logsStatus:    http.StatusBadRequest,
			logsBody:      `{"code":"invalid_argument","message":"Monitor type mismatch: expected http, got heartbeat"}`,
			getStatus:     http.StatusOK,
			getBody:       heartbeat,
			wantHeartbeat: true,
		},
		{
			name:          "plan rejection on a heartbeat is explained",
			logsStatus:    http.StatusForbidden,
			logsBody:      `{"code":"permission_denied","message":"Upgrade for response logs"}`,
			getStatus:     http.StatusOK,
			getBody:       heartbeat,
			wantHeartbeat: true,
		},
		{
			name:        "other monitor types keep the API error",
			logsStatus:  http.StatusForbidden,
			logsBody:    `{"code":"permission_denied","message":"Upgrade for response logs"}`,
			getStatus:   http.StatusOK,
			getBody:     httpMonitor,
			wantInError: "permission denied",
		},
		{
			name:        "failed lookup keeps the API error",
			logsStatus:  http.StatusBadRequest,
			logsBody:    `{"code":"invalid_argument","message":"limit out of range"}`,
			getStatus:   http.StatusInternalServerError,
			getBody:     `{"code":"internal","message":"boom"}`,
			wantInError: "limit out of range",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			interceptor := &interceptorHTTPClient{
				f: func(req *http.Request) (*http.Response, error) {
					if strings.HasSuffix(req.URL.Path, "GetMonitor") {
						return respond(tc.getStatus, tc.getBody), nil
					}
					return respond(tc.logsStatus, tc.logsBody), nil
				},
			}

			err := monitors.ListMonitorResponseLogsWithHTTPClient(context.Background(), interceptor.GetHTTPClient(), "test-token", "7", 0, 0, "", "")
			if err == nil {
				t.Fatal("expected an error")
			}
			if got := strings.Contains(err.Error(), "heartbeat monitor"); got != tc.wantHeartbeat {
				t.Errorf("heartbeat explanation = %t, want %t: %v", got, tc.wantHeartbeat, err)
			}
			if !strings.Contains(err.Error(), tc.wantInError) {
				t.Errorf("expected %q in error: %v", tc.wantInError, err)
			}
		})
	}
}

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
