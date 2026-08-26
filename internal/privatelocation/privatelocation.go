package privatelocation

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"sort"
	"strings"

	"buf.build/gen/go/openstatus/api/connectrpc/gosimple/openstatus/monitor/v1/monitorv1connect"
	"buf.build/gen/go/openstatus/api/connectrpc/gosimple/openstatus/private_location/v1/private_locationv1connect"
	monitorv1 "buf.build/gen/go/openstatus/api/protocolbuffers/go/openstatus/monitor/v1"
	private_locationv1 "buf.build/gen/go/openstatus/api/protocolbuffers/go/openstatus/private_location/v1"
	"connectrpc.com/connect"
	"github.com/fatih/color"
	"github.com/olekukonko/tablewriter"
	"github.com/olekukonko/tablewriter/renderer"
	"github.com/olekukonko/tablewriter/tw"
	"github.com/urfave/cli/v3"

	"github.com/openstatusHQ/cli/internal/api"
	output "github.com/openstatusHQ/cli/internal/cli"
)

// Ref is the resolved identity of a private location, embedded in the JSON output of the
// monitors commands.
type Ref struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Status string `json:"status"`
}

func NewPrivateLocationClient(apiKey string) private_locationv1connect.PrivateLocationServiceClient {
	return private_locationv1connect.NewPrivateLocationServiceClient(
		api.DefaultHTTPClient,
		api.ConnectBaseURL,
		connect.WithInterceptors(api.NewAuthInterceptor(apiKey)),
		connect.WithProtoJSON(),
	)
}

func NewPrivateLocationClientWithHTTPClient(httpClient *http.Client, apiKey string) private_locationv1connect.PrivateLocationServiceClient {
	return private_locationv1connect.NewPrivateLocationServiceClient(
		httpClient,
		api.ConnectBaseURL,
		connect.WithInterceptors(api.NewAuthInterceptor(apiKey)),
		connect.WithProtoJSON(),
	)
}

// newMonitorClient is a local copy of the monitor client constructor. internal/monitors imports
// this package to resolve location names, so importing it back would create a cycle.
func newMonitorClient(httpClient *http.Client, apiKey string) monitorv1connect.MonitorServiceClient {
	return monitorv1connect.NewMonitorServiceClient(
		httpClient,
		api.ConnectBaseURL,
		connect.WithInterceptors(api.NewAuthInterceptor(apiKey)),
		connect.WithProtoJSON(),
	)
}

func statusToString(s private_locationv1.PrivateLocationStatus) string {
	switch s {
	case private_locationv1.PrivateLocationStatus_PRIVATE_LOCATION_STATUS_ACTIVE:
		return "active"
	case private_locationv1.PrivateLocationStatus_PRIVATE_LOCATION_STATUS_ERROR:
		return "error"
	default:
		return "unknown"
	}
}

func colorizeStatus(status string) string {
	switch status {
	case "active":
		return color.GreenString("● active")
	case "error":
		return color.RedString("● error")
	default:
		return "● unknown"
	}
}

// formatLastSeen renders an agent heartbeat timestamp. The API returns an empty string when the
// agent has never reported.
func formatLastSeen(rfc3339 string) string {
	if rfc3339 == "" {
		return "never"
	}
	return output.FormatTimestamp(rfc3339)
}

// MaskToken hides all but the last four characters of an agent token.
func MaskToken(token string) string {
	if token == "" {
		return ""
	}
	const visible = 4
	if len(token) <= visible {
		return strings.Repeat("•", len(token))
	}
	return strings.Repeat("•", 12) + token[len(token)-visible:]
}

func formatMetadata(metadata map[string]string) string {
	if len(metadata) == 0 {
		return "none"
	}
	keys := make([]string, 0, len(metadata))
	for k := range metadata {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	pairs := make([]string, 0, len(keys))
	for _, k := range keys {
		pairs = append(pairs, k+"="+metadata[k])
	}
	return strings.Join(pairs, ", ")
}

// Resolve turns private location IDs into display references. Callers that cannot tolerate a
// failure should inspect the error; the returned map is always usable, and IDs with no match are
// simply absent so callers can fall back to the raw ID.
func Resolve(ctx context.Context, client private_locationv1connect.PrivateLocationServiceClient, ids []string) (map[string]Ref, error) {
	refs := make(map[string]Ref, len(ids))
	if len(ids) == 0 {
		return refs, nil
	}

	resp, err := client.ListPrivateLocations(ctx, &private_locationv1.ListPrivateLocationsRequest{})
	if err != nil {
		return refs, err
	}

	wanted := make(map[string]struct{}, len(ids))
	for _, id := range ids {
		wanted[id] = struct{}{}
	}

	for _, l := range resp.GetPrivateLocations() {
		if _, ok := wanted[l.GetId()]; !ok {
			continue
		}
		refs[l.GetId()] = Ref{
			ID:     l.GetId(),
			Name:   l.GetName(),
			Status: statusToString(l.GetStatus()),
		}
	}

	return refs, nil
}

// ResolveWithHTTPClient is the entry point used by internal/monitors.
func ResolveWithHTTPClient(ctx context.Context, httpClient *http.Client, apiKey string, ids []string) (map[string]Ref, error) {
	return Resolve(ctx, NewPrivateLocationClientWithHTTPClient(httpClient, apiKey), ids)
}

// Label renders a resolved location for the monitors display, falling back to the raw ID when the
// lookup did not return it.
func Label(id string, refs map[string]Ref, withStatus bool) string {
	ref, ok := refs[id]
	if !ok {
		return id
	}
	if !withStatus {
		return ref.Name
	}
	return fmt.Sprintf("%s (%s)", ref.Name, colorizeStatus(ref.Status))
}

// Labels renders a list of locations in the order the IDs were given.
func Labels(ids []string, refs map[string]Ref, withStatus bool) []string {
	out := make([]string, 0, len(ids))
	for _, id := range ids {
		out = append(out, Label(id, refs, withStatus))
	}
	return out
}

// Refs returns resolved references in the order the IDs were given, synthesising an entry for any
// ID the lookup did not return.
func Refs(ids []string, refs map[string]Ref) []Ref {
	out := make([]Ref, 0, len(ids))
	for _, id := range ids {
		if ref, ok := refs[id]; ok {
			out = append(out, ref)
			continue
		}
		out = append(out, Ref{ID: id})
	}
	return out
}

func newBlueprintTable() *tablewriter.Table {
	return tablewriter.NewTable(os.Stdout,
		tablewriter.WithRenderer(renderer.NewBlueprint()),
		tablewriter.WithRendition(tw.Rendition{
			Symbols: tw.NewSymbolCustom("custom").WithColumn("="),
			Borders: tw.Border{
				Top:    tw.Off,
				Left:   tw.Off,
				Right:  tw.Off,
				Bottom: tw.Off,
			},
			Settings: tw.Settings{
				Lines: tw.Lines{
					ShowHeaderLine: tw.Off,
					ShowFooterLine: tw.On,
				},
				Separators: tw.Separators{
					BetweenRows:    tw.Off,
					BetweenColumns: tw.On,
				},
			},
		}),
		tablewriter.WithRowAlignment(tw.AlignLeft),
		tablewriter.WithHeaderAlignment(tw.AlignLeft),
	)
}

// monitorNames returns an ID to name map for every monitor in the workspace.
func monitorNames(ctx context.Context, client monitorv1connect.MonitorServiceClient) (map[string]string, error) {
	resp, err := client.ListMonitors(ctx, &monitorv1.ListMonitorsRequest{})
	if err != nil {
		return nil, err
	}

	names := make(map[string]string)
	for _, m := range resp.GetHttpMonitors() {
		names[m.GetId()] = m.GetName()
	}
	for _, m := range resp.GetTcpMonitors() {
		names[m.GetId()] = m.GetName()
	}
	for _, m := range resp.GetDnsMonitors() {
		names[m.GetId()] = m.GetName()
	}
	for _, m := range resp.GetIcmpMonitors() {
		names[m.GetId()] = m.GetName()
	}
	return names, nil
}

func PrivateLocationsCmd() *cli.Command {
	return &cli.Command{
		Name:    "private-locations",
		Aliases: []string{"pl"},
		Usage:   "Manage private locations",
		Description: `Private locations are self-hosted checker agents that run monitors from inside
your own network. This command lists them, shows their details, and creates new ones.

Public regions (Fly.io, Koyeb, Railway) are not listed here; they appear in
'openstatus monitors info' and 'openstatus check'.`,
		Commands: []*cli.Command{
			GetPrivateLocationListCmd(),
			GetPrivateLocationInfoCmd(),
			GetPrivateLocationCreateCmd(),
		},
	}
}
