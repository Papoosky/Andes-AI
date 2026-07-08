package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
)

// helper: build a fresh model at ScreenMenu.
func newTestModel() Model {
	return New(nil, nil, nil, nil, nil)
}

// helper: send a key rune message.
func keyRune(r rune) tea.KeyMsg {
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}}
}

// helper: send a special key message.
func keySpecial(t tea.KeyType) tea.KeyMsg {
	return tea.KeyMsg{Type: t}
}

// ── Cursor navigation ──────────────────────────────────────────────────────

func TestCursorDown_j(t *testing.T) {
	m := newTestModel()
	m.cursor = 0

	next, _ := m.Update(keyRune('j'))
	got := next.(Model).cursor
	if got != 1 {
		t.Errorf("cursor after j: got %d, want 1", got)
	}
}

func TestCursorDown_arrow(t *testing.T) {
	m := newTestModel()
	m.cursor = 0

	next, _ := m.Update(keySpecial(tea.KeyDown))
	got := next.(Model).cursor
	if got != 1 {
		t.Errorf("cursor after ↓: got %d, want 1", got)
	}
}

func TestCursorUp_k(t *testing.T) {
	m := newTestModel()
	m.cursor = 1

	next, _ := m.Update(keyRune('k'))
	got := next.(Model).cursor
	if got != 0 {
		t.Errorf("cursor after k: got %d, want 0", got)
	}
}

func TestCursorUp_arrow(t *testing.T) {
	m := newTestModel()
	m.cursor = 1

	next, _ := m.Update(keySpecial(tea.KeyUp))
	got := next.(Model).cursor
	if got != 0 {
		t.Errorf("cursor after ↑: got %d, want 0", got)
	}
}

func TestCursorClampAtTop(t *testing.T) {
	m := newTestModel()
	m.cursor = 0

	next, _ := m.Update(keyRune('k'))
	got := next.(Model).cursor
	if got != 0 {
		t.Errorf("cursor clamped at top: got %d, want 0", got)
	}
}

func TestCursorClampAtBottom(t *testing.T) {
	m := newTestModel()
	last := len(m.options) - 1
	m.cursor = last

	next, _ := m.Update(keyRune('j'))
	got := next.(Model).cursor
	if got != last {
		t.Errorf("cursor clamped at bottom: got %d, want %d", got, last)
	}
}

// ── Quit on q / ctrl+c ────────────────────────────────────────────────────

func TestQuit_q(t *testing.T) {
	m := newTestModel()
	_, cmd := m.Update(keyRune('q'))
	if cmd == nil {
		t.Fatal("expected a Cmd from q, got nil")
	}
	msg := cmd()
	if _, ok := msg.(tea.QuitMsg); !ok {
		t.Errorf("q should produce tea.QuitMsg, got %T", msg)
	}
}

func TestQuit_ctrlC(t *testing.T) {
	m := newTestModel()
	_, cmd := m.Update(keySpecial(tea.KeyCtrlC))
	if cmd == nil {
		t.Fatal("expected a Cmd from ctrl+c, got nil")
	}
	msg := cmd()
	if _, ok := msg.(tea.QuitMsg); !ok {
		t.Errorf("ctrl+c should produce tea.QuitMsg, got %T", msg)
	}
}

// ── Enter on "quit" option ─────────────────────────────────────────────────

func TestEnterOnQuit(t *testing.T) {
	m := newTestModel()
	// Find the quit option index.
	quitIdx := -1
	for i, o := range m.options {
		if o.id == "quit" {
			quitIdx = i
			break
		}
	}
	if quitIdx < 0 {
		t.Fatal("quit option not found in defaultOptions()")
	}
	m.cursor = quitIdx

	_, cmd := m.Update(keySpecial(tea.KeyEnter))
	if cmd == nil {
		t.Fatal("enter on quit should return a Cmd")
	}
	msg := cmd()
	if _, ok := msg.(tea.QuitMsg); !ok {
		t.Errorf("enter on quit should produce tea.QuitMsg, got %T", msg)
	}
}

// ── Enter on "doctor" → transitions to ScreenOutput ───────────────────────

func TestEnterDoctor_transitionOnResult(t *testing.T) {
	m := newTestModel()
	doctorIdx := -1
	for i, o := range m.options {
		if o.id == "doctor" {
			doctorIdx = i
			break
		}
	}
	if doctorIdx < 0 {
		t.Fatal("doctor option not found")
	}
	m.cursor = doctorIdx

	// Press enter — triggers async command; model should stay on ScreenMenu.
	next, _ := m.Update(keySpecial(tea.KeyEnter))
	m2 := next.(Model)
	if m2.screen != ScreenMenu {
		t.Errorf("after enter on doctor: expected ScreenMenu (async), got %v", m2.screen)
	}

	// Simulate the result message arriving.
	result := cmdResultMsg{cmdID: "doctor", output: "All healthy ✓", err: nil}
	next2, _ := m2.Update(result)
	m3 := next2.(Model)
	if m3.screen != ScreenOutput {
		t.Errorf("after cmdResultMsg: expected ScreenOutput, got %v", m3.screen)
	}
}

// ── Esc on ScreenOutput → back to ScreenMenu ──────────────────────────────

func TestEscFromOutput(t *testing.T) {
	m := newTestModel()
	m.screen = ScreenOutput

	next, _ := m.Update(keySpecial(tea.KeyEsc))
	m2 := next.(Model)
	if m2.screen != ScreenMenu {
		t.Errorf("esc from ScreenOutput: expected ScreenMenu, got %v", m2.screen)
	}
}

// ── q on ScreenOutput → quit ──────────────────────────────────────────────

func TestQuit_onOutput(t *testing.T) {
	m := newTestModel()
	m.screen = ScreenOutput

	_, cmd := m.Update(keyRune('q'))
	if cmd == nil {
		t.Fatal("expected Cmd from q on ScreenOutput")
	}
	msg := cmd()
	if _, ok := msg.(tea.QuitMsg); !ok {
		t.Errorf("q on ScreenOutput should produce tea.QuitMsg, got %T", msg)
	}
}

// ── Freshness banner and u-key ─────────────────────────────────────────────

func TestFreshnessOutdatedShowsBanner(t *testing.T) {
	m := New(nil, nil, nil, nil, nil)
	updated, _ := m.Update(FreshnessMsg{Outdated: true})
	mm := updated.(Model)
	if !strings.Contains(mm.View(), "press u to update") {
		t.Errorf("banner missing from view:\n%s", mm.View())
	}
}

func TestFreshnessOfflineShowsFooterNote(t *testing.T) {
	m := New(nil, nil, nil, nil, nil)
	updated, _ := m.Update(FreshnessMsg{Offline: true})
	mm := updated.(Model)
	if !strings.Contains(mm.View(), "offline") {
		t.Errorf("offline note missing:\n%s", mm.View())
	}
}

func TestPressUWithoutUpdateAvailableDoesNothing(t *testing.T) {
	m := New(nil, nil, nil, nil, nil)
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("u")})
	if cmd != nil {
		t.Error("u without update available should be a no-op")
	}
}

func TestPressUWithUpdateAvailableRunsUpdate(t *testing.T) {
	m := New(func() *cobra.Command { return &cobra.Command{Use: "andes"} }, nil, nil, nil, nil)
	updated, _ := m.Update(FreshnessMsg{Outdated: true})
	mm := updated.(Model)
	_, cmd := mm.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("u")})
	if cmd == nil {
		t.Fatal("u with update available should dispatch the update command")
	}
}

func TestCmdResultClearsUpdateBanner(t *testing.T) {
	m := New(nil, nil, nil, nil, nil)
	updated, _ := m.Update(FreshnessMsg{Outdated: true})
	updated, _ = updated.(Model).Update(cmdResultMsg{cmdID: "update", output: "done"})
	mm := updated.(Model)
	// Back on menu after esc: banner must be gone.
	updated, _ = mm.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if strings.Contains(updated.(Model).View(), "press u to update") {
		t.Error("banner should clear after an update run")
	}
}

func TestPressUWithNilRootIsSafe(t *testing.T) {
	m := New(nil, nil, nil, nil, nil)
	updated, _ := m.Update(FreshnessMsg{Outdated: true})
	_, cmd := updated.(Model).Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("u")})
	if cmd != nil {
		t.Error("u with nil root factory should be a no-op, not a panic or dispatch")
	}
}

// ── Output box hugs content ────────────────────────────────────────────────

// TestOutputBoxHugsShortContent verifies that a short output string results
// in a box width much smaller than the terminal width (content-hugging), while
// a long output is capped at the terminal width (not overflowing).
func TestOutputBoxHugsShortContent(t *testing.T) {
	const termWidth = 80

	// Short content — box must hug, not blow up to terminal width.
	t.Run("short content is hugged", func(t *testing.T) {
		m := New(nil, nil, nil, nil, nil)
		sized, _ := m.Update(tea.WindowSizeMsg{Width: termWidth, Height: 24})
		m = sized.(Model)

		result, _ := m.Update(cmdResultMsg{cmdID: "doctor", output: "All healthy", err: nil})
		m = result.(Model)

		view := m.View()
		firstLine := strings.SplitN(view, "\n", 2)[0]
		boxWidth := lipgloss.Width(firstLine)

		if boxWidth >= 40 {
			t.Errorf("short content: box top border width %d should be < 40 (hugging), terminal=%d\nview:\n%s", boxWidth, termWidth, view)
		}
	})

	// Long content — box must not exceed terminal width.
	t.Run("long content is capped", func(t *testing.T) {
		m := New(nil, nil, nil, nil, nil)
		sized, _ := m.Update(tea.WindowSizeMsg{Width: termWidth, Height: 24})
		m = sized.(Model)

		longLine := strings.Repeat("x", 200)
		longOutput := strings.Join([]string{longLine, longLine, longLine}, "\n")
		result, _ := m.Update(cmdResultMsg{cmdID: "list", output: longOutput, err: nil})
		m = result.(Model)

		view := m.View()
		firstLine := strings.SplitN(view, "\n", 2)[0]
		boxWidth := lipgloss.Width(firstLine)

		if boxWidth > termWidth {
			t.Errorf("long content: box top border width %d exceeds terminal width %d\nview:\n%s", boxWidth, termWidth, view)
		}
	})
}

// ── Frame tests ───────────────────────────────────────────────────────────────

// TestMenuAndOutputAreFramed verifies that both TUI screens are wrapped in the
// shared DoubleBorder frame. We drive a WindowSizeMsg first so the viewport is
// sized, then assert the top-left corner rune of a DoubleBorder (╔) is present.
func TestMenuAndOutputAreFramed(t *testing.T) {
	const doubleBorderCorner = "╔"

	// Give the model a real terminal size so sizing paths are exercised.
	m := New(nil, nil, nil, nil, nil)
	sized, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m = sized.(Model)

	// ScreenMenu
	menuView := m.View()
	if !strings.Contains(menuView, doubleBorderCorner) {
		t.Errorf("ScreenMenu View() missing DoubleBorder corner rune %q", doubleBorderCorner)
	}

	// ScreenOutput — switch via a cmdResultMsg to populate the viewport.
	switched, _ := m.Update(cmdResultMsg{cmdID: "list", output: "skill-a\nskill-b\n"})
	m = switched.(Model)
	if m.screen != ScreenOutput {
		t.Fatal("expected ScreenOutput after cmdResultMsg")
	}
	outputView := m.View()
	if !strings.Contains(outputView, doubleBorderCorner) {
		t.Errorf("ScreenOutput View() missing DoubleBorder corner rune %q", doubleBorderCorner)
	}
}

// TestBoxWidthStableAcrossScreens verifies that both TUI screens render a closed
// DoubleBorder and that neither box exceeds the terminal width (content-hugging mode).
func TestBoxWidthStableAcrossScreens(t *testing.T) {
	const termWidth = 80

	m := New(nil, nil, nil, nil, nil)
	sized, _ := m.Update(tea.WindowSizeMsg{Width: termWidth, Height: 24})
	m = sized.(Model)

	menuView := m.View()
	if !strings.Contains(menuView, "╔") {
		t.Error("ScreenMenu missing DoubleBorder top-left corner ╔")
	}
	if !strings.Contains(menuView, "╚") {
		t.Error("ScreenMenu missing DoubleBorder bottom-left corner ╚")
	}
	if w := lipgloss.Width(menuView); w > termWidth {
		t.Errorf("ScreenMenu box width %d exceeds terminal width %d", w, termWidth)
	}

	switched, _ := m.Update(cmdResultMsg{cmdID: "list", output: "test output\n"})
	m = switched.(Model)
	outputView := m.View()
	if !strings.Contains(outputView, "╔") {
		t.Error("ScreenOutput missing DoubleBorder top-left corner ╔")
	}
	if !strings.Contains(outputView, "╚") {
		t.Error("ScreenOutput missing DoubleBorder bottom-left corner ╚")
	}
	if w := lipgloss.Width(outputView); w > termWidth {
		t.Errorf("ScreenOutput box width %d exceeds terminal width %d", w, termWidth)
	}

	t.Logf("ScreenMenu (first 3 lines):\n%s\n", strings.Join(strings.Split(menuView, "\n")[:3], "\n"))
	t.Logf("ScreenOutput (first 3 lines):\n%s\n", strings.Join(strings.Split(outputView, "\n")[:3], "\n"))
}
