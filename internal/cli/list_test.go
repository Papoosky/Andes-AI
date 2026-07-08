package cli_test

import (
	"strings"
	"testing"
)

func TestListWithoutManifestShowsCatalogAndHint(t *testing.T) {
	home := t.TempDir()
	out, err := runAndes(t, home, "list", "--catalog", fixtureCatalog(t))
	if err != nil {
		t.Fatalf("list error = %v\n%s", err, out)
	}
	for _, want := range []string{"andespath-core", "tri-fleet", "git-conventions", "not installed", "andes init"} {
		if !strings.Contains(out, want) {
			t.Errorf("list output does not contain %q:\n%s", want, out)
		}
	}
}

func TestListAfterInitShowsInstalled(t *testing.T) {
	home := t.TempDir()
	if _, err := runAndes(t, home,
		"init", "--catalog", fixtureCatalog(t), "--profiles", "tri-fleet", "--yes"); err != nil {
		t.Fatal(err)
	}

	out, err := runAndes(t, home, "list")
	if err != nil {
		t.Fatalf("list error = %v\n%s", err, out)
	}
	if !strings.Contains(out, "✓") {
		t.Errorf("list does not show golang as installed:\n%s", out)
	}
	// core profile not installed → its skills show as not installed
	if !strings.Contains(out, "✗") {
		t.Errorf("list does not show uninstalled skills:\n%s", out)
	}
}

func TestListWithoutCatalogAnywhereFails(t *testing.T) {
	home := t.TempDir()
	if _, err := runAndes(t, home, "list"); err == nil {
		t.Error("list without catalog or manifest should fail with actionable error")
	}
}
