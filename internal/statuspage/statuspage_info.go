package statuspage

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"

	status_pagev1 "buf.build/gen/go/openstatus/api/protocolbuffers/go/openstatus/status_page/v1"
	"buf.build/gen/go/openstatus/api/connectrpc/gosimple/openstatus/status_page/v1/status_pagev1connect"
	"github.com/fatih/color"
	"github.com/logrusorgru/aurora/v4"
	"github.com/olekukonko/tablewriter"
	"github.com/olekukonko/tablewriter/renderer"
	"github.com/olekukonko/tablewriter/tw"
	"github.com/openstatusHQ/cli/internal/auth"
	output "github.com/openstatusHQ/cli/internal/cli"
	"github.com/rodaine/table"
	"github.com/urfave/cli/v3"
)

type statusPageDetail struct {
	ID          string                `json:"id"`
	Title       string                `json:"title"`
	Description string                `json:"description,omitempty"`
	URL         string                `json:"url"`
	Published   bool                  `json:"published"`
	AccessType  string                `json:"access_type"`
	Theme       string                `json:"theme"`
	HomepageURL string                `json:"homepage_url,omitempty"`
	ContactURL  string                `json:"contact_url,omitempty"`
	Components  []statusPageComponent `json:"components,omitempty"`
}

type statusPageComponent struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Type        string `json:"type"`
	Group       string `json:"group,omitempty"`
	Order       int32  `json:"order"`
}

func buildComponents(components []*status_pagev1.PageComponent, groups []*status_pagev1.PageComponentGroup) []statusPageComponent {
	groupMap := make(map[string]string, len(groups))
	for _, g := range groups {
		groupMap[g.GetId()] = g.GetName()
	}

	result := make([]statusPageComponent, 0, len(components))
	for _, c := range components {
		comp := statusPageComponent{
			ID:          c.GetId(),
			Name:        c.GetName(),
			Description: c.GetDescription(),
			Type:        componentTypeToString(c.GetType()),
			Order:       c.GetOrder(),
		}
		if gid := c.GetGroupId(); gid != "" {
			comp.Group = groupMap[gid]
		}
		result = append(result, comp)
	}

	sort.Slice(result, func(i, j int) bool {
		if result[i].Group != result[j].Group {
			if result[i].Group == "" {
				return false
			}
			if result[j].Group == "" {
				return true
			}
			return result[i].Group < result[j].Group
		}
		return result[i].Order < result[j].Order
	})

	return result
}

func GetStatusPageInfo(ctx context.Context, client status_pagev1connect.StatusPageServiceClient, pageId string, s *output.Spinner) error {
	if pageId == "" {
		output.StopSpinner(s)
		fmt.Fprintln(os.Stderr, "Usage: openstatus status-page info <page-id>")
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "Example: openstatus status-page info 12345")
		return fmt.Errorf("page ID is required")
	}

	req := &status_pagev1.GetStatusPageContentRequest{}
	req.SetId(pageId)
	resp, err := client.GetStatusPageContent(ctx, req)
	output.StopSpinner(s)
	if err != nil {
		return output.FormatError(err, "status-page", pageId)
	}

	page := resp.GetStatusPage()
	comps := buildComponents(resp.GetComponents(), resp.GetGroups())

	pageURL := "https://" + page.GetSlug() + ".openstatus.dev"
	if cd := page.GetCustomDomain(); cd != "" {
		cd = strings.TrimPrefix(strings.TrimPrefix(cd, "https://"), "http://")
		pageURL = "https://" + cd
	}

	if output.IsJSONOutput() {
		detail := statusPageDetail{
			ID:          page.GetId(),
			Title:       page.GetTitle(),
			Description: page.GetDescription(),
			URL:         pageURL,
			Published:   page.GetPublished(),
			AccessType:  accessTypeToString(page.GetAccessType()),
			Theme:       themeToString(page.GetTheme()),
			HomepageURL: page.GetHomepageUrl(),
			ContactURL:  page.GetContactUrl(),
			Components:  comps,
		}
		return output.PrintJSON(detail)
	}

	fmt.Println(aurora.Bold("Status Page:"))
	tbl := tablewriter.NewTable(os.Stdout,
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

	data := [][]string{
		{"ID", page.GetId()},
		{"Title", page.GetTitle()},
		{"URL", pageURL},
		{"Published", strconv.FormatBool(page.GetPublished())},
	}

	if page.GetDescription() != "" {
		data = append(data, []string{"Description", page.GetDescription()})
	}

	data = append(data, []string{"Access Type", accessTypeToString(page.GetAccessType())})
	data = append(data, []string{"Theme", themeToString(page.GetTheme())})

	if page.GetHomepageUrl() != "" {
		data = append(data, []string{"Homepage URL", page.GetHomepageUrl()})
	}
	if page.GetContactUrl() != "" {
		data = append(data, []string{"Contact URL", page.GetContactUrl()})
	}

	tbl.Bulk(data)
	tbl.Render()

	if len(comps) > 0 {
		fmt.Println(aurora.Bold("\nComponents:"))

		headerFmt := color.New(color.FgGreen, color.Underline).SprintfFunc()
		columnFmt := color.New(color.FgYellow).SprintfFunc()

		compTbl := table.New("Name", "Type", "Group")
		compTbl.WithHeaderFormatter(headerFmt).WithFirstColumnFormatter(columnFmt)

		for _, c := range comps {
			compTbl.AddRow(c.Name, c.Type, c.Group)
		}

		compTbl.Print()
	}

	return nil
}

func GetStatusPageInfoWithHTTPClient(ctx context.Context, httpClient *http.Client, apiKey string, pageId string) error {
	client := NewStatusPageClientWithHTTPClient(httpClient, apiKey)
	return GetStatusPageInfo(ctx, client, pageId, nil)
}

func GetStatusPageInfoCmd() *cli.Command {
	return &cli.Command{
		Name:  "info",
		Usage: "Get status page details",
		UsageText: `openstatus status-page info <PageID>
  openstatus status-page info 12345`,
		Flags: []cli.Flag{
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
			pageId := cmd.Args().Get(0)
			s := output.StartSpinner("Fetching status page...")
			client := NewStatusPageClient(apiKey)
			err = GetStatusPageInfo(ctx, client, pageId, s)
			if err != nil {
				return cli.Exit(err.Error(), 1)
			}
			return nil
		},
	}
}
