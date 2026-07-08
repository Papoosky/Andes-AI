package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func modelWithProfiles(t *testing.T) Model {
	t.Helper()
	m := New(nil, nil, nil)
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
