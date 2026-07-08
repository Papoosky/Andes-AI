package catalog_test

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/andespath/andes-ai/internal/catalog"
)

// gitFixture creates a REAL git repo in a temp dir whose layout mirrors the
// company repo: a catalog/ subdir with catalog.json and skills. Returns the
// repo path (usable as a clone URL) and a commit helper.
func gitFixture(t *testing.T) (repo string, commit func(msg string)) {
	t.Helper()
	repo = t.TempDir()

	run := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = repo
		cmd.Env = append(os.Environ(),
			"GIT_AUTHOR_NAME=test", "GIT_AUTHOR_EMAIL=t@t",
			"GIT_COMMITTER_NAME=test", "GIT_COMMITTER_EMAIL=t@t")
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}

	// Seed: copy the production catalog into catalog/ inside the repo.
	src := fixtureDir(t) // existing helper → ../../catalog
	if err := os.MkdirAll(filepath.Join(repo, "catalog"), 0o755); err != nil {
		t.Fatal(err)
	}
	copyTree(t, src, filepath.Join(repo, "catalog"))

	run("init", "--initial-branch=main")
	run("add", "-A")
	run("commit", "-m", "seed catalog")

	commit = func(msg string) {
		t.Helper()
		run("add", "-A")
		run("commit", "-m", msg)
	}
	return repo, commit
}

func copyTree(t *testing.T, src, dst string) {
	t.Helper()
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
		return os.WriteFile(target, data, info.Mode().Perm())
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestGitRepoEnsureClones(t *testing.T) {
	repo, _ := gitFixture(t)
	g := catalog.GitRepo{URL: repo, Dir: filepath.Join(t.TempDir(), "mirror")}

	if err := g.Ensure(); err != nil {
		t.Fatalf("Ensure() error = %v", err)
	}
	if _, err := os.Stat(filepath.Join(g.Dir, "catalog", "catalog.json")); err != nil {
		t.Errorf("mirror missing catalog.json after Ensure: %v", err)
	}
}

func TestGitRepoEnsureHealsCorruptMirror(t *testing.T) {
	repo, _ := gitFixture(t)
	dir := filepath.Join(t.TempDir(), "mirror")
	// Corrupt mirror: a dir that is NOT a git repo.
	if err := os.MkdirAll(filepath.Join(dir, "junk"), 0o755); err != nil {
		t.Fatal(err)
	}
	g := catalog.GitRepo{URL: repo, Dir: dir}

	if err := g.Ensure(); err != nil {
		t.Fatalf("Ensure() on corrupt mirror error = %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "catalog", "catalog.json")); err != nil {
		t.Errorf("mirror not re-cloned: %v", err)
	}
}

func TestGitRepoHeads(t *testing.T) {
	repo, commit := gitFixture(t)
	g := catalog.GitRepo{URL: repo, Dir: filepath.Join(t.TempDir(), "mirror")}
	if err := g.Ensure(); err != nil {
		t.Fatal(err)
	}

	local, err := g.LocalHead()
	if err != nil || len(local) != 40 {
		t.Fatalf("LocalHead() = %q, %v — want 40-char SHA", local, err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	remote, err := g.RemoteHead(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if remote != local {
		t.Errorf("RemoteHead = %s, LocalHead = %s — should match right after clone", remote, local)
	}

	// New commit upstream → remote moves, local stays.
	if err := os.WriteFile(filepath.Join(repo, "catalog", "skills", "golang", "SKILL.md"), []byte("# golang v2"), 0o644); err != nil {
		t.Fatal(err)
	}
	commit("bump golang")

	remote2, err := g.RemoteHead(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if remote2 == local {
		t.Error("RemoteHead did not move after upstream commit")
	}
}

func TestGitRepoSync(t *testing.T) {
	repo, commit := gitFixture(t)
	g := catalog.GitRepo{URL: repo, Dir: filepath.Join(t.TempDir(), "mirror")}
	if err := g.Ensure(); err != nil {
		t.Fatal(err)
	}
	before, _ := g.LocalHead()

	if err := os.WriteFile(filepath.Join(repo, "catalog", "skills", "golang", "SKILL.md"), []byte("# golang v2"), 0o644); err != nil {
		t.Fatal(err)
	}
	commit("bump golang")

	after, err := g.Sync()
	if err != nil {
		t.Fatalf("Sync() error = %v", err)
	}
	if after == before {
		t.Error("Sync() did not advance the mirror")
	}
	data, _ := os.ReadFile(filepath.Join(g.Dir, "catalog", "skills", "golang", "SKILL.md"))
	if string(data) != "# golang v2" {
		t.Errorf("mirror content not updated: %q", data)
	}
}

func TestGitRepoImplementsSource(t *testing.T) {
	repo, _ := gitFixture(t)
	g := catalog.GitRepo{URL: repo, Dir: filepath.Join(t.TempDir(), "mirror")}

	var _ catalog.Source = g // compile-time check

	c, err := g.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if len(c.Profiles) == 0 {
		t.Error("Load() returned empty catalog")
	}
	if p := g.SkillPath("golang"); p != filepath.Join(g.Dir, "catalog", "skills", "golang") {
		t.Errorf("SkillPath = %q", p)
	}
}

func TestGitRepoMissingGitBinary(t *testing.T) {
	// Force LookPath failure by pointing PATH at an empty dir.
	t.Setenv("PATH", t.TempDir())

	g := catalog.GitRepo{URL: "ignored", Dir: filepath.Join(t.TempDir(), "mirror")}
	err := g.Ensure()
	if err == nil {
		t.Fatal("Ensure() without git should fail")
	}
	if !strings.Contains(err.Error(), "git is required") {
		t.Errorf("error = %q, want the actionable git-required message", err)
	}
}
