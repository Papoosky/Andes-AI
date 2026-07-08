package cli

import "github.com/andespath/andes-ai/internal/tui"

// ExportedBuildCallbacks exposes buildCallbacks to external test packages.
var ExportedBuildCallbacks func() (tui.CatalogProfilesFunc, tui.PlanInstallFunc, tui.ApplyInstallFunc) = buildCallbacks
