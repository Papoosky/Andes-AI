package catalog

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// GitRepo is a catalog Source backed by a git repository. It manages its own
// mirror clone under Dir and delegates all catalog reads to LocalDir over
// the mirror's catalog/ subdirectory. Auth is whatever git already has.
type GitRepo struct {
	URL string // repo URL (or local path — git treats both the same)
	Dir string // managed mirror, e.g. ~/.andes/catalog
}

func (g GitRepo) local() LocalDir {
	return LocalDir{Root: filepath.Join(g.Dir, "catalog")}
}

// Ensure guarantees a clean, valid mirror: clones if missing or corrupt
// (self-healing), resets any local drift otherwise. Dir is andes-private,
// so a hard reset is always safe.
func (g GitRepo) Ensure() error {
	if _, err := g.git("-C", g.Dir, "rev-parse", "--git-dir"); err != nil {
		// Missing or not a valid repo → wipe and clone fresh.
		if err := os.RemoveAll(g.Dir); err != nil {
			return fmt.Errorf("could not clean the catalog mirror at %s: %w", g.Dir, err)
		}
		if _, err := g.git("clone", g.URL, g.Dir); err != nil {
			return fmt.Errorf("could not reach the catalog repo — check your GitHub access (SSH key or token): %w", err)
		}
		return nil
	}
	if _, err := g.git("-C", g.Dir, "reset", "--hard"); err != nil {
		return err
	}
	_, err := g.git("-C", g.Dir, "clean", "-fd")
	return err
}

// LocalHead returns the mirror's current commit SHA.
func (g GitRepo) LocalHead() (string, error) {
	return g.git("-C", g.Dir, "rev-parse", "HEAD")
}

// RemoteHead returns the remote's HEAD SHA without downloading content.
// The caller bounds latency via ctx (the TUI uses a 2s timeout).
func (g GitRepo) RemoteHead(ctx context.Context) (string, error) {
	out, err := g.gitCtx(ctx, "ls-remote", g.URL, "HEAD")
	if err != nil {
		return "", err
	}
	fields := strings.Fields(out)
	if len(fields) == 0 {
		return "", fmt.Errorf("unexpected ls-remote output from %s", g.URL)
	}
	return fields[0], nil
}

// Sync fast-forwards the mirror to the remote and returns the new HEAD SHA.
func (g GitRepo) Sync() (string, error) {
	if _, err := g.git("-C", g.Dir, "fetch", "origin"); err != nil {
		return "", fmt.Errorf("could not reach the catalog repo — check your GitHub access (SSH key or token): %w", err)
	}
	if _, err := g.git("-C", g.Dir, "reset", "--hard", "origin/HEAD"); err != nil {
		return "", err
	}
	return g.LocalHead()
}

// Load implements Source: ensures the mirror, then delegates to LocalDir.
func (g GitRepo) Load() (*Catalog, error) {
	if err := g.Ensure(); err != nil {
		return nil, err
	}
	return g.local().Load()
}

// SkillPath implements Source by delegating to LocalDir over the mirror.
func (g GitRepo) SkillPath(id string) string {
	return g.local().SkillPath(id)
}

func (g GitRepo) git(args ...string) (string, error) {
	return g.gitCtx(context.Background(), args...)
}

func (g GitRepo) gitCtx(ctx context.Context, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		if errors.Is(err, exec.ErrNotFound) {
			return "", errors.New("git is required — install it and retry")
		}
		return "", fmt.Errorf("git %s: %s", strings.Join(args, " "), firstLine(out))
	}
	return strings.TrimSpace(string(out)), nil
}

// firstLine keeps error messages short and actionable instead of dumping
// full git output.
func firstLine(out []byte) string {
	s := strings.TrimSpace(string(out))
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		return s[:i]
	}
	if s == "" {
		return "unknown git error"
	}
	return s
}
