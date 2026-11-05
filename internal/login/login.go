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
		Usage:     "Log in with your browser, or save an API token",
		UsageText: "openstatus login [--no-browser | --with-token]",
		Description: `Opens the OpenStatus dashboard in your browser, where you sign in and choose
a workspace, then returns you to the CLI. The login is saved and refreshed
automatically until you run 'openstatus logout'.

Where a browser isn't available, such as CI or an SSH session, use an API token
from the dashboard: pass --with-token to be prompted for it, or pipe it to stdin.`,
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:  "no-browser",
				Usage: "Print the login URL instead of opening a browser",
			},
			&cli.BoolFlag{
				Name:  "with-token",
				Usage: "Save an API token from the dashboard instead of logging in with a browser",
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			if cmd.Bool("no-browser") && cmd.Bool("with-token") {
				return output.UsageErrorf("--no-browser and --with-token cannot be used together")
			}
			interactive := term.IsTerminal(int(os.Stdin.Fd()))

			// Read before saving, which replaces it.
			previous, err := auth.LoadOAuth()
			if err != nil {
				// A corrupt file shouldn't stand in the way of logging in again.
				previous = nil
			}

			var workspace whoami.Whoami
			if usesTokenLogin(cmd.Bool("with-token"), cmd.Bool("no-browser"), interactive) {
				workspace, err = loginWithToken(ctx, interactive)
			} else {
				flow := &browserFlow{
					httpClient: api.DefaultHTTPClient,
					stderr:     os.Stderr,
					open:       openBrowser,
					timeout:    browserLoginTimeout,
				}
				// Over SSH a browser would open on the wrong machine, if at all.
				if cmd.Bool("no-browser") || os.Getenv("SSH_TTY") != "" {
					flow.open = nil
				}
				workspace, err = flow.run(ctx)
			}
			if err != nil {
				return err
			}

			if previous != nil {
				if err := auth.RevokeOAuth(ctx, api.DefaultHTTPClient, previous); err != nil {
					fmt.Fprintf(os.Stderr, "Warning: could not revoke your previous login on the server (%v). You can revoke it from Settings / Integrations in the dashboard\n", err)
				}
			}

			fmt.Printf("Logged in to workspace %s (%s). You can now use openstatus commands without --access-token\n", workspace.Name, workspace.Slug)
			if os.Getenv("OPENSTATUS_API_TOKEN") != "" {
				fmt.Fprintln(os.Stderr, "Warning: OPENSTATUS_API_TOKEN is set and takes priority over this login until you unset it")
			}
			return nil
		},
	}
}

// usesTokenLogin picks the login method. Without a terminal, stdin is taken to
// hold a piped token, unless --no-browser explicitly asks for the browser flow.
func usesTokenLogin(withToken, noBrowser, interactive bool) bool {
	return withToken || (!interactive && !noBrowser)
}

// loginWithToken reads an API token from stdin, verifies it and saves it.
func loginWithToken(ctx context.Context, interactive bool) (whoami.Whoami, error) {
	if interactive {
		fmt.Fprintln(os.Stderr, "Enter your OpenStatus API token (from your dashboard):")
		fmt.Fprint(os.Stderr, "> ")
	}

	var token string
	if interactive {
		raw, err := term.ReadPassword(int(os.Stdin.Fd()))
		if err != nil {
			return whoami.Whoami{}, cli.Exit("Failed to read token", 1)
		}
		token = string(raw)
		fmt.Fprintln(os.Stderr)
	} else {
		reader := bufio.NewReader(os.Stdin)
		// Input may not end with a line break (e.g. `printf "$TOKEN" | openstatus login`),
		// in which case ReadString returns the token along with io.EOF.
		line, err := reader.ReadString('\n')
		if err != nil && !errors.Is(err, io.EOF) {
			return whoami.Whoami{}, cli.Exit("Failed to read token", 1)
		}
		token = line
	}

	token = strings.TrimSpace(token)
	if token == "" {
		return whoami.Whoami{}, output.UsageErrorf("Token cannot be empty")
	}

	fmt.Fprintln(os.Stderr, "Verifying token...")
	workspace, err := whoami.Fetch(ctx, api.DefaultHTTPClient, token)
	if errors.Is(err, whoami.ErrUnauthorized) {
		return whoami.Whoami{}, output.NewError(output.KindUnauthenticated, "Invalid token. Could not authenticate with OpenStatus API")
	}
	if err != nil {
		return whoami.Whoami{}, fmt.Errorf("could not verify token: %w", output.FormatError(err, "", ""))
	}

	if err := auth.SaveToken(token); err != nil {
		return whoami.Whoami{}, cli.Exit(fmt.Sprintf("Failed to save token: %v", err), 1)
	}
	return workspace, nil
}

func LogoutCmd() *cli.Command {
	return &cli.Command{
		Name:      "logout",
		Usage:     "Log out and remove saved credentials",
		UsageText: "openstatus logout",
		Action: func(ctx context.Context, cmd *cli.Command) error {
			creds, err := auth.LoadOAuth()
			if err != nil {
				// A corrupt file can't be revoked, but it can still be removed.
				creds = nil
			}
			if creds != nil {
				if err := auth.RevokeOAuth(ctx, api.DefaultHTTPClient, creds); err != nil {
					fmt.Fprintf(os.Stderr, "Warning: could not revoke your login on the server (%v). You can revoke it from Settings / Integrations in the dashboard\n", err)
				}
			}

			removed, err := auth.RemoveToken()
			if err != nil {
				return cli.Exit(fmt.Sprintf("Failed to remove token: %v", err), 1)
			}
			if removed {
				fmt.Println("Logged out successfully")
			} else {
				fmt.Println("No saved login found")
			}
			if os.Getenv("OPENSTATUS_API_TOKEN") != "" {
				fmt.Fprintln(os.Stderr, "Warning: OPENSTATUS_API_TOKEN is still set; commands will keep using it until you unset it")
			}
			return nil
		},
	}
}
