package statuspage

import (
	"context"
	"fmt"
	"net/http"

	status_pagev1 "buf.build/gen/go/openstatus/api/protocolbuffers/go/openstatus/status_page/v1"
	"buf.build/gen/go/openstatus/api/connectrpc/gosimple/openstatus/status_page/v1/status_pagev1connect"
	"github.com/fatih/color"
	"github.com/openstatusHQ/cli/internal/auth"
	output "github.com/openstatusHQ/cli/internal/cli"
	"github.com/rodaine/table"
	"github.com/urfave/cli/v3"
)

type statusPageListEntry struct {
	ID    string `json:"id"`
	Title string `json:"title"`
	URL   string `json:"url"`
}

func ListStatusPages(ctx context.Context, client status_pagev1connect.StatusPageServiceClient, limit int, s *output.Spinner) error {
	req := &status_pagev1.ListStatusPagesRequest{}

	if limit > 0 {
		l := int32(limit)
		req.SetLimit(l)
	}

	resp, err := client.ListStatusPages(ctx, req)
	output.StopSpinner(s)
	if err != nil {
		return output.FormatError(err, "status-page", "")
	}

	pages := resp.GetStatusPages()

	if output.IsJSONOutput() {
		entries := make([]statusPageListEntry, 0, len(pages))
		for _, p := range pages {
			entries = append(entries, statusPageListEntry{
				ID:    p.GetId(),
				Title: p.GetTitle(),
				URL:   "https://" + p.GetSlug() + ".openstatus.dev",
			})
		}
		return output.PrintJSON(entries)
	}

	if len(pages) == 0 {
		if !output.IsQuiet() {
			fmt.Println("No status pages found")
		}
		return nil
	}

	headerFmt := color.New(color.FgGreen, color.Underline).SprintfFunc()
	columnFmt := color.New(color.FgYellow).SprintfFunc()

	tbl := table.New("ID", "Title", "URL")
	tbl.WithHeaderFormatter(headerFmt).WithFirstColumnFormatter(columnFmt)

	for _, p := range pages {
		tbl.AddRow(
			p.GetId(),
			p.GetTitle(),
			"https://"+p.GetSlug()+".openstatus.dev",
		)
	}

	tbl.Print()
	return nil
}

func ListStatusPagesWithHTTPClient(ctx context.Context, httpClient *http.Client, apiKey string, limit int) error {
	client := NewStatusPageClientWithHTTPClient(httpClient, apiKey)
	return ListStatusPages(ctx, client, limit, nil)
}

func GetStatusPageListCmd() *cli.Command {
	return &cli.Command{
		Name:  "list",
		Usage: "List all status pages",
		UsageText: `openstatus status-page list
  openstatus status-page list --limit 10`,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "access-token",
				Usage:   "OpenStatus API Access Token",
				Aliases: []string{"t"},
				Sources: cli.EnvVars("OPENSTATUS_API_TOKEN"),
			},
			&cli.IntFlag{
				Name:  "limit",
				Usage: "Maximum number of pages to return (1-100)",
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			apiKey, err := auth.ResolveAccessToken(cmd)
			if err != nil {
				return cli.Exit(err.Error(), 1)
			}
			s := output.StartSpinner("Fetching status pages...")
			client := NewStatusPageClient(apiKey)
			err = ListStatusPages(ctx, client, int(cmd.Int("limit")), s)
			if err != nil {
				return cli.Exit(err.Error(), 1)
			}
			return nil
		},
	}
}
