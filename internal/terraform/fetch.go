package terraform

import (
	"context"
	"fmt"

	monitorv1 "buf.build/gen/go/openstatus/api/protocolbuffers/go/openstatus/monitor/v1"
	"buf.build/gen/go/openstatus/api/connectrpc/gosimple/openstatus/monitor/v1/monitorv1connect"
	notificationv1 "buf.build/gen/go/openstatus/api/protocolbuffers/go/openstatus/notification/v1"
	"buf.build/gen/go/openstatus/api/connectrpc/gosimple/openstatus/notification/v1/notificationv1connect"
	status_pagev1 "buf.build/gen/go/openstatus/api/protocolbuffers/go/openstatus/status_page/v1"
	"buf.build/gen/go/openstatus/api/connectrpc/gosimple/openstatus/status_page/v1/status_pagev1connect"
	"connectrpc.com/connect"
	"github.com/openstatusHQ/cli/internal/api"
)

type StatusPageData struct {
	Page       *status_pagev1.StatusPage
	Components []*status_pagev1.PageComponent
	Groups     []*status_pagev1.PageComponentGroup
}

type WorkspaceData struct {
	HTTPMonitors  []*monitorv1.HTTPMonitor
	TCPMonitors   []*monitorv1.TCPMonitor
	DNSMonitors   []*monitorv1.DNSMonitor
	Notifications []*notificationv1.Notification
	StatusPages   []StatusPageData
}

func FetchWorkspaceData(ctx context.Context, apiKey string) (*WorkspaceData, error) {
	interceptor := connect.WithInterceptors(api.NewAuthInterceptor(apiKey))
	protoJSON := connect.WithProtoJSON()

	data := &WorkspaceData{}

	// Monitors
	monitorClient := monitorv1connect.NewMonitorServiceClient(api.DefaultHTTPClient, api.ConnectBaseURL, interceptor, protoJSON)
	monitorResp, err := monitorClient.ListMonitors(ctx, &monitorv1.ListMonitorsRequest{})
	if err != nil {
		return nil, fmt.Errorf("failed to list monitors: %w", err)
	}
	data.HTTPMonitors = monitorResp.GetHttpMonitors()
	data.TCPMonitors = monitorResp.GetTcpMonitors()
	data.DNSMonitors = monitorResp.GetDnsMonitors()

	// Notifications
	notifClient := notificationv1connect.NewNotificationServiceClient(api.DefaultHTTPClient, api.ConnectBaseURL, interceptor, protoJSON)
	notifResp, err := notifClient.ListNotifications(ctx, &notificationv1.ListNotificationsRequest{})
	if err != nil {
		return nil, fmt.Errorf("failed to list notifications: %w", err)
	}
	for _, summary := range notifResp.GetNotifications() {
		req := &notificationv1.GetNotificationRequest{}
		req.SetId(summary.GetId())
		resp, err := notifClient.GetNotification(ctx, req)
		if err != nil {
			return nil, fmt.Errorf("failed to get notification %s: %w", summary.GetId(), err)
		}
		data.Notifications = append(data.Notifications, resp.GetNotification())
	}

	// Status Pages
	pageClient := status_pagev1connect.NewStatusPageServiceClient(api.DefaultHTTPClient, api.ConnectBaseURL, interceptor, protoJSON)
	pageResp, err := pageClient.ListStatusPages(ctx, &status_pagev1.ListStatusPagesRequest{})
	if err != nil {
		return nil, fmt.Errorf("failed to list status pages: %w", err)
	}
	for _, summary := range pageResp.GetStatusPages() {
		req := &status_pagev1.GetStatusPageContentRequest{}
		req.SetId(summary.GetId())
		resp, err := pageClient.GetStatusPageContent(ctx, req)
		if err != nil {
			return nil, fmt.Errorf("failed to get status page %s: %w", summary.GetId(), err)
		}
		data.StatusPages = append(data.StatusPages, StatusPageData{
			Page:       resp.GetStatusPage(),
			Components: resp.GetComponents(),
			Groups:     resp.GetGroups(),
		})
	}

	return data, nil
}
