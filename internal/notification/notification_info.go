package notification

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"

	"buf.build/gen/go/openstatus/api/connectrpc/gosimple/openstatus/notification/v1/notificationv1connect"
	notificationv1 "buf.build/gen/go/openstatus/api/protocolbuffers/go/openstatus/notification/v1"
	"github.com/logrusorgru/aurora/v4"
	"github.com/olekukonko/tablewriter"
	"github.com/olekukonko/tablewriter/renderer"
	"github.com/olekukonko/tablewriter/tw"
	"github.com/openstatusHQ/cli/internal/auth"
	output "github.com/openstatusHQ/cli/internal/cli"
	"github.com/urfave/cli/v3"
)

type notificationDetail struct {
	ID         string         `json:"id"`
	Name       string         `json:"name"`
	Provider   string         `json:"provider"`
	Data       map[string]any `json:"data"`
	MonitorIDs []string       `json:"monitor_ids"`
	CreatedAt  string         `json:"created_at"`
	UpdatedAt  string         `json:"updated_at"`
}

func extractNotificationData(data *notificationv1.NotificationData) (map[string]any, [][]string) {
	if data == nil {
		return map[string]any{}, [][]string{}
	}

	switch d := data.GetData().(type) {
	case *notificationv1.NotificationData_Discord:
		url := d.Discord.GetWebhookUrl()
		return map[string]any{"discord": map[string]any{"webhook_url": url}},
			[][]string{{"Webhook URL", url}}

	case *notificationv1.NotificationData_Email:
		email := d.Email.GetEmail()
		return map[string]any{"email": map[string]any{"email": email}},
			[][]string{{"Email", email}}

	case *notificationv1.NotificationData_Slack:
		url := d.Slack.GetWebhookUrl()
		return map[string]any{"slack": map[string]any{"webhook_url": url}},
			[][]string{{"Webhook URL", url}}

	case *notificationv1.NotificationData_GoogleChat:
		url := d.GoogleChat.GetWebhookUrl()
		return map[string]any{"google_chat": map[string]any{"webhook_url": url}},
			[][]string{{"Webhook URL", url}}

	case *notificationv1.NotificationData_GrafanaOncall:
		url := d.GrafanaOncall.GetWebhookUrl()
		return map[string]any{"grafana_oncall": map[string]any{"webhook_url": url}},
			[][]string{{"Webhook URL", url}}

	case *notificationv1.NotificationData_Pagerduty:
		key := d.Pagerduty.GetIntegrationKey()
		return map[string]any{"pagerduty": map[string]any{"integration_key": key}},
			[][]string{{"Integration Key", key}}

	case *notificationv1.NotificationData_Opsgenie:
		apiKey := d.Opsgenie.GetApiKey()
		region := opsgenieRegionToString(d.Opsgenie.GetRegion())
		return map[string]any{"opsgenie": map[string]any{"api_key": apiKey, "region": region}},
			[][]string{{"API Key", apiKey}, {"Region", region}}

	case *notificationv1.NotificationData_Ntfy:
		topic := d.Ntfy.GetTopic()
		serverUrl := d.Ntfy.GetServerUrl()
		hasToken := d.Ntfy.HasToken()

		jsonMap := map[string]any{"topic": topic, "has_token": hasToken}
		rows := [][]string{{"Topic", topic}}

		if serverUrl != "" {
			jsonMap["server_url"] = serverUrl
			rows = append(rows, []string{"Server URL", serverUrl})
		}

		tokenStatus := "not configured"
		if hasToken {
			tokenStatus = "configured"
		}
		rows = append(rows, []string{"Token", tokenStatus})

		return map[string]any{"ntfy": jsonMap}, rows

	case *notificationv1.NotificationData_Sms:
		phone := d.Sms.GetPhoneNumber()
		return map[string]any{"sms": map[string]any{"phone_number": phone}},
			[][]string{{"Phone Number", phone}}

	case *notificationv1.NotificationData_Telegram:
		chatId := d.Telegram.GetChatId()
		return map[string]any{"telegram": map[string]any{"chat_id": chatId}},
			[][]string{{"Chat ID", chatId}}

	case *notificationv1.NotificationData_Webhook:
		endpoint := d.Webhook.GetEndpoint()
		jsonMap := map[string]any{"endpoint": endpoint}
		rows := [][]string{{"Endpoint", endpoint}}

		headers := d.Webhook.GetHeaders()
		if len(headers) > 0 {
			headerList := make([]map[string]any, 0, len(headers))
			for _, h := range headers {
				headerList = append(headerList, map[string]any{"key": h.GetKey(), "value": h.GetValue()})
				rows = append(rows, []string{"Header", h.GetKey() + ": " + h.GetValue()})
			}
			jsonMap["headers"] = headerList
		}

		return map[string]any{"webhook": jsonMap}, rows

	case *notificationv1.NotificationData_Whatsapp:
		phone := d.Whatsapp.GetPhoneNumber()
		return map[string]any{"whatsapp": map[string]any{"phone_number": phone}},
			[][]string{{"Phone Number", phone}}

	default:
		return map[string]any{}, [][]string{}
	}
}

func GetNotificationInfo(ctx context.Context, client notificationv1connect.NotificationServiceClient, notificationId string, s *output.Spinner) error {
	if notificationId == "" {
		output.StopSpinner(s)
		fmt.Fprintln(os.Stderr, "Usage: openstatus notification info <notification-id>")
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "Example: openstatus notification info 12345")
		return fmt.Errorf("notification ID is required")
	}

	req := &notificationv1.GetNotificationRequest{}
	req.SetId(notificationId)
	resp, err := client.GetNotification(ctx, req)
	output.StopSpinner(s)
	if err != nil {
		return output.FormatError(err, "notification", notificationId)
	}

	n := resp.GetNotification()
	dataMap, dataRows := extractNotificationData(n.GetData())

	if output.IsJSONOutput() {
		monitorIDs := n.GetMonitorIds()
		if monitorIDs == nil {
			monitorIDs = []string{}
		}
		detail := notificationDetail{
			ID:         n.GetId(),
			Name:       n.GetName(),
			Provider:   providerToString(n.GetProvider()),
			Data:       dataMap,
			MonitorIDs: monitorIDs,
			CreatedAt:  n.GetCreatedAt(),
			UpdatedAt:  n.GetUpdatedAt(),
		}
		return output.PrintJSON(detail)
	}

	fmt.Println(aurora.Bold("Notification:"))
	tbl := tablewriter.NewTable(os.Stdout,
		tablewriter.WithRenderer(renderer.NewBlueprint()),
		tablewriter.WithRendition(tw.Rendition{
			Symbols: tw.NewSymbolCustom("custom").WithColumn("="),
			Borders: tw.Border{
				Top:    tw.Off,
				Left:   tw.Off,
				Right:  tw.Off,
				Bottom: tw.Off,
			},
			Settings: tw.Settings{
				Lines: tw.Lines{
					ShowHeaderLine: tw.Off,
					ShowFooterLine: tw.On,
				},
				Separators: tw.Separators{
					BetweenRows:    tw.Off,
					BetweenColumns: tw.On,
				},
			},
		}),
		tablewriter.WithRowAlignment(tw.AlignLeft),
		tablewriter.WithHeaderAlignment(tw.AlignLeft),
	)

	data := [][]string{
		{"ID", n.GetId()},
		{"Name", n.GetName()},
		{"Provider", providerToString(n.GetProvider())},
	}

	data = append(data, dataRows...)

	monitorIDs := n.GetMonitorIds()
	if len(monitorIDs) > 0 {
		data = append(data, []string{"Monitor IDs", strings.Join(monitorIDs, ", ")})
	} else {
		data = append(data, []string{"Monitor IDs", "none"})
	}

	data = append(data, []string{"Created At", output.FormatTimestamp(n.GetCreatedAt())})
	data = append(data, []string{"Updated At", output.FormatTimestamp(n.GetUpdatedAt())})

	tbl.Bulk(data)
	tbl.Render()

	return nil
}

func GetNotificationInfoWithHTTPClient(ctx context.Context, httpClient *http.Client, apiKey string, notificationId string) error {
	client := NewNotificationClientWithHTTPClient(httpClient, apiKey)
	return GetNotificationInfo(ctx, client, notificationId, nil)
}

func GetNotificationInfoCmd() *cli.Command {
	return &cli.Command{
		Name:  "info",
		Usage: "Get notification details",
		UsageText: `openstatus notification info <NotificationID>
  openstatus notification info 12345`,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "access-token",
				Usage:   "OpenStatus API Access Token",
				Aliases: []string{"t"},
				Sources: cli.EnvVars("OPENSTATUS_API_TOKEN"),
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			apiKey, err := auth.ResolveAccessToken(cmd)
			if err != nil {
				return cli.Exit(err.Error(), 1)
			}
			notificationId := cmd.Args().Get(0)
			s := output.StartSpinner("Fetching notification...")
			client := NewNotificationClient(apiKey)
			err = GetNotificationInfo(ctx, client, notificationId, s)
			if err != nil {
				return cli.Exit(err.Error(), 1)
			}
			return nil
		},
	}
}
