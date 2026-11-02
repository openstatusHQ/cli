package main

import (
	"context"

	"github.com/joho/godotenv"

	"github.com/openstatusHQ/cli/internal/api"
	output "github.com/openstatusHQ/cli/internal/cli"
	cmd "github.com/openstatusHQ/cli/internal/cmd"
)

func main() {
	_ = godotenv.Load()
	if err := api.LoadBaseURL(); err != nil {
		output.HandleExitError(context.Background(), nil, output.UsageErrorWithHint(err.Error(), ""))
	}

	app := cmd.NewApp()

	// Command errors are printed and exit inside app.Run via ExitErrHandler;
	// this catches the few urfave/cli returns without calling it (e.g.
	// missing required flags).
	if err := cmd.RunApp(app); err != nil {
		output.HandleExitError(context.Background(), app, err)
	}
}
