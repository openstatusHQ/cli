package statusreport

import (
	"testing"
)

func Test_notEmpty(t *testing.T) {
	t.Parallel()

	validator := notEmpty("title")

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

func Test_buildSummary(t *testing.T) {
	t.Parallel()

	lines := [][2]string{
		{"Page", "Production"},
		{"Status", "investigating"},
	}
	result := buildSummary(lines)
	if result == "" {
		t.Error("expected non-empty summary")
	}
	if !contains(result, "Page") || !contains(result, "Production") {
		t.Errorf("summary missing expected content: %s", result)
	}
	if !contains(result, "Status") || !contains(result, "investigating") {
		t.Errorf("summary missing expected content: %s", result)
	}
}

func Test_statusSelectOptions(t *testing.T) {
	t.Parallel()

	opts := statusSelectOptions()
	if len(opts) != 4 {
		t.Errorf("expected 4 options, got %d", len(opts))
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && searchString(s, substr)
}

func searchString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
