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

func getWhoamiCmd(httpClient *http.Client, apiKey string) error {
	url := "https://api.openstatus.dev/v1/whoami"

	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Add("x-openstatus-key", apiKey)
	res, err := httpClient.Do(req)
	if err != nil {
		return err
	}

	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("Failed to get workspace information")
	}
	body, _ := io.ReadAll(res.Body)
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
		Name:    "whoami",
		Aliases: []string{"w"},
		Usage:   "Get your current workspace information",
		Action: func(ctx context.Context, cmd *cli.Command) error {
			fmt.Println("Your current workspace information")
			err := getWhoamiCmd(http.DefaultClient, cmd.String("access-token"))
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
