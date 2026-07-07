package cli_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDoctorHealthyAfterInit(t *testing.T) {
	home := t.TempDir()
	if _, err := runAndes(t, home,
		"init", "--catalog", fixtureCatalog(t), "--profiles", "tri-fleet", "--yes"); err != nil {
		t.Fatal(err)
	}

	out, err := runAndes(t, home, "doctor")
	if err != nil {
		t.Fatalf("doctor sano debería salir 0: %v\n%s", err, out)
	}
	if !strings.Contains(out, "✓") {
		t.Errorf("doctor no reporta salud:\n%s", out)
	}
}

func TestDoctorDetectsMissingSkill(t *testing.T) {
	home := t.TempDir()
	if _, err := runAndes(t, home,
		"init", "--catalog", fixtureCatalog(t), "--profiles", "tri-fleet", "--yes"); err != nil {
		t.Fatal(err)
	}
	os.RemoveAll(filepath.Join(home, ".claude", "skills", "golang"))

	out, err := runAndes(t, home, "doctor")
	if err == nil {
		t.Errorf("doctor con skill faltante debería fallar (exit != 0):\n%s", out)
	}
	if !strings.Contains(out, "falta") {
		t.Errorf("doctor no reporta la skill faltante:\n%s", out)
	}
}

func TestDoctorWithoutManifest(t *testing.T) {
	home := t.TempDir()
	out, err := runAndes(t, home, "doctor")
	if err == nil {
		t.Error("doctor sin manifiesto debería fallar")
	}
	_ = out
}

func TestDoctorInaccessibleCatalog(t *testing.T) {
	home := t.TempDir()
	// init against a catalog that later disappears
	tmpCat := filepath.Join(t.TempDir(), "cat")
	copyFixture(t, tmpCat)
	if _, err := runAndes(t, home,
		"init", "--catalog", tmpCat, "--profiles", "tri-fleet", "--yes"); err != nil {
		t.Fatal(err)
	}
	os.RemoveAll(tmpCat)

	out, err := runAndes(t, home, "doctor")
	if err == nil {
		t.Errorf("doctor con catálogo inaccesible debería fallar:\n%s", out)
	}
}

// copyFixture clones the fixture catalog so tests can mutate/delete it.
func copyFixture(t *testing.T, dst string) {
	t.Helper()
	src := fixtureCatalog(t)
	err := filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(src, path)
		target := filepath.Join(dst, rel)
		if info.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(target, data, 0o644)
	})
	if err != nil {
		t.Fatal(err)
	}
}
