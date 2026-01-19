package whoami

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/urfave/cli/v3"
)

type Whoami struct {
	Name string `json:"name"`
	Slug string `json:"slug"`
	Plan string `json:"plan"`
}

func GetWhoamiCmd(httpClient *http.Client, apiKey string) error {
	url := "https://api.openstatus.dev/v1/whoami" // Using monitors.APIBaseURL would create circular import

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Add("x-openstatus-key", apiKey)
	res, err := httpClient.Do(req)
	if err != nil {
		return err
	}

	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("Failed to get workspace information")
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
	fmt.Println("Name: ", whoami.Name)
	fmt.Println("Slug: ", whoami.Slug)
	fmt.Println("Plan: ", whoami.Plan)

	return nil
}

func WhoamiCmd() *cli.Command {
	whoamiCmd := cli.Command{
		Name:        "whoami",
		Usage:       "Get your workspace information",
		Aliases:     []string{"w"},
		UsageText:   "openstatus whoami [options]",
		Description: "Get your current workspace information, display the workspace name, slug, and plan",
		Action: func(ctx context.Context, cmd *cli.Command) error {
			fmt.Println("Your current workspace information")
			err := GetWhoamiCmd(http.DefaultClient, cmd.String("access-token"))
			if err != nil {
				return cli.Exit("Failed to get workspace information", 1)
			}
			return nil
		},
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "access-token",
				Usage:    "OpenStatus API Access Token",
				Aliases:  []string{"t"},
				Sources:  cli.EnvVars("OPENSTATUS_API_TOKEN"),
				Required: true,
			}},
	}
	return &whoamiCmd
}
