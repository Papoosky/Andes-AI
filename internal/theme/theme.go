// Package theme defines the shared Andean cold palette and frame style used
// across all andes UI surfaces (static banner, interactive TUI screens).
// No package may redefine these constants — import theme and use them.
package theme

import "github.com/charmbracelet/lipgloss"

// Andean cold palette.
const (
	ColorSnow     = "#e0def4" // bold titles, headings
	ColorIce      = "#9ccfd8" // command names (highlight)
	ColorDeepBlue = "#31748f" // borders
	ColorSlate    = "#6e6a86" // muted text, footer, descriptions
	ColorRose     = "#eb6f92" // errors, destructive actions
)

// Frame returns the shared double-border box style used by banner and TUI
// screens. Callers wrap their inner content string with Frame().Render(inner).
func Frame() lipgloss.Style {
	return lipgloss.NewStyle().
		Border(lipgloss.DoubleBorder()).
		BorderForeground(lipgloss.Color(ColorDeepBlue)).
		Padding(0, 2)
}
