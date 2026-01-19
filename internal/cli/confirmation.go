package cli

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

func AskForConfirmation(s string) (bool, error) {
	reader := bufio.NewReader(os.Stdin)
	fmt.Printf("%s [y/N]: ", s)
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
