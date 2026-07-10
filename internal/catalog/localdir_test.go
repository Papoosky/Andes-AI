package catalog_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/andespath/andes-ai/internal/catalog"
)

func fixtureDir(t *testing.T) string {
	t.Helper()
	abs, err := filepath.Abs("../../catalog")
	if err != nil {
		t.Fatal(err)
	}
	return abs
}

func TestLocalDirLoadValid(t *testing.T) {
	src := catalog.LocalDir{Root: fixtureDir(t)}
	c, err := src.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if c.Name != "andespath" {
		t.Errorf("Name = %q, want %q", c.Name, "andespath")
	}
	if len(c.Profiles) != 2 {
		t.Errorf("len(Profiles) = %d, want 2", len(c.Profiles))
	}
	if got := c.Profiles["andespath-core"].Skills; len(got) != 2 {
		t.Errorf("andespath-core skills = %v, want 2 skills", got)
	}
}

func TestLocalDirLoadErrors(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(t *testing.T) string // returns catalog root
		wantErr string
	}{
		{
			name: "folder without catalog.json",
			setup: func(t *testing.T) string {
				return t.TempDir()
			},
			wantErr: "could not read the catalog",
		},
		{
			name: "invalid json",
			setup: func(t *testing.T) string {
				dir := t.TempDir()
				if err := os.WriteFile(filepath.Join(dir, "catalog.json"), []byte("{no es json"), 0o644); err != nil {
					t.Fatal(err)
				}
				return dir
			},
			wantErr: "invalid catalog.json",
		},
		{
			name: "profile references non-existent skill",
			setup: func(t *testing.T) string {
				dir := t.TempDir()
				if err := os.WriteFile(filepath.Join(dir, "catalog.json"), []byte(`{
					"name": "x",
					"profiles": {"p1": {"description": "d", "skills": ["fantasma"]}}
				}`), 0o644); err != nil {
					t.Fatal(err)
				}
				return dir
			},
			wantErr: "fantasma",
		},
		{
			name: "skill id with path traversal ../evil",
			setup: func(t *testing.T) string {
				dir := t.TempDir()
				if err := os.WriteFile(filepath.Join(dir, "catalog.json"), []byte(`{
					"name": "x",
					"profiles": {"p1": {"description": "d", "skills": ["../evil"]}}
				}`), 0o644); err != nil {
					t.Fatal(err)
				}
				return dir
			},
			wantErr: "invalid id",
		},
		{
			name: "skill id with separator a/b",
			setup: func(t *testing.T) string {
				dir := t.TempDir()
				if err := os.WriteFile(filepath.Join(dir, "catalog.json"), []byte(`{
					"name": "x",
					"profiles": {"p1": {"description": "d", "skills": ["a/b"]}}
				}`), 0o644); err != nil {
					t.Fatal(err)
				}
				return dir
			},
			wantErr: "invalid id",
		},
		{
			name: "skill id dotdot",
			setup: func(t *testing.T) string {
				dir := t.TempDir()
				if err := os.WriteFile(filepath.Join(dir, "catalog.json"), []byte(`{
					"name": "x",
					"profiles": {"p1": {"description": "d", "skills": [".."]}}
				}`), 0o644); err != nil {
					t.Fatal(err)
				}
				return dir
			},
			wantErr: "invalid id",
		},
		{
			name: "skill md symlink rejected",
			setup: func(t *testing.T) string {
				dir := t.TempDir()
				if err := os.WriteFile(filepath.Join(dir, "catalog.json"), []byte(`{
					"name": "x",
					"profiles": {"p1": {"description": "d", "skills": ["linked"]}}
				}`), 0o644); err != nil {
					t.Fatal(err)
				}
				skillDir := filepath.Join(dir, "skills", "linked")
				if err := os.MkdirAll(skillDir, 0o755); err != nil {
					t.Fatal(err)
				}
				target := filepath.Join(dir, "target.md")
				if err := os.WriteFile(target, []byte("# linked"), 0o644); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink(target, filepath.Join(skillDir, "SKILL.md")); err != nil {
					t.Fatal(err)
				}
				return dir
			},
			wantErr: "symlinks are not allowed",
		},
		{
			name: "extra symlink in skill rejected",
			setup: func(t *testing.T) string {
				dir := t.TempDir()
				if err := os.WriteFile(filepath.Join(dir, "catalog.json"), []byte(`{
					"name": "x",
					"profiles": {"p1": {"description": "d", "skills": ["linked"]}}
				}`), 0o644); err != nil {
					t.Fatal(err)
				}
				skillDir := filepath.Join(dir, "skills", "linked")
				if err := os.MkdirAll(skillDir, 0o755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte("# linked"), 0o644); err != nil {
					t.Fatal(err)
				}
				target := filepath.Join(dir, "outside.txt")
				if err := os.WriteFile(target, []byte("outside"), 0o644); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink(target, filepath.Join(skillDir, "outside-link.txt")); err != nil {
					t.Fatal(err)
				}
				return dir
			},
			wantErr: "symlinks are not allowed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			src := catalog.LocalDir{Root: tt.setup(t)}
			_, err := src.Load()
			if err == nil {
				t.Fatal("Load() = nil error, want error")
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("error = %q, want contains %q", err, tt.wantErr)
			}
		})
	}
}

func TestSkillPath(t *testing.T) {
	src := catalog.LocalDir{Root: "/tmp/cat"}
	got := src.SkillPath("golang")
	want := filepath.Join("/tmp/cat", "skills", "golang")
	if got != want {
		t.Errorf("SkillPath = %q, want %q", got, want)
	}
}
