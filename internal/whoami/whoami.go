package whoami

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/urfave/cli/v2"
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
		Action: func(cCtx *cli.Context) error {
			fmt.Println("Your current workspace information")

			getWhoamiCmd(http.DefaultClient, cCtx.String("access-token"))
			return nil
		},
	}
	return &whoamiCmd
}
