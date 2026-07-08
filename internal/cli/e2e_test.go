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

	// 1. Onboarding: install with both profiles
	out, err := runAndes(t, home,
		"install", "--catalog", fixtureCatalog(t), "--profiles", "andespath-core,tri-fleet", "--yes")
	if err != nil {
		t.Fatalf("init: %v\n%s", err, out)
	}

	// 2. list shows everything installed
	out, err = runAndes(t, home, "list")
	if err != nil {
		t.Fatalf("list: %v\n%s", err, out)
	}
	if strings.Contains(out, "✗") {
		t.Errorf("after full init there should be no uninstalled skills:\n%s", out)
	}

	// 3. doctor healthy
	if out, err = runAndes(t, home, "doctor"); err != nil {
		t.Fatalf("healthy doctor: %v\n%s", err, out)
	}

	// 4. simulate local edit → doctor catches modified
	skillMD := filepath.Join(home, ".claude", "skills", "golang", "SKILL.md")
	os.WriteFile(skillMD, []byte("# manually edited"), 0o644)

	out, err = runAndes(t, home, "doctor")
	if err == nil {
		t.Errorf("doctor should detect the modified skill:\n%s", out)
	}
	if !strings.Contains(out, "modified") {
		t.Errorf("doctor did not classify as modified:\n%s", out)
	}

	// 5. re-install repairs
	if out, err = runAndes(t, home,
		"install", "--profiles", "andespath-core,tri-fleet", "--yes"); err != nil {
		t.Fatalf("re-install: %v\n%s", err, out)
	}
	if out, err = runAndes(t, home, "doctor"); err != nil {
		t.Fatalf("doctor after re-install should be healthy: %v\n%s", err, out)
	}
}
