package cli

import (
	"strings"

	"github.com/andespath/andes-ai/internal/logo"
	"github.com/andespath/andes-ai/internal/theme"
	"github.com/charmbracelet/lipgloss"
)

// renderBanner assembles the full andes welcome banner and wraps it in a
// double-border box styled with the Andean cold palette.
func renderBanner() string {
	bold := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(theme.ColorSnow))
	muted := lipgloss.NewStyle().Foreground(lipgloss.Color(theme.ColorSlate))
	cmdName := lipgloss.NewStyle().Foreground(lipgloss.Color(theme.ColorIce))
	heading := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(theme.ColorSnow))

	// Command list — name in highlight color, description muted
	type entry struct{ name, desc string }
	commands := []entry{
		{"init", "install skills from the catalog"},
		{"list", "show catalog and install status"},
		{"doctor", "diagnose drift between manifest and disk"},
	}

	// Inner width = widest text line; the logo block is centered against it.
	title := "andes — andespath skills, one command"
	footer := "run andes <command> --help for details"
	innerWidth := lipgloss.Width(title)
	if w := lipgloss.Width(footer); w > innerWidth {
		innerWidth = w
	}
	for _, c := range commands {
		if w := 2 + 10 + lipgloss.Width(c.desc); w > innerWidth {
			innerWidth = w
		}
	}

	var sb strings.Builder

	// Logo (gradient applied inside logo.Render), centered, blank line below
	sb.WriteString(logo.Render(innerWidth))
	sb.WriteRune('\n')

	// Title line
	sb.WriteString(bold.Render("andes"))
	sb.WriteString(muted.Render(" — andespath skills, one command"))
	sb.WriteRune('\n')
	sb.WriteRune('\n')

	// Commands heading
	sb.WriteString(heading.Render("Commands"))
	sb.WriteRune('\n')

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

	return theme.Frame().Render(sb.String())
}
