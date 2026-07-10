// Package manifest reads and writes the andes install receipt (~/.claude/andes.json).
package manifest

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

type CatalogRef struct {
	Type string `json:"type"`           // "local" | "git"
	Path string `json:"path,omitempty"` // local: absolute folder path
	URL  string `json:"url,omitempty"`  // git: repo URL
	Ref  string `json:"ref,omitempty"`  // git: last applied catalog HEAD; used for freshness, not pinned checkouts
}

type InstalledSkill struct {
	Hash    string `json:"hash"`
	Profile string `json:"profile"`
}

type Manifest struct {
	Version   int                       `json:"version"`
	Catalog   CatalogRef                `json:"catalog"`
	Profiles  []string                  `json:"profiles"`
	Installed map[string]InstalledSkill `json:"installed"`
}

// DefaultPath returns ~/.claude/andes.json.
func DefaultPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("could not resolve home directory: %w", err)
	}
	return filepath.Join(home, ".claude", "andes.json"), nil
}

// Load reads the manifest. A missing file is not an error: it means
// install never ran, so Load returns (nil, nil).
func Load(path string) (*Manifest, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("could not read manifest at %s: %w", path, err)
	}
	var m Manifest
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("corrupted manifest at %s: delete it and re-run `andes install` (%w)", path, err)
	}
	if err := m.Validate(); err != nil {
		return nil, fmt.Errorf("invalid manifest at %s: delete it and re-run `andes install` (%w)", path, err)
	}
	return &m, nil
}

// Validate checks the semantic manifest contract after JSON parsing.
func (m *Manifest) Validate() error {
	if m.Version != 1 {
		return fmt.Errorf("unsupported manifest version %d", m.Version)
	}
	switch m.Catalog.Type {
	case "local":
		if m.Catalog.Path == "" {
			return errors.New("local catalog path is required")
		}
	case "git":
		if m.Catalog.URL == "" {
			return errors.New("git catalog URL is required")
		}
	default:
		return fmt.Errorf("invalid catalog type %q", m.Catalog.Type)
	}
	if m.Installed == nil {
		return errors.New("installed map is required")
	}
	return nil
}

// Save writes atomically: temp file in the same dir, then rename.
// A crash mid-write leaves the previous manifest intact.
func (m *Manifest) Save(path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("could not create directory for %s: %w", path, err)
	}
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".andes-*.tmp")
	if err != nil {
		return fmt.Errorf("could not create temp file for %s: %w", path, err)
	}
	defer os.Remove(tmp.Name()) // no-op after successful rename
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), path)
}
