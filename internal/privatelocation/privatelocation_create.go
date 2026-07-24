package privatelocation

import (
	"context"
	"fmt"
	"net/http"
	"sort"
	"strings"

	"buf.build/gen/go/openstatus/api/connectrpc/gosimple/openstatus/monitor/v1/monitorv1connect"
	"buf.build/gen/go/openstatus/api/connectrpc/gosimple/openstatus/private_location/v1/private_locationv1connect"
	private_locationv1 "buf.build/gen/go/openstatus/api/protocolbuffers/go/openstatus/private_location/v1"
	"github.com/charmbracelet/huh"
	"github.com/logrusorgru/aurora/v4"
	"github.com/urfave/cli/v3"

	"github.com/openstatusHQ/cli/internal/api"
	"github.com/openstatusHQ/cli/internal/auth"
	output "github.com/openstatusHQ/cli/internal/cli"
	"github.com/openstatusHQ/cli/internal/wizard"
)

type CreateParams struct {
	Name       string
	MonitorIDs []string
	Metadata   map[string]string
}

type createOutput struct {
	ID         string            `json:"id"`
	Name       string            `json:"name"`
	Token      string            `json:"token"`
	MonitorIDs []string          `json:"monitor_ids"`
	Metadata   map[string]string `json:"metadata,omitempty"`
	CreatedAt  string            `json:"created_at"`
}

// parseMetadataFlag turns repeated key=value pairs into a map.
func parseMetadataFlag(values []string) (map[string]string, error) {
	if len(values) == 0 {
		return nil, nil
	}
	metadata := make(map[string]string, len(values))
	for _, v := range values {
		key, value, found := strings.Cut(v, "=")
		key = strings.TrimSpace(key)
		if !found || key == "" {
			return nil, fmt.Errorf("invalid --metadata %q: expected key=value", v)
		}
		metadata[key] = value
	}
	return metadata, nil
}

func CreatePrivateLocation(ctx context.Context, client private_locationv1connect.PrivateLocationServiceClient, p CreateParams) (*private_locationv1.PrivateLocation, error) {
	req := &private_locationv1.CreatePrivateLocationRequest{}
	req.SetName(p.Name)
	if len(p.MonitorIDs) > 0 {
		req.SetMonitorIds(p.MonitorIDs)
	}
	if len(p.Metadata) > 0 {
		req.SetMetadata(p.Metadata)
	}

	resp, err := client.CreatePrivateLocation(ctx, req)
	if err != nil {
		return nil, output.FormatError(err, "private location", "")
	}

	return resp.GetPrivateLocation(), nil
}

func renderCreated(location *private_locationv1.PrivateLocation) error {
	if output.IsJSONOutput() {
		return output.PrintJSON(createOutput{
			ID:         location.GetId(),
			Name:       location.GetName(),
			Token:      location.GetToken(),
			MonitorIDs: location.GetMonitorIds(),
			Metadata:   location.GetMetadata(),
			CreatedAt:  location.GetCreatedAt(),
		})
	}

	if output.IsQuiet() {
		fmt.Println(location.GetId())
		return nil
	}

	fmt.Println(aurora.Bold("Private location created:"))
	tbl := newBlueprintTable()
	tbl.Bulk([][]string{
		{"ID", location.GetId()},
		{"Name", location.GetName()},
		{"Monitors", fmt.Sprintf("%d", len(location.GetMonitorIds()))},
		{"Metadata", formatMetadata(location.GetMetadata())},
		{"Token", location.GetToken()},
	})
	tbl.Render()

	fmt.Println("\nStore this token securely. Retrieve it again with:")
	fmt.Printf("  openstatus pl info %s --show-token\n", location.GetId())

	return nil
}

func CreatePrivateLocationWithHTTPClient(ctx context.Context, httpClient *http.Client, apiKey string, p CreateParams) error {
	client := NewPrivateLocationClientWithHTTPClient(httpClient, apiKey)
	location, err := CreatePrivateLocation(ctx, client, p)
	if err != nil {
		return err
	}
	return renderCreated(location)
}

// runCreateWizard prompts for anything the flags did not supply.
func runCreateWizard(ctx context.Context, monitorClient monitorv1connect.MonitorServiceClient, prefilled CreateParams) (CreateParams, error) {
	inputs := prefilled

	var fields []huh.Field

	if inputs.Name == "" {
		fields = append(fields, huh.NewInput().
			Title("Name").
			Description("A label for this private location, e.g. office-paris").
			Validate(wizard.NotEmpty("name")).
			Value(&inputs.Name))
	}

	if len(inputs.MonitorIDs) == 0 {
		s := output.StartSpinner("Fetching monitors...")
		names, err := monitorNames(ctx, monitorClient)
		output.StopSpinner(s)
		if err != nil {
			return inputs, fmt.Errorf("could not fetch monitors: %w", err)
		}

		if len(names) > 0 {
			ids := make([]string, 0, len(names))
			for id := range names {
				ids = append(ids, id)
			}
			sort.Strings(ids)

			options := make([]huh.Option[string], 0, len(ids))
			for _, id := range ids {
				options = append(options, huh.NewOption(fmt.Sprintf("%s (%s)", names[id], id), id))
			}

			fields = append(fields, huh.NewMultiSelect[string]().
				Title("Monitors").
				Description("Monitors this private location should run (optional)").
				Options(options...).
				Value(&inputs.MonitorIDs))
		}
	}

	if len(fields) == 0 {
		return inputs, nil
	}

	form := huh.NewForm(huh.NewGroup(fields...)).WithTheme(huh.ThemeBase())
	if err := form.Run(); err != nil {
		return inputs, wizard.HandleFormError(err)
	}

	return inputs, nil
}

func GetPrivateLocationCreateCmd() *cli.Command {
	return &cli.Command{
		Name:  "create",
		Usage: "Create a private location",
		UsageText: `openstatus private-locations create --name office-paris
  openstatus pl create --name office-paris --monitor-ids 12345,12346
  openstatus pl create --name office-paris --metadata env=prod --metadata team=infra`,
		Description: `Create a private location and print its agent token. The token is shown in full
because you need it to configure the agent; retrieve it later with 'openstatus pl info --show-token'.

Run without --name in an interactive terminal to be prompted, including a
picker for the monitors to attach.`,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "access-token",
				Usage:   "OpenStatus API Access Token",
				Aliases: []string{"t"},
				Sources: cli.EnvVars("OPENSTATUS_API_TOKEN"),
			},
			&cli.StringFlag{
				Name:  "name",
				Usage: "Name of the private location",
			},
			&cli.StringSliceFlag{
				Name:  "monitor-ids",
				Usage: "Monitor IDs to attach, comma-separated or repeated",
			},
			&cli.StringSliceFlag{
				Name:  "metadata",
				Usage: "Key/value label, repeatable: --metadata <key>=<value>",
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			apiKey, err := auth.ResolveAccessToken(cmd)
			if err != nil {
				return cli.Exit(err.Error(), 1)
			}

			metadata, err := parseMetadataFlag(cmd.StringSlice("metadata"))
			if err != nil {
				return cli.Exit(err.Error(), 1)
			}

			params := CreateParams{
				Name:       cmd.String("name"),
				MonitorIDs: cmd.StringSlice("monitor-ids"),
				Metadata:   metadata,
			}

			if params.Name == "" {
				if output.IsJSONOutput() || !output.IsStdinTerminal() {
					return cli.Exit("missing required flags: --name", 1)
				}
				params, err = runCreateWizard(ctx, newMonitorClient(api.DefaultHTTPClient, apiKey), params)
				if err != nil {
					return cli.Exit(err.Error(), 1)
				}
			}

			s := output.StartSpinner("Creating private location...")
			client := NewPrivateLocationClient(apiKey)
			location, err := CreatePrivateLocation(ctx, client, params)
			output.StopSpinner(s)
			if err != nil {
				return cli.Exit(err.Error(), 1)
			}

			if err := renderCreated(location); err != nil {
				return cli.Exit(err.Error(), 1)
			}
			return nil
		},
	}
}
