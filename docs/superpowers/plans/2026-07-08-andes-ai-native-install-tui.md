# andes-ai Native Install TUI Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax.

**Goal:** Rename `init` → `install`, and rebuild the install flow as NATIVE Bubbletea screens inside the TUI (no subprocess, no huh, no aesthetic drop) — catalog input (only if unknown), profile multiselect, plan+confirm, in-process apply.

**Architecture:** A screen state machine in the existing `internal/tui` Model. Selecting "install" from the menu walks ScreenInstallProfiles → ScreenInstallPlan → apply in-process → ScreenOutput, all inside the shared bordered frame + palette. Reuses `cli` resolution + `installer.Plan/Apply` via injected callbacks (tui must NOT import cli). The CLI `andes install` command (subprocess/huh, for shell/CI) stays fully functional — only the TUI path becomes native.

**Tech Stack:** Go, Bubbletea, bubbles/textinput, lipgloss. Fixtures: git repos + local catalogs in t.TempDir().

## Global Constraints

- All user-facing text ENGLISH, actionable. Conventional Commits, no AI attribution, no Co-Authored-By. TDD.
- `tui` must NOT import `cli` (no import cycle). New behavior the TUI needs from cli is passed as injected function values (same pattern as `newRoot func() *cobra.Command`).
- Shared frame + palette from internal/theme; logo from internal/logo. One source each — no duplication.
- The CLI `install` command behavior is identical to today's `init` (only the name changes).
- Nobody has the repo yet → clean rename, NO `init` alias.

---

### Task 1: Rename the `init` command to `install`

**Files:**
- Rename: `internal/cli/init.go` → `internal/cli/install.go`; `internal/cli/init_test.go` → `internal/cli/install_test.go`
- Modify: `internal/cli/root.go` (registration), `internal/cli/update.go` (any "run `andes init`" message text), `internal/cli/doctor.go` (same), `internal/cli/prompts.go` (any message), `internal/tui/model.go` (menu option id/label "init"→"install", and the `tea.ExecProcess(exec.Command(exe, "init"))` → "install"), `README.md`, `install.sh` (any `andes init` reference in echoed help)
- Test: the renamed test file + any test asserting on the string "init" as a command

**Interfaces:**
- Produces: cobra command `install` (was `init`); function `newInstallCmd()` (was `newInitCmd`); `runInstall` (was `runInit`). `installAndSave` keeps its name (already generic).

- [ ] **Step 1: Sweep every `init` command reference to `install`**

Run to find them all first:
```bash
grep -rn --include='*.go' --include='*.md' --include='*.sh' -e '"init"' -e 'newInitCmd' -e 'runInit' -e 'andes init' -e 'exec.Command(exe, "init")' . | grep -v '.superpowers' | grep -v 'docs/superpowers'
```

Rename in code:
- `git mv internal/cli/init.go internal/cli/install.go && git mv internal/cli/init_test.go internal/cli/install_test.go`
- In install.go: `newInitCmd`→`newInstallCmd`, `runInit`→`runInstall`, cobra `Use: "install"`, `Short: "Install skills from the catalog according to profiles"`.
- root.go: `newInitCmd()` → `newInstallCmd()` in AddCommand.
- All user-facing strings mentioning "andes init" → "andes install" (update.go "you haven't run `andes init` yet" → "andes install"; doctor.go advice "re-run `andes init`" → "re-run `andes install`"; prompts.go; the no-op message from the bug fix "run `andes install`" if present).
- tui/model.go: menu option `{id:"init", label:"init", desc:...}` → `{id:"install", label:"install", desc:"install skills from the catalog"}`; and `exec.Command(exe, "init")` → `exec.Command(exe, "install")` (this line is REMOVED entirely in Task 3, but rename it here so the build stays green in between).
- README.md and install.sh: `andes init` → `andes install`.

- [ ] **Step 2: Update test assertions**

Any test asserting the command name or the advice strings (e.g. tests checking output contains "andes init", or `TestListWithoutManifestShowsCatalogAndHint` which suggests "andes init") must switch to "andes install". Grep the test files for `init` and update assertions + the banner/help command list in banner.go/model.go (the Commands list shows "init" → "install").

- [ ] **Step 3: Full gate**

Run: `go test ./... && go vet ./... && gofmt -l .`
Expected: green, silent. `andes install --help` works; `andes init` no longer exists.

- [ ] **Step 4: Commit**

```bash
git add -A
git commit -m "refactor: rename init command to install"
```

---

### Task 2: Profile multiselect + catalog input screens

**Files:**
- Create: `internal/tui/install.go` (install-flow screens, state, key handling, views)
- Modify: `internal/tui/model.go` (add Screen constants, install-flow fields to Model, route in Update/View)
- Modify: `go.mod` (`go get github.com/charmbracelet/bubbles/textinput` — already indirectly present via bubbles; make direct if needed)
- Test: `internal/tui/install_test.go`

**Interfaces:**
- Consumes: injected callbacks (added to Model + New/Run signature) so tui stays decoupled from cli:
  - `catalogProfiles func() (names []string, descs map[string]string, installed []string, catalogKnown bool, err error)` — resolves the catalog (flag/manifest/default), loads it, returns profile names sorted, their descriptions, the currently-installed profile set, and whether the catalog location is already known (so the flow can skip the catalog-input screen).
  - (catalog input is only needed when `catalogKnown == false`.)
- Produces (Task 3 consumes):
  - New `Screen` values: `ScreenInstallCatalog`, `ScreenInstallProfiles`, `ScreenInstallPlan`.
  - Model fields: `catInput textinput.Model`, `profiles []string`, `profileDesc map[string]string`, `profileChecked map[string]bool`, `profileCursor int`, `installErr error`.
  - `func (m Model) startInstall() (tea.Model, tea.Cmd)` — entry from the menu; triggers a `tea.Cmd` that calls `catalogProfiles()` and returns an `installProfilesMsg`.
  - Messages: `installProfilesMsg{names, descs, installed, catalogKnown, err}`.

- [ ] **Step 1: Write failing tests for navigation + selection**

`internal/tui/install_test.go`:

```go
package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func modelWithProfiles(t *testing.T) Model {
	t.Helper()
	m := New(nil, nil)
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
	m, _ = m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
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
```

- [ ] **Step 2: Run tests, verify they fail**

Run: `go test ./internal/tui/ -run TestInstallProfiles`
Expected: FAIL — undefined `installProfilesMsg`, `ScreenInstallProfiles`, fields.

- [ ] **Step 3: Implement `internal/tui/install.go` + Model wiring**

Add to model.go: `ScreenInstallCatalog`, `ScreenInstallProfiles`, `ScreenInstallPlan` in the Screen const block; the Model fields listed in Interfaces; route these screens in `Update` (delegate to install-flow handlers) and `View` (delegate to install-flow views). Extend `New` to accept the `catalogProfiles` callback and store it (update all call sites incl. tests and cli/root.go — pass a real impl in root.go, `nil` acceptable in unit tests that inject msgs directly).

Implement in install.go:
- `installProfilesMsg` struct and handling in Update: populate profiles/descs/profileChecked (pre-check installed), set cursor 0, screen = ScreenInstallProfiles (or ScreenInstallCatalog if `!catalogKnown` — show textinput first, then on enter fire catalogProfiles again with the entered path).
- `updateInstallProfiles(msg tea.KeyMsg)`: up/k, down/j (clamped), space toggles `profileChecked[name]` under cursor, enter → build the selected list and transition to plan (Task 3 provides the plan step; for THIS task, enter can no-op or set a placeholder screen — but to keep the task self-contained, enter transitions to ScreenInstallPlan which Task 3 fills; here just switch screen and store selection in a field `selectedProfiles []string`), esc → ScreenMenu, q/ctrl+c → tea.Quit.
- `viewInstallProfiles()`: framed (theme.Frame), title "Install skills", one line per profile `[x] name  desc` / `[ ] name  desc` with `▸` cursor marker, footer "space: toggle • enter: continue • esc: back".
- Catalog input screen (`viewInstallCatalog` + handling): a `textinput.Model` prompt "Catalog path or git URL", framed; enter submits → re-run catalogProfiles with the path.

Wire the menu: selecting "install" calls `startInstall()` (Task 3 replaces the old ExecProcess path; in THIS task, add `startInstall` and call it from selectOption's "install" case, replacing the ExecProcess block).

- [ ] **Step 4: Run tests, verify pass**

Run: `go test ./internal/tui/ -v -run TestInstall` then `go test ./...`
Expected: PASS. (cli/root.go must pass a real `catalogProfiles` impl — see Task 3 for the impl; for THIS task, add a minimal impl in cli that returns catalogKnown from manifest/default and loads profiles; if that's cleaner to land in Task 3, have root.go pass a stub that errors, and gate only tui tests here. Prefer landing the real impl now so the build is honest.)

- [ ] **Step 5: Commit**

```bash
git add -A
git commit -m "feat: native tui profile selection and catalog input screens"
```

---

### Task 3: Plan/confirm/apply screen + wire the full native flow

**Files:**
- Modify: `internal/tui/install.go` (plan screen, apply command, result)
- Modify: `internal/tui/model.go` (remove the old `exec.ExecProcess` install path from selectOption)
- Create: `internal/cli/tuiwire.go` (the `catalogProfiles` + `applyInstall` implementations passed into tui.Run/New — lives in cli, injected, so tui stays decoupled)
- Modify: `internal/cli/root.go` (build and inject the callbacks)
- Test: `internal/tui/install_test.go` (plan + apply), `internal/cli` integration if feasible

**Interfaces:**
- Consumes: `installer.Plan/Apply`, `resolveSource`/`SourceFromManifest`, `manifest` (all in cli, wrapped by the injected callbacks).
- Produces:
  - Injected `applyInstall func(profiles []string) (summary string, err error)` — resolves source, plans, applies in-process (implicit yes), saves manifest, returns a human summary ("✓ 3 skills up to date" or "Everything is already up to date"). NO prompts (the TUI plan screen already confirmed).
  - Messages: `installDoneMsg{summary string, err error}`.
  - `ScreenInstallPlan` view showing the computed plan (install/update/unchanged counts) with confirm; enter → run applyInstall via tea.Cmd → installDoneMsg → ScreenOutput; esc → ScreenMenu.

- [ ] **Step 1: Write failing tests**

Add to `internal/tui/install_test.go`:

```go
func TestInstallPlanConfirmRunsApply(t *testing.T) {
	called := false
	m := New(func() *cobra.Command { return &cobra.Command{Use: "andes"} }, nil)
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
	m := New(nil, nil)
	m, _ = m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m2, _ := m.Update(installDoneMsg{summary: "✓ 2 skills up to date"})
	mm := m2.(Model)
	if mm.screen != ScreenOutput {
		t.Fatalf("screen = %v, want ScreenOutput", mm.screen)
	}
	if !contains(mm.View(), "up to date") {
		t.Errorf("output should show the summary:\n%s", mm.View())
	}
}
```

(add `cobra` import; note `New` must expose or allow setting `applyInstall` — make the field settable within-package, and add it as a param to `New` alongside `catalogProfiles`, updating all call sites.)

- [ ] **Step 2: Run, verify fail**

Run: `go test ./internal/tui/ -run 'TestInstallPlan|TestInstallDone'`
Expected: FAIL — undefined `applyInstall`, `installDoneMsg`, `ScreenInstallPlan` handling.

- [ ] **Step 3: Implement**

- install.go: `updateInstallPlan` (enter → `func() tea.Msg { s, err := m.applyInstall(m.selectedProfiles); return installDoneMsg{s, err} }`; esc → menu). `viewInstallPlan` framed: heading "Review", the selected profiles, footer "enter: apply • esc: back". Handle `installDoneMsg` in Update: put summary (or err) into the viewport, screen = ScreenOutput (reuse existing output screen). 
- model.go: in `selectOption`, replace the entire `case "install":` ExecProcess block with `return m.startInstall()`. Delete the now-unused `os/exec` import if nothing else uses it (check: init subprocess was the only exec user — verify and remove).
- cli/tuiwire.go: implement `catalogProfiles()` (resolve source via manifest/default, load catalog, return sorted profile names + descriptions + installed set from manifest + catalogKnown) and `applyInstall(profiles)` (resolveSource, Plan, Apply in-process, finalizeRef, Save, return summary; if no changes → "Everything is already up to date"). These close over nothing cli-private that tui sees — tui only holds the func values.
- root.go: `tui.Run(NewRootCmd, checkCatalogFreshness, catalogProfiles, applyInstall)` (widen Run/New signature accordingly; update tui tests' New(...) calls — pass nils where not exercised).

- [ ] **Step 4: Verify + manual note**

Run: `go test ./... && go vet ./... && gofmt -l .`
Expected: green, silent. Build `go build -o andes ./cmd/andes`. NOTE in report: the interactive native flow needs human TTY verification (drive with real keystrokes) — automated tests cover Update/View, not live rendering.

- [ ] **Step 5: Commit**

```bash
git add -A
git commit -m "feat: native in-process install flow with plan and apply screens"
```

---

## Self-Review (applied)

- **Spec coverage:** rename → Task 1; catalog-unknown input + profile multiselect (pre-checked, re-selectable) → Task 2; plan/confirm + in-process apply inside the frame + wiring + removing the subprocess path → Task 3. The bug-#2 no-op behavior lands separately in installAndSave (CLI) and is mirrored by applyInstall's "Everything is already up to date".
- **Decoupling:** tui never imports cli — `catalogProfiles`/`applyInstall` are injected func values, same pattern as `newRoot`. Verified in every task's interface block.
- **No placeholders in tests:** all test code is complete. Implementation steps for the TUI screens are spec-level (screen fields, key bindings, view contents, injected signatures) rather than line-complete — these tasks use a sonnet implementer because the view/layout code needs idiomatic judgment; the interfaces, messages, state fields, and test expectations are fully pinned so neighbors compose.
- **Type consistency:** `installProfilesMsg`/`installDoneMsg`, `ScreenInstall*`, `catalogProfiles`/`applyInstall`, `selectedProfiles`/`profileChecked` used consistently across Tasks 2 and 3.
