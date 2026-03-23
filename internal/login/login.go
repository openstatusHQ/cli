package login

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/openstatusHQ/cli/internal/api"
	"github.com/openstatusHQ/cli/internal/auth"
	"github.com/openstatusHQ/cli/internal/whoami"
	"github.com/urfave/cli/v3"
	"golang.org/x/term"
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
				line, err := reader.ReadString('\n')
				if err != nil {
					return cli.Exit("Failed to read token", 1)
				}
				token = line
			}

			token = strings.TrimSpace(token)
			if token == "" {
				return cli.Exit("Token cannot be empty", 1)
			}

			fmt.Fprintln(os.Stderr, "Verifying token...")
			err := whoami.GetWhoamiCmd(ctx, api.DefaultHTTPClient, token, nil)
			if err != nil {
				return cli.Exit("Invalid token. Could not authenticate with OpenStatus API", 1)
			}

			if err := auth.SaveToken(token); err != nil {
				return cli.Exit(fmt.Sprintf("Failed to save token: %v", err), 1)
			}

			fmt.Println("Token saved successfully. You can now use openstatus commands without --access-token")
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
			if err := auth.RemoveToken(); err != nil {
				return cli.Exit(fmt.Sprintf("Failed to remove token: %v", err), 1)
			}
			fmt.Println("Token removed successfully")
			return nil
		},
	}
}
