package cli

import (
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
	return func(catalogOverride string, profiles []string) (string, error) {
		mPath, err := manifest.DefaultPath()
		if err != nil {
			return "", err
		}
		prev, err := manifest.Load(mPath)
		if err != nil {
			return "", err
		}

		// Resolve catalog source: user-typed override → manifest → baked default → error.
		src, catRef, err := resolveSource(catalogOverride, prev, true)
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

		_, summary, err := applyActions(src, actions, prev, profiles, catRef)
		if err != nil {
			return "", err
		}

		if changeCount == 0 {
			return "Everything is already up to date.", nil
		}
		return summary, nil
	}
}

// buildPlanInstallFunc returns a tui.PlanInstallFunc that resolves the catalog
// source and runs installer.Plan without applying — used to show counts on the
// plan screen before the user confirms.
func buildPlanInstallFunc() tui.PlanInstallFunc {
	return func(catalogOverride string, profiles []string) (install, update, unchanged int, err error) {
		mPath, err := manifest.DefaultPath()
		if err != nil {
			return 0, 0, 0, err
		}
		prev, err := manifest.Load(mPath)
		if err != nil {
			return 0, 0, 0, err
		}

		src, _, err := resolveSource(catalogOverride, prev, true)
		if err != nil {
			return 0, 0, 0, err
		}

		cat, err := src.Load()
		if err != nil {
			return 0, 0, 0, err
		}

		actions, err := installer.Plan(src, cat, prev, profiles)
		if err != nil {
			return 0, 0, 0, err
		}

		for _, a := range actions {
			switch a.Type {
			case installer.ActionInstall:
				install++
			case installer.ActionUpdate:
				update++
			default:
				unchanged++
			}
		}
		return install, update, unchanged, nil
	}
}

// buildCallbacks wires all TUI callbacks from real CLI implementations.
func buildCallbacks() (tui.CatalogProfilesFunc, tui.PlanInstallFunc, tui.ApplyInstallFunc) {
	return buildCatalogProfilesFunc(), buildPlanInstallFunc(), buildApplyInstallFunc()
}
