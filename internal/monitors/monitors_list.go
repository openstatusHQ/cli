package monitors

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"

	"buf.build/gen/go/openstatus/api/connectrpc/gosimple/openstatus/monitor/v1/monitorv1connect"
	monitorv1 "buf.build/gen/go/openstatus/api/protocolbuffers/go/openstatus/monitor/v1"
	"github.com/fatih/color"
	"github.com/rodaine/table"
	"github.com/urfave/cli/v3"

	"github.com/openstatusHQ/cli/internal/api"
	"github.com/openstatusHQ/cli/internal/auth"
	output "github.com/openstatusHQ/cli/internal/cli"
	"github.com/openstatusHQ/cli/internal/privatelocation"
)

type monitorListEntry struct {
	ID               string                `json:"id"`
	Name             string                `json:"name"`
	URL              string                `json:"url"`
	Kind             string                `json:"kind"`
	PrivateLocations []privatelocation.Ref `json:"private_locations,omitempty"`

	privateLocationIDs []string
}

func ListMonitors(ctx context.Context, client monitorv1connect.MonitorServiceClient, resolver PrivateLocationResolver, showAll bool, s *output.Spinner) error {
	resp, err := client.ListMonitors(ctx, &monitorv1.ListMonitorsRequest{})
	output.StopSpinner(s)
	if err != nil {
		return output.FormatError(err, "monitors", "")
	}

	var entries []monitorListEntry

	for _, monitor := range resp.GetHttpMonitors() {
		if monitor.GetActive() || showAll {
			entries = append(entries, monitorListEntry{
				ID:                 monitor.GetId(),
				Name:               monitor.GetName(),
				URL:                monitor.GetUrl(),
				Kind:               "http",
				privateLocationIDs: monitor.GetPrivateLocationIds(),
			})
		}
	}

	for _, monitor := range resp.GetTcpMonitors() {
		if monitor.GetActive() || showAll {
			entries = append(entries, monitorListEntry{
				ID:                 monitor.GetId(),
				Name:               monitor.GetName(),
				URL:                monitor.GetUri(),
				Kind:               "tcp",
				privateLocationIDs: monitor.GetPrivateLocationIds(),
			})
		}
	}

	for _, monitor := range resp.GetDnsMonitors() {
		if monitor.GetActive() || showAll {
			entries = append(entries, monitorListEntry{
				ID:                 monitor.GetId(),
				Name:               monitor.GetName(),
				URL:                monitor.GetUri(),
				Kind:               "dns",
				privateLocationIDs: monitor.GetPrivateLocationIds(),
			})
		}
	}

	for _, monitor := range resp.GetIcmpMonitors() {
		if monitor.GetActive() || showAll {
			entries = append(entries, monitorListEntry{
				ID:                 monitor.GetId(),
				Name:               monitor.GetName(),
				URL:                monitor.GetUri(),
				Kind:               "icmp",
				privateLocationIDs: monitor.GetPrivateLocationIds(),
			})
		}
	}

	// The private-location column is only rendered when something is attached, so workspaces
	// without the feature keep the original four-column output and pay for no extra request.
	var allIDs []string
	for _, e := range entries {
		allIDs = append(allIDs, e.privateLocationIDs...)
	}

	hasPrivateLocations := len(allIDs) > 0
	var refs map[string]privatelocation.Ref
	if hasPrivateLocations && resolver != nil {
		resolved, err := resolver(ctx, allIDs)
		if err != nil {
			fmt.Fprintln(os.Stderr, "Warning: could not resolve private locations:", err)
		}
		refs = resolved
	}

	if hasPrivateLocations {
		for i := range entries {
			if len(entries[i].privateLocationIDs) > 0 {
				entries[i].PrivateLocations = privatelocation.Refs(entries[i].privateLocationIDs, refs)
			}
		}
	}

	if output.IsJSONOutput() {
		return output.PrintJSON(entries)
	}

	headerFmt := color.New(color.FgGreen, color.Underline).SprintfFunc()
	columnFmt := color.New(color.FgYellow).SprintfFunc()

	columns := []any{"ID", "Name", "Url", "Kind"}
	if hasPrivateLocations {
		columns = append(columns, "Private Locations")
	}

	tbl := table.New(columns...)
	tbl.WithHeaderFormatter(headerFmt).WithFirstColumnFormatter(columnFmt)

	for _, e := range entries {
		if !hasPrivateLocations {
			tbl.AddRow(e.ID, e.Name, e.URL, e.Kind)
			continue
		}
		locations := "—"
		if len(e.privateLocationIDs) > 0 {
			locations = strings.Join(privatelocation.Labels(e.privateLocationIDs, refs, false), ", ")
		}
		tbl.AddRow(e.ID, e.Name, e.URL, e.Kind, locations)
	}

	tbl.Print()

	return nil
}

// PrivateLocationResolver turns private location IDs into display references. It is injected so
// the monitors package stays independent of how the lookup is performed.
type PrivateLocationResolver func(ctx context.Context, ids []string) (map[string]privatelocation.Ref, error)

func newPrivateLocationResolver(httpClient *http.Client, apiKey string) PrivateLocationResolver {
	return func(ctx context.Context, ids []string) (map[string]privatelocation.Ref, error) {
		return privatelocation.ResolveWithHTTPClient(ctx, httpClient, apiKey, ids)
	}
}

func ListMonitorsWithHTTPClient(ctx context.Context, httpClient *http.Client, apiKey string) error {
	client := NewMonitorClientWithHTTPClient(httpClient, apiKey)
	return ListMonitors(ctx, client, newPrivateLocationResolver(httpClient, apiKey), false, nil)
}

func GetMonitorsListCmd() *cli.Command {
	monitorsListCmd := cli.Command{
		Name:  "list",
		Usage: "List all monitors",
		Description: `List all monitors. The list shows all your monitors attached to your workspace.
It displays the ID, name, URL, and kind of each monitor.`,
		UsageText: `openstatus monitors list
  openstatus monitors list --all`,
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:  "all",
				Usage: "List all monitors including inactive ones",
			},
			&cli.StringFlag{
				Name:    "access-token",
				Usage:   "OpenStatus API Access Token",
				Aliases: []string{"t"},
				Sources: cli.EnvVars("OPENSTATUS_API_TOKEN"),
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			apiKey, err := auth.ResolveAccessToken(cmd)
			if err != nil {
				return cli.Exit(err.Error(), 1)
			}
			if !output.IsQuiet() && !output.IsJSONOutput() {
				fmt.Println("List of all monitors")
			}
			s := output.StartSpinner("Fetching monitors...")
			client := NewMonitorClient(apiKey)
			err = ListMonitors(ctx, client, newPrivateLocationResolver(api.DefaultHTTPClient, apiKey), cmd.Bool("all"), s)
			if err != nil {
				return cli.Exit(err.Error(), 1)
			}
			return nil
		},
	}
	return &monitorsListCmd
}
