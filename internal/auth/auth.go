package auth

import (
	"fmt"
	"os"
	"strings"

	"github.com/openstatusHQ/cli/internal/config"
	clilib "github.com/urfave/cli/v3"
)

// ResolveAccessToken extracts the access token from CLI flags or falls back to saved token.
func ResolveAccessToken(cmd *clilib.Command) (string, error) {
	return ResolveToken(cmd.String("access-token"))
}

func ResolveToken(flagValue string) (string, error) {
	if flagValue != "" {
		return flagValue, nil
	}

	tokenPath, err := config.TokenPath()
	if err == nil {
		data, readErr := os.ReadFile(tokenPath)
		if readErr == nil {
			token := strings.TrimSpace(string(data))
			if token != "" {
				return token, nil
			}
		}
	}

	return "", fmt.Errorf("no API token found. Set OPENSTATUS_API_TOKEN env var, or run 'openstatus login'")
}

func SaveToken(token string) error {
	dir, err := config.ConfigDir()
	if err != nil {
		return fmt.Errorf("failed to determine config directory: %w", err)
	}
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	tokenPath, err := config.TokenPath()
	if err != nil {
		return fmt.Errorf("failed to determine token path: %w", err)
	}

	tmpFile, err := os.CreateTemp(dir, ".token-*")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	tmpPath := tmpFile.Name()

	if _, err := tmpFile.WriteString(token); err != nil {
		tmpFile.Close()
		os.Remove(tmpPath)
		return fmt.Errorf("failed to write token: %w", err)
	}
	if err := tmpFile.Close(); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("failed to close temp file: %w", err)
	}
	if err := os.Chmod(tmpPath, 0600); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("failed to set token file permissions: %w", err)
	}
	if err := os.Rename(tmpPath, tokenPath); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("failed to save token: %w", err)
	}
	return nil
}

func RemoveToken() error {
	tokenPath, err := config.TokenPath()
	if err != nil {
		return fmt.Errorf("failed to determine token path: %w", err)
	}
	err = os.Remove(tokenPath)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("failed to remove token: %w", err)
	}
	return nil
}

