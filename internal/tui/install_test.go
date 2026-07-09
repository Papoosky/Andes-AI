package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
)

// TestCatalogInputThreadsPathToFunc verifies C1: when the user types a catalog
// path on ScreenInstallCatalog and presses Enter, the injected catalogProfiles
// func is called with THAT path (not "").
func TestCatalogInputThreadsPathToFunc(t *testing.T) {
	var gotOverride string
	fakeCatalogProfiles := func(override string) ([]string, map[string]string, []string, bool, error) {
		gotOverride = override
		// Return a valid catalog so the flow advances to ScreenInstallProfiles.
		return []string{"test-profile"}, map[string]string{"test-profile": "desc"}, nil, true, nil
	}

	m := New(nil, nil, fakeCatalogProfiles, nil, nil)
	m.screen = ScreenInstallCatalog
	// Simulate the textinput having a value — set it directly.
	ti := m.catInput
	ti.SetValue("/tmp/my-catalog")
	m.catInput = ti

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("Enter on catalog screen should dispatch a Cmd")
	}

	// Execute the cmd — it calls catalogProfiles(override) and returns an installProfilesMsg.
	msg := cmd()
	if _, ok := msg.(installProfilesMsg); !ok {
		t.Fatalf("expected installProfilesMsg, got %T", msg)
	}

	if gotOverride != "/tmp/my-catalog" {
		t.Errorf("catalogProfiles called with override=%q, want %q", gotOverride, "/tmp/my-catalog")
	}
}

func TestInstallPlanConfirmRunsApply(t *testing.T) {
	called := false
	m := New(func() *cobra.Command { return &cobra.Command{Use: "andes"} }, nil, nil, nil, nil)
	m.applyInstall = func(catalogOverride string, profiles []string) (string, error) {
		called = true
		return "✓ 1 skill up to date", nil
	}
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

// TestInstallPlanShowsSkillNames verifies the Review screen lists the actual
// skill names (with action + profile), not just profile counts.
func TestInstallPlanShowsSkillNames(t *testing.T) {
	m := New(nil, nil, nil, nil, nil)
	sized, _ := m.Update(tea.WindowSizeMsg{Width: 100, Height: 40})
	m = sized.(Model)
	m.screen = ScreenInstallPlan
	m.selectedProfiles = []string{"tri-fleet", "andespath-core"}

	m2, _ := m.Update(planDoneMsg{items: []PlanItem{
		{SkillID: "golang", Action: "install", Profile: "tri-fleet"},
		{SkillID: "git-conventions", Action: "unchanged", Profile: "andespath-core"},
	}})
	mm := m2.(Model)

	view := mm.View()
	if !contains(view, "golang") {
		t.Errorf("Review must show skill name 'golang':\n%s", view)
	}
	if !contains(view, "git-conventions") {
		t.Errorf("Review must show skill name 'git-conventions':\n%s", view)
	}
	// Still shows the derived counts.
	if !contains(view, "1 to install") {
		t.Errorf("Review must still show derived counts:\n%s", view)
	}
}

func TestInstallDoneShowsOutput(t *testing.T) {
	m := New(nil, nil, nil, nil, nil)
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
	m := New(nil, nil, nil, nil, nil)
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

// TestCatalogQRuneDoesNotQuit verifies Critical 2: typing 'q' on the catalog
// input screen must NOT quit the app — it must go to the text input instead.
func TestCatalogQRuneDoesNotQuit(t *testing.T) {
	// Reach catalog screen via the normal installProfilesMsg path (catalogKnown=false).
	m := New(nil, nil, nil, nil, nil)
	m2, _ := m.Update(installProfilesMsg{catalogKnown: false})
	mm := m2.(Model)
	if mm.screen != ScreenInstallCatalog {
		t.Fatalf("expected ScreenInstallCatalog, got %v", mm.screen)
	}

	_, cmd := mm.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	if cmd == nil {
		// textinput returned nil cmd — fine, as long as the screen didn't change.
		if mm.screen != ScreenInstallCatalog {
			t.Error("typing 'q' changed screen away from catalog")
		}
		return
	}
	msg := cmd()
	if _, ok := msg.(tea.QuitMsg); ok {
		t.Error("typing 'q' on catalog screen must not produce tea.QuitMsg")
	}
}

// TestCatalogPathWithQ verifies Critical 2 extended: a full path containing
// 'q' is correctly captured (not cut short by a quit).
func TestCatalogPathWithQ(t *testing.T) {
	m := New(nil, nil, nil, nil, nil)
	m2, _ := m.Update(installProfilesMsg{catalogKnown: false})
	mm := m2.(Model)
	if mm.screen != ScreenInstallCatalog {
		t.Fatalf("expected ScreenInstallCatalog, got %v", mm.screen)
	}

	// Type each rune of the path "/tmp/qcatalog/q-path".
	// We verify that the screen stays on ScreenInstallCatalog for every rune.
	path := "/tmp/qcatalog/q-path"
	cur := mm
	for _, r := range path {
		m3, _ := cur.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		cur = m3.(Model)
		if cur.screen != ScreenInstallCatalog {
			t.Fatalf("screen changed to %v after typing rune %q — q must not quit", cur.screen, r)
		}
	}
}

// TestInstallPlanInFlightGuard verifies Important 4: pressing enter twice
// while an install is in-flight only dispatches one apply command.
func TestInstallPlanInFlightGuard(t *testing.T) {
	callCount := 0
	m := New(nil, nil, nil, nil, func(catalogOverride string, profiles []string) (string, error) {
		callCount++
		return "✓ 1 skill up to date", nil
	})
	m.screen = ScreenInstallPlan
	m.selectedProfiles = []string{"tri-fleet"}

	// First enter: dispatches apply, sets installing=true.
	m2, cmd1 := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	mm := m2.(Model)
	if cmd1 == nil {
		t.Fatal("first enter should dispatch a Cmd")
	}
	if !mm.installing {
		t.Error("installing flag should be true after first enter")
	}

	// Second enter while in-flight: must be no-op.
	_, cmd2 := mm.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd2 != nil {
		t.Error("second enter while installing must not dispatch another Cmd")
	}

	// Execute the first cmd.
	cmd1()
	if callCount != 1 {
		t.Errorf("applyInstall called %d times, want exactly 1", callCount)
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
