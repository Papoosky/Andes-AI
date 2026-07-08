package cli_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/andespath/andes-ai/internal/manifest"
)

// TestGitLifecycle walks the full v2 story: init from the company repo →
// upstream change → update detects and applies → doctor healthy → ref advanced.
func TestGitLifecycle(t *testing.T) {
	home := t.TempDir()
	repo, commit := gitFixture(t)
	url := "file://" + repo

	// 1. Fresh dev: install from git.
	out, err := runAndes(t, home,
		"install", "--catalog", url, "--profiles", "andespath-core,tri-fleet", "--yes")
	if err != nil {
		t.Fatalf("init: %v\n%s", err, out)
	}

	// 2. List shows installed skills.
	if out, err = runAndes(t, home, "list"); err != nil {
		t.Fatalf("list: %v\n%s", err, out)
	}

	// 3. Doctor healthy against the mirror.
	if out, err = runAndes(t, home, "doctor"); err != nil {
		t.Fatalf("doctor: %v\n%s", err, out)
	}
	m1, _ := manifest.Load(filepath.Join(home, ".claude", "andes.json"))

	// 4. Upstream: a skill changes.
	os.WriteFile(filepath.Join(repo, "catalog", "skills", "code-review", "SKILL.md"),
		[]byte("# code review v2"), 0o644)
	commit("update code-review")

	// 5. Update pulls it.
	out, err = runAndes(t, home, "update", "--yes")
	if err != nil {
		t.Fatalf("update: %v\n%s", err, out)
	}
	if !strings.Contains(out, "code-review") {
		t.Errorf("update plan should mention code-review:\n%s", out)
	}
	data, _ := os.ReadFile(filepath.Join(home, ".claude", "skills", "code-review", "SKILL.md"))
	if string(data) != "# code review v2" {
		t.Errorf("skill not refreshed: %q", data)
	}

	// 6. Ref advanced; doctor still healthy.
	m2, _ := manifest.Load(filepath.Join(home, ".claude", "andes.json"))
	if m2.Catalog.Ref == m1.Catalog.Ref {
		t.Error("manifest ref did not advance after update")
	}
	if out, err = runAndes(t, home, "doctor"); err != nil {
		t.Fatalf("doctor after update: %v\n%s", err, out)
	}
}
