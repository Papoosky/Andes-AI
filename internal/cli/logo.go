package cli

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// logoLines holds the braille mountain-range silhouette.
// Generated from a 70×20 bitmap (35 braille chars wide × 5 rows tall).
// Peaks: one tall off-center-left, two smaller peaks right; sun dot upper-right.
var logoLines = []string{
	"⠀⠀⠀⠀⠀⠀⠀⠀⠀⢀⣄⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠈⠀⠀⠀⠀⠀⠀⠀⠀",
	"⠀⠀⠀⠀⠀⠀⠀⢀⣴⣿⣿⣷⣄⠀⠀⠀⠀⠀⠀⠀⣠⡀⠀⠀⠀⢀⣄⠀⠀⠀⠀⠀⠀⠀⠀",
	"⠀⠀⠀⠀⠀⢀⣴⣿⣿⣿⣿⣿⣿⣷⣄⣀⣀⣀⣠⣾⣿⣿⣦⣀⣴⣿⣿⣷⣄⣀⠀⠀⠀⠀⠀",
	"⠀⠀⠀⢀⣴⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⠀⠀⠀⠀⠀",
	"⠀⢀⣴⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⠀⠀⠀⠀⠀",
}

// gradientColors defines the Andean cold palette applied top-to-bottom.
// Band assignment: row 0 = snow-white, rows 1-2 = ice, rows 3 = deep-blue, row 4 = slate.
var gradientColors = []string{
	"#e0def4", // snow white  — top peak / sky
	"#9ccfd8", // ice blue    — upper slopes
	"#9ccfd8", // ice blue    — mid slopes
	"#31748f", // deep blue   — lower slopes
	"#6e6a86", // slate       — base / foothills
}

// RenderLogo applies a top-to-bottom gradient over the braille logo rows.
func RenderLogo() string {
	var sb strings.Builder
	for i, line := range logoLines {
		color := gradientColors[i%len(gradientColors)]
		rendered := lipgloss.NewStyle().
			Foreground(lipgloss.Color(color)).
			Render(line)
		sb.WriteString(rendered)
		sb.WriteRune('\n')
	}
	return sb.String()
}
