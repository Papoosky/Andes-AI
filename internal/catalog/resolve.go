package catalog

import (
	"fmt"

	"github.com/andespath/andes-ai/internal/manifest"
)

// ResolveSkills expands profiles into skillID → profile that brought it in.
// The first profile (in requested order) wins on shared skills.
func ResolveSkills(c *Catalog, profiles []string) (map[string]string, error) {
	resolved := map[string]string{}
	for _, pname := range profiles {
		p, ok := c.Profiles[pname]
		if !ok {
			return nil, fmt.Errorf("profile %q does not exist in the catalog; run `andes list` to see available ones", pname)
		}
		for _, id := range p.Skills {
			if _, seen := resolved[id]; !seen {
				resolved[id] = pname
			}
		}
	}
	return resolved, nil
}

// SourceFromManifest resolves the catalog Source from a manifest based on its type.
// For git catalogs, it returns a GitRepo with the URL and the provided mirror directory.
// For local catalogs, it returns a LocalDir with the catalog path.
func SourceFromManifest(m *manifest.Manifest, mirrorDir string) Source {
	if m.Catalog.Type == "git" {
		return GitRepo{URL: m.Catalog.URL, Dir: mirrorDir}
	}
	return LocalDir{Root: m.Catalog.Path}
}
