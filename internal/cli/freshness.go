package cli

import (
	"context"
	"time"

	"github.com/andespath/andes-ai/internal/catalog"
	"github.com/andespath/andes-ai/internal/manifest"
	"github.com/andespath/andes-ai/internal/tui"
)

// checkCatalogFreshness compares the remote catalog HEAD against the
// installed ref. Best-effort: any failure reports offline, never blocks.
func checkCatalogFreshness() tui.FreshnessMsg {
	mPath, err := manifest.DefaultPath()
	if err != nil {
		return tui.FreshnessMsg{}
	}
	m, err := manifest.Load(mPath)
	if err != nil || m == nil || m.Catalog.Type != "git" {
		return tui.FreshnessMsg{}
	}
	dir, err := mirrorDir()
	if err != nil {
		return tui.FreshnessMsg{}
	}
	g := catalog.GitRepo{URL: m.Catalog.URL, Dir: dir}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	remote, err := g.RemoteHead(ctx)
	if err != nil {
		return tui.FreshnessMsg{Offline: true}
	}
	return tui.FreshnessMsg{Outdated: remote != m.Catalog.Ref}
}
