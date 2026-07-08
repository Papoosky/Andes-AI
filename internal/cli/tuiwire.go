package cli

import (
	"fmt"

	"github.com/andespath/andes-ai/internal/installer"
	"github.com/andespath/andes-ai/internal/manifest"
	"github.com/andespath/andes-ai/internal/tui"
)

// buildApplyInstallFunc returns a tui.ApplyInstallFunc that resolves the
// catalog source, plans and applies the install in-process (no confirmation
// prompts — the TUI plan screen already confirmed), saves the manifest, and
// returns a human-readable summary. It is injected into tui.Model so that
// tui stays decoupled from cli.
func buildApplyInstallFunc() tui.ApplyInstallFunc {
	return func(profiles []string) (string, error) {
		mPath, err := manifest.DefaultPath()
		if err != nil {
			return "", err
		}
		prev, err := manifest.Load(mPath)
		if err != nil {
			return "", err
		}

		// Resolve catalog source using the same precedence as runInstall:
		// manifest → baked default → error (no interactive prompt in TUI path).
		src, catRef, err := resolveSource("", prev, true)
		if err != nil {
			return "", err
		}

		cat, err := src.Load()
		if err != nil {
			return "", err
		}

		actions, err := installer.Plan(src, cat, prev, profiles)
		if err != nil {
			return "", err
		}

		// Count non-skip actions.
		changeCount := 0
		for _, a := range actions {
			if a.Type != installer.ActionSkip {
				changeCount++
			}
		}

		if changeCount == 0 {
			return "Everything is already up to date.", nil
		}

		sDir, err := skillsDir()
		if err != nil {
			return "", err
		}
		installed, err := installer.Apply(src, actions, sDir)
		if err != nil {
			return "", err
		}

		catRef, err = finalizeRef(src, catRef)
		if err != nil {
			return "", err
		}

		next := &manifest.Manifest{
			Version:   1,
			Catalog:   catRef,
			Profiles:  profiles,
			Installed: installed,
		}
		if err := next.Save(mPath); err != nil {
			return "", err
		}

		return fmt.Sprintf("✓ %d skills up to date in %s", len(installed), sDir), nil
	}
}
