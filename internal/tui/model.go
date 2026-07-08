// Package tui implements the interactive Bubbletea TUI for the andes CLI.
// It provides a two-screen experience:
//   - ScreenMenu: braille logo + navigable command list
//   - ScreenOutput: scrollable output from a chosen command
package tui

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
)

// ── Palette (mirrors banner.go) ────────────────────────────────────────────

const (
	colorSnow     = "#e0def4"
	colorIce      = "#9ccfd8"
	colorDeepBlue = "#31748f"
	colorSlate    = "#6e6a86"
)

// ── Screens ────────────────────────────────────────────────────────────────

type Screen int

const (
	ScreenMenu Screen = iota
	ScreenOutput
)

// ── Menu options ───────────────────────────────────────────────────────────

type menuOption struct {
	id    string
	label string
	desc  string
}

func defaultOptions() []menuOption {
	return []menuOption{
		{id: "init", label: "init", desc: "install skills from the catalog"},
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

// ── Model ──────────────────────────────────────────────────────────────────

// Model holds all TUI state. newRoot is a factory used to build a fresh
// cobra command for in-process execution — this breaks the cli→tui import
// cycle because tui never imports cli's package-level symbols; the factory
// is injected from outside.
type Model struct {
	screen   Screen
	cursor   int
	options  []menuOption
	newRoot  func() *cobra.Command
	vp       viewport.Model
	cmdTitle string
	width    int
	height   int
}

// New builds a Model ready to run.
func New(newRoot func() *cobra.Command) Model {
	vp := viewport.New(80, 20)
	return Model{
		screen:  ScreenMenu,
		cursor:  0,
		options: defaultOptions(),
		newRoot: newRoot,
		vp:      vp,
		width:   80,
		height:  24,
	}
}

// Init satisfies tea.Model; no I/O at startup.
func (m Model) Init() tea.Cmd {
	return nil
}

// ── Update ─────────────────────────────────────────────────────────────────

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.vp.Width = msg.Width
		m.vp.Height = msg.Height - 4 // leave room for header/footer
		return m, nil

	case cmdResultMsg:
		// Async result arrived — switch to output screen.
		out := msg.output
		if msg.err != nil {
			out = strings.TrimRight(out, "\n") + "\n" + msg.err.Error()
		}
		m.vp.SetContent(out)
		m.vp.GotoTop()
		m.cmdTitle = msg.cmdID
		m.screen = ScreenOutput
		return m, nil

	case tea.KeyMsg:
		switch m.screen {
		case ScreenMenu:
			return m.updateMenu(msg)
		case ScreenOutput:
			return m.updateOutput(msg)
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

// selectOption handles Enter on the menu.
func (m Model) selectOption() (tea.Model, tea.Cmd) {
	opt := m.options[m.cursor]

	switch opt.id {
	case "quit":
		return m, tea.Quit

	case "init":
		// Interactive — suspend TUI and hand the terminal to the subprocess.
		exe, err := os.Executable()
		if err != nil {
			// Fall back to showing the error.
			result := cmdResultMsg{cmdID: "init", output: "", err: fmt.Errorf("cannot resolve executable: %w", err)}
			return m, func() tea.Msg { return result }
		}
		return m, tea.ExecProcess(
			exec.Command(exe, "init"),
			func(err error) tea.Msg {
				// After init returns, go back to menu (no output screen needed).
				return tea.KeyMsg{Type: tea.KeyEsc}
			},
		)

	case "list", "doctor":
		// Run in-process, async.
		newRoot := m.newRoot
		cmdID := opt.id
		return m, func() tea.Msg {
			var buf bytes.Buffer
			root := newRoot()
			root.SetArgs([]string{cmdID})
			root.SetOut(&buf)
			root.SetErr(&buf)
			execErr := root.Execute()
			output := buf.String()
			if execErr != nil {
				// Append the error string so it appears in the output pane.
				if output != "" && !strings.HasSuffix(output, "\n") {
					output += "\n"
				}
				output += execErr.Error()
			}
			return cmdResultMsg{cmdID: cmdID, output: output, err: nil}
		}
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
	}
	return ""
}

func (m Model) viewMenu() string {
	bold := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(colorSnow))
	muted := lipgloss.NewStyle().Foreground(lipgloss.Color(colorSlate))
	selected := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(colorIce))

	var sb strings.Builder

	// Logo rendered inline using the shared logo data in this package.
	sb.WriteString(renderLogo(40))

	// Title.
	sb.WriteString(bold.Render("andes"))
	sb.WriteString(muted.Render(" — andespath skills, one command"))
	sb.WriteString("\n\n")

	// Menu items.
	for i, opt := range m.options {
		if i == m.cursor {
			sb.WriteString(selected.Render("▸ " + opt.label))
			sb.WriteString("  ")
			sb.WriteString(muted.Render(opt.desc))
		} else {
			sb.WriteString(muted.Render("  " + opt.label))
			sb.WriteString("  ")
			sb.WriteString(muted.Render(opt.desc))
		}
		sb.WriteRune('\n')
	}

	sb.WriteRune('\n')
	sb.WriteString(muted.Render("↑/k ↓/j: navigate • enter: select • q: quit"))
	sb.WriteRune('\n')

	return sb.String()
}

func (m Model) viewOutput() string {
	bold := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(colorSnow))
	muted := lipgloss.NewStyle().Foreground(lipgloss.Color(colorSlate))

	var sb strings.Builder

	// Header.
	sb.WriteString(bold.Render("andes " + m.cmdTitle))
	sb.WriteString("\n\n")

	// Scrollable content.
	sb.WriteString(m.vp.View())
	sb.WriteRune('\n')

	// Footer.
	sb.WriteString(muted.Render("esc: back • q: quit"))
	sb.WriteRune('\n')

	return sb.String()
}

// ── Entry point ────────────────────────────────────────────────────────────

// Run starts the Bubbletea program. newRoot is a factory for a fresh
// cobra root command, used for in-process command execution.
// cli imports tui, so tui MUST NOT import cli at the package level for
// NewRootCmd — the factory is passed in, breaking any import cycle.
// (tui does import cli for RenderLogo; cli does NOT import tui.)
func Run(newRoot func() *cobra.Command) error {
	p := tea.NewProgram(New(newRoot), tea.WithAltScreen())
	_, err := p.Run()
	return err
}
