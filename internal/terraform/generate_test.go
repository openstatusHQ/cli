package terraform

import (
	"strings"
	"testing"

	monitorv1 "buf.build/gen/go/openstatus/api/protocolbuffers/go/openstatus/monitor/v1"
	notificationv1 "buf.build/gen/go/openstatus/api/protocolbuffers/go/openstatus/notification/v1"
	status_pagev1 "buf.build/gen/go/openstatus/api/protocolbuffers/go/openstatus/status_page/v1"
)

func TestGenerateProviderFile(t *testing.T) {
	content := string(GenerateProviderFile())
	mustContain(t, content, `source  = "openstatusHQ/openstatus"`)
	mustContain(t, content, `version = "~> 0.2"`)
	mustContain(t, content, `provider "openstatus" {}`)
	mustContain(t, content, `OPENSTATUS_API_TOKEN`)
}

func TestGenerateMonitorsFile_HTTP(t *testing.T) {
	m := &monitorv1.HTTPMonitor{}
	m.SetId("123")
	m.SetName("API Health")
	m.SetUrl("https://api.example.com/health")
	m.SetPeriodicity(monitorv1.Periodicity_PERIODICITY_5M)
	m.SetMethod(monitorv1.HTTPMethod_HTTP_METHOD_POST)
	m.SetTimeout(45000)
	m.SetRetry(3)
	m.SetFollowRedirects(true)
	m.SetActive(true)
	m.SetRegions([]monitorv1.Region{monitorv1.Region_REGION_FLY_IAD})

	data := &WorkspaceData{HTTPMonitors: []*monitorv1.HTTPMonitor{m}}
	gen := NewGenerator(data)
	content := string(gen.GenerateMonitorsFile().Bytes())

	mustContain(t, content, `resource "openstatus_http_monitor" "api_health"`)
	mustContain(t, content, `name        = "API Health"`)
	mustContain(t, content, `periodicity = "5m"`)
	mustContain(t, content, `method      = "POST"`)
	mustContain(t, content, `active      = true`)
	mustContain(t, content, `"fly-iad"`)
	// timeout, retry, follow_redirects should be omitted at defaults
	mustNotContain(t, content, "timeout")
	mustNotContain(t, content, "retry")
	mustNotContain(t, content, "follow_redirects")
}

func TestGenerateMonitorsFile_DNS(t *testing.T) {
	m := &monitorv1.DNSMonitor{}
	m.SetId("456")
	m.SetName("DNS Check")
	m.SetUri("example.com")
	m.SetPeriodicity(monitorv1.Periodicity_PERIODICITY_10M)
	m.SetActive(true)

	a := &monitorv1.RecordAssertion{}
	a.SetRecord("A")
	a.SetTarget("93.184.216.34")
	a.SetComparator(monitorv1.RecordComparator_RECORD_COMPARATOR_EQUAL)
	m.SetRecordAssertions([]*monitorv1.RecordAssertion{a})

	data := &WorkspaceData{DNSMonitors: []*monitorv1.DNSMonitor{m}}
	gen := NewGenerator(data)
	content := string(gen.GenerateMonitorsFile().Bytes())

	mustContain(t, content, `resource "openstatus_dns_monitor" "dns_check"`)
	mustContain(t, content, `record_assertions`)
	mustContain(t, content, `record     = "A"`)
	mustContain(t, content, `target     = "93.184.216.34"`)
}

func TestGenerateNotificationsFile_Slack(t *testing.T) {
	n := &notificationv1.Notification{}
	n.SetId("789")
	n.SetName("Slack Alerts")
	n.SetProvider(notificationv1.NotificationProvider_NOTIFICATION_PROVIDER_SLACK)
	n.SetMonitorIds([]string{"123"})

	nd := &notificationv1.NotificationData{}
	sd := &notificationv1.SlackData{}
	sd.SetWebhookUrl("https://hooks.slack.com/xxx")
	nd.SetSlack(sd)
	n.SetData(nd)

	data := &WorkspaceData{Notifications: []*notificationv1.Notification{n}}
	gen := NewGenerator(data)
	content := string(gen.GenerateNotificationsFile().Bytes())

	mustContain(t, content, `resource "openstatus_notification" "slack_alerts"`)
	mustContain(t, content, `provider_type = "slack"`)
	mustContain(t, content, `"123"`)
	mustContain(t, content, `webhook_url`)
}

func TestGenerateStatusPagesFile(t *testing.T) {
	page := &status_pagev1.StatusPage{}
	page.SetId("p1")
	page.SetTitle("My Status Page")
	page.SetSlug("my-status")

	comp := &status_pagev1.PageComponent{}
	comp.SetId("c1")
	comp.SetPageId("p1")
	comp.SetName("API")
	comp.SetType(status_pagev1.PageComponentType_PAGE_COMPONENT_TYPE_STATIC)
	comp.SetOrder(1)

	data := &WorkspaceData{
		StatusPages: []StatusPageData{
			{Page: page, Components: []*status_pagev1.PageComponent{comp}},
		},
	}
	gen := NewGenerator(data)
	content := string(gen.GenerateStatusPagesFile().Bytes())

	mustContain(t, content, `resource "openstatus_status_page" "my_status_page"`)
	mustContain(t, content, `title = "My Status Page"`)
	mustContain(t, content, `slug  = "my-status"`)
	mustContain(t, content, `resource "openstatus_status_page_component" "api"`)
	mustContain(t, content, `openstatus_status_page.my_status_page.id`)
}

func TestGenerateImportsFile(t *testing.T) {
	m := &monitorv1.HTTPMonitor{}
	m.SetId("123")
	m.SetName("API")

	page := &status_pagev1.StatusPage{}
	page.SetId("p1")
	page.SetTitle("Page")

	comp := &status_pagev1.PageComponent{}
	comp.SetId("c1")
	comp.SetPageId("p1")
	comp.SetName("Comp")
	comp.SetType(status_pagev1.PageComponentType_PAGE_COMPONENT_TYPE_STATIC)

	data := &WorkspaceData{
		HTTPMonitors: []*monitorv1.HTTPMonitor{m},
		StatusPages:  []StatusPageData{{Page: page, Components: []*status_pagev1.PageComponent{comp}}},
	}
	gen := NewGenerator(data)
	content := string(gen.GenerateImportsFile().Bytes())

	mustContain(t, content, `openstatus_http_monitor.api`)
	mustContain(t, content, `id = "123"`)
	mustContain(t, content, `id = "p1/c1"`)
}

func TestEmptyWorkspace(t *testing.T) {
	data := &WorkspaceData{}
	gen := NewGenerator(data)

	if gen.TotalResourceCount() != 0 {
		t.Errorf("TotalResourceCount() = %d, want 0", gen.TotalResourceCount())
	}
	if gen.HasMonitors() {
		t.Error("HasMonitors() should be false")
	}
	if gen.HasNotifications() {
		t.Error("HasNotifications() should be false")
	}
	if gen.HasStatusPages() {
		t.Error("HasStatusPages() should be false")
	}
}

func TestCrossReferenceMonitorInComponent(t *testing.T) {
	m := &monitorv1.HTTPMonitor{}
	m.SetId("mon-1")
	m.SetName("API Monitor")

	page := &status_pagev1.StatusPage{}
	page.SetId("p1")
	page.SetTitle("Page")

	comp := &status_pagev1.PageComponent{}
	comp.SetId("c1")
	comp.SetPageId("p1")
	comp.SetName("API Comp")
	comp.SetType(status_pagev1.PageComponentType_PAGE_COMPONENT_TYPE_MONITOR)
	comp.SetMonitorId("mon-1")

	data := &WorkspaceData{
		HTTPMonitors: []*monitorv1.HTTPMonitor{m},
		StatusPages:  []StatusPageData{{Page: page, Components: []*status_pagev1.PageComponent{comp}}},
	}
	gen := NewGenerator(data)
	content := string(gen.GenerateStatusPagesFile().Bytes())

	mustContain(t, content, `openstatus_http_monitor.api_monitor.id`)
}

func TestCrossReferenceFallbackToString(t *testing.T) {
	page := &status_pagev1.StatusPage{}
	page.SetId("p1")
	page.SetTitle("Page")

	comp := &status_pagev1.PageComponent{}
	comp.SetId("c1")
	comp.SetPageId("p1")
	comp.SetName("Unknown Monitor Comp")
	comp.SetType(status_pagev1.PageComponentType_PAGE_COMPONENT_TYPE_MONITOR)
	comp.SetMonitorId("nonexistent-id")

	data := &WorkspaceData{
		StatusPages: []StatusPageData{{Page: page, Components: []*status_pagev1.PageComponent{comp}}},
	}
	gen := NewGenerator(data)
	content := string(gen.GenerateStatusPagesFile().Bytes())

	mustContain(t, content, `monitor_id = "nonexistent-id"`)
}

func TestGenerateNotificationsFile_MsTeams(t *testing.T) {
	n := &notificationv1.Notification{}
	n.SetId("1")
	n.SetName("Teams Alerts")

	nd := &notificationv1.NotificationData{}
	td := &notificationv1.MsTeamsData{}
	td.SetWebhookUrl("https://prod.example.com/webhook")
	nd.SetMsTeams(td)
	n.SetData(nd)

	data := &WorkspaceData{Notifications: []*notificationv1.Notification{n}}
	gen := NewGenerator(data)
	content := string(gen.GenerateNotificationsFile().Bytes())

	mustContain(t, content, `resource "openstatus_notification" "teams_alerts"`)
	mustContain(t, content, `provider_type = "ms_teams"`)
	mustContain(t, content, "ms_teams {")
	mustContain(t, content, `webhook_url = "https://prod.example.com/webhook"`)
}

func TestGenerateNotificationsFile_WebhookHeaders(t *testing.T) {
	n := &notificationv1.Notification{}
	n.SetId("1")
	n.SetName("Webhook")

	nd := &notificationv1.NotificationData{}
	wd := &notificationv1.WebhookData{}
	wd.SetEndpoint("https://example.com/hook")
	h1 := &notificationv1.WebhookHeader{}
	h1.SetKey("Authorization")
	h1.SetValue("Bearer xyz")
	h2 := &notificationv1.WebhookHeader{}
	h2.SetKey("X-Source")
	h2.SetValue("openstatus")
	wd.SetHeaders([]*notificationv1.WebhookHeader{h1, h2})
	nd.SetWebhook(wd)
	n.SetData(nd)

	data := &WorkspaceData{Notifications: []*notificationv1.Notification{n}}
	gen := NewGenerator(data)
	content := string(gen.GenerateNotificationsFile().Bytes())

	mustContain(t, content, `headers = [`)
	mustContain(t, content, `key   = "Authorization"`)
	mustContain(t, content, `value = "Bearer xyz"`)
	mustContain(t, content, `key   = "X-Source"`)
	mustNotContain(t, content, "headers {")
}

func TestGenerateNotificationsFile_MonitorIdsTraversal(t *testing.T) {
	m := &monitorv1.HTTPMonitor{}
	m.SetId("mon-known")
	m.SetName("API")

	n := &notificationv1.Notification{}
	n.SetId("n1")
	n.SetName("Slack")
	n.SetMonitorIds([]string{"mon-unknown", "mon-known"})

	nd := &notificationv1.NotificationData{}
	sd := &notificationv1.SlackData{}
	sd.SetWebhookUrl("https://hooks.example.com/xxx")
	nd.SetSlack(sd)
	n.SetData(nd)

	data := &WorkspaceData{
		HTTPMonitors:  []*monitorv1.HTTPMonitor{m},
		Notifications: []*notificationv1.Notification{n},
	}
	gen := NewGenerator(data)
	content := string(gen.GenerateNotificationsFile().Bytes())

	mustContain(t, content, "openstatus_http_monitor.api.id")
	mustContain(t, content, `"mon-unknown"`)
	idx := strings.Index(content, "monitor_ids")
	if idx == -1 {
		t.Fatalf("monitor_ids not found in output:\n%s", content)
	}
	line := content[idx : strings.Index(content[idx:], "\n")+idx]
	if strings.Index(line, "openstatus_http_monitor.api.id") > strings.Index(line, `"mon-unknown"`) {
		t.Errorf("expected sorted order (mon-known first, mon-unknown second), got: %s", line)
	}
}

func TestGenerateNotificationsFile_UnknownProviderSkipped(t *testing.T) {
	n := &notificationv1.Notification{}
	n.SetId("n1")
	n.SetName("Mystery")

	data := &WorkspaceData{Notifications: []*notificationv1.Notification{n}}
	gen := NewGenerator(data)
	nContent := string(gen.GenerateNotificationsFile().Bytes())
	importsContent := string(gen.GenerateImportsFile().Bytes())

	if nContent != "" {
		t.Errorf("expected empty notifications file for skipped resource, got:\n%s", nContent)
	}
	mustNotContain(t, importsContent, "openstatus_notification.mystery")
	mustNotContain(t, importsContent, `id = "n1"`)
}

func TestGenerateMonitorsFile_HTTP_OpenTelemetry(t *testing.T) {
	ot := &monitorv1.OpenTelemetryConfig{}
	ot.SetEndpoint("https://otel.example.com/v1/metrics")
	h := &monitorv1.Headers{}
	h.SetKey("X-Api-Key")
	h.SetValue("secret")
	ot.SetHeaders([]*monitorv1.Headers{h})

	m := &monitorv1.HTTPMonitor{}
	m.SetId("1")
	m.SetName("API")
	m.SetUrl("https://api.example.com")
	m.SetPeriodicity(monitorv1.Periodicity_PERIODICITY_5M)
	m.SetActive(true)
	m.SetOpenTelemetry(ot)

	data := &WorkspaceData{HTTPMonitors: []*monitorv1.HTTPMonitor{m}}
	gen := NewGenerator(data)
	content := string(gen.GenerateMonitorsFile().Bytes())

	mustContain(t, content, "open_telemetry {")
	mustContain(t, content, `endpoint = "https://otel.example.com/v1/metrics"`)
	mustContain(t, content, `key   = "X-Api-Key"`)
	mustContain(t, content, `value = "secret"`)
}

func TestGenerateMonitorsFile_TCP_OpenTelemetry(t *testing.T) {
	ot := &monitorv1.OpenTelemetryConfig{}
	ot.SetEndpoint("https://otel.example.com/v1/metrics")

	m := &monitorv1.TCPMonitor{}
	m.SetId("1")
	m.SetName("TCP")
	m.SetUri("example.com:443")
	m.SetPeriodicity(monitorv1.Periodicity_PERIODICITY_5M)
	m.SetActive(true)
	m.SetOpenTelemetry(ot)

	data := &WorkspaceData{TCPMonitors: []*monitorv1.TCPMonitor{m}}
	gen := NewGenerator(data)
	content := string(gen.GenerateMonitorsFile().Bytes())

	mustContain(t, content, "open_telemetry {")
	mustContain(t, content, `endpoint = "https://otel.example.com/v1/metrics"`)
}

func TestGenerateMonitorsFile_DNS_OpenTelemetry(t *testing.T) {
	ot := &monitorv1.OpenTelemetryConfig{}
	ot.SetEndpoint("https://otel.example.com/v1/metrics")

	m := &monitorv1.DNSMonitor{}
	m.SetId("1")
	m.SetName("DNS")
	m.SetUri("example.com")
	m.SetPeriodicity(monitorv1.Periodicity_PERIODICITY_5M)
	m.SetActive(true)
	m.SetOpenTelemetry(ot)

	data := &WorkspaceData{DNSMonitors: []*monitorv1.DNSMonitor{m}}
	gen := NewGenerator(data)
	content := string(gen.GenerateMonitorsFile().Bytes())

	mustContain(t, content, "open_telemetry {")
	mustContain(t, content, `endpoint = "https://otel.example.com/v1/metrics"`)
}

func TestGenerateMonitorsFile_OpenTelemetry_SkippedWhenEmpty(t *testing.T) {
	m := &monitorv1.HTTPMonitor{}
	m.SetId("1")
	m.SetName("API")
	m.SetUrl("https://api.example.com")
	m.SetPeriodicity(monitorv1.Periodicity_PERIODICITY_5M)
	m.SetActive(true)
	m.SetOpenTelemetry(&monitorv1.OpenTelemetryConfig{})

	data := &WorkspaceData{HTTPMonitors: []*monitorv1.HTTPMonitor{m}}
	gen := NewGenerator(data)
	content := string(gen.GenerateMonitorsFile().Bytes())

	mustNotContain(t, content, "open_telemetry")
}

func TestGenerateMonitorsFile_RegionsSorted(t *testing.T) {
	m := &monitorv1.HTTPMonitor{}
	m.SetId("1")
	m.SetName("API")
	m.SetUrl("https://api.example.com")
	m.SetPeriodicity(monitorv1.Periodicity_PERIODICITY_5M)
	m.SetActive(true)
	m.SetRegions([]monitorv1.Region{
		monitorv1.Region_REGION_FLY_SYD,
		monitorv1.Region_REGION_FLY_AMS,
		monitorv1.Region_REGION_FLY_IAD,
	})

	data := &WorkspaceData{HTTPMonitors: []*monitorv1.HTTPMonitor{m}}
	gen := NewGenerator(data)
	content := string(gen.GenerateMonitorsFile().Bytes())

	mustContain(t, content, `["fly-ams", "fly-iad", "fly-syd"]`)
}

func TestGenerateStatusPagesFile_ComponentGroupDefaultOpen(t *testing.T) {
	page := &status_pagev1.StatusPage{}
	page.SetId("p1")
	page.SetTitle("Page")
	page.SetSlug("page")

	grpOpen := &status_pagev1.PageComponentGroup{}
	grpOpen.SetId("g1")
	grpOpen.SetPageId("p1")
	grpOpen.SetName("Infrastructure")
	grpOpen.SetDefaultOpen(true)

	grpClosed := &status_pagev1.PageComponentGroup{}
	grpClosed.SetId("g2")
	grpClosed.SetPageId("p1")
	grpClosed.SetName("Other")

	data := &WorkspaceData{
		StatusPages: []StatusPageData{{
			Page:   page,
			Groups: []*status_pagev1.PageComponentGroup{grpOpen, grpClosed},
		}},
	}
	gen := NewGenerator(data)
	content := string(gen.GenerateStatusPagesFile().Bytes())

	mustContain(t, content, `resource "openstatus_status_page_component_group" "infrastructure"`)
	mustContain(t, content, `default_open = true`)
	mustContain(t, content, `resource "openstatus_status_page_component_group" "other"`)

	otherStart := strings.Index(content, `"other"`)
	if otherStart == -1 {
		t.Fatalf("expected 'other' resource not found")
	}
	otherBlock := content[otherStart:]
	if strings.Contains(otherBlock[:strings.Index(otherBlock, "}")], "default_open") {
		t.Errorf("default_open should not be emitted for the 'other' group, got:\n%s", otherBlock)
	}
}

func TestGenerateStatusPagesFile_IPAccess(t *testing.T) {
	page := &status_pagev1.StatusPage{}
	page.SetId("p1")
	page.SetTitle("Internal")
	page.SetSlug("internal")
	page.SetAccessType(status_pagev1.PageAccessType_PAGE_ACCESS_TYPE_IP_RESTRICTED)
	page.SetAllowedIpRanges("10.0.0.0/8,192.168.0.0/16")

	data := &WorkspaceData{StatusPages: []StatusPageData{{Page: page}}}
	gen := NewGenerator(data)
	content := string(gen.GenerateStatusPagesFile().Bytes())

	mustContain(t, content, `access_type       = "ip"`)
	mustContain(t, content, `allowed_ip_ranges = "10.0.0.0/8,192.168.0.0/16"`)
	mustNotContain(t, content, "REPLACE_ME")
}

func TestGenerateStatusPagesFile_IPAccessEmptyFallback(t *testing.T) {
	page := &status_pagev1.StatusPage{}
	page.SetId("p1")
	page.SetTitle("Internal")
	page.SetSlug("internal")
	page.SetAccessType(status_pagev1.PageAccessType_PAGE_ACCESS_TYPE_IP_RESTRICTED)

	data := &WorkspaceData{StatusPages: []StatusPageData{{Page: page}}}
	gen := NewGenerator(data)
	content := string(gen.GenerateStatusPagesFile().Bytes())

	mustContain(t, content, `access_type = "ip"`)
	mustContain(t, content, `allowed_ip_ranges = "REPLACE_ME"`)
	mustContain(t, content, "# TODO:")
}

func TestGenerateStatusPagesFile_EmailDomainAccess(t *testing.T) {
	page := &status_pagev1.StatusPage{}
	page.SetId("p1")
	page.SetTitle("Internal")
	page.SetSlug("internal")
	page.SetAccessType(status_pagev1.PageAccessType_PAGE_ACCESS_TYPE_AUTHENTICATED)
	page.SetAuthEmailDomains([]string{"example.com", "acme.com"})

	data := &WorkspaceData{StatusPages: []StatusPageData{{Page: page}}}
	gen := NewGenerator(data)
	content := string(gen.GenerateStatusPagesFile().Bytes())

	mustContain(t, content, `access_type        = "email-domain"`)
	mustContain(t, content, `auth_email_domains = ["acme.com", "example.com"]`)
	mustNotContain(t, content, "REPLACE_ME")
}

func TestGenerateStatusPagesFile_EmailDomainEmptyFallback(t *testing.T) {
	page := &status_pagev1.StatusPage{}
	page.SetId("p1")
	page.SetTitle("Internal")
	page.SetSlug("internal")
	page.SetAccessType(status_pagev1.PageAccessType_PAGE_ACCESS_TYPE_AUTHENTICATED)

	data := &WorkspaceData{StatusPages: []StatusPageData{{Page: page}}}
	gen := NewGenerator(data)
	content := string(gen.GenerateStatusPagesFile().Bytes())

	mustContain(t, content, `access_type = "email-domain"`)
	mustContain(t, content, `auth_email_domains = ["REPLACE_ME"]`)
	mustContain(t, content, "# TODO:")
}

func TestGenerateStatusPagesFile_ThemeLocaleAllowIndex(t *testing.T) {
	page := &status_pagev1.StatusPage{}
	page.SetId("p1")
	page.SetTitle("Themed")
	page.SetSlug("themed")
	page.SetTheme(status_pagev1.PageTheme_PAGE_THEME_DARK)
	page.SetDefaultLocale(status_pagev1.Locale_LOCALE_FR)
	page.SetLocales([]status_pagev1.Locale{
		status_pagev1.Locale_LOCALE_FR,
		status_pagev1.Locale_LOCALE_EN,
	})
	page.SetAllowIndex(true)

	data := &WorkspaceData{StatusPages: []StatusPageData{{Page: page}}}
	gen := NewGenerator(data)
	content := string(gen.GenerateStatusPagesFile().Bytes())

	mustContain(t, content, `theme          = "dark"`)
	mustContain(t, content, `default_locale = "fr"`)
	mustContain(t, content, `locales        = ["en", "fr"]`)
	mustContain(t, content, `allow_index    = true`)
}

func TestGenerateStatusPagesFile_DefaultsOmitted(t *testing.T) {
	page := &status_pagev1.StatusPage{}
	page.SetId("p1")
	page.SetTitle("Plain")
	page.SetSlug("plain")
	page.SetTheme(status_pagev1.PageTheme_PAGE_THEME_SYSTEM)
	page.SetDefaultLocale(status_pagev1.Locale_LOCALE_EN)

	data := &WorkspaceData{StatusPages: []StatusPageData{{Page: page}}}
	gen := NewGenerator(data)
	content := string(gen.GenerateStatusPagesFile().Bytes())

	mustNotContain(t, content, "theme")
	mustNotContain(t, content, "default_locale")
	mustNotContain(t, content, "locales")
	mustNotContain(t, content, "allow_index")
}

func mustContain(t *testing.T, content, substr string) {
	t.Helper()
	if !strings.Contains(content, substr) {
		t.Errorf("expected output to contain %q, got:\n%s", substr, content)
	}
}

func mustNotContain(t *testing.T, content, substr string) {
	t.Helper()
	if strings.Contains(content, substr) {
		t.Errorf("expected output NOT to contain %q, got:\n%s", substr, content)
	}
}
