//go:build smoke

// Run with: go test -tags=smoke ./internal/terraform/
//
// This smoke test exercises the generator's output against a real terraform
// binary and the live openstatusHQ/openstatus provider from the public
// registry. It does NOT require an API token and does NOT call any
// OpenStatus RPC — only `terraform init` (which fetches the provider) and
// `terraform validate` (which checks schema/syntax fit).
//
// Skipped automatically when the terraform binary is not on PATH.

package terraform

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	monitorv1 "buf.build/gen/go/openstatus/api/protocolbuffers/go/openstatus/monitor/v1"
	notificationv1 "buf.build/gen/go/openstatus/api/protocolbuffers/go/openstatus/notification/v1"
	status_pagev1 "buf.build/gen/go/openstatus/api/protocolbuffers/go/openstatus/status_page/v1"
)

func TestSmokeValidate(t *testing.T) {
	if _, err := exec.LookPath("terraform"); err != nil {
		t.Skipf("terraform binary not in PATH; install terraform 1.0+ to run this test")
	}

	dir := t.TempDir()
	data := smokeFixture()
	gen := NewGenerator(data)

	files := map[string][]byte{
		"provider.tf":      GenerateProviderFile(),
		"monitors.tf":      gen.GenerateMonitorsFile().Bytes(),
		"notifications.tf": gen.GenerateNotificationsFile().Bytes(),
		"status_pages.tf":  gen.GenerateStatusPagesFile().Bytes(),
		"imports.tf":       gen.GenerateImportsFile().Bytes(),
	}
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(dir, name), content, 0644); err != nil {
			t.Fatalf("writing %s: %v", name, err)
		}
	}

	initCmd := exec.Command("terraform", "init", "-upgrade", "-no-color")
	initCmd.Dir = dir
	if out, err := initCmd.CombinedOutput(); err != nil {
		t.Fatalf("terraform init failed: %v\noutput:\n%s", err, out)
	}

	validateCmd := exec.Command("terraform", "validate", "-no-color")
	validateCmd.Dir = dir
	if out, err := validateCmd.CombinedOutput(); err != nil {
		t.Fatalf("terraform validate failed: %v\noutput:\n%s", err, out)
	}
}

func smokeFixture() *WorkspaceData {
	httpMon := &monitorv1.HTTPMonitor{}
	httpMon.SetId("mon-http")
	httpMon.SetName("API Health")
	httpMon.SetUrl("https://api.example.com/health")
	httpMon.SetPeriodicity(monitorv1.Periodicity_PERIODICITY_5M)
	httpMon.SetMethod(monitorv1.HTTPMethod_HTTP_METHOD_GET)
	httpMon.SetTimeout(45000)
	httpMon.SetRetry(3)
	httpMon.SetFollowRedirects(true)
	httpMon.SetActive(true)
	httpMon.SetRegions([]monitorv1.Region{monitorv1.Region_REGION_FLY_IAD})

	otelConfig := &monitorv1.OpenTelemetryConfig{}
	otelConfig.SetEndpoint("https://otel.example.com/v1/metrics")
	otelHeader := &monitorv1.Headers{}
	otelHeader.SetKey("X-Api-Key")
	otelHeader.SetValue("secret")
	otelConfig.SetHeaders([]*monitorv1.Headers{otelHeader})
	httpMon.SetOpenTelemetry(otelConfig)

	statusAssertion := &monitorv1.StatusCodeAssertion{}
	statusAssertion.SetTarget(200)
	statusAssertion.SetComparator(monitorv1.NumberComparator_NUMBER_COMPARATOR_EQUAL)
	httpMon.SetStatusCodeAssertions([]*monitorv1.StatusCodeAssertion{statusAssertion})

	tcpMon := &monitorv1.TCPMonitor{}
	tcpMon.SetId("mon-tcp")
	tcpMon.SetName("DB TCP")
	tcpMon.SetUri("db.example.com:5432")
	tcpMon.SetPeriodicity(monitorv1.Periodicity_PERIODICITY_5M)
	tcpMon.SetTimeout(45000)
	tcpMon.SetRetry(3)
	tcpMon.SetActive(true)

	dnsMon := &monitorv1.DNSMonitor{}
	dnsMon.SetId("mon-dns")
	dnsMon.SetName("DNS Check")
	dnsMon.SetUri("example.com")
	dnsMon.SetPeriodicity(monitorv1.Periodicity_PERIODICITY_10M)
	dnsMon.SetTimeout(45000)
	dnsMon.SetRetry(3)
	dnsMon.SetActive(true)
	dnsRecord := &monitorv1.RecordAssertion{}
	dnsRecord.SetRecord("A")
	dnsRecord.SetTarget("93.184.216.34")
	dnsRecord.SetComparator(monitorv1.RecordComparator_RECORD_COMPARATOR_EQUAL)
	dnsMon.SetRecordAssertions([]*monitorv1.RecordAssertion{dnsRecord})

	slackNotif := newNotification("notif-slack", "Slack Alerts", []string{"mon-http"}, func(d *notificationv1.NotificationData) {
		sd := &notificationv1.SlackData{}
		sd.SetWebhookUrl("https://hooks.example.com/slack")
		d.SetSlack(sd)
	})
	teamsNotif := newNotification("notif-teams", "Teams Alerts", nil, func(d *notificationv1.NotificationData) {
		td := &notificationv1.MsTeamsData{}
		td.SetWebhookUrl("https://example.com/teams")
		d.SetMsTeams(td)
	})
	webhookNotif := newNotification("notif-webhook", "Webhook Alerts", nil, func(d *notificationv1.NotificationData) {
		wd := &notificationv1.WebhookData{}
		wd.SetEndpoint("https://example.com/hook")
		h := &notificationv1.WebhookHeader{}
		h.SetKey("Authorization")
		h.SetValue("Bearer xyz")
		wd.SetHeaders([]*notificationv1.WebhookHeader{h})
		d.SetWebhook(wd)
	})

	page := &status_pagev1.StatusPage{}
	page.SetId("page-1")
	page.SetTitle("Public Status")
	page.SetSlug("public-status")
	page.SetTheme(status_pagev1.PageTheme_PAGE_THEME_DARK)
	page.SetDefaultLocale(status_pagev1.Locale_LOCALE_EN)
	page.SetLocales([]status_pagev1.Locale{
		status_pagev1.Locale_LOCALE_EN,
		status_pagev1.Locale_LOCALE_FR,
	})
	page.SetAllowIndex(true)

	group := &status_pagev1.PageComponentGroup{}
	group.SetId("group-1")
	group.SetPageId("page-1")
	group.SetName("Infrastructure")
	group.SetDefaultOpen(true)

	component := &status_pagev1.PageComponent{}
	component.SetId("comp-1")
	component.SetPageId("page-1")
	component.SetName("API")
	component.SetType(status_pagev1.PageComponentType_PAGE_COMPONENT_TYPE_MONITOR)
	component.SetMonitorId("mon-http")
	component.SetOrder(1)
	component.SetGroupId("group-1")
	component.SetGroupOrder(1)

	ipPage := &status_pagev1.StatusPage{}
	ipPage.SetId("page-2")
	ipPage.SetTitle("Internal")
	ipPage.SetSlug("internal")
	ipPage.SetAccessType(status_pagev1.PageAccessType_PAGE_ACCESS_TYPE_IP_RESTRICTED)
	ipPage.SetAllowedIpRanges("10.0.0.0/8,192.168.0.0/16")

	return &WorkspaceData{
		HTTPMonitors:  []*monitorv1.HTTPMonitor{httpMon},
		TCPMonitors:   []*monitorv1.TCPMonitor{tcpMon},
		DNSMonitors:   []*monitorv1.DNSMonitor{dnsMon},
		Notifications: []*notificationv1.Notification{slackNotif, teamsNotif, webhookNotif},
		StatusPages: []StatusPageData{
			{
				Page:       page,
				Groups:     []*status_pagev1.PageComponentGroup{group},
				Components: []*status_pagev1.PageComponent{component},
			},
			{Page: ipPage},
		},
	}
}

func newNotification(id, name string, monitorIDs []string, setData func(*notificationv1.NotificationData)) *notificationv1.Notification {
	n := &notificationv1.Notification{}
	n.SetId(id)
	n.SetName(name)
	if len(monitorIDs) > 0 {
		n.SetMonitorIds(monitorIDs)
	}
	nd := &notificationv1.NotificationData{}
	setData(nd)
	n.SetData(nd)
	return n
}
