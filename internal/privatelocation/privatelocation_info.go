package privatelocation

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync"

	private_locationv1 "buf.build/gen/go/openstatus/api/protocolbuffers/go/openstatus/private_location/v1"
	"connectrpc.com/connect"
	"github.com/logrusorgru/aurora/v4"
	"github.com/urfave/cli/v3"

	"github.com/openstatusHQ/cli/internal/api"
	"github.com/openstatusHQ/cli/internal/auth"
	output "github.com/openstatusHQ/cli/internal/cli"
)

type monitorRef struct {
	ID   string `json:"id"`
	Name string `json:"name,omitempty"`
}

type infoOutput struct {
	ID         string            `json:"id"`
	Name       string            `json:"name"`
	Status     string            `json:"status"`
	Token      string            `json:"token"`
	Monitors   []monitorRef      `json:"monitors"`
	Metadata   map[string]string `json:"metadata,omitempty"`
	LastSeenAt string            `json:"last_seen_at"`
	CreatedAt  string            `json:"created_at"`
	UpdatedAt  string            `json:"updated_at"`
}

func GetPrivateLocationInfo(ctx context.Context, httpClient *http.Client, apiKey string, id string, showToken bool, s *output.Spinner) error {
	if id == "" {
		output.StopSpinner(s)
		fmt.Fprintln(os.Stderr, "Usage: openstatus private-locations info <private-location-id>")
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "Example: openstatus pl info pl_1a2b3c")
		return errors.New("private location ID is required")
	}

	locationClient := NewPrivateLocationClientWithHTTPClient(httpClient, apiKey)
	monitorClient := newMonitorClient(httpClient, apiKey)

	var (
		locationResp *private_locationv1.GetPrivateLocationResponse
		names        map[string]string
		locationErr  error
		namesErr     error
		wg           sync.WaitGroup
	)
	wg.Add(2)
	go func() {
		defer wg.Done()
		req := &private_locationv1.GetPrivateLocationRequest{}
		req.SetId(id)
		locationResp, locationErr = locationClient.GetPrivateLocation(ctx, req)
	}()
	go func() {
		defer wg.Done()
		names, namesErr = monitorNames(ctx, monitorClient)
	}()
	wg.Wait()

	output.StopSpinner(s)

	if locationErr != nil {
		var connectErr *connect.Error
		if errors.As(locationErr, &connectErr) && connectErr.Code() == connect.CodeNotFound {
			return fmt.Errorf("private location %q not found. Run 'openstatus pl list' to see available locations", id)
		}
		return output.FormatError(locationErr, "private location", id)
	}

	if namesErr != nil {
		fmt.Fprintln(os.Stderr, "Warning: could not resolve monitor names:", namesErr)
		names = map[string]string{}
	}

	location := locationResp.GetPrivateLocation()
	monitorIDs := location.GetMonitorIds()

	monitors := make([]monitorRef, 0, len(monitorIDs))
	for _, monitorID := range monitorIDs {
		monitors = append(monitors, monitorRef{ID: monitorID, Name: names[monitorID]})
	}

	token := location.GetToken()
	if !showToken {
		token = MaskToken(token)
	}

	if output.IsJSONOutput() {
		return output.PrintJSON(infoOutput{
			ID:         location.GetId(),
			Name:       location.GetName(),
			Status:     statusToString(location.GetStatus()),
			Token:      token,
			Monitors:   monitors,
			Metadata:   location.GetMetadata(),
			LastSeenAt: location.GetLastSeenAt(),
			CreatedAt:  location.GetCreatedAt(),
			UpdatedAt:  location.GetUpdatedAt(),
		})
	}

	fmt.Println(aurora.Bold("Private Location:"))
	tbl := newBlueprintTable()

	data := [][]string{
		{"ID", location.GetId()},
		{"Name", location.GetName()},
		{"Status", colorizeStatus(statusToString(location.GetStatus()))},
		{"Last Seen", formatLastSeen(location.GetLastSeenAt())},
		{"Monitors", formatMonitors(monitors)},
		{"Metadata", formatMetadata(location.GetMetadata())},
		{"Token", token},
	}

	tbl.Bulk(data)
	tbl.Render()

	if !showToken && location.GetToken() != "" && !output.IsQuiet() {
		fmt.Printf("\nRun with --show-token to reveal the agent token.\n")
	}

	return nil
}

func formatMonitors(monitors []monitorRef) string {
	if len(monitors) == 0 {
		return "none"
	}
	labels := make([]string, 0, len(monitors))
	for _, m := range monitors {
		if m.Name == "" {
			labels = append(labels, m.ID)
			continue
		}
		labels = append(labels, fmt.Sprintf("%s (%s)", m.Name, m.ID))
	}
	return fmt.Sprintf("%d: %s", len(monitors), strings.Join(labels, ", "))
}

func GetPrivateLocationInfoWithHTTPClient(ctx context.Context, httpClient *http.Client, apiKey string, id string, showToken bool) error {
	return GetPrivateLocationInfo(ctx, httpClient, apiKey, id, showToken, nil)
}

func GetPrivateLocationInfoCmd() *cli.Command {
	return &cli.Command{
		Name:  "info",
		Usage: "Get private location details",
		UsageText: `openstatus private-locations info <PrivateLocationID>
  openstatus pl info pl_1a2b3c
  openstatus pl info pl_1a2b3c --show-token`,
		Description: `Fetch a private location including the monitors it runs and its agent token.
The token is masked unless --show-token is passed.`,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "access-token",
				Usage:   "OpenStatus API Access Token",
				Aliases: []string{"t"},
				Sources: cli.EnvVars("OPENSTATUS_API_TOKEN"),
			},
			&cli.BoolFlag{
				Name:  "show-token",
				Usage: "Reveal the agent token instead of masking it",
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			apiKey, err := auth.ResolveAccessToken(cmd)
			if err != nil {
				return cli.Exit(err.Error(), 1)
			}
			id := cmd.Args().Get(0)
			s := output.StartSpinner("Fetching private location...")
			if err := GetPrivateLocationInfo(ctx, api.DefaultHTTPClient, apiKey, id, cmd.Bool("show-token"), s); err != nil {
				return cli.Exit(err.Error(), 1)
			}
			return nil
		},
	}
}
