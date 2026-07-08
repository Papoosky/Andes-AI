package cli

import (
	"sort"

	"github.com/andespath/andes-ai/internal/manifest"
	"github.com/andespath/andes-ai/internal/tui"
)

// buildCatalogProfilesFunc returns a tui.CatalogProfilesFunc that resolves the
// catalog using the same logic as runInstall (manifest → baked default →
// unknown). It is injected into the TUI model so tui stays decoupled from cli.
func buildCatalogProfilesFunc() tui.CatalogProfilesFunc {
	return func() (names []string, descs map[string]string, installed []string, catalogKnown bool, err error) {
		mPath, err := manifest.DefaultPath()
		if err != nil {
			return nil, nil, nil, false, err
		}
		prev, err := manifest.Load(mPath)
		if err != nil {
			return nil, nil, nil, false, err
		}

		// Determine whether the catalog location is already known (manifest or
		// baked default). We use the same precedence as resolveSource but
		// without prompting — that's the TUI's job.
		knownSrc := false
		if prev != nil && (prev.Catalog.Type == "git" || (prev.Catalog.Type == "local" && prev.Catalog.Path != "")) {
			knownSrc = true
		} else if defaultCatalogURL != "" {
			knownSrc = true
		}

		// If location is unknown the TUI will show the catalog-input screen;
		// we can still return empty names with catalogKnown=false.
		if !knownSrc {
			return nil, nil, nil, false, nil
		}

		// Load the catalog — yes flag=false (non-interactive here; errors surface to TUI).
		src, _, err := resolveSource("", prev, true)
		if err != nil {
			return nil, nil, nil, false, err
		}
		cat, err := src.Load()
		if err != nil {
			return nil, nil, nil, false, err
		}

		// Collect sorted profile names and descriptions.
		profileNames := make([]string, 0, len(cat.Profiles))
		profileDescs := make(map[string]string, len(cat.Profiles))
		for name, profile := range cat.Profiles {
			profileNames = append(profileNames, name)
			profileDescs[name] = profile.Description
		}
		sort.Strings(profileNames)

		// Collect currently-installed profile names from manifest.
		var installedProfiles []string
		if prev != nil {
			installedProfiles = prev.Profiles
		}

		return profileNames, profileDescs, installedProfiles, true, nil
	}
}
