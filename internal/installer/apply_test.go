package installer_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/andespath/andes-ai/internal/catalog"
	"github.com/andespath/andes-ai/internal/installer"
	"github.com/andespath/andes-ai/internal/manifest"
)

func TestApplyCopiesSkills(t *testing.T) {
	src := makeCatalog(t)
	cat := loadCat(t, src)
	skillsDir := t.TempDir()

	actions, err := installer.Plan(src, cat, nil, []string{"core"})
	if err != nil {
		t.Fatal(err)
	}

	installed, err := installer.Apply(src, actions, skillsDir)
	if err != nil {
		t.Fatalf("Apply() error = %v", err)
	}

	for _, id := range []string{"git-conventions", "code-review"} {
		skillMD := filepath.Join(skillsDir, id, "SKILL.md")
		if _, err := os.Stat(skillMD); err != nil {
			t.Errorf("missing %s after Apply", skillMD)
		}
		if installed[id].Hash == "" {
			t.Errorf("installed[%q] has no hash", id)
		}
	}
}

func TestApplySkipReinstallsMissing(t *testing.T) {
	src := makeCatalog(t)
	cat := loadCat(t, src)
	skillsDir := t.TempDir()

	// Plan yields skip because manifest hash matches catalog.
	// But skill is NOT on disk: Apply must reinstall it.
	actions, err := installer.Plan(src, cat, nil, []string{"tri"})
	if err != nil {
		t.Fatal(err)
	}
	// Force actions to skip (simulate manifest already up-to-date).
	for i := range actions {
		actions[i].Type = installer.ActionSkip
	}
	// skillsDir is empty — golang dir does not exist.

	installed, err := installer.Apply(src, actions, skillsDir)
	if err != nil {
		t.Fatalf("Apply() error = %v", err)
	}

	skillMD := filepath.Join(skillsDir, "golang", "SKILL.md")
	if _, err := os.Stat(skillMD); err != nil {
		t.Errorf("golang/SKILL.md should have been reinstalled, got: %v", err)
	}
	if installed["golang"].Hash == "" {
		t.Errorf("installed[golang] hash should be non-empty, got %+v", installed["golang"])
	}
}

func TestApplySkipRepairsModified(t *testing.T) {
	src := makeCatalog(t)
	cat := loadCat(t, src)
	skillsDir := t.TempDir()

	// First install.
	actions, err := installer.Plan(src, cat, nil, []string{"tri"})
	if err != nil {
		t.Fatal(err)
	}
	installed, err := installer.Apply(src, actions, skillsDir)
	if err != nil {
		t.Fatal(err)
	}

	// Overwrite SKILL.md with junk.
	skillMD := filepath.Join(skillsDir, "golang", "SKILL.md")
	if err := os.WriteFile(skillMD, []byte("junk content"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Re-run Plan with current manifest: hash matches catalog → skip.
	mf := &manifest.Manifest{
		Version:   1,
		Installed: map[string]manifest.InstalledSkill{"golang": {Hash: installed["golang"].Hash, Profile: "tri"}},
	}
	actions2, err := installer.Plan(src, cat, mf, []string{"tri"})
	if err != nil {
		t.Fatal(err)
	}
	if len(actions2) != 1 || actions2[0].Type != installer.ActionSkip {
		t.Fatalf("expected 1 skip action, got %+v", actions2)
	}

	// Apply: disk hash differs from catalog hash → should repair.
	if _, err := installer.Apply(src, actions2, skillsDir); err != nil {
		t.Fatalf("Apply() error = %v", err)
	}

	data, err := os.ReadFile(skillMD)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) == "junk content" {
		t.Error("SKILL.md should have been restored to catalog content, but still has junk")
	}
	if string(data) != "# golang" {
		t.Errorf("SKILL.md content = %q, want %q", string(data), "# golang")
	}
}

func TestApplySkipUntouchedWhenClean(t *testing.T) {
	src := makeCatalog(t)
	cat := loadCat(t, src)
	skillsDir := t.TempDir()

	// First install.
	actions, err := installer.Plan(src, cat, nil, []string{"tri"})
	if err != nil {
		t.Fatal(err)
	}
	installed, err := installer.Apply(src, actions, skillsDir)
	if err != nil {
		t.Fatal(err)
	}

	skillMD := filepath.Join(skillsDir, "golang", "SKILL.md")
	beforeBytes, err := os.ReadFile(skillMD)
	if err != nil {
		t.Fatal(err)
	}

	// Re-run Plan+Apply with matching manifest: disk is clean, should skip efficiently.
	mf := &manifest.Manifest{
		Version:   1,
		Installed: map[string]manifest.InstalledSkill{"golang": {Hash: installed["golang"].Hash, Profile: "tri"}},
	}
	actions2, err := installer.Plan(src, cat, mf, []string{"tri"})
	if err != nil {
		t.Fatal(err)
	}
	installed2, err := installer.Apply(src, actions2, skillsDir)
	if err != nil {
		t.Fatalf("Apply() error = %v", err)
	}

	afterBytes, err := os.ReadFile(skillMD)
	if err != nil {
		t.Fatal(err)
	}
	if string(afterBytes) != string(beforeBytes) {
		t.Errorf("SKILL.md content changed on clean skip: before=%q after=%q", beforeBytes, afterBytes)
	}
	if installed2["golang"].Hash != installed["golang"].Hash {
		t.Errorf("installed hash changed on clean skip: got %q", installed2["golang"].Hash)
	}
}

func TestApplyPreservesExecBit(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "catalog.json"), []byte(`{
		"name": "exectest",
		"profiles": {
			"exec": {"description": "exec", "skills": ["execskill"]}
		}
	}`), 0o644); err != nil {
		t.Fatal(err)
	}
	skillDir := filepath.Join(root, "skills", "execskill")
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte("# execskill"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(skillDir, "run.sh"), []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}

	src := catalog.LocalDir{Root: root}
	cat, err := src.Load()
	if err != nil {
		t.Fatal(err)
	}
	actions, err := installer.Plan(src, cat, nil, []string{"exec"})
	if err != nil {
		t.Fatal(err)
	}
	skillsDir := t.TempDir()
	if _, err := installer.Apply(src, actions, skillsDir); err != nil {
		t.Fatalf("Apply() error = %v", err)
	}

	fi, err := os.Stat(filepath.Join(skillsDir, "execskill", "run.sh"))
	if err != nil {
		t.Fatal(err)
	}
	m := fi.Mode().Perm()
	if m&0o100 == 0 {
		t.Errorf("run.sh mode = %04o, exec bit not preserved", m)
	}
}

func TestApplyUpdateReplacesStaleFiles(t *testing.T) {
	src := makeCatalog(t)
	cat := loadCat(t, src)
	skillsDir := t.TempDir()

	// Pre-existing stale install with an extra file that must disappear.
	stale := filepath.Join(skillsDir, "golang")
	os.MkdirAll(stale, 0o755)
	os.WriteFile(filepath.Join(stale, "basura.md"), []byte("viejo"), 0o644)

	actions, err := installer.Plan(src, cat, nil, []string{"tri"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := installer.Apply(src, actions, skillsDir); err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(filepath.Join(stale, "basura.md")); !os.IsNotExist(err) {
		t.Error("stale file should have been removed (clean copy)")
	}
	if _, err := os.Stat(filepath.Join(stale, "SKILL.md")); err != nil {
		t.Error("SKILL.md missing after update")
	}
}
