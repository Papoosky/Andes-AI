package logo

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// logoLines holds the braille silhouette of the Andespath company logo.
// Generated via: chafa -f symbols --symbols braille -c none --size 32x16 ap-min-logo-background-500x500.png
// Trimmed to first/last non-blank rows; centering applied at render time.
var logoLines = []string{
	"⠀⠀⠀⠀⠀⠀⠀⣤⣤⡀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀",
	"⠀⠀⠀⠀⠀⠀⣼⣿⣿⣷⡀⠀⠀⠀⠀⠀⠀⠀⠀⠀",
	"⠀⠀⠀⠀⢀⣾⣿⣿⣿⣿⣷⠀⠀⠀⢀⡀⠀⠀⠀⠀",
	"⠀⠀⠀⢀⣾⣿⣿⣿⣿⣿⣿⣷⣶⣾⣿⣿⣄⠀⠀⠀",
	"⠀⠀⢀⣾⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣆⠀⠀",
	"⠀⢀⣾⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣆⠀",
	"⢠⣿⣿⣿⣿⣿⡿⠿⠛⠋⠉⠙⠛⠻⠿⢿⣿⣿⣿⠂",
	"⠀⠙⠟⠛⠉⠁⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠈⠉⠁⠀",
}

// gradientColors defines the Andean cold palette applied top-to-bottom.
var gradientColors = []string{
	"#e0def4", // snow white  — top peak / sky
	"#c5e8ed", // light ice   — upper slopes
	"#9ccfd8", // ice blue    — upper-mid slopes
	"#9ccfd8", // ice blue    — mid slopes
	"#56a0bc", // mid blue    — lower slopes
	"#31748f", // deep blue   — base
	"#6e6a86", // slate       — foothills
	"#6e6a86", // slate       — bottom detail
}

// Render applies a top-to-bottom gradient over the braille logo rows and
// centers the logo BLOCK within the given width. The block is centered as a
// unit (uniform left pad for every row) — centering rows individually would
// distort the silhouette.
func Render(width int) string {
	maxW := 0
	for _, line := range logoLines {
		if w := lipgloss.Width(line); w > maxW {
			maxW = w
		}
	}
	leftPad := 0
	if width > maxW {
		leftPad = (width - maxW) / 2
	}
	pad := strings.Repeat(" ", leftPad)

	var sb strings.Builder
	for i, line := range logoLines {
		color := gradientColors[i%len(gradientColors)]
		rendered := lipgloss.NewStyle().
			Foreground(lipgloss.Color(color)).
			Render(line)
		sb.WriteString(pad)
		sb.WriteString(rendered)
		sb.WriteRune('\n')
	}
	return sb.String()
}
