package check_test

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/openstatusHQ/cli/internal/check"
)

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func newClient(rt roundTripperFunc) *http.Client {
	return &http.Client{Transport: rt}
}

func mustReadFixture(t *testing.T, name string) []byte {
	t.Helper()
	data, err := os.ReadFile("testdata/" + name)
	if err != nil {
		t.Fatalf("read fixture %s: %v", name, err)
	}
	return data
}

func TestRun_HappyPath(t *testing.T) {
	t.Parallel()
	fixture := mustReadFixture(t, "happy.ndjson")

	rt := roundTripperFunc(func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodPost {
			t.Errorf("method = %q, want POST", req.Method)
		}
		if got := req.Header.Get("Content-Type"); got != "application/json" {
			t.Errorf("Content-Type = %q, want application/json", got)
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(bytes.NewReader(fixture)),
			Header:     make(http.Header),
		}, nil
	})

	var rows []check.RegionResult
	results, id, err := check.Run(context.Background(), newClient(rt), check.Payload{URL: "https://example.com"}, func(r check.RegionResult) {
		rows = append(rows, r)
	})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if len(results) != 28 {
		t.Errorf("results = %d, want 28", len(results))
	}
	if len(rows) != 28 {
		t.Errorf("onRow calls = %d, want 28", len(rows))
	}
	if id == "" {
		t.Errorf("checkID empty, want non-empty hex")
	}
	for _, r := range results {
		if !r.Succeeded() {
			t.Errorf("region %q state %q, want success", r.Region, r.State)
		}
	}
}

func TestRun_BadURL_FailureRows(t *testing.T) {
	t.Parallel()
	fixture := mustReadFixture(t, "bad_url.ndjson")

	rt := roundTripperFunc(func(*http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(bytes.NewReader(fixture)),
			Header:     make(http.Header),
		}, nil
	})

	results, id, err := check.Run(context.Background(), newClient(rt), check.Payload{URL: "https://invalid.example"}, nil)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if id == "" {
		t.Errorf("checkID empty")
	}
	if len(results) != 28 {
		t.Errorf("results = %d, want 28", len(results))
	}
	for _, r := range results {
		if r.Succeeded() {
			t.Errorf("region %q unexpectedly succeeded", r.Region)
		}
		if r.Message == "" {
			t.Errorf("region %q missing message", r.Region)
		}
		if r.Status != 0 || r.Latency != 0 || r.Timing != nil {
			t.Errorf("region %q has unexpected success fields", r.Region)
		}
	}
}

func TestRun_RateLimit(t *testing.T) {
	t.Parallel()
	rt := roundTripperFunc(func(*http.Request) (*http.Response, error) {
		hdr := make(http.Header)
		hdr.Set("Retry-After", "16")
		return &http.Response{
			StatusCode: http.StatusTooManyRequests,
			Header:     hdr,
			Body:       io.NopCloser(strings.NewReader(`{"error":"You have exceeded the rate limit of 3 requests per 60 seconds","code":"RATE_LIMIT_EXCEEDED","reset":0}`)),
		}, nil
	})

	_, _, err := check.Run(context.Background(), newClient(rt), check.Payload{URL: "https://example.com"}, nil)
	var rl *check.RateLimitError
	if !errors.As(err, &rl) {
		t.Fatalf("err = %v, want *RateLimitError", err)
	}
	if rl.RetryAfter != 16*time.Second {
		t.Errorf("retry-after = %s, want 16s", rl.RetryAfter)
	}
	if !strings.Contains(rl.Message, "rate limit") {
		t.Errorf("message = %q, want to contain \"rate limit\"", rl.Message)
	}
}

func TestRun_RateLimit_ResetFallback(t *testing.T) {
	t.Parallel()
	resetAt := time.Now().Add(20 * time.Second).UnixMilli()
	body := `{"error":"limited","reset":` + itoa(resetAt) + `}`

	rt := roundTripperFunc(func(*http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusTooManyRequests,
			Header:     make(http.Header),
			Body:       io.NopCloser(strings.NewReader(body)),
		}, nil
	})

	_, _, err := check.Run(context.Background(), newClient(rt), check.Payload{URL: "https://example.com"}, nil)
	var rl *check.RateLimitError
	if !errors.As(err, &rl) {
		t.Fatalf("err = %v, want *RateLimitError", err)
	}
	if rl.RetryAfter <= 0 || rl.RetryAfter > 25*time.Second {
		t.Errorf("retry-after = %s, want ~20s", rl.RetryAfter)
	}
}

func TestRun_BadRequest(t *testing.T) {
	t.Parallel()
	rt := roundTripperFunc(func(*http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusBadRequest,
			Header:     make(http.Header),
			Body:       io.NopCloser(strings.NewReader(`{"error":"could not determine client IP"}`)),
		}, nil
	})

	_, _, err := check.Run(context.Background(), newClient(rt), check.Payload{URL: "https://example.com"}, nil)
	var br *check.BadRequestError
	if !errors.As(err, &br) {
		t.Fatalf("err = %v, want *BadRequestError", err)
	}
	if !strings.Contains(br.Message, "client IP") {
		t.Errorf("message = %q, want to contain client IP", br.Message)
	}
}

func TestRun_ServerError(t *testing.T) {
	t.Parallel()
	rt := roundTripperFunc(func(*http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusServiceUnavailable,
			Header:     make(http.Header),
			Body:       io.NopCloser(strings.NewReader("upstream down")),
		}, nil
	})

	_, _, err := check.Run(context.Background(), newClient(rt), check.Payload{URL: "https://example.com"}, nil)
	var se *check.ServerError
	if !errors.As(err, &se) {
		t.Fatalf("err = %v, want *ServerError", err)
	}
	if se.Status != http.StatusServiceUnavailable {
		t.Errorf("status = %d, want 503", se.Status)
	}
}

func TestRun_TruncatedStream(t *testing.T) {
	t.Parallel()
	fixture := mustReadFixture(t, "happy.ndjson")
	cut := bytes.SplitN(fixture, []byte("\n"), 10)
	truncated := bytes.Join(cut[:9], []byte("\n"))

	rt := roundTripperFunc(func(*http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(bytes.NewReader(truncated)),
			Header:     make(http.Header),
		}, nil
	})

	results, id, err := check.Run(context.Background(), newClient(rt), check.Payload{URL: "https://example.com"}, nil)
	if !errors.Is(err, check.ErrStreamTruncated) {
		t.Fatalf("err = %v, want ErrStreamTruncated", err)
	}
	if id != "" {
		t.Errorf("checkID = %q, want empty", id)
	}
	if len(results) != 9 {
		t.Errorf("results = %d, want 9", len(results))
	}
}

func TestRun_NonJSONLineTerminatesStream(t *testing.T) {
	t.Parallel()
	body := `{"region":"fra","state":"success","status":200,"latency":10,"timing":{"dns":1,"connection":1,"tls":1,"ttfb":7,"transfer":0},"index":0}
abc123checkid
{"region":"iad","state":"success","status":200,"latency":20,"timing":{"dns":1,"connection":1,"tls":1,"ttfb":17,"transfer":0},"index":1}`

	rt := roundTripperFunc(func(*http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(body)),
			Header:     make(http.Header),
		}, nil
	})

	results, id, err := check.Run(context.Background(), newClient(rt), check.Payload{URL: "https://example.com"}, nil)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if len(results) != 1 {
		t.Errorf("results = %d, want 1 (parser stops at first non-JSON line)", len(results))
	}
	if id != "abc123checkid" {
		t.Errorf("checkID = %q, want \"abc123checkid\"", id)
	}
}

func TestRun_UnparseableJSONSkipped(t *testing.T) {
	t.Parallel()
	body := `{"region":"fra","state":"success","status":200,"latency":10,"timing":{"dns":1,"connection":1,"tls":1,"ttfb":7,"transfer":0},"index":0}
{"region":bad json
{"region":"iad","state":"success","status":200,"latency":20,"timing":{"dns":1,"connection":1,"tls":1,"ttfb":17,"transfer":0},"index":1}
abc123checkid`

	rt := roundTripperFunc(func(*http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(body)),
			Header:     make(http.Header),
		}, nil
	})

	results, id, err := check.Run(context.Background(), newClient(rt), check.Payload{URL: "https://example.com"}, nil)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if len(results) != 2 {
		t.Errorf("results = %d, want 2 (unparseable {…} line skipped, others kept)", len(results))
	}
	if id != "abc123checkid" {
		t.Errorf("checkID = %q, want \"abc123checkid\"", id)
	}
}

func TestRun_ContextCancel(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())

	rt := roundTripperFunc(func(req *http.Request) (*http.Response, error) {
		<-req.Context().Done()
		return nil, req.Context().Err()
	})

	go func() {
		time.Sleep(10 * time.Millisecond)
		cancel()
	}()

	_, _, err := check.Run(ctx, newClient(rt), check.Payload{URL: "https://example.com"}, nil)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("err = %v, want context.Canceled", err)
	}
}

func itoa(n int64) string {
	if n == 0 {
		return "0"
	}
	var buf [20]byte
	i := len(buf)
	neg := n < 0
	if neg {
		n = -n
	}
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}
