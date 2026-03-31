package statusreport

import (
	"context"
	"fmt"
	"os"
	"strings"

	status_reportv1 "buf.build/gen/go/openstatus/api/protocolbuffers/go/openstatus/status_report/v1"
	"github.com/charmbracelet/huh"
	output "github.com/openstatusHQ/cli/internal/cli"
	"github.com/openstatusHQ/cli/internal/statuspage"
	"github.com/openstatusHQ/cli/internal/wizard"
)

type createInputs struct {
	PageID         string
	PageName       string
	Title          string
	Status         string
	Message        string
	ComponentIDs   []string
	componentNames map[string]string
	Notify         bool
	Confirmed      bool
}

type addUpdateInputs struct {
	ReportID   string
	ReportName string
	Status     string
	Message    string
	Notify     bool
	Confirmed  bool
}

func statusSelectOptions() []huh.Option[string] {
	return []huh.Option[string]{
		huh.NewOption("Investigating", "investigating"),
		huh.NewOption("Identified", "identified"),
		huh.NewOption("Monitoring", "monitoring"),
		huh.NewOption("Resolved", "resolved"),
	}
}

func fetchStatusReports(ctx context.Context, apiKey string) ([]*status_reportv1.StatusReportSummary, error) {
	client := NewStatusReportClient(apiKey)

	req := &status_reportv1.ListStatusReportsRequest{}
	l := int32(20)
	req.SetLimit(l)
	req.SetStatuses([]status_reportv1.StatusReportStatus{
		status_reportv1.StatusReportStatus_STATUS_REPORT_STATUS_INVESTIGATING,
		status_reportv1.StatusReportStatus_STATUS_REPORT_STATUS_IDENTIFIED,
		status_reportv1.StatusReportStatus_STATUS_REPORT_STATUS_MONITORING,
	})

	resp, err := client.ListStatusReports(ctx, req)
	if err != nil {
		return nil, output.FormatError(err, "status-report", "")
	}

	reports := resp.GetStatusReports()
	if len(reports) > 0 {
		return reports, nil
	}

	reqAll := &status_reportv1.ListStatusReportsRequest{}
	reqAll.SetLimit(l)
	resp, err = client.ListStatusReports(ctx, reqAll)
	if err != nil {
		return nil, output.FormatError(err, "status-report", "")
	}
	return resp.GetStatusReports(), nil
}

// Wizard: sr create

func runCreateWizard(ctx context.Context, apiKey string, prefilled *createInputs) (*createInputs, error) {
	inputs := *prefilled

	s := output.StartSpinner("Fetching status pages...")
	pages, err := wizard.FetchStatusPages(ctx, apiKey)
	output.StopSpinner(s)
	if err != nil {
		return nil, err
	}
	if len(pages) == 0 {
		return nil, fmt.Errorf("no status pages found. Create one at https://www.openstatus.dev first, then run this command again")
	}

	if inputs.PageID == "" {
		pageOptions := make([]huh.Option[string], 0, len(pages))
		for _, p := range pages {
			label := p.GetTitle() + " (" + statuspage.StatusPageURL(p) + ")"
			pageOptions = append(pageOptions, huh.NewOption(label, p.GetId()))
		}

		form1 := huh.NewForm(
			huh.NewGroup(
				huh.NewSelect[string]().
					Title("Status page").
					Options(pageOptions...).
					Value(&inputs.PageID),
			),
		).WithTheme(huh.ThemeBase())

		if err := form1.Run(); err != nil {
			return nil, wizard.HandleFormError(err)
		}
	}

	for _, p := range pages {
		if p.GetId() == inputs.PageID {
			inputs.PageName = p.GetTitle()
			break
		}
	}

	s = output.StartSpinner("Fetching page components...")
	components, groups, err := wizard.FetchPageComponents(ctx, apiKey, inputs.PageID)
	output.StopSpinner(s)
	if err != nil {
		return nil, err
	}

	var fields []huh.Field

	if len(components) > 0 {
		groupMap := make(map[string]string, len(groups))
		for _, g := range groups {
			groupMap[g.GetId()] = g.GetName()
		}

		inputs.componentNames = make(map[string]string, len(components))
		compOptions := make([]huh.Option[string], 0, len(components))
		for _, c := range components {
			label := c.GetName()
			if gid := c.GetGroupId(); gid != "" {
				if gname, ok := groupMap[gid]; ok {
					label += " (" + gname + ")"
				}
			}
			inputs.componentNames[c.GetId()] = label
			compOptions = append(compOptions, huh.NewOption(label, c.GetId()))
		}

		if len(prefilled.ComponentIDs) == 0 {
			fields = append(fields, huh.NewMultiSelect[string]().
				Title("Components").
				Options(compOptions...).
				Value(&inputs.ComponentIDs))
		}
	}

	if inputs.Title == "" {
		fields = append(fields, huh.NewInput().
			Title("Title").
			Validate(wizard.NotEmpty("title")).
			Value(&inputs.Title))
	}

	if inputs.Status == "" {
		fields = append(fields, huh.NewSelect[string]().
			Title("Status").
			Options(statusSelectOptions()...).
			Value(&inputs.Status))
	}

	if inputs.Message == "" {
		fields = append(fields, huh.NewText().
			Title("Message").
			Validate(wizard.NotEmpty("message")).
			Value(&inputs.Message))
	}

	fields = append(fields, huh.NewConfirm().
		Title("Notify subscribers?").
		Value(&inputs.Notify))

	summaryNote := huh.NewNote().
		Title("Summary").
		DescriptionFunc(func() string {
			lines := [][2]string{
				{"Page", inputs.PageName},
			}
			if len(inputs.ComponentIDs) > 0 {
				names := make([]string, 0, len(inputs.ComponentIDs))
				for _, id := range inputs.ComponentIDs {
					if name, ok := inputs.componentNames[id]; ok {
						names = append(names, name)
					} else {
						names = append(names, id)
					}
				}
				lines = append(lines, [2]string{"Components", strings.Join(names, ", ")})
			}
			lines = append(lines,
				[2]string{"Title", inputs.Title},
				[2]string{"Status", inputs.Status},
				[2]string{"Message", inputs.Message},
			)
			notifyStr := "no"
			if inputs.Notify {
				notifyStr = "yes"
			}
			lines = append(lines, [2]string{"Notify", notifyStr})
			return wizard.BuildSummary(lines)
		}, &inputs)

	form2 := huh.NewForm(
		huh.NewGroup(fields...),
		huh.NewGroup(
			summaryNote,
			huh.NewConfirm().
				Title("Create this status report?").
				Value(&inputs.Confirmed),
		),
	).WithTheme(huh.ThemeBase())

	if err := form2.Run(); err != nil {
		return nil, wizard.HandleFormError(err)
	}

	if !inputs.Confirmed {
		fmt.Fprintln(os.Stderr, "Aborted.")
		os.Exit(130)
	}

	return &inputs, nil
}

// Wizard: sr add-update

func runAddUpdateWizard(ctx context.Context, apiKey string, prefilled *addUpdateInputs) (*addUpdateInputs, error) {
	inputs := *prefilled

	s := output.StartSpinner("Fetching status reports...")
	reports, err := fetchStatusReports(ctx, apiKey)
	output.StopSpinner(s)
	if err != nil {
		return nil, err
	}

	if inputs.ReportID == "" {
		if len(reports) == 0 {
			return nil, fmt.Errorf("no status reports found")
		}

		reportOptions := make([]huh.Option[string], 0, len(reports))
		for _, r := range reports {
			label := r.GetTitle() + " (" + statusToString(r.GetStatus()) + ")"
			reportOptions = append(reportOptions, huh.NewOption(label, r.GetId()))
		}

		form1 := huh.NewForm(
			huh.NewGroup(
				huh.NewSelect[string]().
					Title("Select a status report").
					Options(reportOptions...).
					Value(&inputs.ReportID),
			),
		).WithTheme(huh.ThemeBase())

		if err := form1.Run(); err != nil {
			return nil, wizard.HandleFormError(err)
		}
	}

	for _, r := range reports {
		if r.GetId() == inputs.ReportID {
			inputs.ReportName = r.GetTitle()
			break
		}
	}
	if inputs.ReportName == "" {
		inputs.ReportName = inputs.ReportID
	}

	var fields []huh.Field

	if inputs.Status == "" {
		fields = append(fields, huh.NewSelect[string]().
			Title("Status").
			Options(statusSelectOptions()...).
			Value(&inputs.Status))
	}

	if inputs.Message == "" {
		fields = append(fields, huh.NewInput().
			Title("Message").
			Validate(wizard.NotEmpty("message")).
			Value(&inputs.Message))
	}

	fields = append(fields, huh.NewConfirm().
		Title("Notify subscribers?").
		Value(&inputs.Notify))

	summaryNote := huh.NewNote().
		Title("Summary").
		DescriptionFunc(func() string {
			lines := [][2]string{
				{"Report", inputs.ReportName},
				{"Status", inputs.Status},
				{"Message", inputs.Message},
			}
			notifyStr := "no"
			if inputs.Notify {
				notifyStr = "yes"
			}
			lines = append(lines, [2]string{"Notify", notifyStr})
			return wizard.BuildSummary(lines)
		}, &inputs)

	form2 := huh.NewForm(
		huh.NewGroup(fields...),
		huh.NewGroup(
			summaryNote,
			huh.NewConfirm().
				Title("Add this update?").
				Value(&inputs.Confirmed),
		),
	).WithTheme(huh.ThemeBase())

	if err := form2.Run(); err != nil {
		return nil, wizard.HandleFormError(err)
	}

	if !inputs.Confirmed {
		fmt.Fprintln(os.Stderr, "Aborted.")
		os.Exit(130)
	}

	return &inputs, nil
}

