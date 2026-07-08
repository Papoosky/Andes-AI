package hashdir_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/andespath/andes-ai/internal/hashdir"
)

func writeFile(t *testing.T, dir, name, content string) {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestHashDeterministic(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "SKILL.md", "# hola")
	writeFile(t, dir, "extra/notes.md", "detalle")

	h1, err := hashdir.Hash(dir)
	if err != nil {
		t.Fatal(err)
	}
	h2, err := hashdir.Hash(dir)
	if err != nil {
		t.Fatal(err)
	}
	if h1 != h2 {
		t.Errorf("non-deterministic hash: %s != %s", h1, h2)
	}
	if !strings.HasPrefix(h1, "sha256:") {
		t.Errorf("hash missing sha256 prefix: %s", h1)
	}
}

func TestHashChangesOnContentChange(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "SKILL.md", "# v1")
	h1, err := hashdir.Hash(dir)
	if err != nil {
		t.Fatal(err)
	}

	writeFile(t, dir, "SKILL.md", "# v2")
	h2, err := hashdir.Hash(dir)
	if err != nil {
		t.Fatal(err)
	}

	if h1 == h2 {
		t.Error("hash did not change after content change")
	}
}

func TestHashChangesOnNewFile(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "SKILL.md", "# v1")
	h1, err := hashdir.Hash(dir)
	if err != nil {
		t.Fatal(err)
	}

	writeFile(t, dir, "other.md", "new")
	h2, err := hashdir.Hash(dir)
	if err != nil {
		t.Fatal(err)
	}

	if h1 == h2 {
		t.Error("hash did not change after adding a file")
	}
}

func TestHashEqualDirsEqualHash(t *testing.T) {
	dir1 := t.TempDir()
	dir2 := t.TempDir()
	for _, d := range []string{dir1, dir2} {
		writeFile(t, d, "SKILL.md", "same content")
	}
	h1, err := hashdir.Hash(dir1)
	if err != nil {
		t.Fatal(err)
	}
	h2, err := hashdir.Hash(dir2)
	if err != nil {
		t.Fatal(err)
	}
	if h1 != h2 {
		t.Errorf("equal dirs have different hash: %s != %s", h1, h2)
	}
}

func TestHashChangesOnRename(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "a.md", "content")
	h1, err := hashdir.Hash(dir)
	if err != nil {
		t.Fatal(err)
	}

	if err := os.Rename(filepath.Join(dir, "a.md"), filepath.Join(dir, "b.md")); err != nil {
		t.Fatal(err)
	}
	h2, err := hashdir.Hash(dir)
	if err != nil {
		t.Fatal(err)
	}

	if h1 == h2 {
		t.Error("hash did not change after renaming a file")
	}
}

func TestHashMissingDir(t *testing.T) {
	_, err := hashdir.Hash(filepath.Join(t.TempDir(), "does-not-exist"))
	if err == nil {
		t.Error("Hash of non-existent dir should fail")
	}
}
