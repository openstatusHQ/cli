package main

import (
	cmd "github.com/openstatusHQ/cli/internal/cmd"
	"github.com/joho/godotenv"
	"log"
)

func main() {
	_ = godotenv.Load()

	app := cmd.NewApp()

	if err := cmd.RunApp(app); err != nil {
		log.Fatal(err)
	}
}
