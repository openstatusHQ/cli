package cmd

import (
	"strings"

	"github.com/urfave/cli/v3"
)

// Commands are grouped in the root help around the three OpenStatus pillars,
// followed by infrastructure as code and account. Anything without one of
// these categories (version, help) is listed last under "Other".
const (
	categoryMonitor     = "Monitor"
	categoryRespond     = "Respond"
	categoryCommunicate = "Communicate"
	categoryIaC         = "Infrastructure as Code"
	categoryAccount     = "Account"
)

// pillars is the order the categories appear in the root help. urfave/cli
// sorts categories by name, so the template walks them in this order instead.
var pillars = []string{categoryMonitor, categoryRespond, categoryCommunicate, categoryIaC, categoryAccount}

// withCategory sets the help category of cmds and returns them.
func withCategory(category string, cmds ...*cli.Command) []*cli.Command {
	for _, c := range cmds {
		c.Category = category
	}
	return cmds
}

// rootHelpTemplate is cli.RootCommandHelpTemplate with the COMMANDS section
// replaced by one that lists the pillars first, in order.
var rootHelpTemplate = strings.Replace(
	cli.RootCommandHelpTemplate,
	`COMMANDS:{{template "visibleCommandCategoryTemplate" .}}`,
	`COMMANDS:`+pillarCommandsTemplate(),
	1,
)

func pillarCommandsTemplate() string {
	var b strings.Builder
	for _, p := range pillars {
		b.WriteString(`{{range .VisibleCategories}}{{if eq .Name "` + p + `"}}` + categoryBlock(`{{.Name}}`) + `{{end}}{{end}}`)
	}
	b.WriteString(`{{range .VisibleCategories}}{{if not (or`)
	for _, p := range pillars {
		b.WriteString(` (eq .Name "` + p + `")`)
	}
	b.WriteString(`)}}` + categoryBlock(`{{if .Name}}{{.Name}}{{else}}Other{{end}}`) + `{{end}}{{end}}`)
	return b.String()
}

func categoryBlock(title string) string {
	return `

   ` + title + `:{{range .VisibleCommands}}
     {{join .Names ", "}}{{"\t"}}{{.Usage}}{{end}}`
}
