package wizard

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/huh"
)

func NotEmpty(fieldName string) func(string) error {
	return func(s string) error {
		if strings.TrimSpace(s) == "" {
			return fmt.Errorf("%s cannot be empty", fieldName)
		}
		return nil
	}
}

func HandleFormError(err error) error {
	if errors.Is(err, huh.ErrUserAborted) {
		fmt.Fprintln(os.Stderr, "Aborted.")
		os.Exit(130)
	}
	return err
}

func BuildSummary(lines [][2]string) string {
	var sb strings.Builder
	for _, line := range lines {
		sb.WriteString(fmt.Sprintf("  %s: %s\n", line[0], line[1]))
	}
	return sb.String()
}
