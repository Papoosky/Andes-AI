package cli_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// gitFixture creates a real git repo in a temp dir whose layout mirrors the
// company repo: a catalog/ subdir with catalog.json and skills. Returns the
// repo path (usable as a clone source).
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
	src := fixtureCatalog(t)
	if err := os.MkdirAll(filepath.Join(repo, "catalog"), 0o755); err != nil {
		t.Fatal(err)
	}
	copyTreeCLI(t, src, filepath.Join(repo, "catalog"))

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

// copyTreeCLI copies a directory tree from src to dst.
func copyTreeCLI(t *testing.T, src, dst string) {
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
