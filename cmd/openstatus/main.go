package main

import (
	"log"

	"github.com/joho/godotenv"

	"github.com/openstatusHQ/cli/internal/api"
	cmd "github.com/openstatusHQ/cli/internal/cmd"
)

func main() {
	_ = godotenv.Load()
	if err := api.LoadBaseURL(); err != nil {
		log.Fatal(err)
	}

	app := cmd.NewApp()

	if err := cmd.RunApp(app); err != nil {
		log.Fatal(err)
	}
}
