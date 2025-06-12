package main

import (
	"context"
	"log"
	"os"
	cmd "github.com/openstatusHQ/cli/internal/cmd"
)

func main() {
	app := cmd.NewApp()

	if err := app.Run(context.Background(), os.Args); err != nil {
		log.Fatal(err)
	}
}
