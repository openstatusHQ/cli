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
	mustContain(t, content, `version = "~> 0.1.0"`)
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
