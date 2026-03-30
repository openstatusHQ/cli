package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"sync/atomic"

	"github.com/fatih/color"
	"github.com/mattn/go-isatty"
)

var (
	jsonOutput atomic.Bool
	quietMode  atomic.Bool
	debugMode  atomic.Bool
)

func SetJSONOutput(v bool)  { jsonOutput.Store(v) }
func SetQuietMode(v bool)   { quietMode.Store(v) }
func SetDebugMode(v bool)   { debugMode.Store(v) }
func IsJSONOutput() bool    { return jsonOutput.Load() }
func IsQuiet() bool         { return quietMode.Load() }
func IsDebug() bool         { return debugMode.Load() }
func IsTerminal() bool { return isatty.IsTerminal(os.Stdout.Fd()) || isatty.IsCygwinTerminal(os.Stdout.Fd()) }
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

func InitColorSettings(noColorFlag bool) {
	if noColorFlag || os.Getenv("NO_COLOR") != "" || os.Getenv("TERM") == "dumb" || !IsTerminal() {
		color.NoColor = true
	}
}
