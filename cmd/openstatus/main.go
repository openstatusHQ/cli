package main

import (
	"github.com/joho/godotenv"
	cmd "github.com/openstatusHQ/cli/internal/cmd"
	"log"
)

func main() {
	_ = godotenv.Load()

	app := cmd.NewApp()

	if err := cmd.RunApp(app); err != nil {
		log.Fatal(err)
	}
}
