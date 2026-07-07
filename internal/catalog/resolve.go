package catalog

import "fmt"

// ResolveSkills expands profiles into skillID → profile that brought it in.
// The first profile (in requested order) wins on shared skills.
func ResolveSkills(c *Catalog, profiles []string) (map[string]string, error) {
	resolved := map[string]string{}
	for _, pname := range profiles {
		p, ok := c.Profiles[pname]
		if !ok {
			return nil, fmt.Errorf("el perfil %q no existe en el catálogo; corré `andes list` para ver los disponibles", pname)
		}
		for _, id := range p.Skills {
			if _, seen := resolved[id]; !seen {
				resolved[id] = pname
			}
		}
	}
	return resolved, nil
}
