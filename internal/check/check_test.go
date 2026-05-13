package check

import (
	"errors"
	"strings"
	"testing"
)

func TestValidateURL(t *testing.T) {
	t.Parallel()
	good := []string{
		"https://example.com",
		"http://example.com:8080/path?query=1",
		"https://api.example.com/v1/health",
	}
	for _, u := range good {
		if err := validateURL(u); err != nil {
			t.Errorf("validateURL(%q) = %v, want nil", u, err)
		}
	}
	bad := []struct {
		url   string
		anyOf []string
	}{
		{"", []string{"missing scheme or host"}},
		{"example.com", []string{"missing scheme or host"}},
		{"://example.com", []string{"missing scheme or host", "missing protocol scheme"}},
		{"ftp://host", []string{"only http and https"}},
		{"ssh://host:22", []string{"only http and https"}},
	}
	for _, c := range bad {
		err := validateURL(c.url)
		if err == nil {
			t.Errorf("validateURL(%q) = nil, want error", c.url)
			continue
		}
		var matched bool
		for _, want := range c.anyOf {
			if strings.Contains(err.Error(), want) {
				matched = true
				break
			}
		}
		if !matched {
			t.Errorf("validateURL(%q) error = %q, want one of %v", c.url, err.Error(), c.anyOf)
		}
	}
}

func TestParseHeaders(t *testing.T) {
	t.Parallel()
	t.Run("nil", func(t *testing.T) {
		t.Parallel()
		h, err := parseHeaders(nil)
		if err != nil || h != nil {
			t.Errorf("parseHeaders(nil) = %v,%v", h, err)
		}
	})
	t.Run("empty slice", func(t *testing.T) {
		t.Parallel()
		h, err := parseHeaders([]string{})
		if err != nil || h != nil {
			t.Errorf("parseHeaders([]) = %v,%v", h, err)
		}
	})
	t.Run("valid pairs trim whitespace", func(t *testing.T) {
		t.Parallel()
		h, err := parseHeaders([]string{"Authorization: Bearer abc", "x-foo:bar"})
		if err != nil {
			t.Fatalf("err = %v", err)
		}
		if h["Authorization"] != "Bearer abc" {
			t.Errorf("Authorization = %q", h["Authorization"])
		}
		if h["x-foo"] != "bar" {
			t.Errorf("x-foo = %q", h["x-foo"])
		}
	})
	t.Run("colon in value preserved", func(t *testing.T) {
		t.Parallel()
		h, err := parseHeaders([]string{"X-URL: https://example.com:8080/x"})
		if err != nil {
			t.Fatalf("err = %v", err)
		}
		if h["X-URL"] != "https://example.com:8080/x" {
			t.Errorf("value = %q", h["X-URL"])
		}
	})
	t.Run("missing colon errors", func(t *testing.T) {
		t.Parallel()
		_, err := parseHeaders([]string{"no-colon"})
		if err == nil || !strings.Contains(err.Error(), "expected") {
			t.Errorf("expected error, got %v", err)
		}
	})
	t.Run("empty key errors", func(t *testing.T) {
		t.Parallel()
		_, err := parseHeaders([]string{": value"})
		if err == nil || !strings.Contains(err.Error(), "key is empty") {
			t.Errorf("expected empty-key error, got %v", err)
		}
	})
	t.Run("duplicate keys last wins", func(t *testing.T) {
		t.Parallel()
		h, err := parseHeaders([]string{"X-Foo: a", "X-Foo: b"})
		if err != nil {
			t.Fatalf("err = %v", err)
		}
		if h["X-Foo"] != "b" {
			t.Errorf("X-Foo = %q, want \"b\"", h["X-Foo"])
		}
	})
}

func TestResolveBody(t *testing.T) {
	t.Parallel()
	t.Run("empty", func(t *testing.T) {
		t.Parallel()
		b, err := resolveBody("")
		if err != nil || b != "" {
			t.Errorf("resolveBody(\"\") = %q,%v", b, err)
		}
	})
	t.Run("inline string", func(t *testing.T) {
		t.Parallel()
		b, err := resolveBody(`{"ping":true}`)
		if err != nil || b != `{"ping":true}` {
			t.Errorf("inline = %q,%v", b, err)
		}
	})
	t.Run("@file reads file", func(t *testing.T) {
		t.Parallel()
		b, err := resolveBody("@testdata/body.json")
		if err != nil {
			t.Fatalf("err = %v", err)
		}
		if !strings.Contains(b, `"ping":true`) {
			t.Errorf("body = %q", b)
		}
	})
	t.Run("@file missing errors", func(t *testing.T) {
		t.Parallel()
		_, err := resolveBody("@no-such-file.json")
		if err == nil || !strings.Contains(err.Error(), "read body file") {
			t.Errorf("expected read error, got %v", err)
		}
	})
	t.Run("@- on TTY errors", func(t *testing.T) {
		t.Parallel()
		_, err := resolveBody("@-")
		if err == nil {
			t.Skip("stdin is not a TTY in this test runner; cannot assert TTY-error branch")
		}
		if !strings.Contains(err.Error(), "stdin") {
			t.Errorf("expected stdin error, got %v", err)
		}
	})
}

func TestShareURL_EdgeCases(t *testing.T) {
	t.Parallel()
	if got := shareURL(""); got != "" {
		t.Errorf("shareURL(\"\") = %q", got)
	}
	if got := shareURL("xyz"); got != "https://www.openstatus.dev/play/checker/xyz" {
		t.Errorf("shareURL = %q", got)
	}
}

func TestFormatRunError_RateLimit(t *testing.T) {
	t.Parallel()
	err := formatRunError(&RateLimitError{RetryAfter: 16_000_000_000, Message: "rate limit"})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "Retry after") {
		t.Errorf("err = %q, want \"Retry after\" substring", err.Error())
	}
}

func TestFormatRunError_ClientIPHint(t *testing.T) {
	t.Parallel()
	err := formatRunError(&BadRequestError{Status: 400, Message: "could not determine client IP", Body: `{"error":"could not determine client IP"}`})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "VPN") {
		t.Errorf("err = %q, want VPN hint", err.Error())
	}
}

func TestFormatRunError_ServerError(t *testing.T) {
	t.Parallel()
	err := formatRunError(&ServerError{Status: 503})
	if err == nil || !strings.Contains(err.Error(), "503") {
		t.Errorf("err = %v, want 503 message", err)
	}
}

func TestFormatRunError_StreamTruncated(t *testing.T) {
	t.Parallel()
	err := formatRunError(ErrStreamTruncated)
	if !strings.Contains(err.Error(), "Stream ended") {
		t.Errorf("err = %v, want stream-ended message", err)
	}
}

func TestFormatRunError_Unknown(t *testing.T) {
	t.Parallel()
	err := formatRunError(errors.New("dial tcp: no route to host"))
	if !strings.Contains(err.Error(), "Could not reach OpenStatus") {
		t.Errorf("err = %v, want reach-error wrap", err)
	}
}
