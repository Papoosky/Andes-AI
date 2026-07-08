// Package doctor diagnoses drift between manifest (declared), disk (real)
// and catalog (source). It never modifies anything.
package doctor

import (
	"errors"
	"fmt"
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
	StatusMissing  Status = "missing"
	StatusModified Status = "modified"
	StatusOutdated Status = "outdated"
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

		_, statErr := os.Stat(diskPath)
		if errors.Is(statErr, fs.ErrNotExist) {
			findings = append(findings, Finding{id, StatusMissing,
				"run `andes init` to reinstall it"})
			continue
		}
		if statErr != nil {
			return nil, fmt.Errorf("could not verify skill %q at %s: %w", id, diskPath, statErr)
		}

		diskHash, err := hashdir.Hash(diskPath)
		if err != nil {
			return nil, fmt.Errorf("could not read skill %q on disk: %w", id, err)
		}
		if diskHash != inst.Hash {
			findings = append(findings, Finding{id, StatusModified,
				"manually edited; re-running `andes init` OVERWRITES your changes — decide first"})
			continue
		}

		catHash, err := hashdir.Hash(src.SkillPath(id))
		if err != nil {
			return nil, fmt.Errorf("could not read skill %q from catalog: %w", id, err)
		}
		if catHash != inst.Hash {
			findings = append(findings, Finding{id, StatusOutdated,
				"catalog has a newer version; run `andes init`"})
			continue
		}

		findings = append(findings, Finding{id, StatusOK, ""})
	}
	return findings, nil
}
