package terraform

import (
	"context"
	"fmt"
	"net/http"
	"os"

	"buf.build/gen/go/openstatus/api/connectrpc/gosimple/openstatus/monitor/v1/monitorv1connect"
	"buf.build/gen/go/openstatus/api/connectrpc/gosimple/openstatus/notification/v1/notificationv1connect"
	"buf.build/gen/go/openstatus/api/connectrpc/gosimple/openstatus/private_location/v1/private_locationv1connect"
	"buf.build/gen/go/openstatus/api/connectrpc/gosimple/openstatus/status_page/v1/status_pagev1connect"
	monitorv1 "buf.build/gen/go/openstatus/api/protocolbuffers/go/openstatus/monitor/v1"
	notificationv1 "buf.build/gen/go/openstatus/api/protocolbuffers/go/openstatus/notification/v1"
	private_locationv1 "buf.build/gen/go/openstatus/api/protocolbuffers/go/openstatus/private_location/v1"
	status_pagev1 "buf.build/gen/go/openstatus/api/protocolbuffers/go/openstatus/status_page/v1"
	"connectrpc.com/connect"

	"github.com/openstatusHQ/cli/internal/api"
)

const privateLocationPageSize = 100

type StatusPageData struct {
	Page       *status_pagev1.StatusPage
	Components []*status_pagev1.PageComponent
	Groups     []*status_pagev1.PageComponentGroup
}

type WorkspaceData struct {
	HTTPMonitors     []*monitorv1.HTTPMonitor
	TCPMonitors      []*monitorv1.TCPMonitor
	DNSMonitors      []*monitorv1.DNSMonitor
	Notifications    []*notificationv1.Notification
	StatusPages      []StatusPageData
	PrivateLocations []*private_locationv1.PrivateLocation
}

func FetchWorkspaceData(ctx context.Context, apiKey string) (*WorkspaceData, error) {
	return FetchWorkspaceDataWithHTTPClient(ctx, api.DefaultHTTPClient, apiKey)
}

func FetchWorkspaceDataWithHTTPClient(ctx context.Context, httpClient *http.Client, apiKey string) (*WorkspaceData, error) {
	interceptor := connect.WithInterceptors(api.NewAuthInterceptor(apiKey))
	protoJSON := connect.WithProtoJSON()

	data := &WorkspaceData{}

	// Monitors
	monitorClient := monitorv1connect.NewMonitorServiceClient(httpClient, api.ConnectBaseURL, interceptor, protoJSON)
	monitorResp, err := monitorClient.ListMonitors(ctx, &monitorv1.ListMonitorsRequest{})
	if err != nil {
		return nil, fmt.Errorf("failed to list monitors: %w", err)
	}
	data.HTTPMonitors = monitorResp.GetHttpMonitors()
	data.TCPMonitors = monitorResp.GetTcpMonitors()
	data.DNSMonitors = monitorResp.GetDnsMonitors()

	// Notifications
	notifClient := notificationv1connect.NewNotificationServiceClient(httpClient, api.ConnectBaseURL, interceptor, protoJSON)
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
	pageClient := status_pagev1connect.NewStatusPageServiceClient(httpClient, api.ConnectBaseURL, interceptor, protoJSON)
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

	// Private Locations
	plClient := private_locationv1connect.NewPrivateLocationServiceClient(httpClient, api.ConnectBaseURL, interceptor, protoJSON)
	locations, err := fetchPrivateLocations(ctx, plClient)
	switch {
	case err == nil:
		data.PrivateLocations = locations
	case isFeatureUnavailable(err):
		// Partial results are dropped on purpose: Terraform owns monitor_ids, so
		// an incomplete set would detach monitors on the next apply.
		fmt.Fprintf(os.Stderr, "warning: skipping private locations — %v\n", err)
	default:
		return nil, fmt.Errorf("failed to fetch private locations: %w", err)
	}

	return data, nil
}

// fetchPrivateLocations returns every private location with its monitor_ids.
// ListPrivateLocations only reports monitor_count, so each summary needs a Get.
// Errors are returned unwrapped so the caller can inspect the Connect code.
func fetchPrivateLocations(ctx context.Context, client private_locationv1connect.PrivateLocationServiceClient) ([]*private_locationv1.PrivateLocation, error) {
	var locations []*private_locationv1.PrivateLocation

	for offset := int32(0); ; {
		listReq := &private_locationv1.ListPrivateLocationsRequest{}
		listReq.SetLimit(privateLocationPageSize)
		listReq.SetOffset(offset)

		listResp, err := client.ListPrivateLocations(ctx, listReq)
		if err != nil {
			return nil, err
		}

		summaries := listResp.GetPrivateLocations()
		if len(summaries) == 0 {
			break
		}

		for _, summary := range summaries {
			getReq := &private_locationv1.GetPrivateLocationRequest{}
			getReq.SetId(summary.GetId())
			getResp, err := client.GetPrivateLocation(ctx, getReq)
			if err != nil {
				return nil, err
			}
			locations = append(locations, getResp.GetPrivateLocation())
		}

		offset += int32(len(summaries))
		if offset >= listResp.GetTotalSize() {
			break
		}
	}

	return locations, nil
}

// isFeatureUnavailable reports whether the workspace simply cannot use private
// locations, as opposed to a failure worth aborting the whole export for.
func isFeatureUnavailable(err error) bool {
	switch connect.CodeOf(err) {
	case connect.CodePermissionDenied, connect.CodeUnimplemented:
		return true
	default:
		return false
	}
}
