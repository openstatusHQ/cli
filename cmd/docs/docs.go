package main

import (
	"os"

	cmd "github.com/openstatusHQ/cli/internal/cmd"
	docs "github.com/urfave/cli-docs/v3"
)

func main() {
	app := cmd.NewApp()
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
