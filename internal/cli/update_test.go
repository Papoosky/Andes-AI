package cli_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/andespath/andes-ai/internal/manifest"
)

func TestUpdateNoManifest(t *testing.T) {
	home := t.TempDir()
	if _, err := runAndes(t, home, "update", "--yes"); err == nil {
		t.Error("update without manifest should fail with actionable error")
	}
}

func TestUpdateLocalCatalogNothingToDo(t *testing.T) {
	home := t.TempDir()
	if _, err := runAndes(t, home,
		"init", "--catalog", fixtureCatalog(t), "--profiles", "tri-fleet", "--yes"); err != nil {
		t.Fatal(err)
	}
	out, err := runAndes(t, home, "update", "--yes")
	if err == nil || !strings.Contains(err.Error()+out, "local catalog") {
		t.Errorf("update on local catalog should explain there is nothing to update: %v\n%s", err, out)
	}
}

func TestUpdateAlreadyUpToDate(t *testing.T) {
	home := t.TempDir()
	repo, _ := gitFixture(t)
	if _, err := runAndes(t, home,
		"init", "--catalog", "file://"+repo, "--profiles", "tri-fleet", "--yes"); err != nil {
		t.Fatal(err)
	}
	out, err := runAndes(t, home, "update", "--yes")
	if err != nil {
		t.Fatalf("update: %v\n%s", err, out)
	}
	if !strings.Contains(out, "Already up to date") {
		t.Errorf("want 'Already up to date':\n%s", out)
	}
}

func TestUpdatePullsNewSkillVersion(t *testing.T) {
	home := t.TempDir()
	repo, commit := gitFixture(t)
	if _, err := runAndes(t, home,
		"init", "--catalog", "file://"+repo, "--profiles", "tri-fleet", "--yes"); err != nil {
		t.Fatal(err)
	}

	// Upstream change.
	os.WriteFile(filepath.Join(repo, "catalog", "skills", "golang", "SKILL.md"), []byte("# golang v2"), 0o644)
	commit("bump golang")

	out, err := runAndes(t, home, "update", "--yes")
	if err != nil {
		t.Fatalf("update: %v\n%s", err, out)
	}
	if !strings.Contains(out, "update") || !strings.Contains(out, "golang") {
		t.Errorf("plan should show golang as update:\n%s", out)
	}

	// Installed skill actually refreshed.
	data, _ := os.ReadFile(filepath.Join(home, ".claude", "skills", "golang", "SKILL.md"))
	if string(data) != "# golang v2" {
		t.Errorf("installed skill not refreshed: %q", data)
	}
	// Manifest ref advanced.
	m, _ := manifest.Load(filepath.Join(home, ".claude", "andes.json"))
	before := m.Catalog.Ref
	if before == "" {
		t.Fatal("manifest ref empty")
	}
	out2, err := runAndes(t, home, "update", "--yes")
	if err != nil || !strings.Contains(out2, "Already up to date") {
		t.Errorf("second update should be up to date: %v\n%s", err, out2)
	}
}
