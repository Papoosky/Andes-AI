package tui

// logo.go holds the braille mountain silhouette and gradient renderer.
// This is intentionally a local copy of the same data in internal/cli/logo.go
// so that internal/tui does NOT import internal/cli — keeping the import
// graph acyclic: cli → tui is allowed; tui → cli is not.

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var logoLines = []string{
	"⠀⠀⠀⠀⠀⠀⠀⠀⢀⣄",
	"⠀⠀⠀⠀⠀⠀⢀⣴⣿⣿⣷⣄⠀⠀⠀⠀⠀⠀⠀⣠⡀⠀⠀⠀⢀⣄",
	"⠀⠀⠀⠀⢀⣴⣿⣿⣿⣿⣿⣿⣷⣄⣀⣀⣀⣠⣾⣿⣿⣦⣀⣴⣿⣿⣷⣄⣀",
	"⠀⠀⢀⣴⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿",
	"⢀⣴⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿",
}

var gradientColors = []string{
	"#e0def4", // snow white  — top peak
	"#9ccfd8", // ice blue    — upper slopes
	"#9ccfd8", // ice blue    — mid slopes
	"#31748f", // deep blue   — lower slopes
	"#6e6a86", // slate       — base
}

// renderLogo applies a top-to-bottom gradient over the braille logo and
// centers the block within the given width.
func renderLogo(width int) string {
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
