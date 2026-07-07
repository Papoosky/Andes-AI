package doctor_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/andespath/andes-ai/internal/catalog"
	"github.com/andespath/andes-ai/internal/doctor"
	"github.com/andespath/andes-ai/internal/hashdir"
	"github.com/andespath/andes-ai/internal/installer"
	"github.com/andespath/andes-ai/internal/manifest"
)

// setup installs profile "tri" (skill golang) into a temp skillsDir and
// returns everything a doctor check needs.
func setup(t *testing.T) (catalog.LocalDir, *manifest.Manifest, string) {
	t.Helper()
	root := t.TempDir()
	os.WriteFile(filepath.Join(root, "catalog.json"), []byte(`{
		"name": "andespath",
		"profiles": {"tri": {"description": "tri", "skills": ["golang"]}}
	}`), 0o644)
	dir := filepath.Join(root, "skills", "golang")
	os.MkdirAll(dir, 0o755)
	os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte("# golang v1"), 0o644)

	src := catalog.LocalDir{Root: root}
	cat, err := src.Load()
	if err != nil {
		t.Fatal(err)
	}
	skillsDir := t.TempDir()
	actions, err := installer.Plan(src, cat, nil, []string{"tri"})
	if err != nil {
		t.Fatal(err)
	}
	installed, err := installer.Apply(src, actions, skillsDir)
	if err != nil {
		t.Fatal(err)
	}
	m := &manifest.Manifest{Version: 1, Profiles: []string{"tri"}, Installed: installed}
	return src, m, skillsDir
}

func TestCheckAllHealthy(t *testing.T) {
	src, m, skillsDir := setup(t)
	findings, err := doctor.Check(src, m, skillsDir)
	if err != nil {
		t.Fatal(err)
	}
	if len(findings) != 1 || findings[0].Status != doctor.StatusOK {
		t.Errorf("findings = %+v, want 1 OK", findings)
	}
}

func TestCheckMissingOnDisk(t *testing.T) {
	src, m, skillsDir := setup(t)
	os.RemoveAll(filepath.Join(skillsDir, "golang"))

	findings, err := doctor.Check(src, m, skillsDir)
	if err != nil {
		t.Fatal(err)
	}
	if len(findings) != 1 || findings[0].Status != doctor.StatusMissing {
		t.Errorf("findings = %+v, want 1 falta", findings)
	}
}

func TestCheckLocallyModified(t *testing.T) {
	src, m, skillsDir := setup(t)
	os.WriteFile(filepath.Join(skillsDir, "golang", "SKILL.md"), []byte("# editado a mano"), 0o644)

	findings, err := doctor.Check(src, m, skillsDir)
	if err != nil {
		t.Fatal(err)
	}
	if len(findings) != 1 || findings[0].Status != doctor.StatusModified {
		t.Errorf("findings = %+v, want 1 modificada", findings)
	}
}

func TestCheckOutdated(t *testing.T) {
	src, m, skillsDir := setup(t)
	// Catalog moves forward; disk + manifest stay at v1.
	os.WriteFile(filepath.Join(src.Root, "skills", "golang", "SKILL.md"), []byte("# golang v2"), 0o644)

	findings, _ := doctor.Check(src, m, skillsDir)
	if len(findings) != 1 || findings[0].Status != doctor.StatusOutdated {
		t.Errorf("findings = %+v, want 1 desactualizada", findings)
	}
}

func TestCheckNeverWrites(t *testing.T) {
	src, m, skillsDir := setup(t)
	before, err := hashdir.Hash(skillsDir)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := doctor.Check(src, m, skillsDir); err != nil {
		t.Fatal(err)
	}
	after, _ := hashdir.Hash(skillsDir)
	if before != after {
		t.Error("Check() modificó el disco — jamás debe escribir")
	}
}
