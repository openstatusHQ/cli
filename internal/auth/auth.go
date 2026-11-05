package auth

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	clilib "github.com/urfave/cli/v3"

	"github.com/openstatusHQ/cli/internal/api"
	output "github.com/openstatusHQ/cli/internal/cli"
	"github.com/openstatusHQ/cli/internal/config"
)

// ResolveAccessToken extracts the access token from CLI flags or falls back to saved token.
func ResolveAccessToken(ctx context.Context, cmd *clilib.Command) (string, error) {
	return ResolveToken(ctx, cmd.String("access-token"))
}

func ResolveToken(ctx context.Context, flagValue string) (string, error) {
	if flagValue != "" {
		return flagValue, nil
	}

	creds, err := LoadOAuth()
	if err != nil {
		return "", err
	}
	if creds != nil {
		return resolveOAuthToken(ctx, api.DefaultHTTPClient, creds)
	}

	tokenPath, err := config.TokenPath()
	if err == nil {
		data, readErr := os.ReadFile(tokenPath)
		if readErr == nil {
			token := strings.TrimSpace(string(data))
			if token != "" {
				return token, nil
			}
		} else if !os.IsNotExist(readErr) {
			return "", fmt.Errorf("failed to read token file %s: %w", tokenPath, readErr)
		}
	}

	return "", output.NewError(output.KindUnauthenticated, "no API token found. Set OPENSTATUS_API_TOKEN env var, or run 'openstatus login'")
}

// SaveToken stores an API key, replacing any browser login.
func SaveToken(token string) error {
	tokenPath, err := config.TokenPath()
	if err != nil {
		return fmt.Errorf("failed to determine token path: %w", err)
	}
	if err := writeSecret(tokenPath, []byte(token)); err != nil {
		return err
	}
	oauthPath, err := config.OAuthPath()
	if err != nil {
		return fmt.Errorf("failed to determine credentials path: %w", err)
	}
	if err := os.Remove(oauthPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove previous login: %w", err)
	}
	return nil
}

// writeSecret atomically writes data to path, readable only by the user.
func writeSecret(path string, data []byte) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	tmpFile, err := os.CreateTemp(dir, ".token-*")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	tmpPath := tmpFile.Name()

	if _, err := tmpFile.Write(data); err != nil {
		tmpFile.Close()
		os.Remove(tmpPath)
		return fmt.Errorf("failed to write token: %w", err)
	}
	if err := tmpFile.Close(); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("failed to close temp file: %w", err)
	}
	if err := os.Chmod(tmpPath, 0o600); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("failed to set token file permissions: %w", err)
	}
	if err := os.Rename(tmpPath, path); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("failed to save token: %w", err)
	}
	return nil
}

// RemoveToken deletes the saved API key and browser login. It reports false if
// neither was saved.
func RemoveToken() (bool, error) {
	removed := false
	for _, pathFn := range []func() (string, error){config.TokenPath, config.OAuthPath} {
		path, err := pathFn()
		if err != nil {
			return false, fmt.Errorf("failed to determine token path: %w", err)
		}
		err = os.Remove(path)
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return false, fmt.Errorf("failed to remove token: %w", err)
		}
		removed = true
	}
	return removed, nil
}
