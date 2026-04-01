package notification

import (
	"net/http"

	"buf.build/gen/go/openstatus/api/connectrpc/gosimple/openstatus/notification/v1/notificationv1connect"
	notificationv1 "buf.build/gen/go/openstatus/api/protocolbuffers/go/openstatus/notification/v1"
	"connectrpc.com/connect"
	"github.com/openstatusHQ/cli/internal/api"
	"github.com/urfave/cli/v3"
)

func NewNotificationClient(apiKey string) notificationv1connect.NotificationServiceClient {
	return notificationv1connect.NewNotificationServiceClient(
		api.DefaultHTTPClient,
		api.ConnectBaseURL,
		connect.WithInterceptors(api.NewAuthInterceptor(apiKey)),
		connect.WithProtoJSON(),
	)
}

func NewNotificationClientWithHTTPClient(httpClient *http.Client, apiKey string) notificationv1connect.NotificationServiceClient {
	return notificationv1connect.NewNotificationServiceClient(
		httpClient,
		api.ConnectBaseURL,
		connect.WithInterceptors(api.NewAuthInterceptor(apiKey)),
		connect.WithProtoJSON(),
	)
}

func providerToString(p notificationv1.NotificationProvider) string {
	switch p {
	case notificationv1.NotificationProvider_NOTIFICATION_PROVIDER_DISCORD:
		return "discord"
	case notificationv1.NotificationProvider_NOTIFICATION_PROVIDER_EMAIL:
		return "email"
	case notificationv1.NotificationProvider_NOTIFICATION_PROVIDER_GOOGLE_CHAT:
		return "google_chat"
	case notificationv1.NotificationProvider_NOTIFICATION_PROVIDER_GRAFANA_ONCALL:
		return "grafana_oncall"
	case notificationv1.NotificationProvider_NOTIFICATION_PROVIDER_NTFY:
		return "ntfy"
	case notificationv1.NotificationProvider_NOTIFICATION_PROVIDER_PAGERDUTY:
		return "pagerduty"
	case notificationv1.NotificationProvider_NOTIFICATION_PROVIDER_OPSGENIE:
		return "opsgenie"
	case notificationv1.NotificationProvider_NOTIFICATION_PROVIDER_SLACK:
		return "slack"
	case notificationv1.NotificationProvider_NOTIFICATION_PROVIDER_SMS:
		return "sms"
	case notificationv1.NotificationProvider_NOTIFICATION_PROVIDER_TELEGRAM:
		return "telegram"
	case notificationv1.NotificationProvider_NOTIFICATION_PROVIDER_WEBHOOK:
		return "webhook"
	case notificationv1.NotificationProvider_NOTIFICATION_PROVIDER_WHATSAPP:
		return "whatsapp"
	default:
		return "unknown"
	}
}

func opsgenieRegionToString(r notificationv1.OpsgenieRegion) string {
	switch r {
	case notificationv1.OpsgenieRegion_OPSGENIE_REGION_US:
		return "us"
	case notificationv1.OpsgenieRegion_OPSGENIE_REGION_EU:
		return "eu"
	default:
		return "unknown"
	}
}

func NotificationCmd() *cli.Command {
	return &cli.Command{
		Name:    "notification",
		Aliases: []string{"n"},
		Usage:   "Manage notifications",
		Commands: []*cli.Command{
			GetNotificationListCmd(),
			GetNotificationInfoCmd(),
		},
	}
}
