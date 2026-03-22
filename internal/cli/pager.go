package cli

import (
	"io"
	"os"
	"os/exec"
	"strings"
)

// WithPager pipes output through the user's $PAGER (default: "less -FIRX").
// Note: $PAGER is split by whitespace (strings.Fields), so paths with spaces
// are not supported. This matches the behavior of git and gh.
func WithPager(fn func(w io.Writer)) {
	if !IsTerminal() || IsJSONOutput() || IsQuiet() {
		fn(os.Stdout)
		return
	}

	pager := os.Getenv("PAGER")
	if pager == "" {
		pager = "less -FIRX"
	}

	parts := strings.Fields(pager)
	if len(parts) == 0 {
		fn(os.Stdout)
		return
	}

	cmd := exec.Command(parts[0], parts[1:]...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	w, err := cmd.StdinPipe()
	if err != nil {
		fn(os.Stdout)
		return
	}

	if err := cmd.Start(); err != nil {
		fn(os.Stdout)
		return
	}

	fn(w)
	w.Close()
	cmd.Wait()
}
