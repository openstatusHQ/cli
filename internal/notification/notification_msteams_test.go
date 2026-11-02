package notification_test

import (
	"testing"

	notificationv1 "buf.build/gen/go/openstatus/api/protocolbuffers/go/openstatus/notification/v1"

	"github.com/openstatusHQ/cli/internal/notification"
)

func Test_MsTeamsProvider(t *testing.T) {
	t.Parallel()

	if got := notification.ProviderToString(notificationv1.NotificationProvider_NOTIFICATION_PROVIDER_MS_TEAMS); got != "ms_teams" {
		t.Errorf("Expected 'ms_teams', got %q", got)
	}

	data := &notificationv1.NotificationData{}
	data.SetMsTeams(&notificationv1.MsTeamsData{WebhookUrl: "https://example.webhook.office.com/x"})
	jsonMap, rows := notification.ExtractNotificationData(data)
	if _, ok := jsonMap["ms_teams"]; !ok {
		t.Errorf("Expected ms_teams key in JSON data, got %v", jsonMap)
	}
	if len(rows) != 1 || rows[0][1] != "https://example.webhook.office.com/x" {
		t.Errorf("Expected webhook URL row, got %v", rows)
	}
}
