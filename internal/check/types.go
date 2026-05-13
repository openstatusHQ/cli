package check

import (
	"errors"
	"fmt"
	"time"
)

type Payload struct {
	URL     string            `json:"url"`
	Method  string            `json:"method,omitempty"`
	Headers map[string]string `json:"headers,omitempty"`
	Body    string            `json:"body,omitempty"`
}

type Timing struct {
	DNS        int64 `json:"dns"`
	Connection int64 `json:"connection"`
	TLS        int64 `json:"tls"`
	TTFB       int64 `json:"ttfb"`
	Transfer   int64 `json:"transfer"`
}

type RegionResult struct {
	Region    string  `json:"region"`
	State     string  `json:"state"`
	Status    int     `json:"status,omitempty"`
	Latency   int64   `json:"latency,omitempty"`
	Timestamp int64   `json:"timestamp,omitempty"`
	Timing    *Timing `json:"timing,omitempty"`
	Message   string  `json:"message,omitempty"`
}

func (r RegionResult) Succeeded() bool {
	return r.State == "success"
}

type RateLimitError struct {
	RetryAfter time.Duration
	Message    string
}

func (e *RateLimitError) Error() string {
	if e.RetryAfter > 0 {
		return fmt.Sprintf("rate limited: retry after %s", e.RetryAfter)
	}
	return "rate limited"
}

type BadRequestError struct {
	Status  int
	Body    string
	Message string
}

func (e *BadRequestError) Error() string {
	if e.Message != "" {
		return e.Message
	}
	return fmt.Sprintf("bad request (HTTP %d)", e.Status)
}

type ServerError struct {
	Status int
	Body   string
}

func (e *ServerError) Error() string {
	return fmt.Sprintf("checker temporarily unavailable (HTTP %d)", e.Status)
}

var ErrStreamTruncated = errors.New("stream ended before check-id")
