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
	abs, err := filepath.Abs("../../testdata/catalog")
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
		"init", "--catalog", fixtureCatalog(t), "--profiles", "andespath-core,tri-fleet", "--yes")
	if err != nil {
		t.Fatalf("init error = %v\noutput:\n%s", err, out)
	}

	// Skills on disk
	for _, id := range []string{"git-conventions", "code-review", "golang"} {
		p := filepath.Join(home, ".claude", "skills", id, "SKILL.md")
		if _, err := os.Stat(p); err != nil {
			t.Errorf("falta skill instalada: %s", p)
		}
	}

	// Manifest written
	m, err := manifest.Load(filepath.Join(home, ".claude", "andes.json"))
	if err != nil || m == nil {
		t.Fatalf("manifiesto no escrito: %v", err)
	}
	if m.Version != 1 || len(m.Installed) != 3 {
		t.Errorf("manifiesto = %+v, want version 1 con 3 skills", m)
	}
	if m.Catalog.Type != "local" {
		t.Errorf("catalog.type = %q, want local", m.Catalog.Type)
	}
}

func TestInitIsIdempotent(t *testing.T) {
	home := t.TempDir()
	args := []string{"init", "--catalog", fixtureCatalog(t), "--profiles", "tri-fleet", "--yes"}

	if _, err := runAndes(t, home, args...); err != nil {
		t.Fatal(err)
	}
	out, err := runAndes(t, home, args...)
	if err != nil {
		t.Fatalf("segundo init falló: %v", err)
	}
	if !bytes.Contains([]byte(out), []byte("sin cambios")) {
		t.Errorf("segundo init debería reportar 'sin cambios':\n%s", out)
	}
}

func TestInitRemembersCatalogPath(t *testing.T) {
	home := t.TempDir()
	if _, err := runAndes(t, home,
		"init", "--catalog", fixtureCatalog(t), "--profiles", "tri-fleet", "--yes"); err != nil {
		t.Fatal(err)
	}
	// Second run without --catalog: must reuse the manifest's path.
	if _, err := runAndes(t, home, "init", "--profiles", "tri-fleet", "--yes"); err != nil {
		t.Fatalf("init sin --catalog con manifiesto previo falló: %v", err)
	}
}

func TestInitNonInteractiveRequiresFlags(t *testing.T) {
	home := t.TempDir()
	// --yes without --catalog and no previous manifest: actionable error.
	if _, err := runAndes(t, home, "init", "--yes"); err == nil {
		t.Error("init --yes sin catálogo debería fallar con error accionable")
	}
}

func TestInitRequiresProfiles(t *testing.T) {
	home := t.TempDir()
	// --catalog given but no --profiles and no previous manifest: actionable error.
	if _, err := runAndes(t, home,
		"init", "--catalog", fixtureCatalog(t), "--yes"); err == nil {
		t.Error("init --yes sin perfiles debería fallar con error accionable")
	}
}

func TestInitWithoutYesAborts(t *testing.T) {
	home := t.TempDir()
	out, err := runAndes(t, home,
		"init", "--catalog", fixtureCatalog(t), "--profiles", "tri-fleet")
	if err == nil {
		t.Fatal("init sin --yes debería abortar con error explícito")
	}
	// Plan must still be shown before aborting, and nothing installed.
	if !bytes.Contains([]byte(out), []byte("Plan:")) {
		t.Errorf("init sin --yes debería mostrar el plan antes de abortar:\n%s", out)
	}
	if _, statErr := os.Stat(filepath.Join(home, ".claude", "skills", "golang")); statErr == nil {
		t.Error("init sin --yes no debería instalar skills")
	}
	if _, statErr := os.Stat(filepath.Join(home, ".claude", "andes.json")); statErr == nil {
		t.Error("init sin --yes no debería escribir el manifiesto")
	}
}

func TestInitDoesNotTouchForeignSkills(t *testing.T) {
	home := t.TempDir()
	foreign := filepath.Join(home, ".claude", "skills", "mi-skill-personal")
	os.MkdirAll(foreign, 0o755)
	os.WriteFile(filepath.Join(foreign, "SKILL.md"), []byte("# mía"), 0o644)

	if _, err := runAndes(t, home,
		"init", "--catalog", fixtureCatalog(t), "--profiles", "tri-fleet", "--yes"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(foreign, "SKILL.md")); err != nil {
		t.Error("init tocó una skill personal ajena al manifiesto")
	}
}
