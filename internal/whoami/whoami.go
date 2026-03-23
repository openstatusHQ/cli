package whoami

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/openstatusHQ/cli/internal/api"
	"github.com/openstatusHQ/cli/internal/auth"
	output "github.com/openstatusHQ/cli/internal/cli"
	"github.com/urfave/cli/v3"
)

type Whoami struct {
	Name string `json:"name"`
	Slug string `json:"slug"`
	Plan string `json:"plan"`
}

func GetWhoamiCmd(ctx context.Context, httpClient *http.Client, apiKey string, s *output.Spinner) error {
	url := fmt.Sprintf("%s/whoami", api.APIBaseURL)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Add("x-openstatus-key", apiKey)
	res, err := httpClient.Do(req)
	output.StopSpinner(s)
	if err != nil {
		return err
	}

	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to get workspace information. Check your API token with OPENSTATUS_API_TOKEN env var")
	}
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}
	var whoami Whoami
	err = json.Unmarshal(body, &whoami)
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
				return cli.Exit(err.Error(), 1)
			}
			if !output.IsQuiet() && !output.IsJSONOutput() {
				fmt.Println("Your current workspace information")
			}
			s := output.StartSpinner("Fetching workspace info...")
			err = GetWhoamiCmd(ctx, api.DefaultHTTPClient, apiKey, s)
			if err != nil {
				return cli.Exit(err.Error(), 1)
			}
			return nil
		},
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "access-token",
				Usage:   "OpenStatus API Access Token",
				Aliases: []string{"t"},
				Sources: cli.EnvVars("OPENSTATUS_API_TOKEN"),
			}},
	}
	return &whoamiCmd
}
