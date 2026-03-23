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

