
package main

import (
	"os"

	docs "github.com/urfave/cli-docs/v3"
	cmd "github.com/openstatusHQ/cli/internal/cmd"
)

func main() {
	app :=  cmd.NewApp()
	md, err := docs.ToMarkdown(app)
	if err != nil {
		panic(err)
	}

	fi, err := os.Create("./docs/openstatus-docs.md")
	if err != nil {
		panic(err)
	}
	defer fi.Close()
	if _, err := fi.WriteString("# OpenStatus CLI\n\n" + md); err != nil {
		panic(err)
	}
}
