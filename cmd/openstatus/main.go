package main

import (
	"context"
	cmd "github.com/openstatusHQ/cli/internal/cmd"
	"log"
	"os"
)

func main() {
	app := cmd.NewApp()

	if err := app.Run(context.Background(), os.Args); err != nil {
		log.Fatal(err)
	}
}
