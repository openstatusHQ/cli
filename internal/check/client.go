package check

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/openstatusHQ/cli/internal/api"
	output "github.com/openstatusHQ/cli/internal/cli"
)

const defaultTimeout = 30 * time.Second

func debugWriter() io.Writer { return os.Stderr }

type OnRow func(RegionResult)

func Run(ctx context.Context, client *http.Client, payload Payload, onRow OnRow) ([]RegionResult, string, error) {
	if onRow == nil {
		onRow = func(RegionResult) {}
	}
	if client == nil {
		client = &http.Client{Timeout: defaultTimeout}
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, "", fmt.Errorf("encode payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, api.PlayCheckerURL+"?compact=true", bytes.NewReader(body))
	if err != nil {
		return nil, "", fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	if output.IsDebug() {
		fmt.Fprintf(debugWriter(), "[debug] POST %s\n", req.URL.String())
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()

	if err := classifyHTTPError(resp); err != nil {
		return nil, "", err
	}

	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 64*1024), 1<<20)

	var results []RegionResult
	var checkID string

	for scanner.Scan() {
		line := scanner.Bytes()
		trimmed := bytes.TrimSpace(line)
		if len(trimmed) == 0 {
			continue
		}
		if trimmed[0] != '{' {
			checkID = string(trimmed)
			break
		}
		var row RegionResult
		if err := json.Unmarshal(trimmed, &row); err != nil {
			if output.IsDebug() {
				fmt.Fprintf(debugWriter(), "[debug] skipping unparseable line: %v\n", err)
			}
			continue
		}
		results = append(results, row)
		onRow(row)
	}

	if err := scanner.Err(); err != nil {
		return results, checkID, fmt.Errorf("read stream: %w", err)
	}

	if checkID == "" {
		return results, "", ErrStreamTruncated
	}

	return results, checkID, nil
}

func classifyHTTPError(resp *http.Response) error {
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return nil
	}
	body, _ := io.ReadAll(resp.Body)
	bodyStr := string(body)

	switch {
	case resp.StatusCode == http.StatusTooManyRequests:
		return &RateLimitError{
			RetryAfter: parseRetryAfter(resp.Header.Get("Retry-After"), bodyStr),
			Message:    extractErrorMessage(bodyStr),
		}
	case resp.StatusCode >= 400 && resp.StatusCode < 500:
		return &BadRequestError{
			Status:  resp.StatusCode,
			Body:    bodyStr,
			Message: extractErrorMessage(bodyStr),
		}
	default:
		return &ServerError{Status: resp.StatusCode, Body: bodyStr}
	}
}

func parseRetryAfter(header, body string) time.Duration {
	if header != "" {
		if secs, err := strconv.Atoi(strings.TrimSpace(header)); err == nil && secs >= 0 {
			return time.Duration(secs) * time.Second
		}
		if t, err := http.ParseTime(header); err == nil {
			d := time.Until(t)
			if d > 0 {
				return d
			}
		}
	}
	var parsed struct {
		Reset int64 `json:"reset"`
	}
	if err := json.Unmarshal([]byte(body), &parsed); err == nil && parsed.Reset > 0 {
		resetAt := time.UnixMilli(parsed.Reset)
		d := time.Until(resetAt)
		if d > 0 {
			return d
		}
	}
	return 0
}

func extractErrorMessage(body string) string {
	var parsed struct {
		Error   string `json:"error"`
		Message string `json:"message"`
	}
	if err := json.Unmarshal([]byte(body), &parsed); err == nil {
		if parsed.Error != "" {
			return parsed.Error
		}
		if parsed.Message != "" {
			return parsed.Message
		}
	}
	return strings.TrimSpace(body)
}
