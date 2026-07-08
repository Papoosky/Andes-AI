package cli

import (
	"errors"
	"path/filepath"
	"strings"

	"github.com/andespath/andes-ai/internal/catalog"
	"github.com/andespath/andes-ai/internal/manifest"
)

// defaultCatalogURL is baked at build time:
//
//	go build -ldflags "-X github.com/andespath/andes-ai/internal/cli.defaultCatalogURL=git@github.com:andespath/andes-ai.git"
//
// Empty (dev builds, pre-GitHub transition) means: no default, fall back to
// the interactive prompt.
var defaultCatalogURL string

// isGitURL reports whether s looks like a git remote rather than a local path.
func isGitURL(s string) bool {
	return strings.HasPrefix(s, "git@") ||
		strings.HasPrefix(s, "https://") ||
		strings.HasPrefix(s, "http://") ||
		strings.HasPrefix(s, "ssh://") ||
		strings.HasPrefix(s, "file://") ||
		strings.HasSuffix(s, ".git")
}

// resolveSource picks the catalog source: --catalog flag → previous
// manifest → baked default URL → interactive prompt (error under --yes).
// The returned CatalogRef has an empty Ref for git sources — the caller
// fills it with LocalHead() after installing.
func resolveSource(catalogFlag string, prev *manifest.Manifest, yes bool) (catalog.Source, manifest.CatalogRef, error) {
	// 1. Explicit flag.
	if catalogFlag != "" {
		return sourceFor(catalogFlag)
	}
	// 2. Previous manifest.
	if prev != nil {
		switch prev.Catalog.Type {
		case "git":
			return gitSource(prev.Catalog.URL)
		case "local":
			if prev.Catalog.Path != "" {
				return sourceFor(prev.Catalog.Path)
			}
		}
	}
	// 3. Baked company default.
	if defaultCatalogURL != "" {
		return gitSource(defaultCatalogURL)
	}
	// 4. Prompt (or fail under --yes).
	if yes {
		return nil, manifest.CatalogRef{}, errors.New("catalog location unknown: pass --catalog <path or git URL>")
	}
	path, err := promptCatalogPath()
	if err != nil {
		return nil, manifest.CatalogRef{}, err
	}
	return sourceFor(path)
}

func sourceFor(loc string) (catalog.Source, manifest.CatalogRef, error) {
	if isGitURL(loc) {
		return gitSource(loc)
	}
	abs, err := filepath.Abs(loc)
	if err != nil {
		return nil, manifest.CatalogRef{}, err
	}
	return catalog.LocalDir{Root: abs}, manifest.CatalogRef{Type: "local", Path: abs}, nil
}

func gitSource(url string) (catalog.Source, manifest.CatalogRef, error) {
	dir, err := mirrorDir()
	if err != nil {
		return nil, manifest.CatalogRef{}, err
	}
	return catalog.GitRepo{URL: url, Dir: dir}, manifest.CatalogRef{Type: "git", URL: url}, nil
}

// finalizeRef fills Ref for git sources after an install/update.
func finalizeRef(src catalog.Source, ref manifest.CatalogRef) (manifest.CatalogRef, error) {
	if g, ok := src.(catalog.GitRepo); ok {
		head, err := g.LocalHead()
		if err != nil {
			return ref, err
		}
		ref.Ref = head
	}
	return ref, nil
}
