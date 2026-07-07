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
		return nil, fmt.Errorf("no pude leer el catálogo en %s: verificá la ruta (%w)", l.Root, err)
	}
	var c Catalog
	if err := json.Unmarshal(data, &c); err != nil {
		return nil, fmt.Errorf("catalog.json inválido en %s: %w", l.Root, err)
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
			skillMD := filepath.Join(l.SkillPath(id), "SKILL.md")
			if _, err := os.Stat(skillMD); os.IsNotExist(err) {
				problems = append(problems,
					fmt.Sprintf("el perfil %q referencia la skill %q pero falta %s", name, id, skillMD))
			} else if err != nil {
				return fmt.Errorf("no pude verificar %s: %w", skillMD, err)
			}
		}
	}
	if len(problems) > 0 {
		sort.Strings(problems)
		return fmt.Errorf("catálogo inválido:\n  %s", strings.Join(problems, "\n  "))
	}
	return nil
}
