package api

import (
	"context"
	"errors"
	"fmt"

	"buf.build/gen/go/openstatus/api/connectrpc/gosimple/openstatus/monitor/v1/monitorv1connect"
	"buf.build/gen/go/openstatus/api/connectrpc/gosimple/openstatus/notification/v1/notificationv1connect"
	"buf.build/gen/go/openstatus/api/connectrpc/gosimple/openstatus/private_location/v1/private_locationv1connect"
	"buf.build/gen/go/openstatus/api/connectrpc/gosimple/openstatus/status_page/v1/status_pagev1connect"
	monitorv1 "buf.build/gen/go/openstatus/api/protocolbuffers/go/openstatus/monitor/v1"
	notificationv1 "buf.build/gen/go/openstatus/api/protocolbuffers/go/openstatus/notification/v1"
	private_locationv1 "buf.build/gen/go/openstatus/api/protocolbuffers/go/openstatus/private_location/v1"
	status_pagev1 "buf.build/gen/go/openstatus/api/protocolbuffers/go/openstatus/status_page/v1"
)

// PageSize is the largest page the list endpoints accept (1-100); without an
// explicit limit they return only the first 50 items.
const PageSize int32 = 100

// ErrStopPaging can be returned by a FetchAll callback to end paging early
// without an error, e.g. once everything the caller needs has been found.
var ErrStopPaging = errors.New("stop paging")

// MaxPages bounds FetchAll so a server that ignores offset, and so keeps
// returning the same full page, fails loudly instead of looping forever.
const MaxPages = 1000

// ErrTooManyPages is returned when FetchAll gives up after MaxPages pages.
var ErrTooManyPages = fmt.Errorf("pagination did not finish after %d pages; the server may be ignoring offset", MaxPages)

// FetchAll walks a list endpoint page by page. fetch requests the page at
// offset and reports how many items it got. It stops on a short page. The
// server's total is deliberately not used as a stop condition: an
// under-reported total, or items added while paging, would otherwise
// silently drop the remaining items.
func FetchAll(ctx context.Context, fetch func(ctx context.Context, limit, offset int32) (n int, err error)) error {
	offset := int32(0)
	for range MaxPages {
		n, err := fetch(ctx, PageSize, offset)
		if errors.Is(err, ErrStopPaging) {
			return nil
		}
		if err != nil {
			return err
		}
		if n < int(PageSize) {
			return nil
		}
		offset += int32(n)
	}
	return ErrTooManyPages
}

// PageFunc fetches one page of a list endpoint and returns its items and the
// total the server holds.
type PageFunc[T any] func(ctx context.Context, limit, offset int32) ([]T, int32, error)

// ListAll collects every item of a list endpoint.
func ListAll[T any](ctx context.Context, fetch PageFunc[T]) ([]T, error) {
	var all []T
	err := FetchAll(ctx, func(ctx context.Context, limit, offset int32) (int, error) {
		items, _, err := fetch(ctx, limit, offset)
		if err != nil {
			return 0, err
		}
		all = append(all, items...)
		return len(items), nil
	})
	return all, err
}

// List backs the user-facing list commands: with a positive limit it fetches
// that single page, otherwise every page. It also returns the server total so
// callers can say when results were truncated.
func List[T any](ctx context.Context, limit int32, fetch PageFunc[T]) ([]T, int32, error) {
	if limit > 0 {
		return fetch(ctx, limit, 0)
	}
	items, err := ListAll(ctx, fetch)
	return items, int32(len(items)), err
}

// ListAllMonitors returns every monitor in the workspace, merging all pages
// into a single response.
func ListAllMonitors(ctx context.Context, client monitorv1connect.MonitorServiceClient) (*monitorv1.ListMonitorsResponse, error) {
	var (
		httpMonitors []*monitorv1.HTTPMonitor
		tcpMonitors  []*monitorv1.TCPMonitor
		dnsMonitors  []*monitorv1.DNSMonitor
		icmpMonitors []*monitorv1.ICMPMonitor
		grpcMonitors []*monitorv1.GRPCMonitor
		totalSize    int32
	)
	// The API applies limit and offset to all monitor kinds combined, so a
	// page's size is the sum over every kind.
	err := FetchAll(ctx, func(ctx context.Context, limit, offset int32) (int, error) {
		req := &monitorv1.ListMonitorsRequest{}
		req.SetLimit(limit)
		req.SetOffset(offset)
		resp, err := client.ListMonitors(ctx, req)
		if err != nil {
			return 0, err
		}
		httpMonitors = append(httpMonitors, resp.GetHttpMonitors()...)
		tcpMonitors = append(tcpMonitors, resp.GetTcpMonitors()...)
		dnsMonitors = append(dnsMonitors, resp.GetDnsMonitors()...)
		icmpMonitors = append(icmpMonitors, resp.GetIcmpMonitors()...)
		grpcMonitors = append(grpcMonitors, resp.GetGrpcMonitors()...)
		totalSize = resp.GetTotalSize()
		n := len(resp.GetHttpMonitors()) + len(resp.GetTcpMonitors()) + len(resp.GetDnsMonitors()) +
			len(resp.GetIcmpMonitors()) + len(resp.GetGrpcMonitors())
		return n, nil
	})
	if err != nil {
		return nil, err
	}

	merged := &monitorv1.ListMonitorsResponse{}
	merged.SetHttpMonitors(httpMonitors)
	merged.SetTcpMonitors(tcpMonitors)
	merged.SetDnsMonitors(dnsMonitors)
	merged.SetIcmpMonitors(icmpMonitors)
	merged.SetGrpcMonitors(grpcMonitors)
	merged.SetTotalSize(totalSize)
	return merged, nil
}

// StatusPagesPage fetches one page of status page summaries.
func StatusPagesPage(client status_pagev1connect.StatusPageServiceClient) PageFunc[*status_pagev1.StatusPageSummary] {
	return func(ctx context.Context, limit, offset int32) ([]*status_pagev1.StatusPageSummary, int32, error) {
		req := &status_pagev1.ListStatusPagesRequest{}
		req.SetLimit(limit)
		req.SetOffset(offset)
		resp, err := client.ListStatusPages(ctx, req)
		if err != nil {
			return nil, 0, err
		}
		return resp.GetStatusPages(), resp.GetTotalSize(), nil
	}
}

// ListAllStatusPages returns every status page summary in the workspace.
func ListAllStatusPages(ctx context.Context, client status_pagev1connect.StatusPageServiceClient) ([]*status_pagev1.StatusPageSummary, error) {
	return ListAll(ctx, StatusPagesPage(client))
}

// NotificationsPage fetches one page of notification summaries.
func NotificationsPage(client notificationv1connect.NotificationServiceClient) PageFunc[*notificationv1.NotificationSummary] {
	return func(ctx context.Context, limit, offset int32) ([]*notificationv1.NotificationSummary, int32, error) {
		req := &notificationv1.ListNotificationsRequest{}
		req.SetLimit(limit)
		req.SetOffset(offset)
		resp, err := client.ListNotifications(ctx, req)
		if err != nil {
			return nil, 0, err
		}
		return resp.GetNotifications(), resp.GetTotalSize(), nil
	}
}

// ListAllNotifications returns every notification summary in the workspace.
func ListAllNotifications(ctx context.Context, client notificationv1connect.NotificationServiceClient) ([]*notificationv1.NotificationSummary, error) {
	return ListAll(ctx, NotificationsPage(client))
}

// PrivateLocationsPage fetches one page of private location summaries.
func PrivateLocationsPage(client private_locationv1connect.PrivateLocationServiceClient) PageFunc[*private_locationv1.PrivateLocationSummary] {
	return func(ctx context.Context, limit, offset int32) ([]*private_locationv1.PrivateLocationSummary, int32, error) {
		req := &private_locationv1.ListPrivateLocationsRequest{}
		req.SetLimit(limit)
		req.SetOffset(offset)
		resp, err := client.ListPrivateLocations(ctx, req)
		if err != nil {
			return nil, 0, err
		}
		return resp.GetPrivateLocations(), resp.GetTotalSize(), nil
	}
}

// ListAllPrivateLocations returns every private location summary in the
// workspace.
func ListAllPrivateLocations(ctx context.Context, client private_locationv1connect.PrivateLocationServiceClient) ([]*private_locationv1.PrivateLocationSummary, error) {
	return ListAll(ctx, PrivateLocationsPage(client))
}
