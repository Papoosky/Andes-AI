package manifest_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/andespath/andes-ai/internal/manifest"
)

func sample() *manifest.Manifest {
	return &manifest.Manifest{
		Version:  1,
		Catalog:  manifest.CatalogRef{Type: "local", Path: "/tmp/cat"},
		Profiles: []string{"andespath-core"},
		Installed: map[string]manifest.InstalledSkill{
			"git-conventions": {Hash: "sha256:abc", Profile: "andespath-core"},
		},
	}
}

func TestSaveLoadRoundtrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".claude", "andes.json")

	if err := sample().Save(path); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	got, err := manifest.Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if got == nil {
		t.Fatal("Load() = nil, want manifest")
	}
	if got.Version != 1 {
		t.Errorf("Version = %d, want 1", got.Version)
	}
	if got.Catalog.Path != "/tmp/cat" {
		t.Errorf("Catalog.Path = %q, want /tmp/cat", got.Catalog.Path)
	}
	if got.Installed["git-conventions"].Hash != "sha256:abc" {
		t.Errorf("Installed hash = %q, want sha256:abc", got.Installed["git-conventions"].Hash)
	}
}

func TestLoadMissingReturnsNilNil(t *testing.T) {
	got, err := manifest.Load(filepath.Join(t.TempDir(), "no-existe.json"))
	if err != nil {
		t.Fatalf("Load() de archivo inexistente: error = %v, want nil", err)
	}
	if got != nil {
		t.Errorf("Load() = %v, want nil", got)
	}
}

func TestLoadCorruptFails(t *testing.T) {
	path := filepath.Join(t.TempDir(), "andes.json")
	os.WriteFile(path, []byte("{corrupto"), 0o644)

	_, err := manifest.Load(path)
	if err == nil || !strings.Contains(err.Error(), "corrupted") {
		t.Errorf("error = %v, want corrupted manifest message", err)
	}
}

func TestLoadSemanticallyInvalidManifestFails(t *testing.T) {
	tests := []struct {
		name    string
		json    string
		wantErr string
	}{
		{
			name: "unsupported version",
			json: `{
				"version": 2,
				"catalog": {"type": "local", "path": "/tmp/cat"},
				"installed": {}
			}`,
			wantErr: "unsupported manifest version",
		},
		{
			name: "unknown catalog type",
			json: `{
				"version": 1,
				"catalog": {"type": "s3"},
				"installed": {}
			}`,
			wantErr: "invalid catalog type",
		},
		{
			name: "local catalog missing path",
			json: `{
				"version": 1,
				"catalog": {"type": "local"},
				"installed": {}
			}`,
			wantErr: "local catalog path is required",
		},
		{
			name: "git catalog missing url",
			json: `{
				"version": 1,
				"catalog": {"type": "git"},
				"installed": {}
			}`,
			wantErr: "git catalog URL is required",
		},
		{
			name: "installed map missing",
			json: `{
				"version": 1,
				"catalog": {"type": "local", "path": "/tmp/cat"}
			}`,
			wantErr: "installed map is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "andes.json")
			if err := os.WriteFile(path, []byte(tt.json), 0o644); err != nil {
				t.Fatal(err)
			}

			_, err := manifest.Load(path)
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("Load() error = %v, want containing %q", err, tt.wantErr)
			}
		})
	}
}

func TestSaveLeavesNoTempFiles(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "andes.json")
	if err := sample().Save(path); err != nil {
		t.Fatal(err)
	}
	entries, _ := os.ReadDir(dir)
	if len(entries) != 1 {
		t.Errorf("quedaron archivos extra en %s: %v", dir, entries)
	}
}

func TestDefaultPath(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	got, err := manifest.DefaultPath()
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(home, ".claude", "andes.json")
	if got != want {
		t.Errorf("DefaultPath = %q, want %q", got, want)
	}
}

func TestSaveLoadGitCatalogRef(t *testing.T) {
	path := filepath.Join(t.TempDir(), "andes.json")
	m := &manifest.Manifest{
		Version: 1,
		Catalog: manifest.CatalogRef{
			Type: "git",
			URL:  "git@github.com:andespath/andes-ai.git",
			Ref:  "a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2",
		},
		Profiles:  []string{"andespath-core"},
		Installed: map[string]manifest.InstalledSkill{},
	}
	if err := m.Save(path); err != nil {
		t.Fatal(err)
	}
	got, err := manifest.Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if got.Catalog.Type != "git" || got.Catalog.URL != m.Catalog.URL || got.Catalog.Ref != m.Catalog.Ref {
		t.Errorf("git CatalogRef roundtrip = %+v", got.Catalog)
	}
	if got.Catalog.Path != "" {
		t.Errorf("Path should be empty for git refs, got %q", got.Catalog.Path)
	}
}
