package terraform

import (
	"fmt"

	monitorv1 "buf.build/gen/go/openstatus/api/protocolbuffers/go/openstatus/monitor/v1"
	notificationv1 "buf.build/gen/go/openstatus/api/protocolbuffers/go/openstatus/notification/v1"
	status_pagev1 "buf.build/gen/go/openstatus/api/protocolbuffers/go/openstatus/status_page/v1"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/zclconf/go-cty/cty"
)

type resourceRef struct {
	ResourceType string
	Name         string
}

type Generator struct {
	data        *WorkspaceData
	registry    *NameRegistry
	monitorRefs map[string]resourceRef
	pageRefs    map[string]resourceRef
	groupRefs   map[string]resourceRef

	httpMonitorNames  map[string]string
	tcpMonitorNames   map[string]string
	dnsMonitorNames   map[string]string
	notifNames        map[string]string
	pageNames         map[string]string
	componentNames    map[string]string
	groupNames        map[string]string
}

func NewGenerator(data *WorkspaceData) *Generator {
	g := &Generator{
		data:             data,
		registry:         NewNameRegistry(),
		monitorRefs:      make(map[string]resourceRef),
		pageRefs:         make(map[string]resourceRef),
		groupRefs:        make(map[string]resourceRef),
		httpMonitorNames: make(map[string]string),
		tcpMonitorNames:  make(map[string]string),
		dnsMonitorNames:  make(map[string]string),
		notifNames:       make(map[string]string),
		pageNames:        make(map[string]string),
		componentNames:   make(map[string]string),
		groupNames:       make(map[string]string),
	}

	for _, m := range data.HTTPMonitors {
		name := g.registry.Name("openstatus_http_monitor", m.GetName())
		g.httpMonitorNames[m.GetId()] = name
		g.monitorRefs[m.GetId()] = resourceRef{"openstatus_http_monitor", name}
	}
	for _, m := range data.TCPMonitors {
		name := g.registry.Name("openstatus_tcp_monitor", m.GetName())
		g.tcpMonitorNames[m.GetId()] = name
		g.monitorRefs[m.GetId()] = resourceRef{"openstatus_tcp_monitor", name}
	}
	for _, m := range data.DNSMonitors {
		name := g.registry.Name("openstatus_dns_monitor", m.GetName())
		g.dnsMonitorNames[m.GetId()] = name
		g.monitorRefs[m.GetId()] = resourceRef{"openstatus_dns_monitor", name}
	}
	for _, n := range data.Notifications {
		name := g.registry.Name("openstatus_notification", n.GetName())
		g.notifNames[n.GetId()] = name
	}
	for _, sp := range data.StatusPages {
		page := sp.Page
		name := g.registry.Name("openstatus_status_page", page.GetTitle())
		g.pageNames[page.GetId()] = name
		g.pageRefs[page.GetId()] = resourceRef{"openstatus_status_page", name}

		for _, grp := range sp.Groups {
			gName := g.registry.Name("openstatus_status_page_component_group", grp.GetName())
			g.groupNames[grp.GetId()] = gName
			g.groupRefs[grp.GetId()] = resourceRef{"openstatus_status_page_component_group", gName}
		}
		for _, comp := range sp.Components {
			cName := g.registry.Name("openstatus_status_page_component", comp.GetName())
			g.componentNames[comp.GetId()] = cName
		}
	}

	return g
}

func GenerateProviderFile() []byte {
	return []byte(`terraform {
  required_providers {
    openstatus = {
      source  = "openstatusHQ/openstatus"
      version = "~> 0.1.0"
    }
  }
}

# Set OPENSTATUS_API_TOKEN environment variable or configure api_token below
provider "openstatus" {}
`)
}

func (g *Generator) GenerateMonitorsFile() *hclwrite.File {
	f := hclwrite.NewEmptyFile()
	body := f.Body()

	for _, m := range g.data.HTTPMonitors {
		name := g.httpMonitorNames[m.GetId()]
		block := body.AppendNewBlock("resource", []string{"openstatus_http_monitor", name})
		b := block.Body()

		b.SetAttributeValue("name", cty.StringVal(m.GetName()))
		b.SetAttributeValue("url", cty.StringVal(m.GetUrl()))
		b.SetAttributeValue("periodicity", cty.StringVal(periodicityToString(m.GetPeriodicity())))

		if method := httpMethodToString(m.GetMethod()); method != "GET" {
			b.SetAttributeValue("method", cty.StringVal(method))
		}
		if m.GetBody() != "" {
			b.SetAttributeValue("body", cty.StringVal(m.GetBody()))
		}
		if m.GetTimeout() != 45000 {
			b.SetAttributeValue("timeout", cty.NumberIntVal(m.GetTimeout()))
		}
		if m.GetDegradedAt() != 0 {
			b.SetAttributeValue("degraded_at", cty.NumberIntVal(m.GetDegradedAt()))
		}
		if m.GetRetry() != 3 {
			b.SetAttributeValue("retry", cty.NumberIntVal(m.GetRetry()))
		}
		if !m.GetFollowRedirects() {
			b.SetAttributeValue("follow_redirects", cty.BoolVal(false))
		}
		b.SetAttributeValue("active", cty.BoolVal(m.GetActive()))
		b.SetAttributeValue("public", cty.BoolVal(m.GetPublic()))
		if m.GetDescription() != "" {
			b.SetAttributeValue("description", cty.StringVal(m.GetDescription()))
		}

		writeRegions(b, m.GetRegions())
		writeHeaders(b, m.GetHeaders())
		writeStatusCodeAssertions(b, m.GetStatusCodeAssertions())
		writeBodyAssertions(b, m.GetBodyAssertions())
		writeHeaderAssertions(b, m.GetHeaderAssertions())

		body.AppendNewline()
	}

	for _, m := range g.data.TCPMonitors {
		name := g.tcpMonitorNames[m.GetId()]
		block := body.AppendNewBlock("resource", []string{"openstatus_tcp_monitor", name})
		b := block.Body()

		b.SetAttributeValue("name", cty.StringVal(m.GetName()))
		b.SetAttributeValue("uri", cty.StringVal(m.GetUri()))
		b.SetAttributeValue("periodicity", cty.StringVal(periodicityToString(m.GetPeriodicity())))

		if m.GetTimeout() != 45000 {
			b.SetAttributeValue("timeout", cty.NumberIntVal(m.GetTimeout()))
		}
		if m.GetDegradedAt() != 0 {
			b.SetAttributeValue("degraded_at", cty.NumberIntVal(m.GetDegradedAt()))
		}
		if m.GetRetry() != 3 {
			b.SetAttributeValue("retry", cty.NumberIntVal(m.GetRetry()))
		}
		b.SetAttributeValue("active", cty.BoolVal(m.GetActive()))
		b.SetAttributeValue("public", cty.BoolVal(m.GetPublic()))
		if m.GetDescription() != "" {
			b.SetAttributeValue("description", cty.StringVal(m.GetDescription()))
		}

		writeRegions(b, m.GetRegions())

		body.AppendNewline()
	}

	for _, m := range g.data.DNSMonitors {
		name := g.dnsMonitorNames[m.GetId()]
		block := body.AppendNewBlock("resource", []string{"openstatus_dns_monitor", name})
		b := block.Body()

		b.SetAttributeValue("name", cty.StringVal(m.GetName()))
		b.SetAttributeValue("uri", cty.StringVal(m.GetUri()))
		b.SetAttributeValue("periodicity", cty.StringVal(periodicityToString(m.GetPeriodicity())))

		if m.GetTimeout() != 45000 {
			b.SetAttributeValue("timeout", cty.NumberIntVal(m.GetTimeout()))
		}
		if m.GetDegradedAt() != 0 {
			b.SetAttributeValue("degraded_at", cty.NumberIntVal(m.GetDegradedAt()))
		}
		if m.GetRetry() != 3 {
			b.SetAttributeValue("retry", cty.NumberIntVal(m.GetRetry()))
		}
		b.SetAttributeValue("active", cty.BoolVal(m.GetActive()))
		b.SetAttributeValue("public", cty.BoolVal(m.GetPublic()))
		if m.GetDescription() != "" {
			b.SetAttributeValue("description", cty.StringVal(m.GetDescription()))
		}

		writeRegions(b, m.GetRegions())

		for _, a := range m.GetRecordAssertions() {
			ab := b.AppendNewBlock("record_assertions", nil).Body()
			ab.SetAttributeValue("record", cty.StringVal(a.GetRecord()))
			ab.SetAttributeValue("target", cty.StringVal(a.GetTarget()))
			ab.SetAttributeValue("comparator", cty.StringVal(recordComparatorToString(a.GetComparator())))
		}

		body.AppendNewline()
	}

	return f
}

func (g *Generator) GenerateNotificationsFile() *hclwrite.File {
	f := hclwrite.NewEmptyFile()
	body := f.Body()

	for _, n := range g.data.Notifications {
		name := g.notifNames[n.GetId()]
		block := body.AppendNewBlock("resource", []string{"openstatus_notification", name})
		b := block.Body()

		b.SetAttributeValue("name", cty.StringVal(n.GetName()))
		b.SetAttributeValue("provider_type", cty.StringVal(notificationProviderToString(n.GetProvider())))

		if ids := n.GetMonitorIds(); len(ids) > 0 {
			vals := make([]cty.Value, len(ids))
			for i, id := range ids {
				vals[i] = cty.StringVal(id)
			}
			b.SetAttributeValue("monitor_ids", cty.SetVal(vals))
		}

		g.writeNotificationProvider(b, n)

		body.AppendNewline()
	}

	return f
}

func (g *Generator) writeNotificationProvider(b *hclwrite.Body, n *notificationv1.Notification) {
	data := n.GetData()
	if data == nil {
		return
	}

	switch d := data.Data.(type) {
	case *notificationv1.NotificationData_Discord:
		pb := b.AppendNewBlock("discord", nil).Body()
		pb.SetAttributeValue("webhook_url", cty.StringVal(d.Discord.GetWebhookUrl()))
	case *notificationv1.NotificationData_Email:
		pb := b.AppendNewBlock("email", nil).Body()
		pb.SetAttributeValue("email", cty.StringVal(d.Email.GetEmail()))
	case *notificationv1.NotificationData_Slack:
		pb := b.AppendNewBlock("slack", nil).Body()
		pb.SetAttributeValue("webhook_url", cty.StringVal(d.Slack.GetWebhookUrl()))
	case *notificationv1.NotificationData_Pagerduty:
		pb := b.AppendNewBlock("pagerduty", nil).Body()
		appendTODOComment(pb)
		pb.SetAttributeValue("integration_key", cty.StringVal("REPLACE_ME"))
	case *notificationv1.NotificationData_Opsgenie:
		pb := b.AppendNewBlock("opsgenie", nil).Body()
		appendTODOComment(pb)
		pb.SetAttributeValue("api_key", cty.StringVal("REPLACE_ME"))
		pb.SetAttributeValue("region", cty.StringVal(opsgenieRegionToString(d.Opsgenie.GetRegion())))
	case *notificationv1.NotificationData_Webhook:
		pb := b.AppendNewBlock("webhook", nil).Body()
		pb.SetAttributeValue("endpoint", cty.StringVal(d.Webhook.GetEndpoint()))
		for _, h := range d.Webhook.GetHeaders() {
			hb := pb.AppendNewBlock("headers", nil).Body()
			hb.SetAttributeValue("key", cty.StringVal(h.GetKey()))
			hb.SetAttributeValue("value", cty.StringVal(h.GetValue()))
		}
	case *notificationv1.NotificationData_Telegram:
		pb := b.AppendNewBlock("telegram", nil).Body()
		pb.SetAttributeValue("chat_id", cty.StringVal(d.Telegram.GetChatId()))
	case *notificationv1.NotificationData_Sms:
		pb := b.AppendNewBlock("sms", nil).Body()
		pb.SetAttributeValue("phone_number", cty.StringVal(d.Sms.GetPhoneNumber()))
	case *notificationv1.NotificationData_Whatsapp:
		pb := b.AppendNewBlock("whatsapp", nil).Body()
		pb.SetAttributeValue("phone_number", cty.StringVal(d.Whatsapp.GetPhoneNumber()))
	case *notificationv1.NotificationData_GoogleChat:
		pb := b.AppendNewBlock("google_chat", nil).Body()
		pb.SetAttributeValue("webhook_url", cty.StringVal(d.GoogleChat.GetWebhookUrl()))
	case *notificationv1.NotificationData_GrafanaOncall:
		pb := b.AppendNewBlock("grafana_oncall", nil).Body()
		pb.SetAttributeValue("webhook_url", cty.StringVal(d.GrafanaOncall.GetWebhookUrl()))
	case *notificationv1.NotificationData_Ntfy:
		pb := b.AppendNewBlock("ntfy", nil).Body()
		pb.SetAttributeValue("topic", cty.StringVal(d.Ntfy.GetTopic()))
		if d.Ntfy.GetServerUrl() != "" {
			pb.SetAttributeValue("server_url", cty.StringVal(d.Ntfy.GetServerUrl()))
		}
		if d.Ntfy.HasToken() {
			appendTODOComment(pb)
			pb.SetAttributeValue("token", cty.StringVal("REPLACE_ME"))
		}
	}
}

func (g *Generator) GenerateStatusPagesFile() *hclwrite.File {
	f := hclwrite.NewEmptyFile()
	body := f.Body()

	for _, sp := range g.data.StatusPages {
		page := sp.Page
		pageName := g.pageNames[page.GetId()]
		block := body.AppendNewBlock("resource", []string{"openstatus_status_page", pageName})
		b := block.Body()

		b.SetAttributeValue("title", cty.StringVal(page.GetTitle()))
		b.SetAttributeValue("slug", cty.StringVal(page.GetSlug()))
		if page.GetDescription() != "" {
			b.SetAttributeValue("description", cty.StringVal(page.GetDescription()))
		}
		if page.GetHomepageUrl() != "" {
			b.SetAttributeValue("homepage_url", cty.StringVal(page.GetHomepageUrl()))
		}
		if page.GetContactUrl() != "" {
			b.SetAttributeValue("contact_url", cty.StringVal(page.GetContactUrl()))
		}
		if page.GetIcon() != "" {
			b.SetAttributeValue("icon", cty.StringVal(page.GetIcon()))
		}
		if page.GetCustomDomain() != "" {
			b.SetAttributeValue("custom_domain", cty.StringVal(page.GetCustomDomain()))
		}

		accessType := pageAccessTypeToString(page.GetAccessType())
		if accessType != "public" {
			b.SetAttributeValue("access_type", cty.StringVal(accessType))
			if accessType == "password" {
				appendTODOComment(b)
				b.SetAttributeValue("password", cty.StringVal("REPLACE_ME"))
			}
		}

		body.AppendNewline()

		// Component groups first
		for _, grp := range sp.Groups {
			gName := g.groupNames[grp.GetId()]
			gb := body.AppendNewBlock("resource", []string{"openstatus_status_page_component_group", gName}).Body()
			setTraversalOrString(gb, "page_id", g.pageRefs, page.GetId())
			gb.SetAttributeValue("name", cty.StringVal(grp.GetName()))
			body.AppendNewline()
		}

		// Components
		for _, comp := range sp.Components {
			cName := g.componentNames[comp.GetId()]
			cb := body.AppendNewBlock("resource", []string{"openstatus_status_page_component", cName}).Body()
			setTraversalOrString(cb, "page_id", g.pageRefs, page.GetId())
			cb.SetAttributeValue("type", cty.StringVal(pageComponentTypeToString(comp.GetType())))

			if comp.GetType() == status_pagev1.PageComponentType_PAGE_COMPONENT_TYPE_MONITOR && comp.GetMonitorId() != "" {
				setTraversalOrString(cb, "monitor_id", g.monitorRefs, comp.GetMonitorId())
			}
			if comp.GetName() != "" {
				cb.SetAttributeValue("name", cty.StringVal(comp.GetName()))
			}
			if comp.GetDescription() != "" {
				cb.SetAttributeValue("description", cty.StringVal(comp.GetDescription()))
			}
			cb.SetAttributeValue("order", cty.NumberIntVal(int64(comp.GetOrder())))

			if comp.GetGroupId() != "" {
				setTraversalOrString(cb, "group_id", g.groupRefs, comp.GetGroupId())
				cb.SetAttributeValue("group_order", cty.NumberIntVal(int64(comp.GetGroupOrder())))
			}

			body.AppendNewline()
		}
	}

	return f
}

func (g *Generator) GenerateImportsFile() *hclwrite.File {
	f := hclwrite.NewEmptyFile()
	body := f.Body()

	for _, m := range g.data.HTTPMonitors {
		writeImportBlock(body, "openstatus_http_monitor", g.httpMonitorNames[m.GetId()], m.GetId())
	}
	for _, m := range g.data.TCPMonitors {
		writeImportBlock(body, "openstatus_tcp_monitor", g.tcpMonitorNames[m.GetId()], m.GetId())
	}
	for _, m := range g.data.DNSMonitors {
		writeImportBlock(body, "openstatus_dns_monitor", g.dnsMonitorNames[m.GetId()], m.GetId())
	}
	for _, n := range g.data.Notifications {
		writeImportBlock(body, "openstatus_notification", g.notifNames[n.GetId()], n.GetId())
	}
	for _, sp := range g.data.StatusPages {
		page := sp.Page
		writeImportBlock(body, "openstatus_status_page", g.pageNames[page.GetId()], page.GetId())
		for _, grp := range sp.Groups {
			writeImportBlock(body, "openstatus_status_page_component_group", g.groupNames[grp.GetId()], fmt.Sprintf("%s/%s", page.GetId(), grp.GetId()))
		}
		for _, comp := range sp.Components {
			writeImportBlock(body, "openstatus_status_page_component", g.componentNames[comp.GetId()], fmt.Sprintf("%s/%s", page.GetId(), comp.GetId()))
		}
	}

	return f
}

func (g *Generator) TotalResourceCount() int {
	count := len(g.data.HTTPMonitors) + len(g.data.TCPMonitors) + len(g.data.DNSMonitors) + len(g.data.Notifications)
	for _, sp := range g.data.StatusPages {
		count += 1 + len(sp.Components) + len(sp.Groups)
	}
	return count
}

func (g *Generator) HasMonitors() bool {
	return len(g.data.HTTPMonitors) > 0 || len(g.data.TCPMonitors) > 0 || len(g.data.DNSMonitors) > 0
}

func (g *Generator) HasNotifications() bool {
	return len(g.data.Notifications) > 0
}

func (g *Generator) HasStatusPages() bool {
	return len(g.data.StatusPages) > 0
}

// helpers

func writeRegions(b *hclwrite.Body, regions []monitorv1.Region) {
	if len(regions) == 0 {
		return
	}
	vals := make([]cty.Value, len(regions))
	for i, r := range regions {
		vals[i] = cty.StringVal(regionToTerraform(r))
	}
	b.SetAttributeValue("regions", cty.ListVal(vals))
}

func writeHeaders(b *hclwrite.Body, headers []*monitorv1.Headers) {
	for _, h := range headers {
		if h.GetKey() == "" {
			continue
		}
		hb := b.AppendNewBlock("headers", nil).Body()
		hb.SetAttributeValue("key", cty.StringVal(h.GetKey()))
		hb.SetAttributeValue("value", cty.StringVal(h.GetValue()))
	}
}

func writeStatusCodeAssertions(b *hclwrite.Body, assertions []*monitorv1.StatusCodeAssertion) {
	for _, a := range assertions {
		ab := b.AppendNewBlock("status_code_assertions", nil).Body()
		ab.SetAttributeValue("target", cty.NumberIntVal(int64(a.GetTarget())))
		ab.SetAttributeValue("comparator", cty.StringVal(numberComparatorToString(a.GetComparator())))
	}
}

func writeBodyAssertions(b *hclwrite.Body, assertions []*monitorv1.BodyAssertion) {
	for _, a := range assertions {
		ab := b.AppendNewBlock("body_assertions", nil).Body()
		ab.SetAttributeValue("target", cty.StringVal(a.GetTarget()))
		ab.SetAttributeValue("comparator", cty.StringVal(stringComparatorToString(a.GetComparator())))
	}
}

func writeHeaderAssertions(b *hclwrite.Body, assertions []*monitorv1.HeaderAssertion) {
	for _, a := range assertions {
		ab := b.AppendNewBlock("header_assertions", nil).Body()
		ab.SetAttributeValue("key", cty.StringVal(a.GetKey()))
		ab.SetAttributeValue("target", cty.StringVal(a.GetTarget()))
		ab.SetAttributeValue("comparator", cty.StringVal(stringComparatorToString(a.GetComparator())))
	}
}

func writeImportBlock(body *hclwrite.Body, resourceType, name, id string) {
	block := body.AppendNewBlock("import", nil)
	b := block.Body()
	b.SetAttributeRaw("to", traversalTokens(resourceType, name))
	b.SetAttributeValue("id", cty.StringVal(id))
	body.AppendNewline()
}

func setTraversalOrString(b *hclwrite.Body, attr string, refs map[string]resourceRef, id string) {
	if ref, ok := refs[id]; ok {
		b.SetAttributeRaw(attr, traversalTokens(ref.ResourceType, ref.Name, "id"))
	} else {
		b.SetAttributeValue(attr, cty.StringVal(id))
	}
}

func traversalTokens(parts ...string) hclwrite.Tokens {
	tokens := hclwrite.Tokens{}
	for i, part := range parts {
		if i > 0 {
			tokens = append(tokens, &hclwrite.Token{
				Type:  hclsyntax.TokenDot,
				Bytes: []byte("."),
			})
		}
		tokens = append(tokens, &hclwrite.Token{
			Type:  hclsyntax.TokenIdent,
			Bytes: []byte(part),
		})
	}
	tokens = append(tokens, &hclwrite.Token{
		Type:  hclsyntax.TokenNewline,
		Bytes: []byte("\n"),
	})
	return tokens
}

func appendTODOComment(b *hclwrite.Body) {
	b.AppendUnstructuredTokens(hclwrite.Tokens{
		{Type: hclsyntax.TokenComment, Bytes: []byte("# TODO: set the actual value — not available from the API\n")},
	})
}
