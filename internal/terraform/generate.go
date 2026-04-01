package terraform

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/openstatusHQ/cli/internal/auth"
	output "github.com/openstatusHQ/cli/internal/cli"
	"github.com/urfave/cli/v3"
)

func GetTerraformGenerateCmd() *cli.Command {
	return &cli.Command{
		Name:      "generate",
		Aliases:   []string{"gen"},
		Usage:     "Generate Terraform configuration from workspace resources",
		UsageText: "openstatus terraform generate [--output-dir ./openstatus-terraform/]",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "access-token",
				Usage:   "OpenStatus API Access Token",
				Aliases: []string{"t"},
				Sources: cli.EnvVars("OPENSTATUS_API_TOKEN"),
			},
			&cli.StringFlag{
				Name:    "output-dir",
				Usage:   "Directory to write Terraform files into",
				Value:   "./openstatus-terraform/",
				Aliases: []string{"o"},
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			apiKey, err := auth.ResolveAccessToken(cmd)
			if err != nil {
				return cli.Exit(err.Error(), 1)
			}

			outputDir := cmd.String("output-dir")
			if err := os.MkdirAll(outputDir, 0755); err != nil {
				return cli.Exit(fmt.Sprintf("failed to create output directory: %v", err), 1)
			}

			s := output.StartSpinner("Fetching workspace resources...")
			data, err := FetchWorkspaceData(ctx, apiKey)
			output.StopSpinner(s)
			if err != nil {
				return cli.Exit(err.Error(), 1)
			}

			gen := NewGenerator(data)

			// Always write provider.tf
			if err := writeFile(filepath.Join(outputDir, "provider.tf"), GenerateProviderFile()); err != nil {
				return cli.Exit(fmt.Sprintf("failed to write provider.tf: %v", err), 1)
			}

			if gen.TotalResourceCount() == 0 {
				fmt.Printf("No resources found in workspace. Only provider.tf was generated in %s\n", outputDir)
				return nil
			}

			if gen.HasMonitors() {
				if err := writeFile(filepath.Join(outputDir, "monitors.tf"), gen.GenerateMonitorsFile().Bytes()); err != nil {
					return cli.Exit(fmt.Sprintf("failed to write monitors.tf: %v", err), 1)
				}
			}

			if gen.HasNotifications() {
				if err := writeFile(filepath.Join(outputDir, "notifications.tf"), gen.GenerateNotificationsFile().Bytes()); err != nil {
					return cli.Exit(fmt.Sprintf("failed to write notifications.tf: %v", err), 1)
				}
			}

			if gen.HasStatusPages() {
				if err := writeFile(filepath.Join(outputDir, "status_pages.tf"), gen.GenerateStatusPagesFile().Bytes()); err != nil {
					return cli.Exit(fmt.Sprintf("failed to write status_pages.tf: %v", err), 1)
				}
			}

			if err := writeFile(filepath.Join(outputDir, "imports.tf"), gen.GenerateImportsFile().Bytes()); err != nil {
				return cli.Exit(fmt.Sprintf("failed to write imports.tf: %v", err), 1)
			}

			printSummary(outputDir, data)
			return nil
		},
	}
}

func writeFile(path string, content []byte) error {
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.Write(content)
	return err
}

func printSummary(outputDir string, data *WorkspaceData) {
	httpCount := len(data.HTTPMonitors)
	tcpCount := len(data.TCPMonitors)
	dnsCount := len(data.DNSMonitors)
	monitorTotal := httpCount + tcpCount + dnsCount
	notifCount := len(data.Notifications)

	pageCount := len(data.StatusPages)
	compCount := 0
	groupCount := 0
	for _, sp := range data.StatusPages {
		compCount += len(sp.Components)
		groupCount += len(sp.Groups)
	}

	importCount := monitorTotal + notifCount + pageCount + compCount + groupCount

	fmt.Printf("\nGenerated Terraform configuration in %s\n\n", outputDir)
	if monitorTotal > 0 {
		fmt.Printf("  %d monitors (%d HTTP, %d TCP, %d DNS)\n", monitorTotal, httpCount, tcpCount, dnsCount)
	}
	if notifCount > 0 {
		fmt.Printf("  %d notifications\n", notifCount)
	}
	if pageCount > 0 {
		fmt.Printf("  %d status pages (%d components, %d groups)\n", pageCount, compCount, groupCount)
	}
	fmt.Printf("  %d import blocks\n", importCount)
	fmt.Printf("\nNext steps:\n")
	fmt.Printf("  cd %s\n", outputDir)
	fmt.Printf("  terraform init\n")
	fmt.Printf("  terraform plan\n")
}
