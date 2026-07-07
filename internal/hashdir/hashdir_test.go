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
		t.Errorf("hash no determinista: %s != %s", h1, h2)
	}
	if !strings.HasPrefix(h1, "sha256:") {
		t.Errorf("hash sin prefijo sha256: %s", h1)
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
		t.Error("hash no cambió al cambiar contenido")
	}
}

func TestHashChangesOnNewFile(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "SKILL.md", "# v1")
	h1, err := hashdir.Hash(dir)
	if err != nil {
		t.Fatal(err)
	}

	writeFile(t, dir, "otro.md", "nuevo")
	h2, err := hashdir.Hash(dir)
	if err != nil {
		t.Fatal(err)
	}

	if h1 == h2 {
		t.Error("hash no cambió al agregar archivo")
	}
}

func TestHashEqualDirsEqualHash(t *testing.T) {
	dir1 := t.TempDir()
	dir2 := t.TempDir()
	for _, d := range []string{dir1, dir2} {
		writeFile(t, d, "SKILL.md", "mismo contenido")
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
		t.Errorf("dirs iguales con hash distinto: %s != %s", h1, h2)
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
		t.Error("hash no cambió tras renombrar un archivo")
	}
}

func TestHashMissingDir(t *testing.T) {
	_, err := hashdir.Hash(filepath.Join(t.TempDir(), "no-existe"))
	if err == nil {
		t.Error("Hash de dir inexistente debería fallar")
	}
}
