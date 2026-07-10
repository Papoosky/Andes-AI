package catalog

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// LocalDir is a catalog rooted at a local folder.
type LocalDir struct {
	Root string
}

func (l LocalDir) Load() (*Catalog, error) {
	data, err := os.ReadFile(filepath.Join(l.Root, "catalog.json"))
	if err != nil {
		return nil, fmt.Errorf("could not read the catalog at %s: check the path (%w)", l.Root, err)
	}
	var c Catalog
	if err := json.Unmarshal(data, &c); err != nil {
		return nil, fmt.Errorf("invalid catalog.json at %s: %w", l.Root, err)
	}
	if err := l.validate(&c); err != nil {
		return nil, err
	}
	return &c, nil
}

func (l LocalDir) SkillPath(id string) string {
	return filepath.Join(l.Root, "skills", id)
}

// validate ensures every referenced skill exists with a SKILL.md.
// Fails loud at load time so installs never break halfway.
func (l LocalDir) validate(c *Catalog) error {
	var problems []string
	for name, p := range c.Profiles {
		for _, id := range p.Skills {
			if !validSkillID(id) {
				problems = append(problems,
					fmt.Sprintf("profile %q references skill %q: invalid id (cannot contain path separators or '..')", name, id))
				continue
			}
			skillMD := filepath.Join(l.SkillPath(id), "SKILL.md")
			if _, err := os.Stat(skillMD); os.IsNotExist(err) {
				problems = append(problems,
					fmt.Sprintf("profile %q references skill %q but %s is missing", name, id, skillMD))
			} else if err != nil {
				return fmt.Errorf("could not verify %s: %w", skillMD, err)
			}
			if err := rejectSymlinks(l.SkillPath(id)); err != nil {
				problems = append(problems,
					fmt.Sprintf("profile %q references skill %q: %v", name, id, err))
			}
		}
	}
	if len(problems) > 0 {
		sort.Strings(problems)
		return fmt.Errorf("invalid catalog:\n  %s", strings.Join(problems, "\n  "))
	}
	return nil
}

func rejectSymlinks(root string) error {
	return filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.Type()&os.ModeSymlink == 0 {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		return fmt.Errorf("symlinks are not allowed (%s)", filepath.ToSlash(rel))
	})
}

// validSkillID rejects ids that could escape the skills directory.
func validSkillID(id string) bool {
	if id == "" || id == "." || id == ".." {
		return false
	}
	if strings.ContainsAny(id, `/\`) || strings.Contains(id, "..") || filepath.IsAbs(id) {
		return false
	}
	return true
}
