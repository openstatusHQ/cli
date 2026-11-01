package cli_test

import (
	"slices"
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

func TestSplitList(t *testing.T) {
	t.Parallel()
	for in, want := range map[string][]string{
		"":         nil,
		"1":        {"1"},
		"1, 2":     {"1", "2"},
		" 1 ,, 2,": {"1", "2"},
	} {
		got := cli.SplitList(in)
		if !slices.Equal(got, want) {
			t.Errorf("SplitList(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestParseIDList(t *testing.T) {
	t.Parallel()
	for in, want := range map[string]struct {
		ids      []string
		provided bool
		err      bool
	}{
		"":      {nil, false, false},
		"a, b":  {[]string{"a", "b"}, true, false},
		",":     {nil, false, true},
		" , , ": {nil, false, true},
	} {
		ids, provided, err := cli.ParseIDList("component-ids", in)
		if !slices.Equal(ids, want.ids) || provided != want.provided || (err != nil) != want.err {
			t.Errorf("ParseIDList(%q) = %q, %v, %v; want %q, %v, err=%v", in, ids, provided, err, want.ids, want.provided, want.err)
		}
	}
}
