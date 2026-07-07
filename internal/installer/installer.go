// Package installer plans and applies skill installs (catalog → ~/.claude/skills).
package installer

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"

	"github.com/andespath/andes-ai/internal/catalog"
	"github.com/andespath/andes-ai/internal/hashdir"
	"github.com/andespath/andes-ai/internal/manifest"
)

type ActionType string

const (
	ActionInstall ActionType = "instalar"
	ActionUpdate  ActionType = "actualizar"
	ActionSkip    ActionType = "sin cambios"
)

// Action is one planned step for one skill. Hash is the catalog-side hash.
type Action struct {
	SkillID string
	Type    ActionType
	Profile string
	Hash    string
}

// Plan diffs desired state (profiles resolved against the catalog) with the
// manifest. m == nil means first init: everything installs.
func Plan(src catalog.Source, cat *catalog.Catalog, m *manifest.Manifest, profiles []string) ([]Action, error) {
	resolved, err := catalog.ResolveSkills(cat, profiles)
	if err != nil {
		return nil, err
	}
	ids := make([]string, 0, len(resolved))
	for id := range resolved {
		ids = append(ids, id)
	}
	sort.Strings(ids)

	actions := make([]Action, 0, len(ids))
	for _, id := range ids {
		h, err := hashdir.Hash(src.SkillPath(id))
		if err != nil {
			return nil, fmt.Errorf("no pude hashear la skill %q del catálogo: %w", id, err)
		}
		a := Action{SkillID: id, Profile: resolved[id], Hash: h, Type: ActionInstall}
		if m != nil {
			if inst, ok := m.Installed[id]; ok {
				if inst.Hash == h {
					a.Type = ActionSkip
				} else {
					a.Type = ActionUpdate
				}
			}
		}
		actions = append(actions, a)
	}
	return actions, nil
}

// Apply executes the plan: install/update actions copy the skill folder
// (clean copy: destination removed first), skips are left untouched.
// The returned map is the complete `installed` section for the manifest.
func Apply(src catalog.Source, actions []Action, skillsDir string) (map[string]manifest.InstalledSkill, error) {
	installed := make(map[string]manifest.InstalledSkill, len(actions))
	for _, a := range actions {
		if a.Type != ActionSkip {
			dst := filepath.Join(skillsDir, a.SkillID)
			if err := os.RemoveAll(dst); err != nil {
				return nil, fmt.Errorf("no pude limpiar %s: %w", dst, err)
			}
			if err := copyDir(src.SkillPath(a.SkillID), dst); err != nil {
				return nil, fmt.Errorf("no pude instalar la skill %q: %w", a.SkillID, err)
			}
		}
		installed[a.SkillID] = manifest.InstalledSkill{Hash: a.Hash, Profile: a.Profile}
	}
	return installed, nil
}

func copyDir(src, dst string) error {
	return filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(target, data, 0o644)
	})
}
