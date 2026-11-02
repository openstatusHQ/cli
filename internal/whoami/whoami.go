package whoami

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/urfave/cli/v3"

	"github.com/openstatusHQ/cli/internal/api"
	"github.com/openstatusHQ/cli/internal/auth"
	output "github.com/openstatusHQ/cli/internal/cli"
)

type Whoami struct {
	Name string `json:"name"`
	Slug string `json:"slug"`
	Plan string `json:"plan"`
}

// ErrUnauthorized is returned by Fetch when the API rejects the token.
var ErrUnauthorized = errors.New("API token was rejected")

// Fetch returns the workspace the API token belongs to.
func Fetch(ctx context.Context, httpClient *http.Client, apiKey string) (Whoami, error) {
	url := fmt.Sprintf("%s/whoami", api.APIBaseURL)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, http.NoBody)
	if err != nil {
		return Whoami{}, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Add("x-openstatus-key", apiKey)
	res, err := httpClient.Do(req)
	if err != nil {
		return Whoami{}, err
	}
	defer res.Body.Close()

	switch {
	case res.StatusCode == http.StatusUnauthorized || res.StatusCode == http.StatusForbidden:
		return Whoami{}, ErrUnauthorized
	case res.StatusCode != http.StatusOK:
		return Whoami{}, fmt.Errorf("failed to get workspace information (HTTP %d)", res.StatusCode)
	}
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return Whoami{}, fmt.Errorf("failed to read response body: %w", err)
	}
	var whoami Whoami
	if err := json.Unmarshal(body, &whoami); err != nil {
		return Whoami{}, fmt.Errorf("failed to decode workspace information: %w", err)
	}
	return whoami, nil
}

func GetWhoamiCmd(ctx context.Context, httpClient *http.Client, apiKey string, s *output.Spinner) error {
	whoami, err := Fetch(ctx, httpClient, apiKey)
	output.StopSpinner(s)
	if errors.Is(err, ErrUnauthorized) {
		return fmt.Errorf("authentication failed. Check your API token via OPENSTATUS_API_TOKEN env var or --access-token flag")
	}
	if err != nil {
		return err
	}

	if output.IsJSONOutput() {
		return output.PrintJSON(whoami)
	}

	fmt.Println("Name: ", whoami.Name)
	fmt.Println("Slug: ", whoami.Slug)
	fmt.Println("Plan: ", whoami.Plan)

	return nil
}

func WhoamiCmd() *cli.Command {
	whoamiCmd := cli.Command{
		Name:      "whoami",
		Usage:     "Get your workspace information",
		Aliases:   []string{"w"},
		UsageText: "openstatus whoami",
		Description: `Get your current workspace information.
Displays the workspace name, slug, and plan.`,
		Action: func(ctx context.Context, cmd *cli.Command) error {
			apiKey, err := auth.ResolveAccessToken(cmd)
			if err != nil {
				return err
			}
			if !output.IsQuiet() && !output.IsJSONOutput() {
				fmt.Println("Your current workspace information")
			}
			s := output.StartSpinner("Fetching workspace info...")
			err = GetWhoamiCmd(ctx, api.DefaultHTTPClient, apiKey, s)
			if err != nil {
				return err
			}
			return nil
		},
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "access-token",
				Usage:   "OpenStatus API Access Token",
				Aliases: []string{"t"},
				Sources: cli.EnvVars("OPENSTATUS_API_TOKEN"),
			},
		},
	}
	return &whoamiCmd
}
