package maintenance

import (
	"testing"
	"time"
)

func Test_timeWindowStatus(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC()

	t.Run("future window returns scheduled", func(t *testing.T) {
		from := now.Add(1 * time.Hour).Format(time.RFC3339)
		to := now.Add(2 * time.Hour).Format(time.RFC3339)
		if s := timeWindowStatus(from, to); s != "scheduled" {
			t.Errorf("expected 'scheduled', got %q", s)
		}
	})

	t.Run("current window returns in_progress", func(t *testing.T) {
		from := now.Add(-1 * time.Hour).Format(time.RFC3339)
		to := now.Add(1 * time.Hour).Format(time.RFC3339)
		if s := timeWindowStatus(from, to); s != "in_progress" {
			t.Errorf("expected 'in_progress', got %q", s)
		}
	})

	t.Run("past window returns completed", func(t *testing.T) {
		from := now.Add(-2 * time.Hour).Format(time.RFC3339)
		to := now.Add(-1 * time.Hour).Format(time.RFC3339)
		if s := timeWindowStatus(from, to); s != "completed" {
			t.Errorf("expected 'completed', got %q", s)
		}
	})

	t.Run("invalid from returns unknown", func(t *testing.T) {
		to := now.Add(1 * time.Hour).Format(time.RFC3339)
		if s := timeWindowStatus("bad", to); s != "unknown" {
			t.Errorf("expected 'unknown', got %q", s)
		}
	})

	t.Run("invalid to returns unknown", func(t *testing.T) {
		from := now.Add(-1 * time.Hour).Format(time.RFC3339)
		if s := timeWindowStatus(from, "bad"); s != "unknown" {
			t.Errorf("expected 'unknown', got %q", s)
		}
	})

	t.Run("empty strings return unknown", func(t *testing.T) {
		if s := timeWindowStatus("", ""); s != "unknown" {
			t.Errorf("expected 'unknown', got %q", s)
		}
	})
}

func Test_statusColor(t *testing.T) {
	t.Parallel()

	t.Run("scheduled returns non-empty", func(t *testing.T) {
		if s := statusColor("scheduled"); s == "" {
			t.Error("expected non-empty string")
		}
	})

	t.Run("in_progress returns non-empty", func(t *testing.T) {
		if s := statusColor("in_progress"); s == "" {
			t.Error("expected non-empty string")
		}
	})

	t.Run("completed returns non-empty", func(t *testing.T) {
		if s := statusColor("completed"); s == "" {
			t.Error("expected non-empty string")
		}
	})

	t.Run("unknown passes through", func(t *testing.T) {
		if s := statusColor("unknown"); s != "unknown" {
			t.Errorf("expected 'unknown', got %q", s)
		}
	})
}
