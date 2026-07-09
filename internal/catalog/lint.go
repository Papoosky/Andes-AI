package catalog

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Lint performs content checks BEYOND Load's structural validation
// (Load already covers JSON validity, skill existence, and id safety).
// It returns a sorted list of problems; an empty slice means the catalog
// is clean. It assumes Load has already passed, so a skill whose SKILL.md
// is unreadable here is skipped rather than reported twice.
func Lint(src Source, c *Catalog) []string {
	var problems []string

	for name, p := range c.Profiles {
		if len(p.Skills) == 0 {
			problems = append(problems, fmt.Sprintf("profile %q has no skills", name))
		}
		seen := map[string]bool{}
		for _, id := range p.Skills {
			if seen[id] {
				problems = append(problems, fmt.Sprintf("profile %q lists skill %q more than once", name, id))
				continue
			}
			seen[id] = true
		}
	}

	// Check each unique referenced skill's frontmatter once.
	checked := map[string]bool{}
	for _, p := range c.Profiles {
		for _, id := range p.Skills {
			if checked[id] {
				continue
			}
			checked[id] = true
			mdPath := filepath.Join(src.SkillPath(id), "SKILL.md")
			data, err := os.ReadFile(mdPath)
			if err != nil {
				continue // existence is Load's job; skip unreadable here
			}
			name, desc := frontmatterFields(data)
			if name == "" || desc == "" {
				problems = append(problems,
					fmt.Sprintf("skill %q: SKILL.md is missing frontmatter name/description", id))
			}
		}
	}

	sort.Strings(problems)
	return problems
}

// frontmatterFields extracts name/description from a leading YAML frontmatter
// block delimited by --- lines. Lightweight line parse — no YAML dependency.
// Returns empty strings if the block or a field is absent.
func frontmatterFields(md []byte) (name, desc string) {
	s := string(md)
	// Normalize UTF-8 BOM and CRLF line endings so Windows-saved files are handled correctly
	s = strings.TrimPrefix(s, "\xef\xbb\xbf") // strip UTF-8 BOM
	s = strings.ReplaceAll(s, "\r\n", "\n")   // normalize CRLF to LF
	if !strings.HasPrefix(s, "---\n") {
		return "", ""
	}
	rest := s[len("---"):]
	end := strings.Index(rest, "\n---")
	if end < 0 {
		return "", ""
	}
	block := rest[:end]
	for _, line := range strings.Split(block, "\n") {
		line = strings.TrimSpace(line)
		if v, ok := strings.CutPrefix(line, "name:"); ok {
			name = strings.TrimSpace(v)
		}
		if v, ok := strings.CutPrefix(line, "description:"); ok {
			desc = strings.TrimSpace(v)
		}
	}
	return name, desc
}
