package cli_test

import (
	"testing"

	"github.com/openstatusHQ/cli/internal/cli"
)

func Test_FormatTimestamp(t *testing.T) {
	t.Parallel()

	t.Run("valid RFC 3339 returns shortened format", func(t *testing.T) {
		result := cli.FormatTimestamp("2026-04-01T10:00:00Z")
		expected := "2026-04-01 10:00 UTC"
		if result != expected {
			t.Errorf("expected %q, got %q", expected, result)
		}
	})

	t.Run("timezone offset converted to UTC", func(t *testing.T) {
		result := cli.FormatTimestamp("2026-04-01T12:00:00+02:00")
		expected := "2026-04-01 10:00 UTC"
		if result != expected {
			t.Errorf("expected %q, got %q", expected, result)
		}
	})

	t.Run("invalid string returned as-is", func(t *testing.T) {
		result := cli.FormatTimestamp("not-a-date")
		if result != "not-a-date" {
			t.Errorf("expected 'not-a-date', got %q", result)
		}
	})

	t.Run("empty string returned as-is", func(t *testing.T) {
		result := cli.FormatTimestamp("")
		if result != "" {
			t.Errorf("expected empty string, got %q", result)
		}
	})
}
