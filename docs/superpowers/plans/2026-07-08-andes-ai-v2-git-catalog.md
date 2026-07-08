# andes-ai v2 Git Catalog Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** El catálogo vive en el repo git de la empresa: andes gestiona su propio mirror, detecta desactualización (ls-remote), actualiza con `andes update`/tecla `u`, y se instala con `install.sh`.

**Architecture:** `GitRepo` implementa `catalog.Source` como wrapper fino que delega en `LocalDir` sobre un mirror gestionado (`~/.andes/catalog`), shell-out a git (3 operaciones). Resolución de catálogo: flag → manifiesto → default horneado (ldflags) → prompt. La TUI chequea frescura async al abrir (timeout 2s, offline silencioso).

**Tech Stack:** Go 1.23, Cobra, Bubbletea/lipgloss, git CLI (requisito del producto), gh CLI (instalación).

**Spec:** `docs/superpowers/specs/2026-07-08-andes-ai-v2-git-catalog-design.md`

## Global Constraints

- Todo texto user-facing en INGLÉS, accionable (qué pasó + qué hacer). Nunca stack traces.
- Shell out a `git` (nunca go-git ni API GitHub). git ausente → "git is required — install it and retry".
- Mirror gestionado: `~/.andes/catalog` (deriva de `os.UserHomeDir()` — los tests overridean con `t.Setenv("HOME", ...)`). El catálogo DENTRO del mirror está en el subdir `catalog/`.
- El catálogo de producción vive en `catalog/` en la raíz de ESTE repo (movido desde `testdata/catalog`).
- Manifiesto: `catalog.type` `"git"` (con `url` + `ref` = SHA) o `"local"` (con `path`). Manifiestos v1 locales siguen válidos sin migración.
- Detección: SHA remoto (`ls-remote`) vs `manifest.catalog.ref`. Sin bumps manuales de versión.
- TUI: check async al abrir, timeout 2s, offline → silencioso + footer "offline". Banner "⚠ catalog updated — press u to update". JAMÁS bloquea el arranque.
- Tests sin red: fixtures = repos git reales creados en `t.TempDir()` (git requerido en CI — aceptado).
- Commits: Conventional Commits en inglés, sin atribución de AI, sin Co-Authored-By.
- TDD: test primero, verlo fallar, implementar mínimo, verlo pasar.
- Los comandos Cobra NO contienen lógica — orquestan `internal/*`.

---

### Task 1: Mover el catálogo a producción (`testdata/catalog` → `catalog/`)

**Files:**
- Move: `testdata/catalog/` → `catalog/` (git mv)
- Modify: `internal/catalog/localdir_test.go` (helper `fixtureDir`), `internal/cli/init_test.go` (helper `fixtureCatalog`), `internal/cli/prompts.go` (tip del prompt), `README.md` (quickstart paths)

**Interfaces:**
- Consumes: nada
- Produces: `catalog/` en la raíz como catálogo de producción y fixture de tests a la vez.

- [ ] **Step 1: Mover con git mv**

```bash
git mv testdata/catalog catalog
rmdir testdata 2>/dev/null || true
```

- [ ] **Step 2: Actualizar los DOS helpers de test y el tip del prompt**

En `internal/catalog/localdir_test.go`, en `fixtureDir`: reemplazar `"../../testdata/catalog"` por `"../../catalog"`.

En `internal/cli/init_test.go`, en `fixtureCatalog`: reemplazar `"../../testdata/catalog"` por `"../../catalog"`.

En `internal/cli/prompts.go`, en la Description de `promptCatalogPath`: reemplazar `./testdata/catalog` por `./catalog`.

En `README.md`: reemplazar toda ocurrencia de `./testdata/catalog` por `./catalog`.

- [ ] **Step 3: Verificar que TODO sigue verde**

Run: `go test ./... && go vet ./... && gofmt -l .`
Expected: 7 paquetes ok, vet limpio, gofmt silencioso.

- [ ] **Step 4: Commit**

```bash
git add -A
git commit -m "refactor: promote demo catalog to production catalog at repo root"
```

---

### Task 2: `GitRepo` source — mirror gestionado sobre git

**Files:**
- Create: `internal/catalog/gitrepo.go`
- Test: `internal/catalog/gitrepo_test.go`

**Interfaces:**
- Consumes: `LocalDir` (Task previa del MVP), git CLI.
- Produces (Tasks 4-6 y 8 consumen esto EXACTO):
  - `type GitRepo struct { URL string; Dir string }` (implementa `Source`)
  - `func (g GitRepo) Ensure() error` — clona si falta/corrupto (re-clone autocurable); si existe: `reset --hard` + `clean -fd`
  - `func (g GitRepo) LocalHead() (string, error)` — SHA del mirror
  - `func (g GitRepo) RemoteHead(ctx context.Context) (string, error)` — SHA remoto vía ls-remote (el ctx acota el timeout)
  - `func (g GitRepo) Sync() (string, error)` — fetch + reset a origin/HEAD; retorna el SHA nuevo
  - `func (g GitRepo) Load() (*Catalog, error)` — Ensure + delega en LocalDir
  - `func (g GitRepo) SkillPath(id string) string` — delega en LocalDir
  - El catálogo dentro del mirror vive en `filepath.Join(g.Dir, "catalog")`.

- [ ] **Step 1: Escribir el helper de fixture git y los tests que fallan**

`internal/catalog/gitrepo_test.go`:

```go
package catalog_test

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/andespath/andes-ai/internal/catalog"
)

// gitFixture creates a REAL git repo in a temp dir whose layout mirrors the
// company repo: a catalog/ subdir with catalog.json and skills. Returns the
// repo path (usable as a clone URL) and a commit helper.
func gitFixture(t *testing.T) (repo string, commit func(msg string)) {
	t.Helper()
	repo = t.TempDir()

	run := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = repo
		cmd.Env = append(os.Environ(),
			"GIT_AUTHOR_NAME=test", "GIT_AUTHOR_EMAIL=t@t",
			"GIT_COMMITTER_NAME=test", "GIT_COMMITTER_EMAIL=t@t")
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}

	// Seed: copy the production catalog into catalog/ inside the repo.
	src := fixtureDir(t) // existing helper → ../../catalog
	if err := os.MkdirAll(filepath.Join(repo, "catalog"), 0o755); err != nil {
		t.Fatal(err)
	}
	copyTree(t, src, filepath.Join(repo, "catalog"))

	run("init", "--initial-branch=main")
	run("add", "-A")
	run("commit", "-m", "seed catalog")

	commit = func(msg string) {
		t.Helper()
		run("add", "-A")
		run("commit", "-m", msg)
	}
	return repo, commit
}

func copyTree(t *testing.T, src, dst string) {
	t.Helper()
	err := filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(src, path)
		target := filepath.Join(dst, rel)
		if info.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(target, data, info.Mode().Perm())
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestGitRepoEnsureClones(t *testing.T) {
	repo, _ := gitFixture(t)
	g := catalog.GitRepo{URL: repo, Dir: filepath.Join(t.TempDir(), "mirror")}

	if err := g.Ensure(); err != nil {
		t.Fatalf("Ensure() error = %v", err)
	}
	if _, err := os.Stat(filepath.Join(g.Dir, "catalog", "catalog.json")); err != nil {
		t.Errorf("mirror missing catalog.json after Ensure: %v", err)
	}
}

func TestGitRepoEnsureHealsCorruptMirror(t *testing.T) {
	repo, _ := gitFixture(t)
	dir := filepath.Join(t.TempDir(), "mirror")
	// Corrupt mirror: a dir that is NOT a git repo.
	if err := os.MkdirAll(filepath.Join(dir, "junk"), 0o755); err != nil {
		t.Fatal(err)
	}
	g := catalog.GitRepo{URL: repo, Dir: dir}

	if err := g.Ensure(); err != nil {
		t.Fatalf("Ensure() on corrupt mirror error = %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "catalog", "catalog.json")); err != nil {
		t.Errorf("mirror not re-cloned: %v", err)
	}
}

func TestGitRepoHeads(t *testing.T) {
	repo, commit := gitFixture(t)
	g := catalog.GitRepo{URL: repo, Dir: filepath.Join(t.TempDir(), "mirror")}
	if err := g.Ensure(); err != nil {
		t.Fatal(err)
	}

	local, err := g.LocalHead()
	if err != nil || len(local) != 40 {
		t.Fatalf("LocalHead() = %q, %v — want 40-char SHA", local, err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	remote, err := g.RemoteHead(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if remote != local {
		t.Errorf("RemoteHead = %s, LocalHead = %s — should match right after clone", remote, local)
	}

	// New commit upstream → remote moves, local stays.
	os.WriteFile(filepath.Join(repo, "catalog", "skills", "golang", "SKILL.md"), []byte("# golang v2"), 0o644)
	commit("bump golang")

	remote2, err := g.RemoteHead(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if remote2 == local {
		t.Error("RemoteHead did not move after upstream commit")
	}
}

func TestGitRepoSync(t *testing.T) {
	repo, commit := gitFixture(t)
	g := catalog.GitRepo{URL: repo, Dir: filepath.Join(t.TempDir(), "mirror")}
	if err := g.Ensure(); err != nil {
		t.Fatal(err)
	}
	before, _ := g.LocalHead()

	os.WriteFile(filepath.Join(repo, "catalog", "skills", "golang", "SKILL.md"), []byte("# golang v2"), 0o644)
	commit("bump golang")

	after, err := g.Sync()
	if err != nil {
		t.Fatalf("Sync() error = %v", err)
	}
	if after == before {
		t.Error("Sync() did not advance the mirror")
	}
	data, _ := os.ReadFile(filepath.Join(g.Dir, "catalog", "skills", "golang", "SKILL.md"))
	if string(data) != "# golang v2" {
		t.Errorf("mirror content not updated: %q", data)
	}
}

func TestGitRepoImplementsSource(t *testing.T) {
	repo, _ := gitFixture(t)
	g := catalog.GitRepo{URL: repo, Dir: filepath.Join(t.TempDir(), "mirror")}

	var _ catalog.Source = g // compile-time check

	c, err := g.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if len(c.Profiles) == 0 {
		t.Error("Load() returned empty catalog")
	}
	if p := g.SkillPath("golang"); p != filepath.Join(g.Dir, "catalog", "skills", "golang") {
		t.Errorf("SkillPath = %q", p)
	}
}
```

- [ ] **Step 2: Verificar que fallan**

Run: `go test ./internal/catalog/ -run TestGitRepo`
Expected: FAIL — `undefined: catalog.GitRepo`.

- [ ] **Step 3: Implementar**

`internal/catalog/gitrepo.go`:

```go
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
```

- [ ] **Step 4: Verificar que pasan**

Run: `go test ./internal/catalog/ -v -run TestGitRepo`
Expected: PASS los 5.

- [ ] **Step 5: Commit**

```bash
git add internal/catalog/
git commit -m "feat: git-backed catalog source with managed mirror"
```

---

### Task 3: Manifiesto — `CatalogRef` gana `URL` y `Ref`

**Files:**
- Modify: `internal/manifest/manifest.go` (struct `CatalogRef`)
- Test: `internal/manifest/manifest_test.go` (agregar un test)

**Interfaces:**
- Consumes: struct existente.
- Produces: `type CatalogRef struct { Type string; Path string `json:"path,omitempty"`; URL string `json:"url,omitempty"`; Ref string `json:"ref,omitempty"` }` — Tasks 4-6 lo consumen EXACTO.

- [ ] **Step 1: Escribir el test que falla**

Agregar a `internal/manifest/manifest_test.go`:

```go
func TestSaveLoadGitCatalogRef(t *testing.T) {
	path := filepath.Join(t.TempDir(), "andes.json")
	m := &manifest.Manifest{
		Version: 1,
		Catalog: manifest.CatalogRef{
			Type: "git",
			URL:  "git@github.com:andespath/andes-ai.git",
			Ref:  "a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2",
		},
		Profiles:  []string{"andespath-core"},
		Installed: map[string]manifest.InstalledSkill{},
	}
	if err := m.Save(path); err != nil {
		t.Fatal(err)
	}
	got, err := manifest.Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if got.Catalog.Type != "git" || got.Catalog.URL != m.Catalog.URL || got.Catalog.Ref != m.Catalog.Ref {
		t.Errorf("git CatalogRef roundtrip = %+v", got.Catalog)
	}
	if got.Catalog.Path != "" {
		t.Errorf("Path should be empty for git refs, got %q", got.Catalog.Path)
	}
}
```

- [ ] **Step 2: Verificar que falla**

Run: `go test ./internal/manifest/`
Expected: FAIL — `unknown field URL`.

- [ ] **Step 3: Implementar**

En `internal/manifest/manifest.go`, reemplazar el struct `CatalogRef`:

```go
type CatalogRef struct {
	Type string `json:"type"`           // "local" | "git"
	Path string `json:"path,omitempty"` // local: absolute folder path
	URL  string `json:"url,omitempty"`  // git: repo URL
	Ref  string `json:"ref,omitempty"`  // git: commit SHA installed from
}
```

- [ ] **Step 4: Verificar que pasan (incluido roundtrip local existente)**

Run: `go test ./internal/manifest/ -v`
Expected: PASS todos.

- [ ] **Step 5: Commit**

```bash
git add internal/manifest/
git commit -m "feat: manifest catalog ref supports git url and commit sha"
```

---

### Task 4: Resolución de catálogo + init con git (`internal/cli/resolve.go`)

**Files:**
- Create: `internal/cli/resolve.go`
- Modify: `internal/cli/init.go` (usar la resolución; escribir CatalogRef correcto)
- Modify: `internal/cli/paths.go` (agregar `mirrorDir()`)
- Test: `internal/cli/resolve_test.go` + casos en `internal/cli/init_test.go`

**Interfaces:**
- Consumes: `catalog.GitRepo` (Task 2), `manifest.CatalogRef` (Task 3), `promptCatalogPath` existente.
- Produces (Task 5 y 6 consumen esto EXACTO):
  - `var defaultCatalogURL string` — se hornea con `-ldflags "-X github.com/andespath/andes-ai/internal/cli.defaultCatalogURL=<url>"`; vacía = sin default (transición pre-GitHub).
  - `func isGitURL(s string) bool`
  - `func mirrorDir() (string, error)` — `~/.andes/catalog`
  - `func resolveSource(catalogFlag string, prev *manifest.Manifest, yes bool) (catalog.Source, manifest.CatalogRef, error)` — resuelve flag → manifiesto → default horneado → prompt (o error con `--yes`). El `Ref` del CatalogRef retornado viene VACÍO; quien instala lo completa con `LocalHead()`.

- [ ] **Step 1: Escribir tests que fallan**

`internal/cli/resolve_test.go`:

```go
package cli

import "testing"

func TestIsGitURL(t *testing.T) {
	tests := []struct {
		in   string
		want bool
	}{
		{"git@github.com:andespath/andes-ai.git", true},
		{"https://github.com/andespath/andes-ai.git", true},
		{"https://github.com/andespath/andes-ai", true},
		{"ssh://git@github.com/x/y", true},
		{"./catalog", false},
		{"/abs/path/catalog", false},
		{"", false},
	}
	for _, tt := range tests {
		if got := isGitURL(tt.in); got != tt.want {
			t.Errorf("isGitURL(%q) = %v, want %v", tt.in, got, tt.want)
		}
	}
}
```

(Nota: package `cli` interno, no `cli_test` — isGitURL no es exportada. Es la única prueba white-box; el resto de la resolución se cubre vía tests de integración de init en Step 4.)

- [ ] **Step 2: Verificar que falla**

Run: `go test ./internal/cli/ -run TestIsGitURL`
Expected: FAIL — `undefined: isGitURL`.

- [ ] **Step 3: Implementar resolve.go, mirrorDir y el rewiring de init.go**

`internal/cli/paths.go` — agregar:

```go
// mirrorDir returns ~/.andes/catalog — the managed catalog mirror.
func mirrorDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("could not resolve home directory: %w", err)
	}
	return filepath.Join(home, ".andes", "catalog"), nil
}
```

(agregar `"path/filepath"` al import si falta)

`internal/cli/resolve.go`:

```go
package cli

import (
	"errors"
	"fmt"
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
```

`internal/cli/init.go` — reemplazar el bloque de resolución de catálogo (líneas del comentario `// Resolve catalog path...` hasta el `src := catalog.LocalDir{...}` y su `Load`) por:

```go
	src, catRef, err := resolveSource(catalogPath, prev, yes)
	if err != nil {
		return err
	}
	if g, ok := src.(catalog.GitRepo); ok {
		if _, statErr := os.Stat(g.Dir); statErr != nil {
			fmt.Fprintln(cmd.OutOrStdout(), "Fetching the andespath catalog…")
		}
		if err := g.Ensure(); err != nil {
			return err
		}
	}
	cat, err := src.Load()
	if err != nil {
		return err
	}
```

Y reemplazar el bloque final que construye `next` (el `absCatalog` + `manifest.CatalogRef{Type: "local", ...}`) por:

```go
	catRef, err = finalizeRef(src, catRef)
	if err != nil {
		return err
	}
	next := &manifest.Manifest{
		Version:   1,
		Catalog:   catRef,
		Profiles:  profiles,
		Installed: installed,
	}
```

(eliminar el `filepath.Abs` local que queda sin uso; ajustar imports: agregar `"os"`, remover `"path/filepath"` si ya no se usa)

Actualizar el usage del flag: `"path or git URL of the catalog"`.

- [ ] **Step 4: Agregar test de integración: init desde catálogo git**

Agregar a `internal/cli/init_test.go` (usa `gitFixtureCLI`, versión local del helper — cópialo desde el patrón de Task 2 adaptado a package `cli_test`, o expórtalo NO: duplicar el helper de ~40 líneas en `internal/cli/gitfixture_test.go` con nombre `gitFixture`; package distinto, sin export contaminante):

```go
func TestInitFromGitCatalog(t *testing.T) {
	home := t.TempDir()
	repo, _ := gitFixture(t)

	out, err := runAndes(t, home,
		"init", "--catalog", repo, "--profiles", "tri-fleet", "--yes")
	if err != nil {
		t.Fatalf("init from git: %v\n%s", err, out)
	}

	// Skills installed.
	if _, err := os.Stat(filepath.Join(home, ".claude", "skills", "golang", "SKILL.md")); err != nil {
		t.Errorf("skill not installed: %v", err)
	}
	// Mirror created under ~/.andes/catalog.
	if _, err := os.Stat(filepath.Join(home, ".andes", "catalog", "catalog", "catalog.json")); err != nil {
		t.Errorf("managed mirror missing: %v", err)
	}
	// Manifest has git type + 40-char ref.
	m, err := manifest.Load(filepath.Join(home, ".claude", "andes.json"))
	if err != nil || m == nil {
		t.Fatal(err)
	}
	if m.Catalog.Type != "git" || m.Catalog.URL != repo || len(m.Catalog.Ref) != 40 {
		t.Errorf("manifest catalog = %+v", m.Catalog)
	}
}
```

NOTA para el implementer del helper: `gitFixture` en package `cli_test` debe sembrar desde `../../catalog` (helper `fixtureCatalog(t)` ya existente devuelve ese path absoluto).

Run: `go test ./internal/cli/ -run 'TestIsGitURL|TestInitFromGitCatalog' -v`
Expected: PASS ambos. NOTA: `isGitURL("git@...")` es un URL de fixture local en el test de integración — el fixture path NO parece git URL, por eso entra como local… **NO**: el test pasa el path del fixture como `--catalog` y el fixture es un repo git PERO el path es local — `sourceFor` lo tratará como `LocalDir` y fallará porque el catálogo está en el subdir `catalog/`. **El implementer debe pasar la URL con sufijo que la marque como git**: usar `repo` directo NO funciona. Solución mandada: en el test, pasar `--catalog` con el prefijo de file URL git-compatible: `"file://" + repo`. Y agregar `strings.HasPrefix(s, "file://")` a `isGitURL`. git clona file:// URLs perfectamente. Ajustar el test para usar `fileURL := "file://" + repo`.

- [ ] **Step 5: Suite completa**

Run: `go test ./... && go vet ./... && gofmt -l .`
Expected: verde, silencioso. Los tests existentes de init local NO cambian (paths locales siguen siendo LocalDir).

- [ ] **Step 6: Commit**

```bash
git add internal/cli/
git commit -m "feat: catalog resolution with git urls and baked default"
```

---

### Task 5: Comando `andes update`

**Files:**
- Create: `internal/cli/update.go`
- Modify: `internal/cli/init.go` (extraer helper compartido `installAndSave`)
- Modify: `internal/cli/root.go` (registrar subcomando)
- Test: `internal/cli/update_test.go`

**Interfaces:**
- Consumes: `resolveSource`/`finalizeRef` (Task 4), `GitRepo.Sync` (Task 2), `installer.Plan/Apply`, manifiesto.
- Produces:
  - `func newUpdateCmd() *cobra.Command` con flag `--yes`.
  - `func installAndSave(cmd *cobra.Command, src catalog.Source, cat *catalog.Catalog, prev *manifest.Manifest, profiles []string, catRef manifest.CatalogRef, yes bool) error` — el bloque plan→print→confirm→apply→finalizeRef→save extraído de `runInit` (init y update lo comparten). La TUI (Task 6) invoca `update --yes` in-process.

- [ ] **Step 1: Escribir tests que fallan**

`internal/cli/update_test.go`:

```go
package cli_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/andespath/andes-ai/internal/manifest"
)

func TestUpdateNoManifest(t *testing.T) {
	home := t.TempDir()
	if _, err := runAndes(t, home, "update", "--yes"); err == nil {
		t.Error("update without manifest should fail with actionable error")
	}
}

func TestUpdateLocalCatalogNothingToDo(t *testing.T) {
	home := t.TempDir()
	if _, err := runAndes(t, home,
		"init", "--catalog", fixtureCatalog(t), "--profiles", "tri-fleet", "--yes"); err != nil {
		t.Fatal(err)
	}
	out, err := runAndes(t, home, "update", "--yes")
	if err == nil || !strings.Contains(err.Error()+out, "local catalog") {
		t.Errorf("update on local catalog should explain there is nothing to update: %v\n%s", err, out)
	}
}

func TestUpdateAlreadyUpToDate(t *testing.T) {
	home := t.TempDir()
	repo, _ := gitFixture(t)
	if _, err := runAndes(t, home,
		"init", "--catalog", "file://"+repo, "--profiles", "tri-fleet", "--yes"); err != nil {
		t.Fatal(err)
	}
	out, err := runAndes(t, home, "update", "--yes")
	if err != nil {
		t.Fatalf("update: %v\n%s", err, out)
	}
	if !strings.Contains(out, "Already up to date") {
		t.Errorf("want 'Already up to date':\n%s", out)
	}
}

func TestUpdatePullsNewSkillVersion(t *testing.T) {
	home := t.TempDir()
	repo, commit := gitFixture(t)
	if _, err := runAndes(t, home,
		"init", "--catalog", "file://"+repo, "--profiles", "tri-fleet", "--yes"); err != nil {
		t.Fatal(err)
	}

	// Upstream change.
	os.WriteFile(filepath.Join(repo, "catalog", "skills", "golang", "SKILL.md"), []byte("# golang v2"), 0o644)
	commit("bump golang")

	out, err := runAndes(t, home, "update", "--yes")
	if err != nil {
		t.Fatalf("update: %v\n%s", err, out)
	}
	if !strings.Contains(out, "update") || !strings.Contains(out, "golang") {
		t.Errorf("plan should show golang as update:\n%s", out)
	}

	// Installed skill actually refreshed.
	data, _ := os.ReadFile(filepath.Join(home, ".claude", "skills", "golang", "SKILL.md"))
	if string(data) != "# golang v2" {
		t.Errorf("installed skill not refreshed: %q", data)
	}
	// Manifest ref advanced.
	m, _ := manifest.Load(filepath.Join(home, ".claude", "andes.json"))
	before := m.Catalog.Ref
	if before == "" {
		t.Fatal("manifest ref empty")
	}
	out2, err := runAndes(t, home, "update", "--yes")
	if err != nil || !strings.Contains(out2, "Already up to date") {
		t.Errorf("second update should be up to date: %v\n%s", err, out2)
	}
}
```

- [ ] **Step 2: Verificar que fallan**

Run: `go test ./internal/cli/ -run TestUpdate`
Expected: FAIL — `unknown command "update"`.

- [ ] **Step 3: Implementar**

Primero el refactor en `internal/cli/init.go`: mover el bloque desde `actions, err := installer.Plan(...)` hasta el `fmt.Fprintf(... "✓ %d skills up to date ...")` inclusive a una función nueva al final del archivo:

```go
// installAndSave runs the shared plan→confirm→apply→save pipeline used by
// both init and update.
func installAndSave(cmd *cobra.Command, src catalog.Source, cat *catalog.Catalog, prev *manifest.Manifest, profiles []string, catRef manifest.CatalogRef, yes bool) error {
	actions, err := installer.Plan(src, cat, prev, profiles)
	if err != nil {
		return err
	}

	fmt.Fprintln(cmd.OutOrStdout(), "Plan:")
	for _, a := range actions {
		fmt.Fprintf(cmd.OutOrStdout(), "  %-12s %s\n", a.Type, a.SkillID)
	}

	if !yes {
		ok, err := confirmPlan()
		if err != nil {
			return err
		}
		if !ok {
			fmt.Fprintln(cmd.OutOrStdout(), "Aborted — nothing was touched.")
			return nil
		}
	}

	sDir, err := skillsDir()
	if err != nil {
		return err
	}
	installed, err := installer.Apply(src, actions, sDir)
	if err != nil {
		return err
	}

	catRef, err = finalizeRef(src, catRef)
	if err != nil {
		return err
	}
	next := &manifest.Manifest{
		Version:   1,
		Catalog:   catRef,
		Profiles:  profiles,
		Installed: installed,
	}
	mPath, err := manifest.DefaultPath()
	if err != nil {
		return err
	}
	if err := next.Save(mPath); err != nil {
		return err
	}

	fmt.Fprintf(cmd.OutOrStdout(), "✓ %d skills up to date in %s\n", len(installed), sDir)
	return nil
}
```

`runInit` termina llamando `return installAndSave(cmd, src, cat, prev, profiles, catRef, yes)`.

`internal/cli/update.go`:

```go
package cli

import (
	"errors"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/andespath/andes-ai/internal/catalog"
	"github.com/andespath/andes-ai/internal/manifest"
)

func newUpdateCmd() *cobra.Command {
	var yes bool
	cmd := &cobra.Command{
		Use:   "update",
		Short: "Syncs the catalog mirror and refreshes outdated skills",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runUpdate(cmd, yes)
		},
	}
	cmd.Flags().BoolVar(&yes, "yes", false, "apply without confirmation prompt")
	return cmd
}

func runUpdate(cmd *cobra.Command, yes bool) error {
	mPath, err := manifest.DefaultPath()
	if err != nil {
		return err
	}
	prev, err := manifest.Load(mPath)
	if err != nil {
		return err
	}
	if prev == nil {
		return errors.New("no manifest found: you haven't run `andes init` yet")
	}
	if prev.Catalog.Type != "git" {
		return errors.New("nothing to update: local catalog (re-run `andes init` to refresh from a local folder)")
	}

	dir, err := mirrorDir()
	if err != nil {
		return err
	}
	g := catalog.GitRepo{URL: prev.Catalog.URL, Dir: dir}
	if err := g.Ensure(); err != nil {
		return err
	}
	newHead, err := g.Sync()
	if err != nil {
		return err
	}
	if newHead == prev.Catalog.Ref {
		fmt.Fprintln(cmd.OutOrStdout(), "Already up to date")
		return nil
	}

	cat, err := g.Load()
	if err != nil {
		return err
	}
	catRef := manifest.CatalogRef{Type: "git", URL: prev.Catalog.URL}
	return installAndSave(cmd, g, cat, prev, prev.Profiles, catRef, yes)
}
```

En `internal/cli/root.go`: `root.AddCommand(newInitCmd(), newListCmd(), newDoctorCmd(), newUpdateCmd())`.

- [ ] **Step 4: Verificar que pasan**

Run: `go test ./internal/cli/ -v -run TestUpdate` y luego `go test ./...`
Expected: PASS todos.

- [ ] **Step 5: Commit**

```bash
git add internal/cli/
git commit -m "feat: andes update syncs mirror and refreshes skills"
```

---

### Task 6: TUI — check de frescura async, banner y tecla `u`

**Files:**
- Modify: `internal/tui/model.go`
- Modify: `internal/cli/root.go` (pasar el check)
- Create: `internal/cli/freshness.go`
- Test: `internal/tui/model_test.go` (agregar casos)

**Interfaces:**
- Consumes: `GitRepo.RemoteHead` (Task 2), manifiesto (Task 3), comando `update` (Task 5, invocado in-process con `--yes`).
- Produces:
  - En tui: `type FreshnessMsg struct { Outdated bool; Offline bool }`, `type UpdateCheck func() FreshnessMsg`, `func Run(newRoot func() *cobra.Command, check UpdateCheck) error` (firma AMPLIADA — root.go se adapta), `New(newRoot, check)`.
  - En cli: `func checkCatalogFreshness() tui.FreshnessMsg` — lee manifiesto; si type git → RemoteHead con timeout 2s → compara con Ref; errores → Offline; type local o sin manifiesto → ni outdated ni offline.

- [ ] **Step 1: Escribir tests que fallan (Update directo, patrón existente)**

Agregar a `internal/tui/model_test.go`:

```go
func TestFreshnessOutdatedShowsBanner(t *testing.T) {
	m := New(nil, nil)
	updated, _ := m.Update(FreshnessMsg{Outdated: true})
	mm := updated.(Model)
	if !strings.Contains(mm.View(), "press u to update") {
		t.Errorf("banner missing from view:\n%s", mm.View())
	}
}

func TestFreshnessOfflineShowsFooterNote(t *testing.T) {
	m := New(nil, nil)
	updated, _ := m.Update(FreshnessMsg{Offline: true})
	mm := updated.(Model)
	if !strings.Contains(mm.View(), "offline") {
		t.Errorf("offline note missing:\n%s", mm.View())
	}
}

func TestPressUWithoutUpdateAvailableDoesNothing(t *testing.T) {
	m := New(nil, nil)
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("u")})
	if cmd != nil {
		t.Error("u without update available should be a no-op")
	}
}

func TestPressUWithUpdateAvailableRunsUpdate(t *testing.T) {
	m := New(func() *cobra.Command { return &cobra.Command{Use: "andes"} }, nil)
	updated, _ := m.Update(FreshnessMsg{Outdated: true})
	mm := updated.(Model)
	_, cmd := mm.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("u")})
	if cmd == nil {
		t.Fatal("u with update available should dispatch the update command")
	}
}

func TestCmdResultClearsUpdateBanner(t *testing.T) {
	m := New(nil, nil)
	updated, _ := m.Update(FreshnessMsg{Outdated: true})
	updated, _ = updated.(Model).Update(cmdResultMsg{cmdID: "update", output: "done"})
	mm := updated.(Model)
	// Back on menu after esc: banner must be gone.
	updated, _ = mm.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if strings.Contains(updated.(Model).View(), "press u to update") {
		t.Error("banner should clear after an update run")
	}
}
```

(agregar imports que falten: `strings`, `tea`, `cobra`)

- [ ] **Step 2: Verificar que fallan**

Run: `go test ./internal/tui/`
Expected: FAIL — `undefined: FreshnessMsg` y firma de `New`.

- [ ] **Step 3: Implementar en `internal/tui/model.go`**

Cambios puntuales:

1. Tipos nuevos (junto a los messages):

```go
// FreshnessMsg reports the async catalog freshness check result.
type FreshnessMsg struct {
	Outdated bool
	Offline  bool
}

// UpdateCheck is injected by the caller (cli) so tui stays decoupled from
// manifest/git specifics and tests can fake it.
type UpdateCheck func() FreshnessMsg
```

2. `Model` gana campos `check UpdateCheck`, `outdated bool`, `offline bool`.

3. `New(newRoot func() *cobra.Command, check UpdateCheck) Model` — guarda check.

4. `Init()`:

```go
func (m Model) Init() tea.Cmd {
	if m.check == nil {
		return nil
	}
	check := m.check
	return func() tea.Msg { return check() }
}
```

5. En `Update`, caso nuevo:

```go
	case FreshnessMsg:
		m.outdated = msg.Outdated
		m.offline = msg.Offline
		return m, nil
```

6. En `updateMenu`, tecla `u`:

```go
	case msg.Type == tea.KeyRunes && string(msg.Runes) == "u":
		if !m.outdated {
			return m, nil
		}
		return m.runInProcess("update", "--yes")
```

Para eso, extraer del `case "list", "doctor"` de `selectOption` un helper reutilizable:

```go
// runInProcess executes a subcommand with captured output, async.
func (m Model) runInProcess(args ...string) (tea.Model, tea.Cmd) {
	newRoot := m.newRoot
	cmdID := args[0]
	return m, func() tea.Msg {
		var buf bytes.Buffer
		root := newRoot()
		root.SetArgs(args)
		root.SetOut(&buf)
		root.SetErr(&buf)
		execErr := root.Execute()
		output := buf.String()
		if execErr != nil {
			if output != "" && !strings.HasSuffix(output, "\n") {
				output += "\n"
			}
			output += execErr.Error()
		}
		return cmdResultMsg{cmdID: cmdID, output: output, err: nil}
	}
}
```

y `selectOption` para list/doctor pasa a `return m.runInProcess(opt.id)`.

7. En el handler de `cmdResultMsg`: si `msg.cmdID == "update"` → `m.outdated = false` (el banner se limpia tras actualizar).

8. En `viewMenu`: tras el título, si `m.outdated`:

```go
	warn := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#f6c177"))
	sb.WriteString(warn.Render("⚠ catalog updated — press u to update"))
	sb.WriteString("\n\n")
```

y en el footer: si `m.offline`, agregar ` • offline` al help line.

9. `Run(newRoot func() *cobra.Command, check UpdateCheck) error` — pasa check a `New`.

`internal/cli/freshness.go`:

```go
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
```

`internal/cli/root.go`: `return tui.Run(NewRootCmd, checkCatalogFreshness)`.

- [ ] **Step 4: Verificar que pasan**

Run: `go test ./internal/tui/ -v && go test ./...`
Expected: PASS todos (los tests viejos de tui usan `New(...)` — actualizarlos a la firma nueva con `nil` check es parte del step).

- [ ] **Step 5: Commit**

```bash
git add internal/tui/ internal/cli/
git commit -m "feat: tui freshness banner with one-key update"
```

---

### Task 7: `install.sh` + CI de releases + README

**Files:**
- Create: `install.sh` (raíz, ejecutable)
- Create: `.github/workflows/release.yml`
- Modify: `README.md` (sección de instalación + update)

**Interfaces:**
- Consumes: `defaultCatalogURL` ldflag (Task 4).
- Produces: instalación de dos líneas para devs; releases multi-plataforma en tag.

- [ ] **Step 1: Escribir `install.sh`**

```bash
#!/usr/bin/env bash
# andes installer — downloads a release binary via gh, or builds from source.
set -euo pipefail

REPO="andespath/andes-ai"          # update when the repo lands on company GitHub
BIN_DIR="${HOME}/.local/bin"
# Baked at build time when building from source; empty until the repo is on GitHub.
CATALOG_URL="${ANDES_CATALOG_URL:-}"

OS="$(uname -s | tr '[:upper:]' '[:lower:]')"
ARCH="$(uname -m)"
case "$ARCH" in
  x86_64) ARCH="amd64" ;;
  aarch64 | arm64) ARCH="arm64" ;;
esac

mkdir -p "$BIN_DIR"

install_from_release() {
  command -v gh >/dev/null 2>&1 || return 1
  gh release download --repo "$REPO" --pattern "andes-${OS}-${ARCH}" \
    --output "$BIN_DIR/andes" --clobber 2>/dev/null || return 1
  chmod +x "$BIN_DIR/andes"
  echo "installed release binary → $BIN_DIR/andes"
}

install_from_source() {
  command -v go >/dev/null 2>&1 || return 1
  local src_dir
  src_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
  local ldflags=""
  if [ -n "$CATALOG_URL" ]; then
    ldflags="-X github.com/andespath/andes-ai/internal/cli.defaultCatalogURL=${CATALOG_URL}"
  fi
  (cd "$src_dir" && go build -ldflags "$ldflags" -o "$BIN_DIR/andes" ./cmd/andes)
  echo "built from source → $BIN_DIR/andes"
}

if ! install_from_release; then
  echo "no release available (or gh missing) — trying to build from source…"
  if ! install_from_source; then
    echo "error: need either the gh CLI (for release download) or Go (to build)." >&2
    echo "install one of them and re-run ./install.sh" >&2
    exit 1
  fi
fi

case ":$PATH:" in
  *":$BIN_DIR:"*) ;;
  *)
    echo ""
    echo "⚠ $BIN_DIR is not in your PATH. Add this to your shell rc:"
    echo "    export PATH=\"\$PATH:$BIN_DIR\""
    ;;
esac

echo ""
echo "Done — run 'andes' to get started."
```

```bash
chmod +x install.sh
```

- [ ] **Step 2: Verificar el camino from-source localmente**

Run: `HOME=$(mktemp -d) bash install.sh && ls "$HOME/.local/bin"` — NO: HOME temporal rompe gh/go caches. En su lugar:
Run: `bash -n install.sh` (syntax check) y `BIN_DIR_TEST=$(mktemp -d) && sed "s|\${HOME}/.local/bin|$BIN_DIR_TEST|" install.sh | bash && test -x "$BIN_DIR_TEST/andes" && echo INSTALL_OK`
Expected: `INSTALL_OK` (por el camino source — no hay releases aún).

- [ ] **Step 3: Escribir `.github/workflows/release.yml`**

```yaml
name: release
on:
  push:
    tags: ["v*"]
permissions:
  contents: write
jobs:
  release:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: "1.23"
      - name: Test
        run: go test ./...
      - name: Build matrix
        env:
          CATALOG_URL: ${{ vars.ANDES_CATALOG_URL }}
        run: |
          mkdir -p dist
          for os in darwin linux; do
            for arch in amd64 arm64; do
              GOOS=$os GOARCH=$arch go build \
                -ldflags "-X github.com/andespath/andes-ai/internal/cli.defaultCatalogURL=${CATALOG_URL}" \
                -o "dist/andes-$os-$arch" ./cmd/andes
            done
          done
      - name: Create release
        env:
          GH_TOKEN: ${{ github.token }}
        run: gh release create "${GITHUB_REF_NAME}" dist/* --generate-notes
```

- [ ] **Step 4: Actualizar README**

Reemplazar la sección Quickstart por:

```markdown
## Install

```bash
gh repo clone andespath/andes-ai && ./andes-ai/install.sh
```

Then just run:

```bash
andes
```

First run clones the company catalog and walks you through picking profiles.
When the catalog gets new skills, the TUI shows "⚠ catalog updated — press u
to update". From scripts: `andes update --yes`.

## Development

```bash
go build -o andes ./cmd/andes
./andes init --catalog ./catalog --profiles andespath-core --yes   # local catalog
```
```

- [ ] **Step 5: Commit**

```bash
git add install.sh .github/ README.md
git commit -m "feat: installer script and release workflow"
```

---

### Task 8: E2E — ciclo de vida git completo

**Files:**
- Test: `internal/cli/e2e_git_test.go`

**Interfaces:**
- Consumes: todo lo anterior.

- [ ] **Step 1: Escribir el E2E**

```go
package cli_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/andespath/andes-ai/internal/manifest"
)

// TestGitLifecycle walks the full v2 story: init from the company repo →
// upstream change → update detects and applies → doctor healthy → ref advanced.
func TestGitLifecycle(t *testing.T) {
	home := t.TempDir()
	repo, commit := gitFixture(t)
	url := "file://" + repo

	// 1. Fresh dev: init from git.
	out, err := runAndes(t, home,
		"init", "--catalog", url, "--profiles", "andespath-core,tri-fleet", "--yes")
	if err != nil {
		t.Fatalf("init: %v\n%s", err, out)
	}

	// 2. Doctor healthy against the mirror.
	if out, err = runAndes(t, home, "doctor"); err != nil {
		t.Fatalf("doctor: %v\n%s", err, out)
	}
	m1, _ := manifest.Load(filepath.Join(home, ".claude", "andes.json"))

	// 3. Upstream: a skill changes.
	os.WriteFile(filepath.Join(repo, "catalog", "skills", "code-review", "SKILL.md"),
		[]byte("# code review v2"), 0o644)
	commit("update code-review")

	// 4. Update pulls it.
	out, err = runAndes(t, home, "update", "--yes")
	if err != nil {
		t.Fatalf("update: %v\n%s", err, out)
	}
	if !strings.Contains(out, "code-review") {
		t.Errorf("update plan should mention code-review:\n%s", out)
	}
	data, _ := os.ReadFile(filepath.Join(home, ".claude", "skills", "code-review", "SKILL.md"))
	if string(data) != "# code review v2" {
		t.Errorf("skill not refreshed: %q", data)
	}

	// 5. Ref advanced; doctor still healthy.
	m2, _ := manifest.Load(filepath.Join(home, ".claude", "andes.json"))
	if m2.Catalog.Ref == m1.Catalog.Ref {
		t.Error("manifest ref did not advance after update")
	}
	if out, err = runAndes(t, home, "doctor"); err != nil {
		t.Fatalf("doctor after update: %v\n%s", err, out)
	}
}
```

- [ ] **Step 2: Correr TODO el gate**

Run: `go test ./... -v && go vet ./... && gofmt -l .`
Expected: verde total, gofmt silencioso.

- [ ] **Step 3: Commit**

```bash
git add internal/cli/e2e_git_test.go
git commit -m "test: end-to-end git catalog lifecycle"
```

---

## Self-Review (aplicado)

- **Cobertura del spec:** Sección 1 (GitRepo) → Task 2; Sección 2 (manifiesto+detección) → Tasks 3, 6; Sección 3 (init cero-preguntas vía default horneado, update, errores) → Tasks 4, 5; Sección 4 (testing sin red) → fixtures git en Tasks 2/4/5/8; Sección 5 (install.sh + CI) → Task 7; mover catálogo → Task 1; nota de transición (URL vacía → prompt) → `defaultCatalogURL` en Task 4 y `CATALOG_URL` opcional en Task 7.
- **Placeholders:** ninguno; todo step con código/comando completo. El único ajuste dinámico explícito (fixture como `file://` URL) está mandado con su razón.
- **Consistencia de tipos:** `GitRepo{URL, Dir}` (T2) consumido igual en T4/T5/T6; `CatalogRef{Type,Path,URL,Ref}` (T3) en T4/T5; `resolveSource`/`finalizeRef` (T4) en T5; `FreshnessMsg`/`UpdateCheck`/`New(newRoot, check)` (T6) coherentes; `installAndSave` extraída en T5 y usada por ambos comandos; helper `gitFixture` duplicado adrede en `cli_test` (packages distintos, documentado).
