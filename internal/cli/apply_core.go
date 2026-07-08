package cli

import (
	"fmt"

	"github.com/andespath/andes-ai/internal/catalog"
	"github.com/andespath/andes-ai/internal/installer"
	"github.com/andespath/andes-ai/internal/manifest"
)

// applyActions is the single manifest-writing path: it applies the plan
// (always — install/update copy, and skips still repair drift via installer.Apply's
// disk re-check), finalizes the git ref, and saves the manifest. No printing, no prompts.
func applyActions(src catalog.Source, actions []installer.Action, prev *manifest.Manifest, profiles []string, catRef manifest.CatalogRef) (installed map[string]manifest.InstalledSkill, summary string, err error) {
	sDir, err := skillsDir()
	if err != nil {
		return nil, "", err
	}
	installed, err = installer.Apply(src, actions, sDir)
	if err != nil {
		return nil, "", err
	}

	catRef, err = finalizeRef(src, catRef)
	if err != nil {
		return nil, "", err
	}

	next := &manifest.Manifest{
		Version:   1,
		Catalog:   catRef,
		Profiles:  profiles,
		Installed: installed,
	}
	mPath, err := manifest.DefaultPath()
	if err != nil {
		return nil, "", err
	}
	if err := next.Save(mPath); err != nil {
		return nil, "", err
	}

	return installed, fmt.Sprintf("✓ %d skills up to date in %s", len(installed), sDir), nil
}
