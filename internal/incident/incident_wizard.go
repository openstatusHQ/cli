package incident

import (
	"github.com/charmbracelet/huh"

	"github.com/openstatusHQ/cli/internal/wizard"
)

func runDeclareWizard(prefilled DeclareParams) (DeclareParams, error) {
	inputs := prefilled

	var fields []huh.Field

	if inputs.Title == "" {
		fields = append(fields, huh.NewInput().
			Title("Title").
			Description("What is broken, e.g. Checkout API returns 502").
			Validate(validateTitle).
			Value(&inputs.Title))
	}

	if inputs.Severity == "" {
		fields = append(fields, huh.NewSelect[string]().
			Title("Severity").
			Options(
				huh.NewOption("critical", "critical"),
				huh.NewOption("major", "major"),
				huh.NewOption("minor", "minor"),
			).
			Value(&inputs.Severity))
	}

	if inputs.Summary == "" {
		fields = append(fields, huh.NewText().
			Title("Summary").
			Description("Short human summary (optional)").
			Validate(validateSummary).
			Value(&inputs.Summary))
	}

	if inputs.Commander == "" {
		fields = append(fields, huh.NewInput().
			Title("Commander").
			Description("Email of the member leading the response (optional)").
			Validate(func(s string) error {
				if s == "" {
					return nil
				}
				return validateEmail("commander", s)
			}).
			Value(&inputs.Commander))
	}

	form := huh.NewForm(huh.NewGroup(fields...)).WithTheme(huh.ThemeBase())
	if err := form.Run(); err != nil {
		return inputs, wizard.HandleFormError(err)
	}

	return inputs, nil
}
