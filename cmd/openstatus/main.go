package main

import (
	"log"

	"github.com/joho/godotenv"

	cmd "github.com/openstatusHQ/cli/internal/cmd"
)

func main() {
	_ = godotenv.Load()

	app := cmd.NewApp()

	if err := cmd.RunApp(app); err != nil {
		log.Fatal(err)
	}
}
