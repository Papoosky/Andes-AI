package cli_test

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/andespath/andes-ai/internal/cli"
	"github.com/andespath/andes-ai/internal/manifest"
)

func fixtureCatalog(t *testing.T) string {
	t.Helper()
	abs, err := filepath.Abs("../../catalog")
	if err != nil {
		t.Fatal(err)
	}
	return abs
}

// runAndes executes the CLI with a temp HOME and returns combined output.
func runAndes(t *testing.T, home string, args ...string) (string, error) {
	t.Helper()
	t.Setenv("HOME", home)
	root := cli.NewRootCmd()
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs(args)
	err := root.Execute()
	return out.String(), err
}

func TestInitInstallsProfiles(t *testing.T) {
	home := t.TempDir()

	out, err := runAndes(t, home,
		"install", "--catalog", fixtureCatalog(t), "--profiles", "andespath-core,tri-fleet", "--yes")
	if err != nil {
		t.Fatalf("init error = %v\noutput:\n%s", err, out)
	}

	// Skills on disk
	for _, id := range []string{"git-conventions", "code-review", "golang"} {
		p := filepath.Join(home, ".claude", "skills", id, "SKILL.md")
		if _, err := os.Stat(p); err != nil {
			t.Errorf("installed skill missing: %s", p)
		}
	}

	// Manifest written
	m, err := manifest.Load(filepath.Join(home, ".claude", "andes.json"))
	if err != nil || m == nil {
		t.Fatalf("manifest not written: %v", err)
	}
	if m.Version != 1 || len(m.Installed) != 3 {
		t.Errorf("manifest = %+v, want version 1 with 3 skills", m)
	}
	if m.Catalog.Type != "local" {
		t.Errorf("catalog.type = %q, want local", m.Catalog.Type)
	}
}

func TestInitIsIdempotent(t *testing.T) {
	home := t.TempDir()
	args := []string{"install", "--catalog", fixtureCatalog(t), "--profiles", "tri-fleet", "--yes"}

	if _, err := runAndes(t, home, args...); err != nil {
		t.Fatal(err)
	}
	out, err := runAndes(t, home, args...)
	if err != nil {
		t.Fatalf("second init failed: %v", err)
	}
	if !bytes.Contains([]byte(out), []byte("Everything is already up to date.")) {
		t.Errorf("second init should report 'Everything is already up to date.':\n%s", out)
	}
}

func TestInitRemembersCatalogPath(t *testing.T) {
	home := t.TempDir()
	if _, err := runAndes(t, home,
		"install", "--catalog", fixtureCatalog(t), "--profiles", "tri-fleet", "--yes"); err != nil {
		t.Fatal(err)
	}
	// Second run without --catalog: must reuse the manifest's path.
	if _, err := runAndes(t, home, "install", "--profiles", "tri-fleet", "--yes"); err != nil {
		t.Fatalf("install without --catalog with previous manifest failed: %v", err)
	}
}

func TestInitNonInteractiveRequiresFlags(t *testing.T) {
	home := t.TempDir()
	// --yes without --catalog and no previous manifest: actionable error.
	if _, err := runAndes(t, home, "install", "--yes"); err == nil {
		t.Error("install --yes without catalog should fail with actionable error")
	}
}

func TestInitRequiresProfiles(t *testing.T) {
	home := t.TempDir()
	// --catalog given but no --profiles and no previous manifest: actionable error.
	if _, err := runAndes(t, home,
		"install", "--catalog", fixtureCatalog(t), "--yes"); err == nil {
		t.Error("install --yes without profiles should fail with actionable error")
	}
}

func TestInitWithoutYesAborts(t *testing.T) {
	home := t.TempDir()
	out, err := runAndes(t, home,
		"install", "--catalog", fixtureCatalog(t), "--profiles", "tri-fleet")
	if err == nil {
		t.Fatal("install without --yes should abort with explicit error")
	}
	// Plan must still be shown before aborting, and nothing installed.
	if !bytes.Contains([]byte(out), []byte("Plan:")) {
		t.Errorf("install without --yes should show the plan before aborting:\n%s", out)
	}
	if _, statErr := os.Stat(filepath.Join(home, ".claude", "skills", "golang")); statErr == nil {
		t.Error("install without --yes should not install skills")
	}
	if _, statErr := os.Stat(filepath.Join(home, ".claude", "andes.json")); statErr == nil {
		t.Error("install without --yes should not write the manifest")
	}
}

func TestInitFromGitCatalog(t *testing.T) {
	home := t.TempDir()
	repo, _ := gitFixture(t)
	fileURL := "file://" + repo

	out, err := runAndes(t, home,
		"install", "--catalog", fileURL, "--profiles", "tri-fleet", "--yes")
	if err != nil {
		t.Fatalf("install from git: %v\n%s", err, out)
	}

	// Skills installed.
	if _, err := os.Stat(filepath.Join(home, ".claude", "skills", "golang", "SKILL.md")); err != nil {
		t.Errorf("skill not installed: %v", err)
	}
	// Mirror created under ~/.andes/catalog.
	if _, err := os.Stat(filepath.Join(home, ".andes", "catalog", "catalog", "catalog.json")); err != nil {
		t.Errorf("managed mirror missing: %v", err)
	}
	// Manifest has git type + 40-char ref.
	m, err := manifest.Load(filepath.Join(home, ".claude", "andes.json"))
	if err != nil || m == nil {
		t.Fatal(err)
	}
	if m.Catalog.Type != "git" || m.Catalog.URL != fileURL || len(m.Catalog.Ref) != 40 {
		t.Errorf("manifest catalog = %+v", m.Catalog)
	}
}

func TestInitDoesNotTouchForeignSkills(t *testing.T) {
	home := t.TempDir()
	foreign := filepath.Join(home, ".claude", "skills", "mi-skill-personal")
	os.MkdirAll(foreign, 0o755)
	os.WriteFile(filepath.Join(foreign, "SKILL.md"), []byte("# mine"), 0o644)

	if _, err := runAndes(t, home,
		"install", "--catalog", fixtureCatalog(t), "--profiles", "tri-fleet", "--yes"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(foreign, "SKILL.md")); err != nil {
		t.Error("install touched a personal skill not in the manifest")
	}
}
