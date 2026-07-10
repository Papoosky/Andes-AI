package cli_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestDoctorHealthyAfterInstall(t *testing.T) {
	home := t.TempDir()
	if _, err := runAndes(t, home,
		"install", "--catalog", fixtureCatalog(t), "--profiles", "tri-fleet", "--yes"); err != nil {
		t.Fatal(err)
	}

	out, err := runAndes(t, home, "doctor")
	if err != nil {
		t.Fatalf("healthy doctor should exit 0: %v\n%s", err, out)
	}
	if !strings.Contains(out, "✓") {
		t.Errorf("doctor does not report health:\n%s", out)
	}
}

func TestDoctorDetectsMissingSkill(t *testing.T) {
	home := t.TempDir()
	if _, err := runAndes(t, home,
		"install", "--catalog", fixtureCatalog(t), "--profiles", "tri-fleet", "--yes"); err != nil {
		t.Fatal(err)
	}
	os.RemoveAll(filepath.Join(home, ".claude", "skills", "golang"))

	out, err := runAndes(t, home, "doctor")
	if err == nil {
		t.Errorf("doctor with missing skill should fail (exit != 0):\n%s", out)
	}
	if !strings.Contains(out, "missing") {
		t.Errorf("doctor does not report the missing skill:\n%s", out)
	}
}

func TestDoctorDoesNotWarnWhenGitManifestRefMatchesMirror(t *testing.T) {
	home := t.TempDir()
	repo, _ := gitFixture(t)

	if _, err := runAndes(t, home,
		"install", "--catalog", "file://"+repo, "--profiles", "tri-fleet", "--yes"); err != nil {
		t.Fatal(err)
	}

	out, err := runAndes(t, home, "doctor")
	if err != nil {
		t.Fatalf("doctor should remain healthy: %v\n%s", err, out)
	}
	if strings.Contains(out, "Warning: local catalog mirror is at") {
		t.Errorf("doctor should not warn when manifest ref matches mirror HEAD:\n%s", out)
	}
}

func TestDoctorWarnsWhenGitMirrorHeadDiffersFromManifestRef(t *testing.T) {
	home := t.TempDir()
	repo, commit := gitFixture(t)
	url := "file://" + repo

	if _, err := runAndes(t, home,
		"install", "--catalog", url, "--profiles", "tri-fleet", "--yes"); err != nil {
		t.Fatal(err)
	}

	// Advance the upstream repo without changing catalog content, then advance the
	// local mirror outside `andes update`. Doctor should stay read-only and healthy,
	// but warn that it is reading a different catalog HEAD than the installed ref.
	if err := os.WriteFile(filepath.Join(repo, "README.md"), []byte("metadata only"), 0o644); err != nil {
		t.Fatal(err)
	}
	commit("metadata only")

	mirror := filepath.Join(home, ".andes", "catalog")
	for _, args := range [][]string{
		{"-C", mirror, "fetch", "origin"},
		{"-C", mirror, "reset", "--hard", "origin/HEAD"},
	} {
		cmd := exec.Command("git", args...)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}

	out, err := runAndes(t, home, "doctor")
	if err != nil {
		t.Fatalf("doctor should remain healthy for metadata-only catalog ref drift: %v\n%s", err, out)
	}
	if !strings.Contains(out, "Warning: local catalog mirror is at") ||
		!strings.Contains(out, "manifest was installed from") {
		t.Errorf("doctor should warn about catalog ref drift:\n%s", out)
	}
}

func TestDoctorWithoutManifest(t *testing.T) {
	home := t.TempDir()
	out, err := runAndes(t, home, "doctor")
	if err == nil {
		t.Error("doctor without manifest should fail")
	}
	_ = out
}

func TestDoctorInaccessibleCatalog(t *testing.T) {
	home := t.TempDir()
	// install against a catalog that later disappears
	tmpCat := filepath.Join(t.TempDir(), "cat")
	copyFixture(t, tmpCat)
	if _, err := runAndes(t, home,
		"install", "--catalog", tmpCat, "--profiles", "tri-fleet", "--yes"); err != nil {
		t.Fatal(err)
	}
	os.RemoveAll(tmpCat)

	out, err := runAndes(t, home, "doctor")
	if err == nil {
		t.Errorf("doctor with inaccessible catalog should fail:\n%s", out)
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
