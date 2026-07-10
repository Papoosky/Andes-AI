// Package tui implements the interactive Bubbletea TUI for the andes CLI.
// It provides a two-screen experience:
//   - ScreenMenu: braille logo + navigable command list
//   - ScreenOutput: scrollable output from a chosen command
package tui

import (
	"bytes"
	"strings"

	"github.com/andespath/andes-ai/internal/logo"
	"github.com/andespath/andes-ai/internal/theme"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
)

// ── Screens ────────────────────────────────────────────────────────────────

type Screen int

const (
	ScreenMenu Screen = iota
	ScreenOutput
	ScreenInstallCatalog
	ScreenInstallProfiles
	ScreenInstallPlan
)

// ── Menu options ───────────────────────────────────────────────────────────

type menuOption struct {
	id    string
	label string
	desc  string
}

func defaultOptions() []menuOption {
	return []menuOption{
		{id: "install", label: "install", desc: "install skills from the catalog"},
		{id: "list", label: "list", desc: "show catalog and install status"},
		{id: "doctor", label: "doctor", desc: "diagnose drift"},
		{id: "quit", label: "quit", desc: "exit andes"},
	}
}

// ── Messages ───────────────────────────────────────────────────────────────

// cmdResultMsg carries the captured output of a sub-command run in-process.
type cmdResultMsg struct {
	cmdID  string
	output string
	err    error
}

// FreshnessMsg reports the async catalog freshness check result.
type FreshnessMsg struct {
	Outdated bool
	Offline  bool
}

// UpdateCheck is injected by the caller (cli) so tui stays decoupled from
// manifest/git specifics and tests can fake it.
type UpdateCheck func() FreshnessMsg

// ── Model ──────────────────────────────────────────────────────────────────

// CatalogProfilesFunc is injected from cli so tui stays decoupled from
// manifest/git specifics. It resolves the catalog source, loads it, and
// returns sorted profile names, their descriptions, the currently-installed
// profile set, and whether the catalog location is already known (so the
// flow can skip the catalog-input screen).
type CatalogProfilesFunc func(catalogOverride string) (names []string, descs map[string]string, installed []string, catalogKnown bool, err error)

// ApplyInstallFunc is injected from cli. It resolves the catalog source
// (using catalogOverride if non-empty, otherwise manifest/default), plans and
// applies the install in-process (no confirmation prompts — the TUI plan
// screen already confirmed), saves the manifest, and returns a human-readable
// summary ("✓ N skills up to date" or "Everything is already up to date").
// tui never imports cli; the func value is injected.
type ApplyInstallFunc func(catalogOverride string, profiles []string) (summary string, err error)

// PlanItem is one planned skill action shown on the Review screen. It mirrors
// installer.Action but lives in tui so tui stays decoupled from installer.
type PlanItem struct {
	SkillID string
	Action  string // "install" | "update" | "unchanged" | "remove"
	Profile string
}

// PlanInstallFunc is injected from cli. It resolves the catalog source and
// runs installer.Plan without applying — used to preview the per-skill plan
// shown on the Review screen before the user confirms. Counts are derived from
// the returned items.
type PlanInstallFunc func(catalogOverride string, profiles []string) ([]PlanItem, error)

// Model holds all TUI state. newRoot is a factory used to build a fresh
// cobra command for in-process execution — this breaks the cli→tui import
// cycle because tui never imports cli's package-level symbols; the factory
// is injected from outside.
type Model struct {
	screen   Screen
	cursor   int
	options  []menuOption
	newRoot  func() *cobra.Command
	check    UpdateCheck
	vp       viewport.Model
	cmdTitle string
	width    int
	height   int
	outdated bool
	offline  bool

	// install-flow fields
	catalogProfiles  CatalogProfilesFunc
	planInstall      PlanInstallFunc
	applyInstall     ApplyInstallFunc
	catInput         textinput.Model
	profiles         []string
	profileDesc      map[string]string
	profileChecked   map[string]bool
	profileCursor    int
	selectedProfiles []string
	catalogOverride  string
	installing       bool
	planItems        []PlanItem
	installErr       error
}

// New builds a Model ready to run.
// catalogProfiles, planInstall, and applyInstall are optional injected
// callbacks; pass nil in tests that inject messages directly.
func New(newRoot func() *cobra.Command, check UpdateCheck, catalogProfiles CatalogProfilesFunc, planInstall PlanInstallFunc, applyInstall ApplyInstallFunc) Model {
	vp := viewport.New(80, 20)
	return Model{
		screen:          ScreenMenu,
		cursor:          0,
		options:         defaultOptions(),
		newRoot:         newRoot,
		check:           check,
		vp:              vp,
		width:           80,
		height:          24,
		catalogProfiles: catalogProfiles,
		planInstall:     planInstall,
		applyInstall:    applyInstall,
	}
}

// Init satisfies tea.Model; fires the async freshness check if provided.
func (m Model) Init() tea.Cmd {
	if m.check == nil {
		return nil
	}
	check := m.check
	return func() tea.Msg { return check() }
}

// ── Update ─────────────────────────────────────────────────────────────────

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		vpWidth := msg.Width - 6
		if vpWidth > 100 {
			vpWidth = 100
		}
		if vpWidth < 20 {
			vpWidth = 20
		}
		vpHeight := msg.Height - 6
		if vpHeight < 3 {
			vpHeight = 3
		}
		m.vp.Width = vpWidth
		m.vp.Height = vpHeight
		return m, nil

	case FreshnessMsg:
		m.outdated = msg.Outdated
		m.offline = msg.Offline
		return m, nil

	case cmdResultMsg:
		// Async result arrived — switch to output screen.
		if msg.cmdID == "update" {
			m.outdated = false
		}
		out := msg.output
		if msg.err != nil {
			out = strings.TrimRight(out, "\n") + "\n" + msg.err.Error()
		}
		m.fitViewport(out)
		m.vp.SetContent(out)
		m.vp.GotoTop()
		m.cmdTitle = msg.cmdID
		m.screen = ScreenOutput
		return m, nil

	case installProfilesMsg:
		return m.handleInstallProfilesMsg(msg)

	case planDoneMsg:
		if msg.err == nil {
			m.planItems = msg.items
		}
		return m, nil

	case installDoneMsg:
		m.installing = false
		return m.handleInstallDoneMsg(msg)

	case tea.KeyMsg:
		switch m.screen {
		case ScreenMenu:
			return m.updateMenu(msg)
		case ScreenOutput:
			return m.updateOutput(msg)
		case ScreenInstallCatalog:
			return m.updateInstallCatalog(msg)
		case ScreenInstallProfiles:
			return m.updateInstallProfiles(msg)
		case ScreenInstallPlan:
			return m.updateInstallPlan(msg)
		}

	default:
		// Forward non-key messages to the catalog text input so the cursor
		// blinks and window-size changes are handled correctly.
		if m.screen == ScreenInstallCatalog {
			var cmd tea.Cmd
			m.catInput, cmd = m.catInput.Update(msg)
			return m, cmd
		}
	}
	return m, nil
}

func (m Model) updateMenu(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case msg.Type == tea.KeyUp || (msg.Type == tea.KeyRunes && string(msg.Runes) == "k"):
		if m.cursor > 0 {
			m.cursor--
		}
		return m, nil

	case msg.Type == tea.KeyDown || (msg.Type == tea.KeyRunes && string(msg.Runes) == "j"):
		if m.cursor < len(m.options)-1 {
			m.cursor++
		}
		return m, nil

	case msg.Type == tea.KeyEnter:
		return m.selectOption()

	case msg.Type == tea.KeyCtrlC || (msg.Type == tea.KeyRunes && string(msg.Runes) == "q"):
		return m, tea.Quit

	case msg.Type == tea.KeyRunes && string(msg.Runes) == "u":
		if !m.outdated {
			return m, nil
		}
		return m.runInProcess("update", "--yes")
	}
	return m, nil
}

func (m Model) updateOutput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case msg.Type == tea.KeyEsc:
		m.screen = ScreenMenu
		return m, nil
	case msg.Type == tea.KeyCtrlC || (msg.Type == tea.KeyRunes && string(msg.Runes) == "q"):
		return m, tea.Quit
	}
	// Let the viewport handle scroll keys.
	var cmd tea.Cmd
	m.vp, cmd = m.vp.Update(msg)
	return m, cmd
}

// fitViewport sizes the viewport to the content, hugging small output and
// capping large output so the bordered box never exceeds the terminal.
// It must be called before vp.SetContent so the viewport renders at the
// right size from the first frame.
func (m *Model) fitViewport(content string) {
	lines := strings.Split(content, "\n")
	widest := 0
	for _, ln := range lines {
		if w := lipgloss.Width(ln); w > widest {
			widest = w
		}
	}
	// Width: hug the widest line, capped by terminal (minus frame overhead 6)
	// and an absolute max; floored so tiny output isn't cramped.
	// Guard: if m.width == 0 (before first WindowSizeMsg, e.g. unit tests),
	// use a fallback of 80 so tests still get a sane box.
	baseWidth := m.width
	if baseWidth == 0 {
		baseWidth = 80
	}
	maxW := baseWidth - 6
	if maxW > 100 {
		maxW = 100
	}
	w := widest
	if w > maxW {
		w = maxW
	}
	if w < 24 {
		w = 24
	}
	m.vp.Width = w
	// Height: hug line count, capped by terminal (minus header/footer/frame ~6).
	maxH := m.height - 6
	if maxH < 3 {
		maxH = 3
	}
	h := len(lines)
	if h > maxH {
		h = maxH
	}
	if h < 1 {
		h = 1
	}
	m.vp.Height = h
}

// runInProcess executes a subcommand with captured output, async.
func (m Model) runInProcess(args ...string) (tea.Model, tea.Cmd) {
	if m.newRoot == nil {
		return m, nil // defensive: no-op when the command factory is not provided (test models)
	}
	newRoot := m.newRoot
	cmdID := args[0]
	return m, func() tea.Msg {
		var buf bytes.Buffer
		root := newRoot()
		root.SetArgs(args)
		root.SetOut(&buf)
		root.SetErr(&buf)
		execErr := root.Execute()
		output := buf.String()
		if execErr != nil {
			if output != "" && !strings.HasSuffix(output, "\n") {
				output += "\n"
			}
			output += execErr.Error()
		}
		return cmdResultMsg{cmdID: cmdID, output: output, err: nil}
	}
}

// selectOption handles Enter on the menu.
func (m Model) selectOption() (tea.Model, tea.Cmd) {
	opt := m.options[m.cursor]

	switch opt.id {
	case "quit":
		return m, tea.Quit

	case "install":
		return m.startInstall()

	case "list", "doctor":
		return m.runInProcess(opt.id)
	}

	return m, nil
}

// ── View ───────────────────────────────────────────────────────────────────

func (m Model) View() string {
	switch m.screen {
	case ScreenMenu:
		return m.viewMenu()
	case ScreenOutput:
		return m.viewOutput()
	case ScreenInstallCatalog:
		return m.viewInstallCatalog()
	case ScreenInstallProfiles:
		return m.viewInstallProfiles()
	case ScreenInstallPlan:
		return m.viewInstallPlan()
	}
	return ""
}

func (m Model) viewMenu() string {
	bold := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(theme.ColorSnow))
	muted := lipgloss.NewStyle().Foreground(lipgloss.Color(theme.ColorSlate))
	selected := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(theme.ColorIce))
	warn := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#f6c177"))

	// Build the text block first (title, optional banner, menu, footer),
	// left-aligned. The logo is then centered over this block's width so the
	// whole thing reads as a single column: logo on top, text below.
	var body strings.Builder

	body.WriteString(bold.Render("andes"))
	body.WriteString(muted.Render(" — andespath skills, one command"))
	body.WriteString("\n\n")

	if m.outdated {
		body.WriteString(warn.Render("⚠ catalog updated — press u to update"))
		body.WriteString("\n\n")
	}

	for i, opt := range m.options {
		if i == m.cursor {
			body.WriteString(selected.Render("▸ " + opt.label))
			body.WriteString("  ")
			body.WriteString(muted.Render(opt.desc))
		} else {
			body.WriteString(muted.Render("  " + opt.label))
			body.WriteString("  ")
			body.WriteString(muted.Render(opt.desc))
		}
		body.WriteRune('\n')
	}

	body.WriteRune('\n')
	footer := "↑/k ↓/j: navigate • enter: select • q: quit"
	if m.offline {
		footer += " • offline"
	}
	body.WriteString(muted.Render(footer))

	bodyText := body.String()
	bodyWidth := lipgloss.Width(bodyText) // widest line of the text block

	var sb strings.Builder
	// Logo centered over the text block width — no floating to the side.
	sb.WriteString(logo.Render(bodyWidth))
	sb.WriteString(bodyText)
	sb.WriteRune('\n')

	return theme.Frame().Render(sb.String())
}

func (m Model) viewOutput() string {
	bold := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(theme.ColorSnow))
	muted := lipgloss.NewStyle().Foreground(lipgloss.Color(theme.ColorSlate))

	var sb strings.Builder

	// Header — matches menu title style: bold snow "andes <cmd>".
	sb.WriteString(bold.Render("andes " + m.cmdTitle))
	sb.WriteString("\n\n")

	// Scrollable content.
	sb.WriteString(m.vp.View())
	sb.WriteRune('\n')

	// Footer — muted, consistent with menu footer.
	sb.WriteString(muted.Render("esc: back • q: quit"))
	sb.WriteRune('\n')

	return theme.Frame().Render(sb.String())
}

// ── Entry point ────────────────────────────────────────────────────────────

// Run starts the Bubbletea program. newRoot is a factory for a fresh
// cobra root command, used for in-process command execution.
// check is an optional async freshness probe; pass nil to skip it.
// catalogProfiles, planInstall, and applyInstall are optional injected
// callbacks for the install flow. cli imports tui, so tui MUST NOT import cli
// at the package level — all dependencies are injected.
// Both cli and tui import internal/logo (leaf package); neither imports the other.
func Run(newRoot func() *cobra.Command, check UpdateCheck, catalogProfiles CatalogProfilesFunc, planInstall PlanInstallFunc, applyInstall ApplyInstallFunc) error {
	p := tea.NewProgram(New(newRoot, check, catalogProfiles, planInstall, applyInstall), tea.WithAltScreen())
	_, err := p.Run()
	return err
}
