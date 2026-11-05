package api_test

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"buf.build/gen/go/openstatus/api/connectrpc/gosimple/openstatus/monitor/v1/monitorv1connect"
	monitorv1 "buf.build/gen/go/openstatus/api/protocolbuffers/go/openstatus/monitor/v1"

	"github.com/openstatusHQ/cli/internal/api"
)

// pagedMonitorClient serves total HTTP monitors, honouring limit and offset.
type pagedMonitorClient struct {
	monitorv1connect.MonitorServiceClient
	total   int
	offsets []int32
}

func (c *pagedMonitorClient) ListMonitors(_ context.Context, req *monitorv1.ListMonitorsRequest) (*monitorv1.ListMonitorsResponse, error) {
	c.offsets = append(c.offsets, req.GetOffset())
	var page []*monitorv1.HTTPMonitor
	for i := int(req.GetOffset()); i < c.total && len(page) < int(req.GetLimit()); i++ {
		m := &monitorv1.HTTPMonitor{}
		m.SetId(fmt.Sprint(i))
		page = append(page, m)
	}
	resp := &monitorv1.ListMonitorsResponse{}
	resp.SetHttpMonitors(page)
	resp.SetTotalSize(int32(c.total))
	return resp, nil
}

// heartbeatFirstClient serves heartbeat monitors on page one and an HTTP monitor
// on page two, since limit and offset apply to all monitor kinds combined.
type heartbeatFirstClient struct {
	monitorv1connect.MonitorServiceClient
}

func (c *heartbeatFirstClient) ListMonitors(_ context.Context, req *monitorv1.ListMonitorsRequest) (*monitorv1.ListMonitorsResponse, error) {
	resp := &monitorv1.ListMonitorsResponse{}
	if req.GetOffset() > 0 {
		resp.SetHttpMonitors([]*monitorv1.HTTPMonitor{{}})
		return resp, nil
	}
	page := make([]*monitorv1.HeartbeatMonitor, api.PageSize)
	for i := range page {
		page[i] = &monitorv1.HeartbeatMonitor{}
	}
	resp.SetHeartbeatMonitors(page)
	return resp, nil
}

func TestListAllMonitors_HeartbeatsCountTowardPageSize(t *testing.T) {
	t.Parallel()

	resp, err := api.ListAllMonitors(context.Background(), &heartbeatFirstClient{})
	if err != nil {
		t.Fatal(err)
	}
	if got := len(resp.GetHeartbeatMonitors()); got != int(api.PageSize) {
		t.Errorf("got %d heartbeat monitors, want %d", got, api.PageSize)
	}
	if got := len(resp.GetHttpMonitors()); got != 1 {
		t.Errorf("got %d HTTP monitors from the second page, want 1", got)
	}
}

func TestListAllMonitors(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		total       int
		wantOffsets []int32
	}{
		{total: 0, wantOffsets: []int32{0}},
		{total: 30, wantOffsets: []int32{0}},
		{total: 100, wantOffsets: []int32{0, 100}},
		{total: 250, wantOffsets: []int32{0, 100, 200}},
	} {
		client := &pagedMonitorClient{total: tc.total}
		resp, err := api.ListAllMonitors(context.Background(), client)
		if err != nil {
			t.Fatalf("total=%d: %v", tc.total, err)
		}
		if got := len(resp.GetHttpMonitors()); got != tc.total {
			t.Errorf("total=%d: got %d monitors", tc.total, got)
		}
		if fmt.Sprint(client.offsets) != fmt.Sprint(tc.wantOffsets) {
			t.Errorf("total=%d: requested offsets %v, want %v", tc.total, client.offsets, tc.wantOffsets)
		}
	}
}

func TestFetchAll(t *testing.T) {
	t.Parallel()

	// run serves one page per call with the given sizes and returns the
	// requested offsets.
	run := func(sizes []int, stopAfter int) []int32 {
		var offsets []int32
		calls := 0
		err := api.FetchAll(context.Background(), func(_ context.Context, limit, offset int32) (int, error) {
			offsets = append(offsets, offset)
			calls++
			if stopAfter > 0 && calls == stopAfter {
				return 0, api.ErrStopPaging
			}
			if calls > len(sizes) {
				t.Fatalf("unexpected extra request at offset %d", offset)
			}
			return sizes[calls-1], nil
		})
		if err != nil {
			t.Fatalf("FetchAll: %v", err)
		}
		return offsets
	}

	full := int(api.PageSize)
	for name, tc := range map[string]struct {
		got  []int32
		want []int32
	}{
		"pages until a short page":          {run([]int{full, full, 3}, 0), []int32{0, 100, 200}},
		"exact multiple ends on empty page": {run([]int{full, 0}, 0), []int32{0, 100}},
		"single short page":                 {run([]int{3}, 0), []int32{0}},
		"stop sentinel ends early":          {run([]int{full, full, full}, 2), []int32{0, 100}},
	} {
		if fmt.Sprint(tc.got) != fmt.Sprint(tc.want) {
			t.Errorf("%s: offsets %v, want %v", name, tc.got, tc.want)
		}
	}
}

func TestFetchAllIgnoredOffset(t *testing.T) {
	t.Parallel()

	calls := 0
	err := api.FetchAll(context.Background(), func(context.Context, int32, int32) (int, error) {
		calls++
		return int(api.PageSize), nil
	})
	if !errors.Is(err, api.ErrTooManyPages) {
		t.Fatalf("err = %v, want ErrTooManyPages", err)
	}
	if calls != api.MaxPages {
		t.Errorf("calls = %d, want %d", calls, api.MaxPages)
	}
}

// underCountingClient serves total HTTP monitors but reports a smaller
// total_size, as a server with a stale or partial count would.
type underCountingClient struct{ pagedMonitorClient }

func (c *underCountingClient) ListMonitors(ctx context.Context, req *monitorv1.ListMonitorsRequest) (*monitorv1.ListMonitorsResponse, error) {
	resp, err := c.pagedMonitorClient.ListMonitors(ctx, req)
	resp.SetTotalSize(100)
	return resp, err
}

func TestListAllMonitorsIgnoresUnderReportedTotal(t *testing.T) {
	t.Parallel()

	client := &underCountingClient{pagedMonitorClient{total: 150}}
	resp, err := api.ListAllMonitors(context.Background(), client)
	if err != nil {
		t.Fatal(err)
	}
	if got := len(resp.GetHttpMonitors()); got != 150 {
		t.Errorf("got %d monitors, want 150", got)
	}
}
