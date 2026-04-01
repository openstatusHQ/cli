package terraform

import (
	"github.com/urfave/cli/v3"
)

func TerraformCmd() *cli.Command {
	return &cli.Command{
		Name:    "terraform",
		Aliases: []string{"tf"},
		Usage:   "Generate Terraform configuration",
		Commands: []*cli.Command{
			GetTerraformGenerateCmd(),
		},
	}
}
