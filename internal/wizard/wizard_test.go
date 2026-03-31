package wizard_test

import (
	"strings"
	"testing"

	"github.com/openstatusHQ/cli/internal/wizard"
)

func Test_NotEmpty(t *testing.T) {
	t.Parallel()

	validator := wizard.NotEmpty("title")

	t.Run("empty string returns error", func(t *testing.T) {
		if err := validator(""); err == nil {
			t.Error("expected error for empty string")
		}
	})

	t.Run("whitespace-only returns error", func(t *testing.T) {
		if err := validator("   "); err == nil {
			t.Error("expected error for whitespace-only string")
		}
	})

	t.Run("non-empty string returns nil", func(t *testing.T) {
		if err := validator("hello"); err != nil {
			t.Errorf("expected nil, got %v", err)
		}
	})

	t.Run("error message contains field name", func(t *testing.T) {
		err := validator("")
		if err == nil {
			t.Fatal("expected error")
		}
		if err.Error() != "title cannot be empty" {
			t.Errorf("expected 'title cannot be empty', got %q", err.Error())
		}
	})
}

func Test_BuildSummary(t *testing.T) {
	t.Parallel()

	lines := [][2]string{
		{"Page", "Production"},
		{"Status", "investigating"},
	}
	result := wizard.BuildSummary(lines)
	if result == "" {
		t.Error("expected non-empty summary")
	}
	if !strings.Contains(result, "Page") || !strings.Contains(result, "Production") {
		t.Errorf("summary missing expected content: %s", result)
	}
	if !strings.Contains(result, "Status") || !strings.Contains(result, "investigating") {
		t.Errorf("summary missing expected content: %s", result)
	}
}

func Test_ValidRFC3339(t *testing.T) {
	t.Parallel()

	validator := wizard.ValidRFC3339("from")

	t.Run("valid RFC 3339 passes", func(t *testing.T) {
		if err := validator("2026-04-01T10:00:00Z"); err != nil {
			t.Errorf("expected nil, got %v", err)
		}
	})

	t.Run("valid RFC 3339 with offset passes", func(t *testing.T) {
		if err := validator("2026-04-01T12:00:00+02:00"); err != nil {
			t.Errorf("expected nil, got %v", err)
		}
	})

	t.Run("invalid string fails", func(t *testing.T) {
		if err := validator("not-a-date"); err == nil {
			t.Error("expected error for invalid string")
		}
	})

	t.Run("empty string fails", func(t *testing.T) {
		if err := validator(""); err == nil {
			t.Error("expected error for empty string")
		}
	})

	t.Run("error message contains field name", func(t *testing.T) {
		err := validator("bad")
		if err == nil {
			t.Fatal("expected error")
		}
		if !strings.Contains(err.Error(), "from") {
			t.Errorf("expected error to contain 'from', got %q", err.Error())
		}
	})
}
