package cli_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/andespath/andes-ai/internal/cli"
	"github.com/andespath/andes-ai/internal/manifest"
)

// TestTUICallbacksCatalogOverrideInstalls verifies Critical 1 end-to-end:
// buildApplyInstallFunc(catalogOverride, profiles) installs skills under a
// temp HOME when catalogOverride is a real local path (not "").
func TestTUICallbacksCatalogOverrideInstalls(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	cat := fixtureCatalog(t)

	cpf, _, aif := cli.ExportedBuildCallbacks()

	// buildCatalogProfilesFunc with override: must return profiles.
	names, _, _, known, err := cpf(cat)
	if err != nil {
		t.Fatalf("catalogProfiles(override): %v", err)
	}
	if !known {
		t.Fatal("catalogKnown should be true when override is a valid local path")
	}
	if len(names) == 0 {
		t.Fatal("expected at least one profile")
	}

	// buildApplyInstallFunc with catalog override: must install skills to disk.
	summary, err := aif(cat, []string{"tri-fleet"})
	if err != nil {
		t.Fatalf("applyInstall(override): %v", err)
	}
	if summary == "" {
		t.Fatal("expected non-empty summary")
	}

	// Skills on disk.
	golangSkill := filepath.Join(home, ".claude", "skills", "golang", "SKILL.md")
	if _, err := os.Stat(golangSkill); err != nil {
		t.Errorf("skill not installed: %v", err)
	}

	// Manifest written.
	m, err := manifest.Load(filepath.Join(home, ".claude", "andes.json"))
	if err != nil || m == nil {
		t.Fatalf("manifest not written: %v", err)
	}
	if m.Catalog.Type != "local" {
		t.Errorf("catalog.type = %q, want local", m.Catalog.Type)
	}
}

// TestTUICallbacksNoOpRepair verifies Critical 3: when all skills are already
// installed (changeCount==0 per Plan), calling applyInstall still repairs a
// deleted skill via installer.Apply's disk re-check.
func TestTUICallbacksNoOpRepair(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	cat := fixtureCatalog(t)

	_, _, aif := cli.ExportedBuildCallbacks()

	// First install.
	if _, err := aif(cat, []string{"tri-fleet"}); err != nil {
		t.Fatalf("first install: %v", err)
	}

	// Delete an installed skill to simulate drift.
	golangSkill := filepath.Join(home, ".claude", "skills", "golang", "SKILL.md")
	if err := os.Remove(golangSkill); err != nil {
		t.Fatalf("remove skill: %v", err)
	}

	// Second call with same params: Plan returns only skip actions, but Apply
	// must still run and repair the missing skill.
	_, err := aif(cat, []string{"tri-fleet"})
	if err != nil {
		t.Fatalf("repair install: %v", err)
	}

	// Skill must be back on disk.
	if _, err := os.Stat(golangSkill); err != nil {
		t.Errorf("skill was NOT reinstalled after drift: %v", err)
	}
}

// TestTUICallbacksGitCatalogOverrideInstalls verifies Critical 1 end-to-end
// with a git file:// URL as catalogOverride.
func TestTUICallbacksGitCatalogOverrideInstalls(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	repo, _ := gitFixture(t)
	fileURL := "file://" + repo

	_, _, aif := cli.ExportedBuildCallbacks()

	summary, err := aif(fileURL, []string{"tri-fleet"})
	if err != nil {
		t.Fatalf("applyInstall(git override): %v", err)
	}
	if summary == "" {
		t.Fatal("expected non-empty summary")
	}

	golangSkill := filepath.Join(home, ".claude", "skills", "golang", "SKILL.md")
	if _, err := os.Stat(golangSkill); err != nil {
		t.Errorf("skill not installed from git: %v", err)
	}

	m, err := manifest.Load(filepath.Join(home, ".claude", "andes.json"))
	if err != nil || m == nil {
		t.Fatalf("manifest not written: %v", err)
	}
	if m.Catalog.Type != "git" {
		t.Errorf("catalog.type = %q, want git", m.Catalog.Type)
	}
}
