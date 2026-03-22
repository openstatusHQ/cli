package cli

import (
	"fmt"
	"os"
	"time"

	"github.com/briandowns/spinner"
)

// Spinner is a type alias so callers don't need to import the spinner package directly.
type Spinner = spinner.Spinner

func StartSpinner(message string) *Spinner {
	if !IsStderrTerminal() || IsJSONOutput() || IsQuiet() {
		return nil
	}
	s := spinner.New(spinner.CharSets[14], 100*time.Millisecond)
	s.Suffix = " " + message
	s.Start()
	return s
}

func StopSpinner(s *spinner.Spinner) {
	if s != nil {
		s.Stop()
		fmt.Fprintln(os.Stderr)
	}
}
