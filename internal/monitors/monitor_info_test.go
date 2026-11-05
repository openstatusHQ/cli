package monitors_test

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"testing"

	monitorv1 "buf.build/gen/go/openstatus/api/protocolbuffers/go/openstatus/monitor/v1"

	output "github.com/openstatusHQ/cli/internal/cli"
	"github.com/openstatusHQ/cli/internal/monitors"
)

type routeEntry struct {
	suffix string
	body   string
}

func monitorInfoInterceptor(routes []routeEntry) *interceptorHTTPClient {
	return &interceptorHTTPClient{
		f: func(req *http.Request) (*http.Response, error) {
			for _, r := range routes {
				if strings.HasSuffix(req.URL.Path, r.suffix) {
					return &http.Response{
						StatusCode: http.StatusOK,
						Body:       io.NopCloser(bytes.NewReader([]byte(r.body))),
						Header:     http.Header{"Content-Type": []string{"application/json"}},
					}, nil
				}
			}
			return &http.Response{
				StatusCode: http.StatusInternalServerError,
				Body:       io.NopCloser(bytes.NewReader([]byte(`{"code":"internal","message":"unexpected call"}`))),
				Header:     http.Header{"Content-Type": []string{"application/json"}},
			}, nil
		},
	}
}

// Not parallel: subtests redirect the process-wide os.Stdout.
func Test_getMonitorInfo_Heartbeat(t *testing.T) {
	const secret = "secret-ping-token"
	const pinged = `{"monitor":{"heartbeat":{"id":"7","name":"Nightly backup","description":"Dumps the database","period":"86400","grace":"1800","active":true,"status":"MONITOR_STATUS_ERROR","lastPingAt":"2026-11-05T02:00:00.000Z","lastPingKind":"HEARTBEAT_PING_KIND_FAIL","pingUrl":"https://api.openstatus.dev/heartbeat/` + secret + `"}}}`
	const summary = `{"id":"7","lastPingAt":"2026-11-05T02:00:00.000Z","totalSuccessful":"6","totalFailed":"1","timeRange":"TIME_RANGE_7D"}`

	t.Run("Shows schedule, last ping and ping totals", func(t *testing.T) {
		interceptor := monitorInfoInterceptor([]routeEntry{
			{"GetMonitor", pinged},
			{"GetMonitorSummary", summary},
		})

		out := captureStdout(t, func() {
			if err := monitors.GetMonitorInfo(context.Background(), interceptor.GetHTTPClient(), "test", "7", monitorv1.TimeRange_TIME_RANGE_7D, "7d", nil); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})

		for _, want := range []string{"Nightly backup", "heartbeat", "● error", "1d", "30m", "2026-11-05T02:00:00.000Z (fail)", "Summary (7d)", "Successful Pings", "Failed Pings"} {
			if !strings.Contains(out, want) {
				t.Errorf("expected %q in output, got:\n%s", want, out)
			}
		}
		for _, unwanted := range []string{secret, "Latency", "Region Status", "Endpoint", "Frequency", "Timeout"} {
			if strings.Contains(out, unwanted) {
				t.Errorf("did not expect %q in output, got:\n%s", unwanted, out)
			}
		}
	})

	t.Run("Before the first ping", func(t *testing.T) {
		interceptor := monitorInfoInterceptor([]routeEntry{
			{"GetMonitor", `{"monitor":{"heartbeat":{"id":"7","name":"New job","period":"3600","grace":"90","status":"MONITOR_STATUS_ACTIVE"}}}`},
			{"GetMonitorSummary", `{"id":"7"}`},
		})

		out := captureStdout(t, func() {
			if err := monitors.GetMonitorInfo(context.Background(), interceptor.GetHTTPClient(), "test", "7", monitorv1.TimeRange_TIME_RANGE_1D, "1d", nil); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})

		for _, want := range []string{"never", "1h", "90s"} {
			if !strings.Contains(out, want) {
				t.Errorf("expected %q in output, got:\n%s", want, out)
			}
		}
	})

	t.Run("Still shows the monitor when the summary fails", func(t *testing.T) {
		interceptor := monitorInfoInterceptor([]routeEntry{{"GetMonitor", pinged}})

		out := captureStdout(t, func() {
			if err := monitors.GetMonitorInfo(context.Background(), interceptor.GetHTTPClient(), "test", "7", monitorv1.TimeRange_TIME_RANGE_1D, "1d", nil); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})

		if !strings.Contains(out, "Nightly backup") {
			t.Errorf("expected the monitor in output, got:\n%s", out)
		}
		if strings.Contains(out, "Summary") {
			t.Errorf("did not expect a summary, got:\n%s", out)
		}
	})

	t.Run("JSON output", func(t *testing.T) {
		interceptor := monitorInfoInterceptor([]routeEntry{
			{"GetMonitor", pinged},
			{"GetMonitorSummary", summary},
		})
		output.SetJSONOutput(true)
		t.Cleanup(func() { output.SetJSONOutput(false) })

		out := captureStdout(t, func() {
			if err := monitors.GetMonitorInfo(context.Background(), interceptor.GetHTTPClient(), "test", "7", monitorv1.TimeRange_TIME_RANGE_7D, "7d", nil); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})

		for _, want := range []string{`"jobType": "heartbeat"`, `"period": 86400`, `"grace": 1800`, `"last_ping_kind": "fail"`, `"status": "error"`, `"total_failed": 1`} {
			if !strings.Contains(out, want) {
				t.Errorf("expected %q in JSON output, got:\n%s", want, out)
			}
		}
		if strings.Contains(out, secret) {
			t.Errorf("ping URL leaked into JSON output:\n%s", out)
		}
	})
}

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
		err := monitors.GetMonitorInfo(context.Background(), interceptor.GetHTTPClient(), "", "", monitorv1.TimeRange_TIME_RANGE_1D, "1d", nil)
		if err == nil {
			t.Errorf("Expected error, got nil")
		}
	})

	t.Run("Should work with all three RPCs", func(t *testing.T) {
		interceptor := monitorInfoInterceptor([]routeEntry{
			{"GetMonitor", `{"monitor":{"http":{"id":"2260","name":"Vercel Checker Edge","description":"","url":"https://www.openstatus.dev","periodicity":"PERIODICITY_10M","method":"HTTP_METHOD_GET","regions":["REGION_FLY_IAD","REGION_FLY_JNB","REGION_FLY_SYD","REGION_FLY_GRU"],"active":false,"public":false,"timeout":45000}}}`},
			{"GetMonitorStatus", `{"id":"2260","regions":[{"region":"REGION_FLY_IAD","status":"MONITOR_STATUS_ACTIVE"},{"region":"REGION_FLY_JNB","status":"MONITOR_STATUS_ERROR"}]}`},
			{"GetMonitorSummary", `{"id":"2260","totalSuccessful":"1420","totalDegraded":"3","totalFailed":"1","p50":"120","p75":"180","p90":"250","p95":"300","p99":"500","timeRange":"TIME_RANGE_1D","lastPingAt":"2026-03-23T10:00:00Z"}`},
		})

		var bf bytes.Buffer
		log.SetOutput(&bf)
		t.Cleanup(func() {
			log.SetOutput(os.Stdout)
		})
		err := monitors.GetMonitorInfo(context.Background(), interceptor.GetHTTPClient(), "test", "2260", monitorv1.TimeRange_TIME_RANGE_1D, "1d", nil)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
	})

	t.Run("Should populate all fields including degradedAt, body, headers, and assertions", func(t *testing.T) {
		interceptor := monitorInfoInterceptor([]routeEntry{
			{"GetMonitor", `{"monitor":{"http":{"id":"2260","name":"Full Monitor","description":"test desc","url":"https://www.openstatus.dev","periodicity":"PERIODICITY_10M","method":"HTTP_METHOD_POST","regions":["REGION_FLY_IAD"],"active":true,"public":true,"timeout":45000,"degradedAt":"30000","body":"{\"key\":\"value\"}","headers":[{"key":"Content-Type","value":"application/json"},{"key":"Authorization","value":"Bearer token"}],"statusCodeAssertions":[{"target":200,"comparator":"NUMBER_COMPARATOR_EQUAL"}],"bodyAssertions":[{"target":"ok","comparator":"STRING_COMPARATOR_CONTAINS"}],"headerAssertions":[{"key":"x-custom","target":"expected","comparator":"STRING_COMPARATOR_EQUAL"}]}}}`},
			{"GetMonitorStatus", `{"id":"2260","regions":[{"region":"REGION_FLY_IAD","status":"MONITOR_STATUS_ACTIVE"}]}`},
			{"GetMonitorSummary", `{"id":"2260","totalSuccessful":"100","totalDegraded":"0","totalFailed":"0","p50":"50","p75":"75","p90":"90","p95":"95","p99":"99","timeRange":"TIME_RANGE_1D"}`},
		})

		var bf bytes.Buffer
		log.SetOutput(&bf)
		t.Cleanup(func() {
			log.SetOutput(os.Stdout)
		})
		err := monitors.GetMonitorInfo(context.Background(), interceptor.GetHTTPClient(), "test", "2260", monitorv1.TimeRange_TIME_RANGE_1D, "1d", nil)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
	})

	t.Run("Should work with gRPC monitor", func(t *testing.T) {
		interceptor := monitorInfoInterceptor([]routeEntry{
			{"GetMonitor", `{"monitor":{"grpc":{"id":"4001","name":"gRPC Check","uri":"grpc.example.com:443","service":"health.v1","periodicity":"PERIODICITY_5M","regions":["REGION_FLY_IAD"],"active":true,"timeout":10000}}}`},
			{"GetMonitorStatus", `{"id":"4001","regions":[{"region":"REGION_FLY_IAD","status":"MONITOR_STATUS_ACTIVE"}]}`},
			{"GetMonitorSummary", `{"id":"4001","totalSuccessful":"10","totalDegraded":"0","totalFailed":"0","p50":"20","p75":"30","p90":"40","p95":"50","p99":"60","timeRange":"TIME_RANGE_1D"}`},
		})

		err := monitors.GetMonitorInfo(context.Background(), interceptor.GetHTTPClient(), "test", "4001", monitorv1.TimeRange_TIME_RANGE_1D, "1d", nil)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
	})

	t.Run("Should work with TCP monitor and degradedAt", func(t *testing.T) {
		interceptor := monitorInfoInterceptor([]routeEntry{
			{"GetMonitor", `{"monitor":{"tcp":{"id":"3001","name":"TCP Check","description":"tcp test","uri":"example.com:443","periodicity":"PERIODICITY_5M","regions":["REGION_FLY_IAD"],"active":true,"public":false,"timeout":10000,"degradedAt":"5000"}}}`},
			{"GetMonitorStatus", `{"id":"3001","regions":[{"region":"REGION_FLY_IAD","status":"MONITOR_STATUS_ACTIVE"}]}`},
			{"GetMonitorSummary", `{"id":"3001","totalSuccessful":"500","totalDegraded":"0","totalFailed":"0","p50":"20","p75":"30","p90":"40","p95":"50","p99":"60","timeRange":"TIME_RANGE_1D"}`},
		})

		var bf bytes.Buffer
		log.SetOutput(&bf)
		t.Cleanup(func() {
			log.SetOutput(os.Stdout)
		})
		err := monitors.GetMonitorInfo(context.Background(), interceptor.GetHTTPClient(), "test", "3001", monitorv1.TimeRange_TIME_RANGE_1D, "1d", nil)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
	})

	t.Run("Should work with ICMP monitor and degradedAt", func(t *testing.T) {
		interceptor := monitorInfoInterceptor([]routeEntry{
			{"GetMonitor", `{"monitor":{"icmp":{"id":"4001","name":"Gateway Ping","uri":"8.8.8.8","periodicity":"PERIODICITY_5M","regions":["REGION_FLY_IAD"],"active":true,"public":false,"timeout":10000,"degradedAt":"5000"}}}`},
			{"GetMonitorStatus", `{"id":"4001","regions":[{"region":"REGION_FLY_IAD","status":"MONITOR_STATUS_ACTIVE"}]}`},
			{"GetMonitorSummary", `{"id":"4001","totalSuccessful":"500","totalDegraded":"0","totalFailed":"0","p50":"20","p75":"30","p90":"40","p95":"50","p99":"60","timeRange":"TIME_RANGE_1D"}`},
		})

		var bf bytes.Buffer
		log.SetOutput(&bf)
		t.Cleanup(func() {
			log.SetOutput(os.Stdout)
		})
		err := monitors.GetMonitorInfo(context.Background(), interceptor.GetHTTPClient(), "test", "4001", monitorv1.TimeRange_TIME_RANGE_1D, "1d", nil)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
	})

	t.Run("Should gracefully degrade when status RPC fails", func(t *testing.T) {
		interceptor := &interceptorHTTPClient{
			f: func(req *http.Request) (*http.Response, error) {
				if strings.HasSuffix(req.URL.Path, "GetMonitor") {
					return &http.Response{
						StatusCode: http.StatusOK,
						Body:       io.NopCloser(bytes.NewReader([]byte(`{"monitor":{"http":{"id":"2260","name":"Test","description":"","url":"https://example.com","periodicity":"PERIODICITY_1M","method":"HTTP_METHOD_GET","regions":["REGION_FLY_IAD"],"active":true,"public":false,"timeout":30000}}}`))),
						Header:     http.Header{"Content-Type": []string{"application/json"}},
					}, nil
				}
				if strings.HasSuffix(req.URL.Path, "GetMonitorStatus") {
					return &http.Response{
						StatusCode: http.StatusInternalServerError,
						Body:       io.NopCloser(bytes.NewReader([]byte(`{"code":"internal","message":"status unavailable"}`))),
						Header:     http.Header{"Content-Type": []string{"application/json"}},
					}, nil
				}
				if strings.HasSuffix(req.URL.Path, "GetMonitorSummary") {
					return &http.Response{
						StatusCode: http.StatusOK,
						Body:       io.NopCloser(bytes.NewReader([]byte(`{"id":"2260","totalSuccessful":"10","totalDegraded":"0","totalFailed":"0","p50":"50","p75":"75","p90":"90","p95":"95","p99":"99","timeRange":"TIME_RANGE_1D"}`))),
						Header:     http.Header{"Content-Type": []string{"application/json"}},
					}, nil
				}
				return nil, fmt.Errorf("unexpected request: %s", req.URL.Path)
			},
		}

		var bf bytes.Buffer
		log.SetOutput(&bf)
		t.Cleanup(func() {
			log.SetOutput(os.Stdout)
		})
		err := monitors.GetMonitorInfo(context.Background(), interceptor.GetHTTPClient(), "test", "2260", monitorv1.TimeRange_TIME_RANGE_1D, "1d", nil)
		if err != nil {
			t.Errorf("Expected no error (graceful degradation), got %v", err)
		}
	})

	t.Run("Should gracefully degrade when summary RPC fails", func(t *testing.T) {
		interceptor := &interceptorHTTPClient{
			f: func(req *http.Request) (*http.Response, error) {
				if strings.HasSuffix(req.URL.Path, "GetMonitor") {
					return &http.Response{
						StatusCode: http.StatusOK,
						Body:       io.NopCloser(bytes.NewReader([]byte(`{"monitor":{"http":{"id":"2260","name":"Test","description":"","url":"https://example.com","periodicity":"PERIODICITY_1M","method":"HTTP_METHOD_GET","regions":["REGION_FLY_IAD"],"active":true,"public":false,"timeout":30000}}}`))),
						Header:     http.Header{"Content-Type": []string{"application/json"}},
					}, nil
				}
				if strings.HasSuffix(req.URL.Path, "GetMonitorStatus") {
					return &http.Response{
						StatusCode: http.StatusOK,
						Body:       io.NopCloser(bytes.NewReader([]byte(`{"id":"2260","regions":[{"region":"REGION_FLY_IAD","status":"MONITOR_STATUS_ACTIVE"}]}`))),
						Header:     http.Header{"Content-Type": []string{"application/json"}},
					}, nil
				}
				if strings.HasSuffix(req.URL.Path, "GetMonitorSummary") {
					return &http.Response{
						StatusCode: http.StatusInternalServerError,
						Body:       io.NopCloser(bytes.NewReader([]byte(`{"code":"internal","message":"summary unavailable"}`))),
						Header:     http.Header{"Content-Type": []string{"application/json"}},
					}, nil
				}
				return nil, fmt.Errorf("unexpected request: %s", req.URL.Path)
			},
		}

		var bf bytes.Buffer
		log.SetOutput(&bf)
		t.Cleanup(func() {
			log.SetOutput(os.Stdout)
		})
		err := monitors.GetMonitorInfo(context.Background(), interceptor.GetHTTPClient(), "test", "2260", monitorv1.TimeRange_TIME_RANGE_1D, "1d", nil)
		if err != nil {
			t.Errorf("Expected no error (graceful degradation), got %v", err)
		}
	})
}
