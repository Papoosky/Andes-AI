package installer_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/andespath/andes-ai/internal/catalog"
	"github.com/andespath/andes-ai/internal/hashdir"
	"github.com/andespath/andes-ai/internal/installer"
	"github.com/andespath/andes-ai/internal/manifest"
)

// makeCatalog builds a temp catalog with two profiles and three skills.
func makeCatalog(t *testing.T) catalog.LocalDir {
	t.Helper()
	root := t.TempDir()
	os.WriteFile(filepath.Join(root, "catalog.json"), []byte(`{
		"name": "andespath",
		"profiles": {
			"core": {"description": "base", "skills": ["git-conventions", "code-review"]},
			"tri":  {"description": "tri",  "skills": ["golang"]}
		}
	}`), 0o644)
	for _, id := range []string{"git-conventions", "code-review", "golang"} {
		dir := filepath.Join(root, "skills", id)
		os.MkdirAll(dir, 0o755)
		os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte("# "+id), 0o644)
	}
	return catalog.LocalDir{Root: root}
}

func loadCat(t *testing.T, src catalog.LocalDir) *catalog.Catalog {
	t.Helper()
	c, err := src.Load()
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func TestPlanFirstInstall(t *testing.T) {
	src := makeCatalog(t)
	actions, err := installer.Plan(src, loadCat(t, src), nil, []string{"core", "tri"})
	if err != nil {
		t.Fatalf("Plan() error = %v", err)
	}
	if len(actions) != 3 {
		t.Fatalf("len(actions) = %d, want 3", len(actions))
	}
	// Ordered by SkillID: code-review, git-conventions, golang
	wantOrder := []string{"code-review", "git-conventions", "golang"}
	for i, want := range wantOrder {
		if actions[i].SkillID != want {
			t.Errorf("actions[%d].SkillID = %q, want %q", i, actions[i].SkillID, want)
		}
		if actions[i].Type != installer.ActionInstall {
			t.Errorf("actions[%d].Type = %q, want install", i, actions[i].Type)
		}
		if actions[i].Hash == "" {
			t.Errorf("actions[%d].Hash vacío", i)
		}
	}
	if actions[2].Profile != "tri" {
		t.Errorf("golang profile = %q, want tri", actions[2].Profile)
	}
}

func TestPlanIdempotentSkip(t *testing.T) {
	src := makeCatalog(t)
	cat := loadCat(t, src)

	h, err := hashdir.Hash(src.SkillPath("golang"))
	if err != nil {
		t.Fatal(err)
	}
	m := &manifest.Manifest{
		Version:   1,
		Installed: map[string]manifest.InstalledSkill{"golang": {Hash: h, Profile: "tri"}},
	}

	actions, err := installer.Plan(src, cat, m, []string{"tri"})
	if err != nil {
		t.Fatal(err)
	}
	if len(actions) != 1 || actions[0].Type != installer.ActionSkip {
		t.Errorf("actions = %+v, want 1 skip", actions)
	}
}

func TestPlanUpdateOnHashMismatch(t *testing.T) {
	src := makeCatalog(t)
	cat := loadCat(t, src)

	m := &manifest.Manifest{
		Version:   1,
		Installed: map[string]manifest.InstalledSkill{"golang": {Hash: "sha256:viejo", Profile: "tri"}},
	}

	actions, err := installer.Plan(src, cat, m, []string{"tri"})
	if err != nil {
		t.Fatal(err)
	}
	if len(actions) != 1 || actions[0].Type != installer.ActionUpdate {
		t.Errorf("actions = %+v, want 1 update", actions)
	}
}

func TestPlanUnknownProfile(t *testing.T) {
	src := makeCatalog(t)
	_, err := installer.Plan(src, loadCat(t, src), nil, []string{"fantasma"})
	if err == nil {
		t.Error("Plan() con perfil inexistente debería fallar")
	}
}
