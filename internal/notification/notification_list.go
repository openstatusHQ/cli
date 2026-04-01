package notification

import (
	"context"
	"fmt"
	"net/http"

	"buf.build/gen/go/openstatus/api/connectrpc/gosimple/openstatus/notification/v1/notificationv1connect"
	notificationv1 "buf.build/gen/go/openstatus/api/protocolbuffers/go/openstatus/notification/v1"
	"github.com/fatih/color"
	"github.com/openstatusHQ/cli/internal/auth"
	output "github.com/openstatusHQ/cli/internal/cli"
	"github.com/rodaine/table"
	"github.com/urfave/cli/v3"
)

type notificationListEntry struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	Provider     string `json:"provider"`
	MonitorCount int32  `json:"monitor_count"`
	CreatedAt    string `json:"created_at"`
	UpdatedAt    string `json:"updated_at"`
}

func ListNotifications(ctx context.Context, client notificationv1connect.NotificationServiceClient, limit int, s *output.Spinner) error {
	req := &notificationv1.ListNotificationsRequest{}

	if limit > 0 {
		l := int32(limit)
		req.SetLimit(l)
	}

	resp, err := client.ListNotifications(ctx, req)
	output.StopSpinner(s)
	if err != nil {
		return output.FormatError(err, "notification", "")
	}

	notifications := resp.GetNotifications()

	if output.IsJSONOutput() {
		entries := make([]notificationListEntry, 0, len(notifications))
		for _, n := range notifications {
			entries = append(entries, notificationListEntry{
				ID:           n.GetId(),
				Name:         n.GetName(),
				Provider:     providerToString(n.GetProvider()),
				MonitorCount: n.GetMonitorCount(),
				CreatedAt:    n.GetCreatedAt(),
				UpdatedAt:    n.GetUpdatedAt(),
			})
		}
		return output.PrintJSON(entries)
	}

	if len(notifications) == 0 {
		if !output.IsQuiet() {
			fmt.Println("No notifications found")
		}
		return nil
	}

	headerFmt := color.New(color.FgGreen, color.Underline).SprintfFunc()
	columnFmt := color.New(color.FgYellow).SprintfFunc()

	tbl := table.New("ID", "Name", "Provider", "Monitors")
	tbl.WithHeaderFormatter(headerFmt).WithFirstColumnFormatter(columnFmt)

	for _, n := range notifications {
		tbl.AddRow(
			n.GetId(),
			n.GetName(),
			providerToString(n.GetProvider()),
			n.GetMonitorCount(),
		)
	}

	tbl.Print()
	return nil
}

func ListNotificationsWithHTTPClient(ctx context.Context, httpClient *http.Client, apiKey string, limit int) error {
	client := NewNotificationClientWithHTTPClient(httpClient, apiKey)
	return ListNotifications(ctx, client, limit, nil)
}

func GetNotificationListCmd() *cli.Command {
	return &cli.Command{
		Name:  "list",
		Usage: "List all notifications",
		UsageText: `openstatus notification list
  openstatus notification list --limit 10`,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "access-token",
				Usage:   "OpenStatus API Access Token",
				Aliases: []string{"t"},
				Sources: cli.EnvVars("OPENSTATUS_API_TOKEN"),
			},
			&cli.IntFlag{
				Name:  "limit",
				Usage: "Maximum number of notifications to return (1-100)",
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			apiKey, err := auth.ResolveAccessToken(cmd)
			if err != nil {
				return cli.Exit(err.Error(), 1)
			}
			s := output.StartSpinner("Fetching notifications...")
			client := NewNotificationClient(apiKey)
			err = ListNotifications(ctx, client, int(cmd.Int("limit")), s)
			if err != nil {
				return cli.Exit(err.Error(), 1)
			}
			return nil
		},
	}
}
