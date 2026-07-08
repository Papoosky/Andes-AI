package cli

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// logoLines holds the braille mountain-range silhouette.
// Generated from a 70×20 bitmap (braille: 2×4 dots per char).
// Peaks: one tall off-center-left, two smaller peaks right.
// Lines are stored trimmed (no trailing blank braille); centering is
// applied at render time against the banner's inner width.
var logoLines = []string{
	"⠀⠀⠀⠀⠀⠀⠀⠀⢀⣄",
	"⠀⠀⠀⠀⠀⠀⢀⣴⣿⣿⣷⣄⠀⠀⠀⠀⠀⠀⠀⣠⡀⠀⠀⠀⢀⣄",
	"⠀⠀⠀⠀⢀⣴⣿⣿⣿⣿⣿⣿⣷⣄⣀⣀⣀⣠⣾⣿⣿⣦⣀⣴⣿⣿⣷⣄⣀",
	"⠀⠀⢀⣴⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿",
	"⢀⣴⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿",
}

// gradientColors defines the Andean cold palette applied top-to-bottom.
// Band assignment: row 0 = snow-white, rows 1-2 = ice, row 3 = deep-blue, row 4 = slate.
var gradientColors = []string{
	"#e0def4", // snow white  — top peak / sky
	"#9ccfd8", // ice blue    — upper slopes
	"#9ccfd8", // ice blue    — mid slopes
	"#31748f", // deep blue   — lower slopes
	"#6e6a86", // slate       — base / foothills
}

// RenderLogo applies a top-to-bottom gradient over the braille logo rows and
// centers the logo BLOCK within the given width. The block is centered as a
// unit (uniform left pad for every row) — centering rows individually would
// distort the silhouette.
func RenderLogo(width int) string {
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
