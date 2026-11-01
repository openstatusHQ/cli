package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync/atomic"
	"time"

	"github.com/fatih/color"
	"github.com/mattn/go-isatty"
)

var (
	jsonOutput atomic.Bool
	quietMode  atomic.Bool
	debugMode  atomic.Bool
)

func SetJSONOutput(v bool) { jsonOutput.Store(v) }
func SetQuietMode(v bool)  { quietMode.Store(v) }
func SetDebugMode(v bool)  { debugMode.Store(v) }
func IsJSONOutput() bool   { return jsonOutput.Load() }
func IsQuiet() bool        { return quietMode.Load() }
func IsDebug() bool        { return debugMode.Load() }
func IsTerminal() bool {
	return isatty.IsTerminal(os.Stdout.Fd()) || isatty.IsCygwinTerminal(os.Stdout.Fd())
}

func IsStdinTerminal() bool {
	return isatty.IsTerminal(os.Stdin.Fd()) || isatty.IsCygwinTerminal(os.Stdin.Fd())
}

func IsStderrTerminal() bool {
	return isatty.IsTerminal(os.Stderr.Fd()) || isatty.IsCygwinTerminal(os.Stderr.Fd())
}

func PrintJSON(v any) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}
	fmt.Println(string(data))
	return nil
}

func FormatTimestamp(rfc3339 string) string {
	t, err := time.Parse(time.RFC3339, rfc3339)
	if err != nil {
		return rfc3339
	}
	return t.UTC().Format("2006-01-02 15:04 UTC")
}

func InitColorSettings(noColorFlag bool) {
	if noColorFlag || os.Getenv("NO_COLOR") != "" || os.Getenv("TERM") == "dumb" || !IsTerminal() {
		color.NoColor = true
	}
}

// RenderDeleted reports a deleted resource: {"id","deleted"} in JSON mode,
// the bare ID in quiet mode, and the given lines otherwise.
func RenderDeleted(id string, lines ...string) error {
	if IsJSONOutput() {
		return PrintJSON(struct {
			ID      string `json:"id"`
			Deleted bool   `json:"deleted"`
		}{id, true})
	}
	if IsQuiet() {
		fmt.Println(id)
		return nil
	}
	for _, l := range lines {
		fmt.Println(l)
	}
	return nil
}

// SplitList splits a comma-separated flag value, trimming spaces and dropping
// empty entries, so "1, 2," yields ["1" "2"].
func SplitList(raw string) []string {
	var out []string
	for _, part := range strings.Split(raw, ",") {
		if part = strings.TrimSpace(part); part != "" {
			out = append(out, part)
		}
	}
	return out
}

// ParseIDList parses an optional comma-separated ID flag. An empty value means
// the flag was not provided; a value with only separators (e.g. ",") is an
// error rather than an empty list, since update endpoints treat an empty list
// as "remove all".
func ParseIDList(flag, raw string) (ids []string, provided bool, err error) {
	if raw == "" {
		return nil, false, nil
	}
	ids = SplitList(raw)
	if len(ids) == 0 {
		return nil, false, fmt.Errorf("--%s must contain at least one ID", flag)
	}
	return ids, true, nil
}
