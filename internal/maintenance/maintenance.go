package maintenance

import (
	"net/http"
	"time"

	"buf.build/gen/go/openstatus/api/connectrpc/gosimple/openstatus/maintenance/v1/maintenancev1connect"
	"connectrpc.com/connect"
	"github.com/fatih/color"
	"github.com/openstatusHQ/cli/internal/api"
	"github.com/urfave/cli/v3"
)

func NewMaintenanceClient(apiKey string) maintenancev1connect.MaintenanceServiceClient {
	return maintenancev1connect.NewMaintenanceServiceClient(
		api.DefaultHTTPClient,
		api.ConnectBaseURL,
		connect.WithInterceptors(api.NewAuthInterceptor(apiKey)),
		connect.WithProtoJSON(),
	)
}

func NewMaintenanceClientWithHTTPClient(httpClient *http.Client, apiKey string) maintenancev1connect.MaintenanceServiceClient {
	return maintenancev1connect.NewMaintenanceServiceClient(
		httpClient,
		api.ConnectBaseURL,
		connect.WithInterceptors(api.NewAuthInterceptor(apiKey)),
		connect.WithProtoJSON(),
	)
}

func timeWindowStatus(from, to string) string {
	fromTime, err := time.Parse(time.RFC3339, from)
	if err != nil {
		return "unknown"
	}
	toTime, err := time.Parse(time.RFC3339, to)
	if err != nil {
		return "unknown"
	}

	now := time.Now().UTC()
	switch {
	case now.Before(fromTime):
		return "scheduled"
	case now.After(toTime):
		return "completed"
	default:
		return "in_progress"
	}
}

func statusColor(s string) string {
	switch s {
	case "scheduled":
		return color.BlueString(s)
	case "in_progress":
		return color.YellowString(s)
	case "completed":
		return color.GreenString(s)
	default:
		return s
	}
}

func MaintenanceCmd() *cli.Command {
	return &cli.Command{
		Name:    "maintenance",
		Aliases: []string{"mt"},
		Usage:   "Manage maintenance windows",
		Commands: []*cli.Command{
			GetMaintenanceListCmd(),
			GetMaintenanceInfoCmd(),
			GetMaintenanceCreateCmd(),
			GetMaintenanceUpdateCmd(),
			GetMaintenanceDeleteCmd(),
		},
	}
}
