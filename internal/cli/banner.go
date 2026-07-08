package cli

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// Palette — consistent with logo gradient.
const (
	colorSnow     = "#e0def4" // bold title, headings
	colorIce      = "#9ccfd8" // command names (highlight)
	colorDeepBlue = "#31748f" // borders
	colorSlate    = "#6e6a86" // muted text, footer, descriptions
)

// renderBanner assembles the full andes welcome banner and wraps it in a
// double-border box styled with the Andean cold palette.
func renderBanner() string {
	bold := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(colorSnow))
	muted := lipgloss.NewStyle().Foreground(lipgloss.Color(colorSlate))
	cmdName := lipgloss.NewStyle().Foreground(lipgloss.Color(colorIce))
	heading := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(colorSnow))

	var sb strings.Builder

	// Logo (gradient applied inside RenderLogo)
	sb.WriteString(RenderLogo())

	// Title line
	sb.WriteString(bold.Render("andes"))
	sb.WriteString(muted.Render(" — andespath skills, one command"))
	sb.WriteRune('\n')
	sb.WriteRune('\n')

	// Commands heading
	sb.WriteString(heading.Render("Commands"))
	sb.WriteRune('\n')

	// Command list — name in highlight color, description muted
	type entry struct{ name, desc string }
	commands := []entry{
		{"init", "install skills from the catalog"},
		{"list", "show catalog and install status"},
		{"doctor", "diagnose drift between manifest and disk"},
	}
	for _, c := range commands {
		sb.WriteString("  ")
		sb.WriteString(cmdName.Render(c.name))
		padding := strings.Repeat(" ", 10-len(c.name))
		sb.WriteString(padding)
		sb.WriteString(muted.Render(c.desc))
		sb.WriteRune('\n')
	}

	sb.WriteRune('\n')

	// Footer hint
	sb.WriteString(muted.Render("run andes <command> --help for details"))
	sb.WriteRune('\n')

	// Wrap in a double-border box
	box := lipgloss.NewStyle().
		Border(lipgloss.DoubleBorder()).
		BorderForeground(lipgloss.Color(colorDeepBlue)).
		Padding(0, 2)

	return box.Render(sb.String())
}
