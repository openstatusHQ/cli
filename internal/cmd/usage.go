package cmd

import (
	"context"
	"fmt"
	"strings"

	"github.com/urfave/cli/v3"

	output "github.com/openstatusHQ/cli/internal/cli"
)

// handleUsage routes unknown commands and flag errors on every command
// through the root's ExitErrHandler as usage errors. urfave/cli does not
// inherit these hooks, so they are set on each command.
func handleUsage(cmds []*cli.Command) {
	for _, c := range cmds {
		c.OnUsageError = onUsageError
		if len(c.Commands) > 0 {
			c.CommandNotFound = commandNotFound
		}
		handleUsage(c.Commands)
	}
}

func onUsageError(_ context.Context, cmd *cli.Command, err error, _ bool) error {
	msg := err.Error()
	if name, ok := strings.CutPrefix(msg, "flag provided but not defined: -"); ok {
		msg = "unknown flag " + dashed(name)
		if s := suggest(name, flagNames(cmd)); s != "" {
			msg += fmt.Sprintf(". Did you mean %s?", dashed(s))
		}
	}
	return output.UsageErrorWithHint(msg, fmt.Sprintf("Run '%s --help' for usage.", cmd.FullName()))
}

func commandNotFound(ctx context.Context, cmd *cli.Command, name string) {
	msg := fmt.Sprintf("unknown command %q for %q", name, cmd.FullName())
	if s := suggest(name, commandNames(cmd)); s != "" {
		msg += fmt.Sprintf(". Did you mean %q?", s)
	}
	hint := fmt.Sprintf("Run '%s --help' for a list of commands.", cmd.FullName())
	output.HandleExitError(ctx, cmd, output.UsageErrorWithHint(msg, hint))
}

func dashed(name string) string {
	if len(name) == 1 {
		return "-" + name
	}
	return "--" + name
}

// commandNames maps every visible name and alias to its command's name, so a
// typo of an alias still suggests the canonical command.
func commandNames(cmd *cli.Command) map[string]string {
	names := map[string]string{}
	for _, c := range cmd.VisibleCommands() {
		for _, n := range c.Names() {
			names[n] = c.Name
		}
	}
	return names
}

// flagNames covers the command's flags and the global ones it inherits.
func flagNames(cmd *cli.Command) map[string]string {
	names := map[string]string{}
	for _, c := range cmd.Lineage() {
		for _, f := range c.VisibleFlags() {
			for _, n := range f.Names() {
				names[n] = n
			}
		}
	}
	return names
}

// suggest returns the candidate closest to name, or "" when nothing is close
// enough to be a likely typo.
func suggest(name string, candidates map[string]string) string {
	if len(name) < 2 {
		return ""
	}
	best, bestDist := "", -1
	for n, canonical := range candidates {
		d := levenshtein(name, n)
		if strings.HasPrefix(n, name) && len(name) >= 3 {
			d = min(d, 1)
		}
		if d > max(1, len(name)/3) {
			continue
		}
		if bestDist == -1 || d < bestDist || (d == bestDist && canonical < best) {
			best, bestDist = canonical, d
		}
	}
	return best
}

func levenshtein(a, b string) int {
	prev := make([]int, len(b)+1)
	for j := range prev {
		prev[j] = j
	}
	for i := 1; i <= len(a); i++ {
		cur := make([]int, len(b)+1)
		cur[0] = i
		for j := 1; j <= len(b); j++ {
			cost := 1
			if a[i-1] == b[j-1] {
				cost = 0
			}
			cur[j] = min(prev[j]+1, cur[j-1]+1, prev[j-1]+cost)
		}
		prev = cur
	}
	return prev[len(b)]
}
