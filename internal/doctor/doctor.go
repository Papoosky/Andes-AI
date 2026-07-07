// Package doctor diagnoses drift between manifest (declared), disk (real)
// and catalog (source). It never modifies anything.
package doctor

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"sort"

	"github.com/andespath/andes-ai/internal/catalog"
	"github.com/andespath/andes-ai/internal/hashdir"
	"github.com/andespath/andes-ai/internal/manifest"
)

type Status string

const (
	StatusOK       Status = "ok"
	StatusMissing  Status = "falta"
	StatusModified Status = "modificada"
	StatusOutdated Status = "desactualizada"
)

type Finding struct {
	SkillID string
	Status  Status
	Advice  string
}

// Check compares the three states per installed skill, in SkillID order.
// Precedence per skill: missing > modified > outdated > ok.
func Check(src catalog.Source, m *manifest.Manifest, skillsDir string) ([]Finding, error) {
	ids := make([]string, 0, len(m.Installed))
	for id := range m.Installed {
		ids = append(ids, id)
	}
	sort.Strings(ids)

	findings := make([]Finding, 0, len(ids))
	for _, id := range ids {
		inst := m.Installed[id]
		diskPath := filepath.Join(skillsDir, id)

		if _, err := os.Stat(diskPath); errors.Is(err, fs.ErrNotExist) {
			findings = append(findings, Finding{id, StatusMissing,
				"re-corré `andes init` para reinstalarla"})
			continue
		}

		diskHash, err := hashdir.Hash(diskPath)
		if err != nil {
			return nil, err
		}
		if diskHash != inst.Hash {
			findings = append(findings, Finding{id, StatusModified,
				"fue editada a mano; re-correr `andes init` PISA tus cambios — decidí antes"})
			continue
		}

		catHash, err := hashdir.Hash(src.SkillPath(id))
		if err != nil {
			return nil, err
		}
		if catHash != inst.Hash {
			findings = append(findings, Finding{id, StatusOutdated,
				"el catálogo tiene versión nueva; re-corré `andes init`"})
			continue
		}

		findings = append(findings, Finding{id, StatusOK, ""})
	}
	return findings, nil
}
