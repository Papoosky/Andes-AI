package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
)

func TestInstallPlanConfirmRunsApply(t *testing.T) {
	called := false
	m := New(func() *cobra.Command { return &cobra.Command{Use: "andes"} }, nil, nil, nil)
	m.applyInstall = func(profiles []string) (string, error) { called = true; return "✓ 1 skill up to date", nil }
	// jump to plan screen with a selection
	m.screen = ScreenInstallPlan
	m.selectedProfiles = []string{"tri-fleet"}
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("enter on plan should dispatch apply")
	}
	cmd() // execute the tea.Cmd
	if !called {
		t.Error("applyInstall was not invoked")
	}
}

func TestInstallDoneShowsOutput(t *testing.T) {
	m := New(nil, nil, nil, nil)
	sized, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m = sized.(Model)
	m2, _ := m.Update(installDoneMsg{summary: "✓ 2 skills up to date"})
	mm := m2.(Model)
	if mm.screen != ScreenOutput {
		t.Fatalf("screen = %v, want ScreenOutput", mm.screen)
	}
	if !contains(mm.View(), "up to date") {
		t.Errorf("output should show the summary:\n%s", mm.View())
	}
}

func modelWithProfiles(t *testing.T) Model {
	t.Helper()
	m := New(nil, nil, nil, nil)
	m2, _ := m.Update(installProfilesMsg{
		names:        []string{"andespath-core", "tri-fleet"},
		descs:        map[string]string{"andespath-core": "base", "tri-fleet": "TRI"},
		installed:    []string{"tri-fleet"},
		catalogKnown: true,
	})
	return m2.(Model)
}

func TestInstallProfilesScreenPrechecksInstalled(t *testing.T) {
	m := modelWithProfiles(t)
	if m.screen != ScreenInstallProfiles {
		t.Fatalf("screen = %v, want ScreenInstallProfiles", m.screen)
	}
	if !m.profileChecked["tri-fleet"] {
		t.Error("installed profile tri-fleet should be pre-checked")
	}
	if m.profileChecked["andespath-core"] {
		t.Error("non-installed profile should start unchecked")
	}
}

func TestInstallProfilesToggleWithSpace(t *testing.T) {
	m := modelWithProfiles(t)
	// cursor at 0 (andespath-core); space toggles it on
	m2, _ := m.Update(tea.KeyMsg{Type: tea.KeySpace})
	mm := m2.(Model)
	if !mm.profileChecked["andespath-core"] {
		t.Error("space should check the profile under the cursor")
	}
}

func TestInstallProfilesNavigation(t *testing.T) {
	m := modelWithProfiles(t)
	m2, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
	if m2.(Model).profileCursor != 1 {
		t.Errorf("cursor = %d, want 1", m2.(Model).profileCursor)
	}
	// clamp at bottom
	m3, _ := m2.(Model).Update(tea.KeyMsg{Type: tea.KeyDown})
	if m3.(Model).profileCursor != 1 {
		t.Errorf("cursor should clamp at 1, got %d", m3.(Model).profileCursor)
	}
}

func TestInstallProfilesEscReturnsToMenu(t *testing.T) {
	m := modelWithProfiles(t)
	m2, _ := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if m2.(Model).screen != ScreenMenu {
		t.Error("esc should return to the menu")
	}
}

func TestInstallProfilesViewIsFramed(t *testing.T) {
	m := modelWithProfiles(t)
	m2, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m = m2.(Model)
	if got := m.View(); !contains(got, "╔") || !contains(got, "[x]") {
		t.Errorf("profiles view must be framed and show checkboxes:\n%s", got)
	}
}

func contains(s, sub string) bool { return len(s) >= len(sub) && (indexOf(s, sub) >= 0) }
func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
