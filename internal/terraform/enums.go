package terraform

import (
	monitorv1 "buf.build/gen/go/openstatus/api/protocolbuffers/go/openstatus/monitor/v1"
	notificationv1 "buf.build/gen/go/openstatus/api/protocolbuffers/go/openstatus/notification/v1"
	status_pagev1 "buf.build/gen/go/openstatus/api/protocolbuffers/go/openstatus/status_page/v1"
)

func periodicityToString(p monitorv1.Periodicity) string {
	switch p {
	case monitorv1.Periodicity_PERIODICITY_30S:
		return "30s"
	case monitorv1.Periodicity_PERIODICITY_1M:
		return "1m"
	case monitorv1.Periodicity_PERIODICITY_5M:
		return "5m"
	case monitorv1.Periodicity_PERIODICITY_10M:
		return "10m"
	case monitorv1.Periodicity_PERIODICITY_30M:
		return "30m"
	case monitorv1.Periodicity_PERIODICITY_1H:
		return "1h"
	default:
		return "10m"
	}
}

func httpMethodToString(m monitorv1.HTTPMethod) string {
	switch m {
	case monitorv1.HTTPMethod_HTTP_METHOD_GET:
		return "GET"
	case monitorv1.HTTPMethod_HTTP_METHOD_POST:
		return "POST"
	case monitorv1.HTTPMethod_HTTP_METHOD_PUT:
		return "PUT"
	case monitorv1.HTTPMethod_HTTP_METHOD_PATCH:
		return "PATCH"
	case monitorv1.HTTPMethod_HTTP_METHOD_DELETE:
		return "DELETE"
	case monitorv1.HTTPMethod_HTTP_METHOD_HEAD:
		return "HEAD"
	case monitorv1.HTTPMethod_HTTP_METHOD_OPTIONS:
		return "OPTIONS"
	case monitorv1.HTTPMethod_HTTP_METHOD_TRACE:
		return "TRACE"
	case monitorv1.HTTPMethod_HTTP_METHOD_CONNECT:
		return "CONNECT"
	default:
		return "GET"
	}
}

func numberComparatorToString(c monitorv1.NumberComparator) string {
	switch c {
	case monitorv1.NumberComparator_NUMBER_COMPARATOR_EQUAL:
		return "eq"
	case monitorv1.NumberComparator_NUMBER_COMPARATOR_NOT_EQUAL:
		return "neq"
	case monitorv1.NumberComparator_NUMBER_COMPARATOR_GREATER_THAN:
		return "gt"
	case monitorv1.NumberComparator_NUMBER_COMPARATOR_GREATER_THAN_OR_EQUAL:
		return "gte"
	case monitorv1.NumberComparator_NUMBER_COMPARATOR_LESS_THAN:
		return "lt"
	case monitorv1.NumberComparator_NUMBER_COMPARATOR_LESS_THAN_OR_EQUAL:
		return "lte"
	default:
		return "eq"
	}
}

func stringComparatorToString(c monitorv1.StringComparator) string {
	switch c {
	case monitorv1.StringComparator_STRING_COMPARATOR_EQUAL:
		return "eq"
	case monitorv1.StringComparator_STRING_COMPARATOR_NOT_EQUAL:
		return "neq"
	case monitorv1.StringComparator_STRING_COMPARATOR_CONTAINS:
		return "contains"
	case monitorv1.StringComparator_STRING_COMPARATOR_NOT_CONTAINS:
		return "not_contains"
	case monitorv1.StringComparator_STRING_COMPARATOR_EMPTY:
		return "empty"
	case monitorv1.StringComparator_STRING_COMPARATOR_NOT_EMPTY:
		return "not_empty"
	case monitorv1.StringComparator_STRING_COMPARATOR_GREATER_THAN:
		return "gt"
	case monitorv1.StringComparator_STRING_COMPARATOR_GREATER_THAN_OR_EQUAL:
		return "gte"
	case monitorv1.StringComparator_STRING_COMPARATOR_LESS_THAN:
		return "lt"
	case monitorv1.StringComparator_STRING_COMPARATOR_LESS_THAN_OR_EQUAL:
		return "lte"
	default:
		return "eq"
	}
}

func recordComparatorToString(c monitorv1.RecordComparator) string {
	switch c {
	case monitorv1.RecordComparator_RECORD_COMPARATOR_EQUAL:
		return "eq"
	case monitorv1.RecordComparator_RECORD_COMPARATOR_NOT_EQUAL:
		return "neq"
	case monitorv1.RecordComparator_RECORD_COMPARATOR_CONTAINS:
		return "contains"
	case monitorv1.RecordComparator_RECORD_COMPARATOR_NOT_CONTAINS:
		return "not_contains"
	default:
		return "eq"
	}
}

func notificationProviderToString(p notificationv1.NotificationProvider) string {
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
		return "us"
	}
}

func pageComponentTypeToString(t status_pagev1.PageComponentType) string {
	switch t {
	case status_pagev1.PageComponentType_PAGE_COMPONENT_TYPE_MONITOR:
		return "monitor"
	case status_pagev1.PageComponentType_PAGE_COMPONENT_TYPE_STATIC:
		return "static"
	default:
		return "static"
	}
}

func pageAccessTypeToString(t status_pagev1.PageAccessType) string {
	switch t {
	case status_pagev1.PageAccessType_PAGE_ACCESS_TYPE_PUBLIC:
		return "public"
	case status_pagev1.PageAccessType_PAGE_ACCESS_TYPE_PASSWORD_PROTECTED:
		return "password"
	case status_pagev1.PageAccessType_PAGE_ACCESS_TYPE_AUTHENTICATED:
		return "email-domain"
	default:
		return "public"
	}
}
