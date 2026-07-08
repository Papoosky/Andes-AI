package tui

import (
	"fmt"
	"sort"
	"strings"

	"github.com/andespath/andes-ai/internal/theme"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// enterCatalogScreen sets up the catalog text input and returns the model plus
// a blink command so the cursor starts blinking immediately.
// Preserves any previously typed value across error round-trips (Minor 8).
func enterCatalogScreen(m Model) (Model, tea.Cmd) {
	prev := m.catInput.Value()
	ti := textinput.New()
	ti.Placeholder = "git URL or local path"
	if prev != "" {
		ti.SetValue(prev)
	}
	ti.Focus()
	m.catInput = ti
	m.screen = ScreenInstallCatalog
	return m, textinput.Blink
}

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

// installDoneMsg carries the result of an in-process install run. It is
// dispatched by the tea.Cmd returned from updateInstallPlan's enter handler.
type installDoneMsg struct {
	summary string
	err     error
}

// planDoneMsg carries the result of a plan preview computation.
type planDoneMsg struct {
	install   int
	update    int
	unchanged int
	err       error
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
		names, descs, installed, catalogKnown, err := cp("")
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
		return enterCatalogScreen(m)
	}
	// Clear any previous error on successful load (Minor 7).
	m.installErr = nil

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
		return enterCatalogScreen(m)
	}

	m.screen = ScreenInstallProfiles
	return m, nil
}

// ── installDoneMsg handler ─────────────────────────────────────────────────

// handleInstallDoneMsg routes the apply result to the output screen.
func (m Model) handleInstallDoneMsg(msg installDoneMsg) (tea.Model, tea.Cmd) {
	out := msg.summary
	if msg.err != nil {
		if out != "" {
			out += "\n"
		}
		out += msg.err.Error()
	}
	m.fitViewport(out)
	m.vp.SetContent(out)
	m.vp.GotoTop()
	m.cmdTitle = "install"
	m.screen = ScreenOutput
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
		// Collect selected profiles and transition to plan screen.
		selected := make([]string, 0, len(m.profiles))
		for _, n := range m.profiles {
			if m.profileChecked[n] {
				selected = append(selected, n)
			}
		}
		m.selectedProfiles = selected
		m.screen = ScreenInstallPlan
		// Dispatch plan preview (Important 5).
		if m.planInstall != nil && len(selected) > 0 {
			pf := m.planInstall
			override := m.catalogOverride
			profiles := selected
			return m, func() tea.Msg {
				inst, upd, unch, err := pf(override, profiles)
				return planDoneMsg{install: inst, update: upd, unchanged: unch, err: err}
			}
		}

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
		// Store the override so applyInstall can use it (Critical 1).
		m.catalogOverride = path
		// Re-run catalogProfiles with the typed path so the real catalog is
		// loaded. The result drives advancement: if the path resolves to a valid
		// catalog, catalogKnown=true and real profiles are returned → the
		// installProfilesMsg handler advances to ScreenInstallProfiles. If it
		// errors, the error is surfaced on the catalog screen.
		if m.catalogProfiles != nil {
			cp := m.catalogProfiles
			override := path
			return m, func() tea.Msg {
				names, descs, installed, catalogKnown, err := cp(override)
				return installProfilesMsg{
					names:        names,
					descs:        descs,
					installed:    installed,
					catalogKnown: catalogKnown,
					err:          err,
				}
			}
		}
		return m, nil

	case msg.Type == tea.KeyCtrlC:
		// Only CtrlC quits from the catalog screen (Critical 2: 'q' must be typeable in paths).
		return m, tea.Quit

	case msg.Type == tea.KeyEsc:
		// Esc cancels back to menu; clear error (Minor 7).
		m.installErr = nil
		m.screen = ScreenMenu
		return m, nil
	}

	// ALL other keys (including runes like 'q') go to the text input (Critical 2).
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
		errStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(theme.ColorRose))
		sb.WriteString("\n\n")
		sb.WriteString(errStyle.Render("Error: " + m.installErr.Error()))
	}

	sb.WriteString("\n\n")
	sb.WriteString(muted.Render("enter: confirm • esc: back"))

	return theme.Frame().Render(sb.String())
}

// ── ScreenInstallPlan: update & view ──────────────────────────────────────

func (m Model) updateInstallPlan(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case msg.Type == tea.KeyEnter:
		if m.applyInstall == nil {
			return m, nil
		}
		// No profiles selected — disable apply (Minor 9).
		if len(m.selectedProfiles) == 0 {
			return m, nil
		}
		// In-flight guard: ignore additional enters while applying (Important 4).
		if m.installing {
			return m, nil
		}
		m.installing = true
		apply := m.applyInstall
		override := m.catalogOverride
		profiles := m.selectedProfiles
		return m, func() tea.Msg {
			summary, err := apply(override, profiles)
			return installDoneMsg{summary: summary, err: err}
		}

	case msg.Type == tea.KeyEsc:
		m.screen = ScreenMenu

	case msg.Type == tea.KeyCtrlC || (msg.Type == tea.KeyRunes && string(msg.Runes) == "q"):
		return m, tea.Quit
	}
	return m, nil
}

func (m Model) viewInstallPlan() string {
	bold := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(theme.ColorSnow))
	muted := lipgloss.NewStyle().Foreground(lipgloss.Color(theme.ColorSlate))

	var sb strings.Builder
	sb.WriteString(bold.Render("Review"))
	sb.WriteString("\n\n")

	if len(m.selectedProfiles) == 0 {
		sb.WriteString(muted.Render("No profiles selected."))
		sb.WriteString("\n\n")
		sb.WriteString(muted.Render("esc: back"))
		return theme.Frame().Render(sb.String())
	}

	sb.WriteString(muted.Render("Selected profiles:"))
	sb.WriteString("\n")
	for _, p := range m.selectedProfiles {
		sb.WriteString(muted.Render("  • " + p))
		sb.WriteRune('\n')
	}

	// Show plan counts if available (Important 5).
	if m.planInstallCount+m.planUpdateCount+m.planUnchangedCnt > 0 || m.planInstallCount == 0 {
		sb.WriteString("\n")
		sb.WriteString(muted.Render(fmt.Sprintf("%d to install, %d to update, %d unchanged",
			m.planInstallCount, m.planUpdateCount, m.planUnchangedCnt)))
	}

	sb.WriteString("\n\n")
	if m.installing {
		sb.WriteString(bold.Render("Applying…"))
	} else {
		sb.WriteString(muted.Render("enter: apply • esc: back"))
	}

	return theme.Frame().Render(sb.String())
}
