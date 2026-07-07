package installer_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/andespath/andes-ai/internal/installer"
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
			t.Errorf("falta %s tras Apply", skillMD)
		}
		if installed[id].Hash == "" {
			t.Errorf("installed[%q] sin hash", id)
		}
	}
}

func TestApplySkipDoesNotTouchDisk(t *testing.T) {
	src := makeCatalog(t)
	skillsDir := t.TempDir()

	// A skip action for a skill NOT on disk: Apply must not create it,
	// but must still return its manifest entry.
	actions := []installer.Action{
		{SkillID: "golang", Type: installer.ActionSkip, Profile: "tri", Hash: "sha256:x"},
	}
	installed, err := installer.Apply(src, actions, skillsDir)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(skillsDir, "golang")); !os.IsNotExist(err) {
		t.Error("skip no debería tocar el disco")
	}
	if installed["golang"].Hash != "sha256:x" {
		t.Errorf("skip debe conservar la entrada del manifiesto, got %+v", installed["golang"])
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
		t.Error("el archivo viejo debería haber sido eliminado (copia limpia)")
	}
	if _, err := os.Stat(filepath.Join(stale, "SKILL.md")); err != nil {
		t.Error("falta SKILL.md tras update")
	}
}
