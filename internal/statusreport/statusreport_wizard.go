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
	PageID           string
	PageName         string
	Title            string
	Status           string
	Message          string
	ComponentIDs     []string
	ComponentImpacts map[string]string
	componentNames   map[string]string
	Notify           bool
	Confirmed        bool
}

type addUpdateInputs struct {
	ReportID         string
	ReportName       string
	Status           string
	Message          string
	ComponentImpacts map[string]string
	priorImpacts     map[string]string
	Notify           bool
	Confirmed        bool
}

func impactsToMap(in []*status_reportv1.ComponentImpact) map[string]string {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]string, len(in))
	for _, ci := range in {
		out[ci.GetPageComponentId()] = impactToString(ci.GetImpact())
	}
	return out
}

func mapToImpacts(in map[string]string, order []string) ([]*status_reportv1.ComponentImpact, error) {
	if len(in) == 0 {
		return nil, nil
	}
	out := make([]*status_reportv1.ComponentImpact, 0, len(in))
	if len(order) == 0 {
		order = make([]string, 0, len(in))
		for k := range in {
			order = append(order, k)
		}
	}
	for _, id := range order {
		token, ok := in[id]
		if !ok {
			continue
		}
		level, err := impactToSDK(token)
		if err != nil {
			return nil, err
		}
		ci := &status_reportv1.ComponentImpact{}
		ci.SetPageComponentId(id)
		ci.SetImpact(level)
		out = append(out, ci)
	}
	return out, nil
}

func statusSelectOptions() []huh.Option[string] {
	return []huh.Option[string]{
		huh.NewOption("Investigating", "investigating"),
		huh.NewOption("Identified", "identified"),
		huh.NewOption("Monitoring", "monitoring"),
		huh.NewOption("Resolved", "resolved"),
	}
}

func impactSelectOptions() []huh.Option[string] {
	return []huh.Option[string]{
		huh.NewOption("Operational", "operational"),
		huh.NewOption("Degraded performance", "degraded"),
		huh.NewOption("Partial outage", "partial_outage"),
		huh.NewOption("Major outage", "major_outage"),
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

		if len(prefilled.ComponentIDs) == 0 && len(prefilled.ComponentImpacts) == 0 {
			form2 := huh.NewForm(
				huh.NewGroup(
					huh.NewMultiSelect[string]().
						Title("Components").
						Options(compOptions...).
						Value(&inputs.ComponentIDs),
				),
			).WithTheme(huh.ThemeBase())
			if err := form2.Run(); err != nil {
				return nil, wizard.HandleFormError(err)
			}
		}
	}

	if len(inputs.ComponentImpacts) == 0 && len(inputs.ComponentIDs) > 0 {
		inputs.ComponentImpacts = make(map[string]string, len(inputs.ComponentIDs))
		for _, id := range inputs.ComponentIDs {
			inputs.ComponentImpacts[id] = "partial_outage"
		}
	}

	var fields []huh.Field

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

	impactPickers := make(map[string]*string, len(inputs.ComponentIDs))
	for _, id := range inputs.ComponentIDs {
		current := inputs.ComponentImpacts[id]
		impactPickers[id] = &current
		label := inputs.componentNames[id]
		if label == "" {
			label = id
		}
		fields = append(fields, huh.NewSelect[string]().
			Title("Impact for "+label).
			Options(impactSelectOptions()...).
			Value(impactPickers[id]))
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
				rows := make([]string, 0, len(inputs.ComponentIDs))
				for _, id := range inputs.ComponentIDs {
					name := inputs.componentNames[id]
					if name == "" {
						name = id
					}
					token := ""
					if impactPickers[id] != nil {
						token = *impactPickers[id]
					}
					rows = append(rows, name+" ("+token+")")
				}
				lines = append(lines, [2]string{"Components", strings.Join(rows, ", ")})
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

	form3 := huh.NewForm(
		huh.NewGroup(fields...),
		huh.NewGroup(
			summaryNote,
			huh.NewConfirm().
				Title("Create this status report?").
				Value(&inputs.Confirmed),
		),
	).WithTheme(huh.ThemeBase())

	if err := form3.Run(); err != nil {
		return nil, wizard.HandleFormError(err)
	}

	if !inputs.Confirmed {
		fmt.Fprintln(os.Stderr, "Aborted.")
		os.Exit(130)
	}

	for id, ptr := range impactPickers {
		inputs.ComponentImpacts[id] = *ptr
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

	s = output.StartSpinner("Fetching status report details...")
	client := NewStatusReportClient(apiKey)
	getReq := &status_reportv1.GetStatusReportRequest{}
	getReq.SetId(inputs.ReportID)
	detail, err := client.GetStatusReport(ctx, getReq)
	output.StopSpinner(s)
	if err != nil {
		return nil, output.FormatError(err, "status-report", inputs.ReportID)
	}
	report := detail.GetStatusReport()

	prior := currentImpacts(report.GetUpdates())
	inputs.priorImpacts = make(map[string]string, len(prior))
	for id, level := range prior {
		inputs.priorImpacts[id] = impactToString(level)
	}

	affectedIDs := affectedOrder(report)

	var selectedComponents []string
	if len(affectedIDs) > 0 {
		compOptions := make([]huh.Option[string], 0, len(affectedIDs))
		for _, id := range affectedIDs {
			currentLabel := "—"
			if v, ok := prior[id]; ok && v != status_reportv1.PageComponentImpact_PAGE_COMPONENT_IMPACT_UNSPECIFIED {
				currentLabel = impactToString(v)
			}
			compOptions = append(compOptions, huh.NewOption(id+" (current: "+currentLabel+")", id))
		}
		form2 := huh.NewForm(
			huh.NewGroup(
				huh.NewMultiSelect[string]().
					Title("Components to re-impact (leave empty to skip)").
					Options(compOptions...).
					Value(&selectedComponents),
			),
		).WithTheme(huh.ThemeBase())
		if err := form2.Run(); err != nil {
			return nil, wizard.HandleFormError(err)
		}
	}

	if inputs.ComponentImpacts == nil {
		inputs.ComponentImpacts = make(map[string]string, len(selectedComponents))
	}
	for _, id := range selectedComponents {
		if _, ok := inputs.ComponentImpacts[id]; ok {
			continue
		}
		if prev, ok := inputs.priorImpacts[id]; ok && prev != "" {
			inputs.ComponentImpacts[id] = prev
		} else {
			inputs.ComponentImpacts[id] = "partial_outage"
		}
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

	impactPickers := make(map[string]*string, len(selectedComponents))
	for _, id := range selectedComponents {
		current := inputs.ComponentImpacts[id]
		impactPickers[id] = &current
		fields = append(fields, huh.NewSelect[string]().
			Title("Impact for "+id).
			Options(impactSelectOptions()...).
			Value(impactPickers[id]))
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
			if len(selectedComponents) > 0 {
				rows := make([]string, 0, len(selectedComponents))
				for _, id := range selectedComponents {
					prev := inputs.priorImpacts[id]
					if prev == "" {
						prev = "(none)"
					}
					next := ""
					if impactPickers[id] != nil {
						next = *impactPickers[id]
					}
					rows = append(rows, id+": "+prev+" → "+next)
				}
				lines = append(lines, [2]string{"Impacts", strings.Join(rows, ", ")})
			}
			notifyStr := "no"
			if inputs.Notify {
				notifyStr = "yes"
			}
			lines = append(lines, [2]string{"Notify", notifyStr})
			return wizard.BuildSummary(lines)
		}, &inputs)

	form3 := huh.NewForm(
		huh.NewGroup(fields...),
		huh.NewGroup(
			summaryNote,
			huh.NewConfirm().
				Title("Add this update?").
				Value(&inputs.Confirmed),
		),
	).WithTheme(huh.ThemeBase())

	if err := form3.Run(); err != nil {
		return nil, wizard.HandleFormError(err)
	}

	for id, ptr := range impactPickers {
		inputs.ComponentImpacts[id] = *ptr
	}

	if !inputs.Confirmed {
		fmt.Fprintln(os.Stderr, "Aborted.")
		os.Exit(130)
	}

	return &inputs, nil
}
