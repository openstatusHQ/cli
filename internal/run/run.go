package run

import (
	"context"
	"fmt"

	"github.com/urfave/cli/v3"
)

func RunCmd() *cli.Command {
	runCmd := cli.Command{
		Name:    "run",
		Aliases: []string{"r"},
		Usage:   "Run your synthetics tests defined in your configuration file",
		Action: func(ctx context.Context, cmd *cli.Command) error {

			fmt.Println(cmd.String("config"))
			fmt.Println("Test ran 🔥")
			return nil
		},
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        "config",
				Usage:       "The configuration file",
				DefaultText: "config.openstatus.yaml",
				Value:       "config.openstatus.yaml",
			},
			&cli.StringFlag{
				Name:     "access-token",
				Usage:    "OpenStatus API Access Token",
				Aliases:  []string{"t"},
				Sources:  cli.EnvVars("OPENSTATUS_API_TOKEN"),
				Required: true,
			},
		},
	}
	return &runCmd
}
