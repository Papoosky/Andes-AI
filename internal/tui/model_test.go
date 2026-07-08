package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
)

// helper: build a fresh model at ScreenMenu.
func newTestModel() Model {
	return New(nil, nil)
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
	m := New(nil, nil)
	updated, _ := m.Update(FreshnessMsg{Outdated: true})
	mm := updated.(Model)
	if !strings.Contains(mm.View(), "press u to update") {
		t.Errorf("banner missing from view:\n%s", mm.View())
	}
}

func TestFreshnessOfflineShowsFooterNote(t *testing.T) {
	m := New(nil, nil)
	updated, _ := m.Update(FreshnessMsg{Offline: true})
	mm := updated.(Model)
	if !strings.Contains(mm.View(), "offline") {
		t.Errorf("offline note missing:\n%s", mm.View())
	}
}

func TestPressUWithoutUpdateAvailableDoesNothing(t *testing.T) {
	m := New(nil, nil)
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("u")})
	if cmd != nil {
		t.Error("u without update available should be a no-op")
	}
}

func TestPressUWithUpdateAvailableRunsUpdate(t *testing.T) {
	m := New(func() *cobra.Command { return &cobra.Command{Use: "andes"} }, nil)
	updated, _ := m.Update(FreshnessMsg{Outdated: true})
	mm := updated.(Model)
	_, cmd := mm.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("u")})
	if cmd == nil {
		t.Fatal("u with update available should dispatch the update command")
	}
}

func TestCmdResultClearsUpdateBanner(t *testing.T) {
	m := New(nil, nil)
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
	m := New(nil, nil)
	updated, _ := m.Update(FreshnessMsg{Outdated: true})
	_, cmd := updated.(Model).Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("u")})
	if cmd != nil {
		t.Error("u with nil root factory should be a no-op, not a panic or dispatch")
	}
}
