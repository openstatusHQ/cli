package login

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/urfave/cli/v3"
	"golang.org/x/term"

	"github.com/openstatusHQ/cli/internal/api"
	"github.com/openstatusHQ/cli/internal/auth"
	output "github.com/openstatusHQ/cli/internal/cli"
	"github.com/openstatusHQ/cli/internal/whoami"
)

func LoginCmd() *cli.Command {
	return &cli.Command{
		Name:      "login",
		Usage:     "Save your API token",
		UsageText: "openstatus login",
		Description: `Saves your OpenStatus API token for use in subsequent commands.
Get your API token from the OpenStatus dashboard.`,
		Action: func(ctx context.Context, cmd *cli.Command) error {
			interactive := term.IsTerminal(int(os.Stdin.Fd()))

			if interactive {
				fmt.Fprintln(os.Stderr, "Enter your OpenStatus API token (from your dashboard):")
				fmt.Fprint(os.Stderr, "> ")
			}

			var token string
			if interactive {
				raw, err := term.ReadPassword(int(os.Stdin.Fd()))
				if err != nil {
					return cli.Exit("Failed to read token", 1)
				}
				token = string(raw)
				fmt.Fprintln(os.Stderr)
			} else {
				reader := bufio.NewReader(os.Stdin)
				// Input may not end with a line break (e.g. `printf "$TOKEN" | openstatus login`),
				// in which case ReadString returns the token along with io.EOF.
				line, err := reader.ReadString('\n')
				if err != nil && !errors.Is(err, io.EOF) {
					return cli.Exit("Failed to read token", 1)
				}
				token = line
			}

			token = strings.TrimSpace(token)
			if token == "" {
				return output.UsageErrorf("Token cannot be empty")
			}

			fmt.Fprintln(os.Stderr, "Verifying token...")
			workspace, err := whoami.Fetch(ctx, api.DefaultHTTPClient, token)
			if errors.Is(err, whoami.ErrUnauthorized) {
				return output.NewError(output.KindUnauthenticated, "Invalid token. Could not authenticate with OpenStatus API")
			}
			if err != nil {
				return fmt.Errorf("could not verify token: %w", output.FormatError(err, "", ""))
			}

			if err := auth.SaveToken(token); err != nil {
				return cli.Exit(fmt.Sprintf("Failed to save token: %v", err), 1)
			}

			fmt.Printf("Logged in to workspace %s (%s). You can now use openstatus commands without --access-token\n", workspace.Name, workspace.Slug)
			return nil
		},
	}
}

func LogoutCmd() *cli.Command {
	return &cli.Command{
		Name:      "logout",
		Usage:     "Remove saved API token",
		UsageText: "openstatus logout",
		Action: func(ctx context.Context, cmd *cli.Command) error {
			removed, err := auth.RemoveToken()
			if err != nil {
				return cli.Exit(fmt.Sprintf("Failed to remove token: %v", err), 1)
			}
			if removed {
				fmt.Println("Token removed successfully")
			} else {
				fmt.Println("No saved token found")
			}
			if os.Getenv("OPENSTATUS_API_TOKEN") != "" {
				fmt.Fprintln(os.Stderr, "Warning: OPENSTATUS_API_TOKEN is still set; commands will keep using it until you unset it")
			}
			return nil
		},
	}
}
