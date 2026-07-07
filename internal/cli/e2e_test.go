package cli_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestEndToEnd walks the full demo: init → list → doctor healthy →
// break something → doctor catches it.
func TestEndToEnd(t *testing.T) {
	home := t.TempDir()

	// 1. Onboarding: init with both profiles
	out, err := runAndes(t, home,
		"init", "--catalog", fixtureCatalog(t), "--profiles", "andespath-core,tri-fleet", "--yes")
	if err != nil {
		t.Fatalf("init: %v\n%s", err, out)
	}

	// 2. list shows everything installed
	out, err = runAndes(t, home, "list")
	if err != nil {
		t.Fatalf("list: %v\n%s", err, out)
	}
	if strings.Contains(out, "✗") {
		t.Errorf("tras init completo no debería haber skills sin instalar:\n%s", out)
	}

	// 3. doctor healthy
	if out, err = runAndes(t, home, "doctor"); err != nil {
		t.Fatalf("doctor sano: %v\n%s", err, out)
	}

	// 4. simulate local edit → doctor catches modified
	skillMD := filepath.Join(home, ".claude", "skills", "golang", "SKILL.md")
	os.WriteFile(skillMD, []byte("# tocado a mano"), 0o644)

	out, err = runAndes(t, home, "doctor")
	if err == nil {
		t.Errorf("doctor debería detectar la skill modificada:\n%s", out)
	}
	if !strings.Contains(out, "modificada") {
		t.Errorf("doctor no clasificó como modificada:\n%s", out)
	}

	// 5. re-init repairs
	if out, err = runAndes(t, home,
		"init", "--profiles", "andespath-core,tri-fleet", "--yes"); err != nil {
		t.Fatalf("re-init: %v\n%s", err, out)
	}
	if out, err = runAndes(t, home, "doctor"); err != nil {
		t.Fatalf("doctor tras re-init debería estar sano: %v\n%s", err, out)
	}
}
