package tui

import (
	"sort"
	"strings"

	"github.com/andespath/andes-ai/internal/theme"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ── Messages ───────────────────────────────────────────────────────────────

// installProfilesMsg is returned by the catalogProfiles command and carries
// the resolved profile list along with pre-installation state.
type installProfilesMsg struct {
	names        []string
	descs        map[string]string
	installed    []string
	catalogKnown bool
	err          error
}

// ── Entry: startInstall ────────────────────────────────────────────────────

// startInstall is called from selectOption when the user picks "install".
// It fires a tea.Cmd that calls catalogProfiles() and returns an
// installProfilesMsg. If catalogProfiles is nil, returns a no-op.
func (m Model) startInstall() (tea.Model, tea.Cmd) {
	if m.catalogProfiles == nil {
		// No callback injected — safe no-op (e.g. test models or unset).
		return m, nil
	}
	cp := m.catalogProfiles
	return m, func() tea.Msg {
		names, descs, installed, catalogKnown, err := cp()
		return installProfilesMsg{
			names:        names,
			descs:        descs,
			installed:    installed,
			catalogKnown: catalogKnown,
			err:          err,
		}
	}
}

// ── installProfilesMsg handler ─────────────────────────────────────────────

// handleInstallProfilesMsg populates the install-flow state and transitions
// to the correct screen (catalog input if location unknown, profiles list
// otherwise).
func (m Model) handleInstallProfilesMsg(msg installProfilesMsg) (tea.Model, tea.Cmd) {
	if msg.err != nil {
		m.installErr = msg.err
		// Fall back to catalog input so the user can provide a path.
		ti := textinput.New()
		ti.Placeholder = "git URL or local path"
		ti.Focus()
		m.catInput = ti
		m.screen = ScreenInstallCatalog
		return m, nil
	}

	// Sort names for stable ordering.
	names := make([]string, len(msg.names))
	copy(names, msg.names)
	sort.Strings(names)

	// Build installed set for O(1) lookup.
	installedSet := make(map[string]bool, len(msg.installed))
	for _, n := range msg.installed {
		installedSet[n] = true
	}

	checked := make(map[string]bool, len(names))
	for _, n := range names {
		checked[n] = installedSet[n]
	}

	m.profiles = names
	m.profileDesc = msg.descs
	m.profileChecked = checked
	m.profileCursor = 0

	if !msg.catalogKnown {
		ti := textinput.New()
		ti.Placeholder = "git URL or local path"
		ti.Focus()
		m.catInput = ti
		m.screen = ScreenInstallCatalog
		return m, nil
	}

	m.screen = ScreenInstallProfiles
	return m, nil
}

// ── ScreenInstallProfiles: update ─────────────────────────────────────────

func (m Model) updateInstallProfiles(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case msg.Type == tea.KeyUp || (msg.Type == tea.KeyRunes && string(msg.Runes) == "k"):
		if m.profileCursor > 0 {
			m.profileCursor--
		}

	case msg.Type == tea.KeyDown || (msg.Type == tea.KeyRunes && string(msg.Runes) == "j"):
		if m.profileCursor < len(m.profiles)-1 {
			m.profileCursor++
		}

	case msg.Type == tea.KeySpace:
		if len(m.profiles) > 0 {
			name := m.profiles[m.profileCursor]
			m.profileChecked[name] = !m.profileChecked[name]
		}

	case msg.Type == tea.KeyEnter:
		// Collect selected profiles and transition to plan screen (Task 3
		// fills the actual apply logic; here we just store the selection).
		selected := make([]string, 0, len(m.profiles))
		for _, n := range m.profiles {
			if m.profileChecked[n] {
				selected = append(selected, n)
			}
		}
		m.selectedProfiles = selected
		m.screen = ScreenInstallPlan

	case msg.Type == tea.KeyEsc:
		m.screen = ScreenMenu

	case msg.Type == tea.KeyCtrlC || (msg.Type == tea.KeyRunes && string(msg.Runes) == "q"):
		return m, tea.Quit
	}

	return m, nil
}

// ── ScreenInstallProfiles: view ────────────────────────────────────────────

func (m Model) viewInstallProfiles() string {
	bold := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(theme.ColorSnow))
	muted := lipgloss.NewStyle().Foreground(lipgloss.Color(theme.ColorSlate))
	selected := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(theme.ColorIce))

	var sb strings.Builder
	sb.WriteString(bold.Render("Install skills"))
	sb.WriteString("\n\n")

	for i, name := range m.profiles {
		checkbox := "[ ]"
		if m.profileChecked[name] {
			checkbox = "[x]"
		}
		cursor := "  "
		style := muted
		if i == m.profileCursor {
			cursor = "▸ "
			style = selected
		}
		desc := ""
		if m.profileDesc != nil {
			desc = m.profileDesc[name]
		}
		line := cursor + checkbox + " " + name
		if desc != "" {
			line += "  " + desc
		}
		sb.WriteString(style.Render(line))
		sb.WriteRune('\n')
	}

	sb.WriteRune('\n')
	sb.WriteString(muted.Render("space: toggle • enter: continue • esc: back"))

	return theme.Frame().Render(sb.String())
}

// ── ScreenInstallCatalog: update ───────────────────────────────────────────

func (m Model) updateInstallCatalog(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case msg.Type == tea.KeyEnter:
		path := strings.TrimSpace(m.catInput.Value())
		if path == "" {
			return m, nil
		}
		// Re-run catalogProfiles. Task 3 will thread the entered path through
		// the callback. For now, just re-trigger and force catalogKnown=true so
		// we advance to the profiles screen.
		if m.catalogProfiles != nil {
			cp := m.catalogProfiles
			return m, func() tea.Msg {
				names, descs, installed, _, err := cp()
				return installProfilesMsg{
					names:        names,
					descs:        descs,
					installed:    installed,
					catalogKnown: true,
					err:          err,
				}
			}
		}
		return m, nil

	case msg.Type == tea.KeyEsc:
		m.screen = ScreenMenu
		return m, nil

	case msg.Type == tea.KeyCtrlC || (msg.Type == tea.KeyRunes && string(msg.Runes) == "q"):
		return m, tea.Quit
	}

	// Let textinput handle everything else.
	var cmd tea.Cmd
	m.catInput, cmd = m.catInput.Update(msg)
	return m, cmd
}

// ── ScreenInstallCatalog: view ─────────────────────────────────────────────

func (m Model) viewInstallCatalog() string {
	bold := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(theme.ColorSnow))
	muted := lipgloss.NewStyle().Foreground(lipgloss.Color(theme.ColorSlate))

	var sb strings.Builder
	sb.WriteString(bold.Render("Install skills"))
	sb.WriteString("\n\n")
	sb.WriteString(muted.Render("Catalog path or git URL"))
	sb.WriteString("\n")

	sb.WriteString(m.catInput.View())

	if m.installErr != nil {
		errStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#eb6f92"))
		sb.WriteString("\n\n")
		sb.WriteString(errStyle.Render("Error: " + m.installErr.Error()))
	}

	sb.WriteString("\n\n")
	sb.WriteString(muted.Render("enter: confirm • esc: back"))

	return theme.Frame().Render(sb.String())
}

// ── ScreenInstallPlan: update & view ──────────────────────────────────────
// Task 3 fills the apply logic. Here we render a review placeholder.

func (m Model) updateInstallPlan(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case msg.Type == tea.KeyEsc:
		m.screen = ScreenInstallProfiles

	case msg.Type == tea.KeyCtrlC || (msg.Type == tea.KeyRunes && string(msg.Runes) == "q"):
		return m, tea.Quit
	}
	return m, nil
}

func (m Model) viewInstallPlan() string {
	bold := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(theme.ColorSnow))
	muted := lipgloss.NewStyle().Foreground(lipgloss.Color(theme.ColorSlate))

	var sb strings.Builder
	sb.WriteString(bold.Render("Install plan"))
	sb.WriteString("\n\n")

	if len(m.selectedProfiles) == 0 {
		sb.WriteString(muted.Render("No profiles selected."))
	} else {
		sb.WriteString(muted.Render("Selected profiles:"))
		sb.WriteString("\n")
		for _, p := range m.selectedProfiles {
			sb.WriteString(muted.Render("  • " + p))
			sb.WriteRune('\n')
		}
	}

	sb.WriteString("\n")
	sb.WriteString(muted.Render("(apply coming in Task 3)"))
	sb.WriteString("\n\n")
	sb.WriteString(muted.Render("esc: back • q: quit"))

	return theme.Frame().Render(sb.String())
}
