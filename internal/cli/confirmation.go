package cli

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

var isInteractiveStdin = func() bool {
	return IsStdinTerminal()
}

func AskForConfirmation(s string) (bool, error) {
	if !isInteractiveStdin() {
		return false, fmt.Errorf("confirmation required but stdin is not a terminal (use --auto-accept / -y to skip)")
	}
	reader := bufio.NewReader(os.Stdin)
	fmt.Fprintf(os.Stderr, "%s [y/N]: ", s)
	response, err := reader.ReadString('\n')
	if err != nil {
		return false, fmt.Errorf("failed to read user input: %w", err)
	}
	response = strings.ToLower(strings.TrimSpace(response))
	if response == "y" || response == "yes" {
		return true, nil
	}
	return false, nil
}
