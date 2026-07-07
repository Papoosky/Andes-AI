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
	Type string `json:"type"`
	Path string `json:"path"`
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
		return "", fmt.Errorf("no pude resolver tu home dir: %w", err)
	}
	return filepath.Join(home, ".claude", "andes.json"), nil
}

// Load reads the manifest. A missing file is not an error: it means
// init never ran, so Load returns (nil, nil).
func Load(path string) (*Manifest, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var m Manifest
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("manifiesto corrupto en %s: borralo y re-corré `andes init` (%w)", path, err)
	}
	return &m, nil
}

// Save writes atomically: temp file in the same dir, then rename.
// A crash mid-write leaves the previous manifest intact.
func (m *Manifest) Save(path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".andes-*.tmp")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name()) // no-op after successful rename
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), path)
}
