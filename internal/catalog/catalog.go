// Package catalog reads and validates the andespath skills catalog.
package catalog

// Profile is a named bundle of skills.
type Profile struct {
	Description string   `json:"description"`
	Skills      []string `json:"skills"`
}

// Catalog is the parsed catalog.json.
type Catalog struct {
	Name     string             `json:"name"`
	Profiles map[string]Profile `json:"profiles"`
}

// Source abstracts where the catalog lives (LocalDir today, GitRepo in v2).
type Source interface {
	Load() (*Catalog, error)
	SkillPath(id string) string
}
