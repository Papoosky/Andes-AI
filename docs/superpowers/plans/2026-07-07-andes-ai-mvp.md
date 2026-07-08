# andes-ai MVP Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** CLI en Go que instala skills de agentes IA desde un catálogo local hacia `~/.claude/skills/` según perfiles, con manifiesto-recibo y diagnóstico de drift.

**Architecture:** Comandos Cobra sin lógica que orquestan módulos internos (`catalog`, `manifest`, `installer`, `doctor`, `hashdir`). `catalog.Source` es interface (impl `LocalDir` en v1). Modelo COPY + manifiesto atómico en `~/.claude/andes.json` con hash sha256 por skill.

**Tech Stack:** Go 1.22+, spf13/cobra, charmbracelet/huh. Spec: `docs/superpowers/specs/2026-07-07-andes-ai-mvp-design.md`.

## Global Constraints

- Módulo Go: `github.com/andespath/andes-ai`. Go 1.22 como piso.
- Mensajes de error al usuario: en español, accionables (qué pasó + qué hacer). Nunca stack traces.
- Los comandos Cobra NO contienen lógica de negocio — solo flags, llamadas a `internal/*` y formato de salida.
- `andes` solo toca skills listadas en el manifiesto (`installed`) — jamás otras carpetas de `~/.claude/skills/`.
- Escritura del manifiesto SIEMPRE atómica (temp + rename), SIEMPRE al final del install.
- Todos los paths de usuario derivan de `os.UserHomeDir()` (los tests overridean con `t.Setenv("HOME", ...)`).
- Commits: Conventional Commits en inglés, sin atribución de AI, sin Co-Authored-By.
- Hash de skill: `"sha256:" + hex` sobre archivos del dir ordenados lexicográficamente (rel path + `\x00` + contenido + `\x00`).
- TDD: cada task escribe el test primero, lo ve fallar, implementa mínimo, lo ve pasar, commitea.

---

### Task 1: Bootstrap del módulo Go + comando raíz Cobra

**Files:**
- Create: `go.mod` (vía `go mod init`)
- Create: `cmd/andes/main.go`
- Create: `internal/cli/root.go`
- Test: `internal/cli/root_test.go`

**Interfaces:**
- Consumes: nada (greenfield)
- Produces: `cli.NewRootCmd() *cobra.Command` — todos los subcomandos posteriores se agregan acá.

- [ ] **Step 1: Inicializar repo y módulo**

```bash
cd /Users/pablozuniga/Desktop/workspace/andes-ai
git init
printf '/andes\n*.tmp\n' > .gitignore   # /andes ANCLADO: sin la barra, git ignora cmd/andes/ entero
go mod init github.com/andespath/andes-ai
go get github.com/spf13/cobra@v1.8.1
```

- [ ] **Step 2: Escribir el test que falla**

`internal/cli/root_test.go`:

```go
package cli_test

import (
	"bytes"
	"testing"

	"github.com/andespath/andes-ai/internal/cli"
)

func TestRootCmdShowsHelp(t *testing.T) {
	root := cli.NewRootCmd()
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetArgs([]string{"--help"})

	if err := root.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if !bytes.Contains(out.Bytes(), []byte("andes")) {
		t.Errorf("help no menciona 'andes':\n%s", out.String())
	}
}
```

- [ ] **Step 3: Verificar que falla**

Run: `go test ./internal/cli/`
Expected: FAIL — `undefined: cli.NewRootCmd` (error de compilación cuenta como test rojo).

- [ ] **Step 4: Implementación mínima**

`internal/cli/root.go`:

```go
package cli

import "github.com/spf13/cobra"

// NewRootCmd builds the andes root command. Subcommands attach here.
func NewRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:           "andes",
		Short:         "Gestor de skills de agentes IA de andespath",
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	return root
}
```

`cmd/andes/main.go`:

```go
package main

import (
	"fmt"
	"os"

	"github.com/andespath/andes-ai/internal/cli"
)

func main() {
	if err := cli.NewRootCmd().Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
}
```

- [ ] **Step 5: Verificar que pasa y compilar**

Run: `go test ./... && go build ./cmd/andes`
Expected: PASS, binario `andes` generado.

- [ ] **Step 6: Commit**

```bash
git add go.mod go.sum .gitignore cmd/ internal/ docs/
git commit -m "feat: bootstrap andes CLI with cobra root command"
```

---

### Task 2: Catálogo fixture + `internal/catalog` (carga, validación, resolución)

**Files:**
- Create: `testdata/catalog/catalog.json`
- Create: `testdata/catalog/skills/git-conventions/SKILL.md`
- Create: `testdata/catalog/skills/code-review/SKILL.md`
- Create: `testdata/catalog/skills/golang/SKILL.md`
- Create: `internal/catalog/catalog.go`
- Create: `internal/catalog/localdir.go`
- Create: `internal/catalog/resolve.go`
- Test: `internal/catalog/localdir_test.go`, `internal/catalog/resolve_test.go`

**Interfaces:**
- Consumes: nada
- Produces:
  - `type Catalog struct { Name string; Profiles map[string]Profile }`
  - `type Profile struct { Description string; Skills []string }`
  - `type Source interface { Load() (*Catalog, error); SkillPath(id string) string }`
  - `type LocalDir struct { Root string }` (implementa `Source`)
  - `func ResolveSkills(c *Catalog, profiles []string) (map[string]string, error)` — retorna skillID → nombre del perfil que la trajo (primer perfil en el orden pedido gana); error si un perfil no existe.

- [ ] **Step 1: Crear el fixture (contenido real mínimo, no lorem ipsum)**

`testdata/catalog/catalog.json`:

```json
{
  "name": "andespath",
  "profiles": {
    "andespath-core": {
      "description": "Baseline para todos en andespath",
      "skills": ["git-conventions", "code-review"]
    },
    "tri-fleet": {
      "description": "Equipo TRI fleet manager",
      "skills": ["golang"]
    }
  }
}
```

`testdata/catalog/skills/git-conventions/SKILL.md`:

```markdown
---
name: git-conventions
description: Convenciones de git de andespath. Usar al escribir commits o crear branches.
---

# Convenciones de Git — andespath

- Commits en Conventional Commits: `tipo(scope): descripción` (feat, fix, chore, docs, refactor, test).
- Mensajes en inglés, modo imperativo, sin punto final.
- Branches: `tipo/descripcion-corta` (ej. `feat/user-auth`).
- Un commit = un cambio lógico.
```

`testdata/catalog/skills/code-review/SKILL.md`:

```markdown
---
name: code-review
description: Estándar de code review de andespath. Usar al revisar PRs o preparar código para review.
---

# Code Review — andespath

- Todo PR necesita al menos 1 aprobación antes de merge.
- El reviewer verifica: correctitud, tests, naming, y que el PR haga UNA sola cosa.
- Comentarios accionables: proponer alternativa, no solo señalar el problema.
- PRs de más de ~400 líneas se piden dividir.
```

`testdata/catalog/skills/golang/SKILL.md`:

```markdown
---
name: golang
description: Best practices de Go en andespath. Usar al escribir o revisar código Go.
---

# Go — andespath

- Errores: envolver con contexto (`fmt.Errorf("...: %w", err)`), nunca ignorar.
- Tests table-driven para lógica con múltiples casos.
- Interfaces chicas, definidas del lado del consumidor.
- `gofmt` y `go vet` limpios antes de commitear.
```

- [ ] **Step 2: Escribir tests que fallan (carga + validación)**

`internal/catalog/localdir_test.go`:

```go
package catalog_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/andespath/andes-ai/internal/catalog"
)

func fixtureDir(t *testing.T) string {
	t.Helper()
	abs, err := filepath.Abs("../../testdata/catalog")
	if err != nil {
		t.Fatal(err)
	}
	return abs
}

func TestLocalDirLoadValid(t *testing.T) {
	src := catalog.LocalDir{Root: fixtureDir(t)}
	c, err := src.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if c.Name != "andespath" {
		t.Errorf("Name = %q, want %q", c.Name, "andespath")
	}
	if len(c.Profiles) != 2 {
		t.Errorf("len(Profiles) = %d, want 2", len(c.Profiles))
	}
	if got := c.Profiles["andespath-core"].Skills; len(got) != 2 {
		t.Errorf("andespath-core skills = %v, want 2 skills", got)
	}
}

func TestLocalDirLoadErrors(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(t *testing.T) string // returns catalog root
		wantErr string
	}{
		{
			name: "carpeta sin catalog.json",
			setup: func(t *testing.T) string {
				return t.TempDir()
			},
			wantErr: "no pude leer el catálogo",
		},
		{
			name: "json inválido",
			setup: func(t *testing.T) string {
				dir := t.TempDir()
				os.WriteFile(filepath.Join(dir, "catalog.json"), []byte("{no es json"), 0o644)
				return dir
			},
			wantErr: "catalog.json inválido",
		},
		{
			name: "perfil referencia skill inexistente",
			setup: func(t *testing.T) string {
				dir := t.TempDir()
				os.WriteFile(filepath.Join(dir, "catalog.json"), []byte(`{
					"name": "x",
					"profiles": {"p1": {"description": "d", "skills": ["fantasma"]}}
				}`), 0o644)
				return dir
			},
			wantErr: "fantasma",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			src := catalog.LocalDir{Root: tt.setup(t)}
			_, err := src.Load()
			if err == nil {
				t.Fatal("Load() = nil error, want error")
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("error = %q, want contains %q", err, tt.wantErr)
			}
		})
	}
}

func TestSkillPath(t *testing.T) {
	src := catalog.LocalDir{Root: "/tmp/cat"}
	got := src.SkillPath("golang")
	want := filepath.Join("/tmp/cat", "skills", "golang")
	if got != want {
		t.Errorf("SkillPath = %q, want %q", got, want)
	}
}
```

`internal/catalog/resolve_test.go`:

```go
package catalog_test

import (
	"strings"
	"testing"

	"github.com/andespath/andes-ai/internal/catalog"
)

func twoProfileCatalog() *catalog.Catalog {
	return &catalog.Catalog{
		Name: "andespath",
		Profiles: map[string]catalog.Profile{
			"core": {Description: "base", Skills: []string{"git-conventions", "code-review"}},
			"tri":  {Description: "tri", Skills: []string{"golang", "git-conventions"}},
		},
	}
}

func TestResolveSkills(t *testing.T) {
	tests := []struct {
		name     string
		profiles []string
		want     map[string]string
		wantErr  string
	}{
		{
			name:     "un perfil",
			profiles: []string{"core"},
			want:     map[string]string{"git-conventions": "core", "code-review": "core"},
		},
		{
			name:     "dedup: skill compartida queda con el primer perfil",
			profiles: []string{"core", "tri"},
			want: map[string]string{
				"git-conventions": "core",
				"code-review":     "core",
				"golang":          "tri",
			},
		},
		{
			name:     "perfil inexistente",
			profiles: []string{"nope"},
			wantErr:  `el perfil "nope" no existe`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := catalog.ResolveSkills(twoProfileCatalog(), tt.profiles)
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("error = %v, want contains %q", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("ResolveSkills() error = %v", err)
			}
			if len(got) != len(tt.want) {
				t.Fatalf("got %v, want %v", got, tt.want)
			}
			for id, profile := range tt.want {
				if got[id] != profile {
					t.Errorf("skill %q → %q, want %q", id, got[id], profile)
				}
			}
		})
	}
}
```

- [ ] **Step 3: Verificar que fallan**

Run: `go test ./internal/catalog/`
Expected: FAIL — `undefined: catalog.LocalDir`, `undefined: catalog.ResolveSkills`.

- [ ] **Step 4: Implementar**

`internal/catalog/catalog.go`:

```go
// Package catalog reads and validates the andespath skills catalog.
package catalog

// Profile is a named bundle of skills.
type Profile struct {
	Description string   `json:"description"`
	Skills      []string `json:"skills"`
}

// Catalog is the parsed catalog.json.
type Catalog struct {
	Name     string             `json:"name"`
	Profiles map[string]Profile `json:"profiles"`
}

// Source abstracts where the catalog lives (LocalDir today, GitRepo in v2).
type Source interface {
	Load() (*Catalog, error)
	SkillPath(id string) string
}
```

`internal/catalog/localdir.go`:

```go
package catalog

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// LocalDir is a catalog rooted at a local folder.
type LocalDir struct {
	Root string
}

func (l LocalDir) Load() (*Catalog, error) {
	data, err := os.ReadFile(filepath.Join(l.Root, "catalog.json"))
	if err != nil {
		return nil, fmt.Errorf("no pude leer el catálogo en %s: verificá la ruta (%w)", l.Root, err)
	}
	var c Catalog
	if err := json.Unmarshal(data, &c); err != nil {
		return nil, fmt.Errorf("catalog.json inválido en %s: %w", l.Root, err)
	}
	if err := l.validate(&c); err != nil {
		return nil, err
	}
	return &c, nil
}

func (l LocalDir) SkillPath(id string) string {
	return filepath.Join(l.Root, "skills", id)
}

// validate ensures every referenced skill exists with a SKILL.md.
// Fails loud at load time so installs never break halfway.
func (l LocalDir) validate(c *Catalog) error {
	var problems []string
	for name, p := range c.Profiles {
		for _, id := range p.Skills {
			skillMD := filepath.Join(l.SkillPath(id), "SKILL.md")
			if _, err := os.Stat(skillMD); err != nil {
				problems = append(problems,
					fmt.Sprintf("el perfil %q referencia la skill %q pero falta %s", name, id, skillMD))
			}
		}
	}
	if len(problems) > 0 {
		sort.Strings(problems)
		return fmt.Errorf("catálogo inválido:\n  %s", strings.Join(problems, "\n  "))
	}
	return nil
}
```

`internal/catalog/resolve.go`:

```go
package catalog

import "fmt"

// ResolveSkills expands profiles into skillID → profile that brought it in.
// The first profile (in requested order) wins on shared skills.
func ResolveSkills(c *Catalog, profiles []string) (map[string]string, error) {
	resolved := map[string]string{}
	for _, pname := range profiles {
		p, ok := c.Profiles[pname]
		if !ok {
			return nil, fmt.Errorf("el perfil %q no existe en el catálogo; corré `andes list` para ver los disponibles", pname)
		}
		for _, id := range p.Skills {
			if _, seen := resolved[id]; !seen {
				resolved[id] = pname
			}
		}
	}
	return resolved, nil
}
```

- [ ] **Step 5: Verificar que pasan**

Run: `go test ./internal/catalog/ -v`
Expected: PASS todos.

- [ ] **Step 6: Commit**

```bash
git add testdata/ internal/catalog/
git commit -m "feat: catalog loading, validation and profile resolution with fixture"
```

---

### Task 3: `internal/hashdir` — hash determinista de carpetas

**Files:**
- Create: `internal/hashdir/hashdir.go`
- Test: `internal/hashdir/hashdir_test.go`

**Interfaces:**
- Consumes: nada
- Produces: `func Hash(dir string) (string, error)` — retorna `"sha256:<hex>"`; determinista; cambia si cambia contenido, nombre o path relativo de cualquier archivo.

- [ ] **Step 1: Escribir tests que fallan**

`internal/hashdir/hashdir_test.go`:

```go
package hashdir_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/andespath/andes-ai/internal/hashdir"
)

func writeFile(t *testing.T, dir, name, content string) {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestHashDeterministic(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "SKILL.md", "# hola")
	writeFile(t, dir, "extra/notes.md", "detalle")

	h1, err := hashdir.Hash(dir)
	if err != nil {
		t.Fatal(err)
	}
	h2, err := hashdir.Hash(dir)
	if err != nil {
		t.Fatal(err)
	}
	if h1 != h2 {
		t.Errorf("hash no determinista: %s != %s", h1, h2)
	}
	if !strings.HasPrefix(h1, "sha256:") {
		t.Errorf("hash sin prefijo sha256: %s", h1)
	}
}

func TestHashChangesOnContentChange(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "SKILL.md", "# v1")
	h1, _ := hashdir.Hash(dir)

	writeFile(t, dir, "SKILL.md", "# v2")
	h2, _ := hashdir.Hash(dir)

	if h1 == h2 {
		t.Error("hash no cambió al cambiar contenido")
	}
}

func TestHashChangesOnNewFile(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "SKILL.md", "# v1")
	h1, _ := hashdir.Hash(dir)

	writeFile(t, dir, "otro.md", "nuevo")
	h2, _ := hashdir.Hash(dir)

	if h1 == h2 {
		t.Error("hash no cambió al agregar archivo")
	}
}

func TestHashEqualDirsEqualHash(t *testing.T) {
	dir1 := t.TempDir()
	dir2 := t.TempDir()
	for _, d := range []string{dir1, dir2} {
		writeFile(t, d, "SKILL.md", "mismo contenido")
	}
	h1, _ := hashdir.Hash(dir1)
	h2, _ := hashdir.Hash(dir2)
	if h1 != h2 {
		t.Errorf("dirs iguales con hash distinto: %s != %s", h1, h2)
	}
}

func TestHashMissingDir(t *testing.T) {
	_, err := hashdir.Hash(filepath.Join(t.TempDir(), "no-existe"))
	if err == nil {
		t.Error("Hash de dir inexistente debería fallar")
	}
}
```

- [ ] **Step 2: Verificar que fallan**

Run: `go test ./internal/hashdir/`
Expected: FAIL — `undefined: hashdir.Hash`.

- [ ] **Step 3: Implementar**

`internal/hashdir/hashdir.go`:

```go
// Package hashdir computes deterministic content hashes of directory trees.
package hashdir

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
	"io/fs"
	"os"
	"path/filepath"
)

// Hash returns "sha256:<hex>" over the dir tree. WalkDir visits files in
// lexical order, so the result is deterministic. Each file contributes its
// slash-separated relative path and its content, NUL-separated, so renames
// and moves change the hash too.
func Hash(dir string) (string, error) {
	h := sha256.New()
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}
		io.WriteString(h, filepath.ToSlash(rel))
		h.Write([]byte{0})
		f, err := os.Open(path)
		if err != nil {
			return err
		}
		if _, err := io.Copy(h, f); err != nil {
			f.Close()
			return err
		}
		f.Close()
		h.Write([]byte{0})
		return nil
	})
	if err != nil {
		return "", err
	}
	return "sha256:" + hex.EncodeToString(h.Sum(nil)), nil
}
```

- [ ] **Step 4: Verificar que pasan**

Run: `go test ./internal/hashdir/ -v`
Expected: PASS todos.

- [ ] **Step 5: Commit**

```bash
git add internal/hashdir/
git commit -m "feat: deterministic directory content hashing"
```

---

### Task 4: `internal/manifest` — recibo con escritura atómica

**Files:**
- Create: `internal/manifest/manifest.go`
- Test: `internal/manifest/manifest_test.go`

**Interfaces:**
- Consumes: nada
- Produces:
  - `type CatalogRef struct { Type, Path string }`
  - `type InstalledSkill struct { Hash, Profile string }`
  - `type Manifest struct { Version int; Catalog CatalogRef; Profiles []string; Installed map[string]InstalledSkill }`
  - `func Load(path string) (*Manifest, error)` — **retorna `(nil, nil)` si el archivo no existe** (primer init).
  - `func (m *Manifest) Save(path string) error` — atómico (temp + rename), crea el dir padre.
  - `func DefaultPath() (string, error)` — `~/.claude/andes.json` vía `os.UserHomeDir()`.

- [ ] **Step 1: Escribir tests que fallan**

`internal/manifest/manifest_test.go`:

```go
package manifest_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/andespath/andes-ai/internal/manifest"
)

func sample() *manifest.Manifest {
	return &manifest.Manifest{
		Version:  1,
		Catalog:  manifest.CatalogRef{Type: "local", Path: "/tmp/cat"},
		Profiles: []string{"andespath-core"},
		Installed: map[string]manifest.InstalledSkill{
			"git-conventions": {Hash: "sha256:abc", Profile: "andespath-core"},
		},
	}
}

func TestSaveLoadRoundtrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".claude", "andes.json")

	if err := sample().Save(path); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	got, err := manifest.Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if got == nil {
		t.Fatal("Load() = nil, want manifest")
	}
	if got.Version != 1 {
		t.Errorf("Version = %d, want 1", got.Version)
	}
	if got.Catalog.Path != "/tmp/cat" {
		t.Errorf("Catalog.Path = %q, want /tmp/cat", got.Catalog.Path)
	}
	if got.Installed["git-conventions"].Hash != "sha256:abc" {
		t.Errorf("Installed hash = %q, want sha256:abc", got.Installed["git-conventions"].Hash)
	}
}

func TestLoadMissingReturnsNilNil(t *testing.T) {
	got, err := manifest.Load(filepath.Join(t.TempDir(), "no-existe.json"))
	if err != nil {
		t.Fatalf("Load() de archivo inexistente: error = %v, want nil", err)
	}
	if got != nil {
		t.Errorf("Load() = %v, want nil", got)
	}
}

func TestLoadCorruptFails(t *testing.T) {
	path := filepath.Join(t.TempDir(), "andes.json")
	os.WriteFile(path, []byte("{corrupto"), 0o644)

	_, err := manifest.Load(path)
	if err == nil || !strings.Contains(err.Error(), "corrupto") {
		t.Errorf("error = %v, want mensaje de manifiesto corrupto", err)
	}
}

func TestSaveLeavesNoTempFiles(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "andes.json")
	if err := sample().Save(path); err != nil {
		t.Fatal(err)
	}
	entries, _ := os.ReadDir(dir)
	if len(entries) != 1 {
		t.Errorf("quedaron archivos extra en %s: %v", dir, entries)
	}
}

func TestDefaultPath(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	got, err := manifest.DefaultPath()
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(home, ".claude", "andes.json")
	if got != want {
		t.Errorf("DefaultPath = %q, want %q", got, want)
	}
}
```

- [ ] **Step 2: Verificar que fallan**

Run: `go test ./internal/manifest/`
Expected: FAIL — `undefined: manifest.Manifest` etc.

- [ ] **Step 3: Implementar**

`internal/manifest/manifest.go`:

```go
// Package manifest reads and writes the andes install receipt (~/.claude/andes.json).
package manifest

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

type CatalogRef struct {
	Type string `json:"type"`
	Path string `json:"path"`
}

type InstalledSkill struct {
	Hash    string `json:"hash"`
	Profile string `json:"profile"`
}

type Manifest struct {
	Version   int                       `json:"version"`
	Catalog   CatalogRef                `json:"catalog"`
	Profiles  []string                  `json:"profiles"`
	Installed map[string]InstalledSkill `json:"installed"`
}

// DefaultPath returns ~/.claude/andes.json.
func DefaultPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("no pude resolver tu home dir: %w", err)
	}
	return filepath.Join(home, ".claude", "andes.json"), nil
}

// Load reads the manifest. A missing file is not an error: it means
// init never ran, so Load returns (nil, nil).
func Load(path string) (*Manifest, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var m Manifest
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("manifiesto corrupto en %s: borralo y re-corré `andes init` (%w)", path, err)
	}
	return &m, nil
}

// Save writes atomically: temp file in the same dir, then rename.
// A crash mid-write leaves the previous manifest intact.
func (m *Manifest) Save(path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".andes-*.tmp")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name()) // no-op after successful rename
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), path)
}
```

- [ ] **Step 4: Verificar que pasan**

Run: `go test ./internal/manifest/ -v`
Expected: PASS todos.

- [ ] **Step 5: Commit**

```bash
git add internal/manifest/
git commit -m "feat: manifest receipt with atomic save"
```

---

### Task 5: `internal/installer` — Plan (diff deseado vs manifiesto)

**Files:**
- Create: `internal/installer/installer.go`
- Test: `internal/installer/plan_test.go`

**Interfaces:**
- Consumes: `catalog.Source`, `catalog.Catalog`, `catalog.ResolveSkills`, `manifest.Manifest`, `hashdir.Hash`
- Produces:
  - `type ActionType string` con constantes `ActionInstall ("instalar")`, `ActionUpdate ("actualizar")`, `ActionSkip ("sin cambios")`
  - `type Action struct { SkillID string; Type ActionType; Profile string; Hash string }` — `Hash` es el hash lado-catálogo.
  - `func Plan(src catalog.Source, cat *catalog.Catalog, m *manifest.Manifest, profiles []string) ([]Action, error)` — acciones ordenadas por SkillID; `m == nil` significa primer init (todo install).

- [ ] **Step 1: Escribir tests que fallan**

`internal/installer/plan_test.go`:

```go
package installer_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/andespath/andes-ai/internal/catalog"
	"github.com/andespath/andes-ai/internal/hashdir"
	"github.com/andespath/andes-ai/internal/installer"
	"github.com/andespath/andes-ai/internal/manifest"
)

// makeCatalog builds a temp catalog with two profiles and three skills.
func makeCatalog(t *testing.T) catalog.LocalDir {
	t.Helper()
	root := t.TempDir()
	os.WriteFile(filepath.Join(root, "catalog.json"), []byte(`{
		"name": "andespath",
		"profiles": {
			"core": {"description": "base", "skills": ["git-conventions", "code-review"]},
			"tri":  {"description": "tri",  "skills": ["golang"]}
		}
	}`), 0o644)
	for _, id := range []string{"git-conventions", "code-review", "golang"} {
		dir := filepath.Join(root, "skills", id)
		os.MkdirAll(dir, 0o755)
		os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte("# "+id), 0o644)
	}
	return catalog.LocalDir{Root: root}
}

func loadCat(t *testing.T, src catalog.LocalDir) *catalog.Catalog {
	t.Helper()
	c, err := src.Load()
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func TestPlanFirstInstall(t *testing.T) {
	src := makeCatalog(t)
	actions, err := installer.Plan(src, loadCat(t, src), nil, []string{"core", "tri"})
	if err != nil {
		t.Fatalf("Plan() error = %v", err)
	}
	if len(actions) != 3 {
		t.Fatalf("len(actions) = %d, want 3", len(actions))
	}
	// Ordered by SkillID: code-review, git-conventions, golang
	wantOrder := []string{"code-review", "git-conventions", "golang"}
	for i, want := range wantOrder {
		if actions[i].SkillID != want {
			t.Errorf("actions[%d].SkillID = %q, want %q", i, actions[i].SkillID, want)
		}
		if actions[i].Type != installer.ActionInstall {
			t.Errorf("actions[%d].Type = %q, want install", i, actions[i].Type)
		}
		if actions[i].Hash == "" {
			t.Errorf("actions[%d].Hash vacío", i)
		}
	}
	if actions[2].Profile != "tri" {
		t.Errorf("golang profile = %q, want tri", actions[2].Profile)
	}
}

func TestPlanIdempotentSkip(t *testing.T) {
	src := makeCatalog(t)
	cat := loadCat(t, src)

	h, err := hashdir.Hash(src.SkillPath("golang"))
	if err != nil {
		t.Fatal(err)
	}
	m := &manifest.Manifest{
		Version:   1,
		Installed: map[string]manifest.InstalledSkill{"golang": {Hash: h, Profile: "tri"}},
	}

	actions, err := installer.Plan(src, cat, m, []string{"tri"})
	if err != nil {
		t.Fatal(err)
	}
	if len(actions) != 1 || actions[0].Type != installer.ActionSkip {
		t.Errorf("actions = %+v, want 1 skip", actions)
	}
}

func TestPlanUpdateOnHashMismatch(t *testing.T) {
	src := makeCatalog(t)
	cat := loadCat(t, src)

	m := &manifest.Manifest{
		Version:   1,
		Installed: map[string]manifest.InstalledSkill{"golang": {Hash: "sha256:viejo", Profile: "tri"}},
	}

	actions, err := installer.Plan(src, cat, m, []string{"tri"})
	if err != nil {
		t.Fatal(err)
	}
	if len(actions) != 1 || actions[0].Type != installer.ActionUpdate {
		t.Errorf("actions = %+v, want 1 update", actions)
	}
}

func TestPlanUnknownProfile(t *testing.T) {
	src := makeCatalog(t)
	_, err := installer.Plan(src, loadCat(t, src), nil, []string{"fantasma"})
	if err == nil {
		t.Error("Plan() con perfil inexistente debería fallar")
	}
}
```

- [ ] **Step 2: Verificar que fallan**

Run: `go test ./internal/installer/`
Expected: FAIL — `undefined: installer.Plan`.

- [ ] **Step 3: Implementar**

`internal/installer/installer.go`:

```go
// Package installer plans and applies skill installs (catalog → ~/.claude/skills).
package installer

import (
	"fmt"
	"sort"

	"github.com/andespath/andes-ai/internal/catalog"
	"github.com/andespath/andes-ai/internal/hashdir"
	"github.com/andespath/andes-ai/internal/manifest"
)

type ActionType string

const (
	ActionInstall ActionType = "instalar"
	ActionUpdate  ActionType = "actualizar"
	ActionSkip    ActionType = "sin cambios"
)

// Action is one planned step for one skill. Hash is the catalog-side hash.
type Action struct {
	SkillID string
	Type    ActionType
	Profile string
	Hash    string
}

// Plan diffs desired state (profiles resolved against the catalog) with the
// manifest. m == nil means first init: everything installs.
func Plan(src catalog.Source, cat *catalog.Catalog, m *manifest.Manifest, profiles []string) ([]Action, error) {
	resolved, err := catalog.ResolveSkills(cat, profiles)
	if err != nil {
		return nil, err
	}
	ids := make([]string, 0, len(resolved))
	for id := range resolved {
		ids = append(ids, id)
	}
	sort.Strings(ids)

	actions := make([]Action, 0, len(ids))
	for _, id := range ids {
		h, err := hashdir.Hash(src.SkillPath(id))
		if err != nil {
			return nil, fmt.Errorf("no pude hashear la skill %q del catálogo: %w", id, err)
		}
		a := Action{SkillID: id, Profile: resolved[id], Hash: h, Type: ActionInstall}
		if m != nil {
			if inst, ok := m.Installed[id]; ok {
				if inst.Hash == h {
					a.Type = ActionSkip
				} else {
					a.Type = ActionUpdate
				}
			}
		}
		actions = append(actions, a)
	}
	return actions, nil
}
```

- [ ] **Step 4: Verificar que pasan**

Run: `go test ./internal/installer/ -v`
Expected: PASS todos.

- [ ] **Step 5: Commit**

```bash
git add internal/installer/
git commit -m "feat: install planning with hash-based diff"
```

---

### Task 6: `internal/installer` — Apply (copia real de skills)

**Files:**
- Modify: `internal/installer/installer.go` (agregar `Apply` y `copyDir` al final del archivo)
- Test: `internal/installer/apply_test.go`

**Interfaces:**
- Consumes: `Action` y helpers de Task 5, `manifest.InstalledSkill`
- Produces: `func Apply(src catalog.Source, actions []Action, skillsDir string) (map[string]manifest.InstalledSkill, error)` — copia install/update (borrando el destino primero), saltea skips, retorna el map `installed` COMPLETO (incluye skips) listo para el manifiesto.

- [ ] **Step 1: Escribir tests que fallan**

`internal/installer/apply_test.go`:

```go
package installer_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/andespath/andes-ai/internal/installer"
)

func TestApplyCopiesSkills(t *testing.T) {
	src := makeCatalog(t)
	cat := loadCat(t, src)
	skillsDir := t.TempDir()

	actions, err := installer.Plan(src, cat, nil, []string{"core"})
	if err != nil {
		t.Fatal(err)
	}

	installed, err := installer.Apply(src, actions, skillsDir)
	if err != nil {
		t.Fatalf("Apply() error = %v", err)
	}

	for _, id := range []string{"git-conventions", "code-review"} {
		skillMD := filepath.Join(skillsDir, id, "SKILL.md")
		if _, err := os.Stat(skillMD); err != nil {
			t.Errorf("falta %s tras Apply", skillMD)
		}
		if installed[id].Hash == "" {
			t.Errorf("installed[%q] sin hash", id)
		}
	}
}

func TestApplySkipDoesNotTouchDisk(t *testing.T) {
	src := makeCatalog(t)
	skillsDir := t.TempDir()

	// A skip action for a skill NOT on disk: Apply must not create it,
	// but must still return its manifest entry.
	actions := []installer.Action{
		{SkillID: "golang", Type: installer.ActionSkip, Profile: "tri", Hash: "sha256:x"},
	}
	installed, err := installer.Apply(src, actions, skillsDir)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(skillsDir, "golang")); !os.IsNotExist(err) {
		t.Error("skip no debería tocar el disco")
	}
	if installed["golang"].Hash != "sha256:x" {
		t.Errorf("skip debe conservar la entrada del manifiesto, got %+v", installed["golang"])
	}
}

func TestApplyUpdateReplacesStaleFiles(t *testing.T) {
	src := makeCatalog(t)
	cat := loadCat(t, src)
	skillsDir := t.TempDir()

	// Pre-existing stale install with an extra file that must disappear.
	stale := filepath.Join(skillsDir, "golang")
	os.MkdirAll(stale, 0o755)
	os.WriteFile(filepath.Join(stale, "basura.md"), []byte("viejo"), 0o644)

	actions, err := installer.Plan(src, cat, nil, []string{"tri"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := installer.Apply(src, actions, skillsDir); err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(filepath.Join(stale, "basura.md")); !os.IsNotExist(err) {
		t.Error("el archivo viejo debería haber sido eliminado (copia limpia)")
	}
	if _, err := os.Stat(filepath.Join(stale, "SKILL.md")); err != nil {
		t.Error("falta SKILL.md tras update")
	}
}
```

- [ ] **Step 2: Verificar que fallan**

Run: `go test ./internal/installer/`
Expected: FAIL — `undefined: installer.Apply`.

- [ ] **Step 3: Implementar (agregar al final de `internal/installer/installer.go`)**

```go
// Apply executes the plan: install/update actions copy the skill folder
// (clean copy: destination removed first), skips are left untouched.
// The returned map is the complete `installed` section for the manifest.
// NOTA POST-E2E: esta semántica resultó incompleta — el E2E (Task 12) demostró
// que skip debe verificar el DISCO en apply-time y reparar skills modificadas
// o borradas localmente (el spec define re-init como LA reparación). Ver commit
// "fix: re-init repairs missing and modified skills at apply time".
func Apply(src catalog.Source, actions []Action, skillsDir string) (map[string]manifest.InstalledSkill, error) {
	installed := make(map[string]manifest.InstalledSkill, len(actions))
	for _, a := range actions {
		if a.Type != ActionSkip {
			dst := filepath.Join(skillsDir, a.SkillID)
			if err := os.RemoveAll(dst); err != nil {
				return nil, fmt.Errorf("no pude limpiar %s: %w", dst, err)
			}
			if err := copyDir(src.SkillPath(a.SkillID), dst); err != nil {
				return nil, fmt.Errorf("no pude instalar la skill %q: %w", a.SkillID, err)
			}
		}
		installed[a.SkillID] = manifest.InstalledSkill{Hash: a.Hash, Profile: a.Profile}
	}
	return installed, nil
}

func copyDir(src, dst string) error {
	return filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(target, data, 0o644)
	})
}
```

Agregar a los imports del archivo: `"io/fs"`, `"os"`, `"path/filepath"`.

- [ ] **Step 4: Verificar que pasan**

Run: `go test ./internal/installer/ -v`
Expected: PASS todos (los de plan y los de apply).

- [ ] **Step 5: Commit**

```bash
git add internal/installer/
git commit -m "feat: apply plan with clean skill copies"
```

---

### Task 7: `internal/doctor` — motor de diff de 3 estados

**Files:**
- Create: `internal/doctor/doctor.go`
- Test: `internal/doctor/doctor_test.go`

**Interfaces:**
- Consumes: `catalog.Source`, `manifest.Manifest`, `hashdir.Hash`
- Produces:
  - `type Status string` con `StatusOK ("ok")`, `StatusMissing ("falta")`, `StatusModified ("modificada")`, `StatusOutdated ("desactualizada")`
  - `type Finding struct { SkillID string; Status Status; Advice string }`
  - `func Check(src catalog.Source, m *manifest.Manifest, skillsDir string) ([]Finding, error)` — un Finding por skill del manifiesto, ordenado por SkillID. Precedencia: falta > modificada > desactualizada > ok. NUNCA modifica nada.

- [ ] **Step 1: Escribir tests que fallan**

`internal/doctor/doctor_test.go`:

```go
package doctor_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/andespath/andes-ai/internal/catalog"
	"github.com/andespath/andes-ai/internal/doctor"
	"github.com/andespath/andes-ai/internal/hashdir"
	"github.com/andespath/andes-ai/internal/installer"
	"github.com/andespath/andes-ai/internal/manifest"
)

// setup installs profile "tri" (skill golang) into a temp skillsDir and
// returns everything a doctor check needs.
func setup(t *testing.T) (catalog.LocalDir, *manifest.Manifest, string) {
	t.Helper()
	root := t.TempDir()
	os.WriteFile(filepath.Join(root, "catalog.json"), []byte(`{
		"name": "andespath",
		"profiles": {"tri": {"description": "tri", "skills": ["golang"]}}
	}`), 0o644)
	dir := filepath.Join(root, "skills", "golang")
	os.MkdirAll(dir, 0o755)
	os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte("# golang v1"), 0o644)

	src := catalog.LocalDir{Root: root}
	cat, err := src.Load()
	if err != nil {
		t.Fatal(err)
	}
	skillsDir := t.TempDir()
	actions, err := installer.Plan(src, cat, nil, []string{"tri"})
	if err != nil {
		t.Fatal(err)
	}
	installed, err := installer.Apply(src, actions, skillsDir)
	if err != nil {
		t.Fatal(err)
	}
	m := &manifest.Manifest{Version: 1, Profiles: []string{"tri"}, Installed: installed}
	return src, m, skillsDir
}

func TestCheckAllHealthy(t *testing.T) {
	src, m, skillsDir := setup(t)
	findings, err := doctor.Check(src, m, skillsDir)
	if err != nil {
		t.Fatal(err)
	}
	if len(findings) != 1 || findings[0].Status != doctor.StatusOK {
		t.Errorf("findings = %+v, want 1 OK", findings)
	}
}

func TestCheckMissingOnDisk(t *testing.T) {
	src, m, skillsDir := setup(t)
	os.RemoveAll(filepath.Join(skillsDir, "golang"))

	findings, _ := doctor.Check(src, m, skillsDir)
	if len(findings) != 1 || findings[0].Status != doctor.StatusMissing {
		t.Errorf("findings = %+v, want 1 falta", findings)
	}
}

func TestCheckLocallyModified(t *testing.T) {
	src, m, skillsDir := setup(t)
	os.WriteFile(filepath.Join(skillsDir, "golang", "SKILL.md"), []byte("# editado a mano"), 0o644)

	findings, _ := doctor.Check(src, m, skillsDir)
	if len(findings) != 1 || findings[0].Status != doctor.StatusModified {
		t.Errorf("findings = %+v, want 1 modificada", findings)
	}
}

func TestCheckOutdated(t *testing.T) {
	src, m, skillsDir := setup(t)
	// Catalog moves forward; disk + manifest stay at v1.
	os.WriteFile(filepath.Join(src.Root, "skills", "golang", "SKILL.md"), []byte("# golang v2"), 0o644)

	findings, _ := doctor.Check(src, m, skillsDir)
	if len(findings) != 1 || findings[0].Status != doctor.StatusOutdated {
		t.Errorf("findings = %+v, want 1 desactualizada", findings)
	}
}

func TestCheckNeverWrites(t *testing.T) {
	src, m, skillsDir := setup(t)
	before, err := hashdir.Hash(skillsDir)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := doctor.Check(src, m, skillsDir); err != nil {
		t.Fatal(err)
	}
	after, _ := hashdir.Hash(skillsDir)
	if before != after {
		t.Error("Check() modificó el disco — jamás debe escribir")
	}
}
```

- [ ] **Step 2: Verificar que fallan**

Run: `go test ./internal/doctor/`
Expected: FAIL — `undefined: doctor.Check`.

- [ ] **Step 3: Implementar**

`internal/doctor/doctor.go`:

```go
// Package doctor diagnoses drift between manifest (declared), disk (real)
// and catalog (source). It never modifies anything.
package doctor

import (
	"errors"
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
	StatusMissing  Status = "falta"
	StatusModified Status = "modificada"
	StatusOutdated Status = "desactualizada"
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

		if _, err := os.Stat(diskPath); errors.Is(err, fs.ErrNotExist) {
			findings = append(findings, Finding{id, StatusMissing,
				"re-corré `andes init` para reinstalarla"})
			continue
		}

		diskHash, err := hashdir.Hash(diskPath)
		if err != nil {
			return nil, err
		}
		if diskHash != inst.Hash {
			findings = append(findings, Finding{id, StatusModified,
				"fue editada a mano; re-correr `andes init` PISA tus cambios — decidí antes"})
			continue
		}

		catHash, err := hashdir.Hash(src.SkillPath(id))
		if err != nil {
			return nil, err
		}
		if catHash != inst.Hash {
			findings = append(findings, Finding{id, StatusOutdated,
				"el catálogo tiene versión nueva; re-corré `andes init`"})
			continue
		}

		findings = append(findings, Finding{id, StatusOK, ""})
	}
	return findings, nil
}
```

- [ ] **Step 4: Verificar que pasan**

Run: `go test ./internal/doctor/ -v`
Expected: PASS todos.

- [ ] **Step 5: Commit**

```bash
git add internal/doctor/
git commit -m "feat: doctor diff engine over manifest, disk and catalog"
```

---

### Task 8: comando `andes init` (modo no-interactivo) + helpers de paths

**Files:**
- Create: `internal/cli/paths.go`
- Create: `internal/cli/init.go`
- Modify: `internal/cli/root.go` (registrar subcomando)
- Test: `internal/cli/init_test.go`

**Interfaces:**
- Consumes: todo lo anterior (`catalog`, `manifest`, `installer`)
- Produces:
  - `func skillsDir() (string, error)` — `~/.claude/skills`
  - `func newInitCmd() *cobra.Command` con flags `--catalog string`, `--profiles []string`, `--yes bool`
  - En Task 11 se reemplazan DOS branches de error por prompts `huh` — quedan marcados con comentario `// interactivo: Task 11`.

- [ ] **Step 1: Escribir test de integración que falla**

`internal/cli/init_test.go`:

```go
package cli_test

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/andespath/andes-ai/internal/cli"
	"github.com/andespath/andes-ai/internal/manifest"
)

func fixtureCatalog(t *testing.T) string {
	t.Helper()
	abs, err := filepath.Abs("../../testdata/catalog")
	if err != nil {
		t.Fatal(err)
	}
	return abs
}

// runAndes executes the CLI with a temp HOME and returns combined output.
func runAndes(t *testing.T, home string, args ...string) (string, error) {
	t.Helper()
	t.Setenv("HOME", home)
	root := cli.NewRootCmd()
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs(args)
	err := root.Execute()
	return out.String(), err
}

func TestInitInstallsProfiles(t *testing.T) {
	home := t.TempDir()

	out, err := runAndes(t, home,
		"init", "--catalog", fixtureCatalog(t), "--profiles", "andespath-core,tri-fleet", "--yes")
	if err != nil {
		t.Fatalf("init error = %v\noutput:\n%s", err, out)
	}

	// Skills on disk
	for _, id := range []string{"git-conventions", "code-review", "golang"} {
		p := filepath.Join(home, ".claude", "skills", id, "SKILL.md")
		if _, err := os.Stat(p); err != nil {
			t.Errorf("falta skill instalada: %s", p)
		}
	}

	// Manifest written
	m, err := manifest.Load(filepath.Join(home, ".claude", "andes.json"))
	if err != nil || m == nil {
		t.Fatalf("manifiesto no escrito: %v", err)
	}
	if m.Version != 1 || len(m.Installed) != 3 {
		t.Errorf("manifiesto = %+v, want version 1 con 3 skills", m)
	}
	if m.Catalog.Type != "local" {
		t.Errorf("catalog.type = %q, want local", m.Catalog.Type)
	}
}

func TestInitIsIdempotent(t *testing.T) {
	home := t.TempDir()
	args := []string{"init", "--catalog", fixtureCatalog(t), "--profiles", "tri-fleet", "--yes"}

	if _, err := runAndes(t, home, args...); err != nil {
		t.Fatal(err)
	}
	out, err := runAndes(t, home, args...)
	if err != nil {
		t.Fatalf("segundo init falló: %v", err)
	}
	if !bytes.Contains([]byte(out), []byte("sin cambios")) {
		t.Errorf("segundo init debería reportar 'sin cambios':\n%s", out)
	}
}

func TestInitRemembersCatalogPath(t *testing.T) {
	home := t.TempDir()
	if _, err := runAndes(t, home,
		"init", "--catalog", fixtureCatalog(t), "--profiles", "tri-fleet", "--yes"); err != nil {
		t.Fatal(err)
	}
	// Second run without --catalog: must reuse the manifest's path.
	if _, err := runAndes(t, home, "init", "--profiles", "tri-fleet", "--yes"); err != nil {
		t.Fatalf("init sin --catalog con manifiesto previo falló: %v", err)
	}
}

func TestInitNonInteractiveRequiresFlags(t *testing.T) {
	home := t.TempDir()
	// --yes without --catalog and no previous manifest: actionable error.
	if _, err := runAndes(t, home, "init", "--yes"); err == nil {
		t.Error("init --yes sin catálogo debería fallar con error accionable")
	}
}

func TestInitDoesNotTouchForeignSkills(t *testing.T) {
	home := t.TempDir()
	foreign := filepath.Join(home, ".claude", "skills", "mi-skill-personal")
	os.MkdirAll(foreign, 0o755)
	os.WriteFile(filepath.Join(foreign, "SKILL.md"), []byte("# mía"), 0o644)

	if _, err := runAndes(t, home,
		"init", "--catalog", fixtureCatalog(t), "--profiles", "tri-fleet", "--yes"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(foreign, "SKILL.md")); err != nil {
		t.Error("init tocó una skill personal ajena al manifiesto")
	}
}
```

- [ ] **Step 2: Verificar que falla**

Run: `go test ./internal/cli/`
Expected: FAIL — `unknown command "init"`.

- [ ] **Step 3: Implementar**

`internal/cli/paths.go`:

```go
package cli

import (
	"fmt"
	"os"
	"path/filepath"
)

// skillsDir returns ~/.claude/skills (respects $HOME, overridable in tests).
func skillsDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("no pude resolver tu home dir: %w", err)
	}
	return filepath.Join(home, ".claude", "skills"), nil
}
```

`internal/cli/init.go`:

```go
package cli

import (
	"errors"
	"fmt"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/andespath/andes-ai/internal/catalog"
	"github.com/andespath/andes-ai/internal/installer"
	"github.com/andespath/andes-ai/internal/manifest"
)

func newInitCmd() *cobra.Command {
	var catalogPath string
	var profiles []string
	var yes bool

	cmd := &cobra.Command{
		Use:   "init",
		Short: "Instala skills desde el catálogo según perfiles",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runInit(cmd, catalogPath, profiles, yes)
		},
	}
	cmd.Flags().StringVar(&catalogPath, "catalog", "", "ruta a la carpeta del catálogo")
	cmd.Flags().StringSliceVar(&profiles, "profiles", nil, "perfiles a instalar (ej: andespath-core,tri-fleet)")
	cmd.Flags().BoolVar(&yes, "yes", false, "aplicar sin pedir confirmación")
	return cmd
}

func runInit(cmd *cobra.Command, catalogPath string, profiles []string, yes bool) error {
	mPath, err := manifest.DefaultPath()
	if err != nil {
		return err
	}
	prev, err := manifest.Load(mPath)
	if err != nil {
		return err
	}

	// Resolve catalog path: flag → previous manifest → prompt (Task 11).
	if catalogPath == "" && prev != nil {
		catalogPath = prev.Catalog.Path
	}
	if catalogPath == "" {
		// interactivo: Task 11
		return errors.New("no sé dónde está el catálogo: pasá --catalog <ruta>")
	}

	src := catalog.LocalDir{Root: catalogPath}
	cat, err := src.Load()
	if err != nil {
		return err
	}

	// Resolve profiles: flag → previous manifest → prompt (Task 11).
	if len(profiles) == 0 && prev != nil {
		profiles = prev.Profiles
	}
	if len(profiles) == 0 {
		// interactivo: Task 11
		return errors.New("no sé qué perfiles instalar: pasá --profiles a,b (corré `andes list` para verlos)")
	}

	actions, err := installer.Plan(src, cat, prev, profiles)
	if err != nil {
		return err
	}

	fmt.Fprintln(cmd.OutOrStdout(), "Plan:")
	for _, a := range actions {
		fmt.Fprintf(cmd.OutOrStdout(), "  %-12s %s\n", a.Type, a.SkillID)
	}

	if !yes {
		// interactivo: Task 11 (confirmación). Sin --yes hoy: abortar explícito.
		return errors.New("confirmación interactiva no disponible todavía: re-corré con --yes")
	}

	sDir, err := skillsDir()
	if err != nil {
		return err
	}
	installed, err := installer.Apply(src, actions, sDir)
	if err != nil {
		return err
	}

	absCatalog, err := filepath.Abs(catalogPath)
	if err != nil {
		return err
	}
	next := &manifest.Manifest{
		Version:   1,
		Catalog:   manifest.CatalogRef{Type: "local", Path: absCatalog},
		Profiles:  profiles,
		Installed: installed,
	}
	if err := next.Save(mPath); err != nil {
		return err
	}

	fmt.Fprintf(cmd.OutOrStdout(), "✓ %d skills al día en %s\n", len(installed), sDir)
	return nil
}
```

En `internal/cli/root.go`, dentro de `NewRootCmd()` antes del `return`:

```go
	root.AddCommand(newInitCmd())
```

- [ ] **Step 4: Verificar que pasan**

Run: `go test ./internal/cli/ -v`
Expected: PASS todos (incluidos idempotencia y skill personal intacta).

- [ ] **Step 5: Commit**

```bash
git add internal/cli/
git commit -m "feat: andes init non-interactive with plan display and receipt"
```

---

### Task 9: comando `andes list`

**Files:**
- Create: `internal/cli/list.go`
- Modify: `internal/cli/root.go` (registrar subcomando)
- Test: `internal/cli/list_test.go`

**Interfaces:**
- Consumes: `catalog`, `manifest`, `hashdir`
- Produces: `func newListCmd() *cobra.Command` con flag `--catalog string`. Estados por skill: `✓ instalada` (hash manifiesto == catálogo), `⚠ desactualizada` (instalada pero hash ≠), `✗ no instalada`. Nota: el estado del DISCO es asunto de `doctor`, no de `list`.

- [ ] **Step 1: Escribir tests que fallan**

`internal/cli/list_test.go`:

```go
package cli_test

import (
	"strings"
	"testing"
)

func TestListWithoutManifestShowsCatalogAndHint(t *testing.T) {
	home := t.TempDir()
	out, err := runAndes(t, home, "list", "--catalog", fixtureCatalog(t))
	if err != nil {
		t.Fatalf("list error = %v\n%s", err, out)
	}
	for _, want := range []string{"andespath-core", "tri-fleet", "git-conventions", "no instalada", "andes init"} {
		if !strings.Contains(out, want) {
			t.Errorf("list output no contiene %q:\n%s", want, out)
		}
	}
}

func TestListAfterInitShowsInstalled(t *testing.T) {
	home := t.TempDir()
	if _, err := runAndes(t, home,
		"init", "--catalog", fixtureCatalog(t), "--profiles", "tri-fleet", "--yes"); err != nil {
		t.Fatal(err)
	}

	out, err := runAndes(t, home, "list")
	if err != nil {
		t.Fatalf("list error = %v\n%s", err, out)
	}
	if !strings.Contains(out, "✓") {
		t.Errorf("list no muestra golang instalada:\n%s", out)
	}
	// core profile not installed → its skills show as not installed
	if !strings.Contains(out, "✗") {
		t.Errorf("list no muestra skills no instaladas:\n%s", out)
	}
}

func TestListWithoutCatalogAnywhereFails(t *testing.T) {
	home := t.TempDir()
	if _, err := runAndes(t, home, "list"); err == nil {
		t.Error("list sin catálogo ni manifiesto debería fallar con error accionable")
	}
}
```

- [ ] **Step 2: Verificar que fallan**

Run: `go test ./internal/cli/ -run TestList`
Expected: FAIL — `unknown command "list"`.

- [ ] **Step 3: Implementar**

`internal/cli/list.go`:

```go
package cli

import (
	"errors"
	"fmt"
	"sort"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/andespath/andes-ai/internal/catalog"
	"github.com/andespath/andes-ai/internal/hashdir"
	"github.com/andespath/andes-ai/internal/manifest"
)

func newListCmd() *cobra.Command {
	var catalogPath string

	cmd := &cobra.Command{
		Use:   "list",
		Short: "Muestra perfiles y skills del catálogo con su estado",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runList(cmd, catalogPath)
		},
	}
	cmd.Flags().StringVar(&catalogPath, "catalog", "", "ruta a la carpeta del catálogo")
	return cmd
}

func runList(cmd *cobra.Command, catalogPath string) error {
	mPath, err := manifest.DefaultPath()
	if err != nil {
		return err
	}
	m, err := manifest.Load(mPath)
	if err != nil {
		return err
	}

	if catalogPath == "" && m != nil {
		catalogPath = m.Catalog.Path
	}
	if catalogPath == "" {
		return errors.New("no sé dónde está el catálogo: pasá --catalog <ruta> o corré `andes init` primero")
	}

	src := catalog.LocalDir{Root: catalogPath}
	cat, err := src.Load()
	if err != nil {
		return err
	}

	profileNames := make([]string, 0, len(cat.Profiles))
	for name := range cat.Profiles {
		profileNames = append(profileNames, name)
	}
	sort.Strings(profileNames)

	w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 4, 2, ' ', 0)
	fmt.Fprintln(w, "PERFIL\tSKILL\tESTADO")
	for _, pname := range profileNames {
		for _, id := range cat.Profiles[pname].Skills {
			fmt.Fprintf(w, "%s\t%s\t%s\n", pname, id, skillStatus(src, m, id))
		}
	}
	w.Flush()

	if m == nil {
		fmt.Fprintln(cmd.OutOrStdout(), "\nTodavía no corriste `andes init` — corrélo para instalar un perfil.")
	}
	return nil
}

// skillStatus compares manifest hash vs catalog hash. Disk state is
// doctor's job, not list's.
func skillStatus(src catalog.Source, m *manifest.Manifest, id string) string {
	if m == nil {
		return "✗ no instalada"
	}
	inst, ok := m.Installed[id]
	if !ok {
		return "✗ no instalada"
	}
	catHash, err := hashdir.Hash(src.SkillPath(id))
	if err != nil || catHash != inst.Hash {
		return "⚠ desactualizada"
	}
	return "✓ instalada"
}
```

En `internal/cli/root.go`, junto al AddCommand existente:

```go
	root.AddCommand(newInitCmd(), newListCmd())
```

(reemplaza la línea `root.AddCommand(newInitCmd())`)

- [ ] **Step 4: Verificar que pasan**

Run: `go test ./internal/cli/ -v`
Expected: PASS todos.

- [ ] **Step 5: Commit**

```bash
git add internal/cli/
git commit -m "feat: andes list with per-skill install status"
```

---

### Task 10: comando `andes doctor`

**Files:**
- Create: `internal/cli/doctor.go`
- Modify: `internal/cli/root.go` (registrar subcomando)
- Test: `internal/cli/doctor_test.go`

**Interfaces:**
- Consumes: `doctor.Check`, `manifest`, `catalog`
- Produces: `func newDoctorCmd() *cobra.Command`. Exit ≠ 0 (RunE retorna error) si: no hay manifiesto, catálogo inaccesible, o hay findings con problema. Exit 0 solo si todo `ok`. JAMÁS modifica nada.

- [ ] **Step 1: Escribir tests que fallan**

`internal/cli/doctor_test.go`:

```go
package cli_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDoctorHealthyAfterInit(t *testing.T) {
	home := t.TempDir()
	if _, err := runAndes(t, home,
		"init", "--catalog", fixtureCatalog(t), "--profiles", "tri-fleet", "--yes"); err != nil {
		t.Fatal(err)
	}

	out, err := runAndes(t, home, "doctor")
	if err != nil {
		t.Fatalf("doctor sano debería salir 0: %v\n%s", err, out)
	}
	if !strings.Contains(out, "✓") {
		t.Errorf("doctor no reporta salud:\n%s", out)
	}
}

func TestDoctorDetectsMissingSkill(t *testing.T) {
	home := t.TempDir()
	if _, err := runAndes(t, home,
		"init", "--catalog", fixtureCatalog(t), "--profiles", "tri-fleet", "--yes"); err != nil {
		t.Fatal(err)
	}
	os.RemoveAll(filepath.Join(home, ".claude", "skills", "golang"))

	out, err := runAndes(t, home, "doctor")
	if err == nil {
		t.Errorf("doctor con skill faltante debería fallar (exit != 0):\n%s", out)
	}
	if !strings.Contains(out, "falta") {
		t.Errorf("doctor no reporta la skill faltante:\n%s", out)
	}
}

func TestDoctorWithoutManifest(t *testing.T) {
	home := t.TempDir()
	out, err := runAndes(t, home, "doctor")
	if err == nil {
		t.Error("doctor sin manifiesto debería fallar")
	}
	_ = out
}

func TestDoctorInaccessibleCatalog(t *testing.T) {
	home := t.TempDir()
	// init against a catalog that later disappears
	tmpCat := filepath.Join(t.TempDir(), "cat")
	copyFixture(t, tmpCat)
	if _, err := runAndes(t, home,
		"init", "--catalog", tmpCat, "--profiles", "tri-fleet", "--yes"); err != nil {
		t.Fatal(err)
	}
	os.RemoveAll(tmpCat)

	out, err := runAndes(t, home, "doctor")
	if err == nil {
		t.Errorf("doctor con catálogo inaccesible debería fallar:\n%s", out)
	}
}

// copyFixture clones the fixture catalog so tests can mutate/delete it.
func copyFixture(t *testing.T, dst string) {
	t.Helper()
	src := fixtureCatalog(t)
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
		return os.WriteFile(target, data, 0o644)
	})
	if err != nil {
		t.Fatal(err)
	}
}
```

- [ ] **Step 2: Verificar que fallan**

Run: `go test ./internal/cli/ -run TestDoctor`
Expected: FAIL — `unknown command "doctor"`.

- [ ] **Step 3: Implementar**

`internal/cli/doctor.go`:

```go
package cli

import (
	"errors"
	"fmt"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/andespath/andes-ai/internal/catalog"
	"github.com/andespath/andes-ai/internal/doctor"
	"github.com/andespath/andes-ai/internal/manifest"
)

func newDoctorCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "doctor",
		Short: "Diagnostica drift entre manifiesto, disco y catálogo (no modifica nada)",
		RunE:  runDoctor,
	}
}

func runDoctor(cmd *cobra.Command, args []string) error {
	mPath, err := manifest.DefaultPath()
	if err != nil {
		return err
	}
	m, err := manifest.Load(mPath)
	if err != nil {
		return err
	}
	if m == nil {
		return errors.New("no hay manifiesto: nunca corriste `andes init`")
	}

	src := catalog.LocalDir{Root: m.Catalog.Path}
	if _, err := src.Load(); err != nil {
		return fmt.Errorf("catálogo inaccesible en %s: corregí la ruta y re-corré `andes init --catalog <ruta>` (%w)",
			m.Catalog.Path, err)
	}

	sDir, err := skillsDir()
	if err != nil {
		return err
	}
	findings, err := doctor.Check(src, m, sDir)
	if err != nil {
		return err
	}

	problems := 0
	w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 4, 2, ' ', 0)
	fmt.Fprintln(w, "SKILL\tESTADO\tCONSEJO")
	for _, f := range findings {
		mark := "✓"
		if f.Status != doctor.StatusOK {
			mark = "✗"
			problems++
		}
		fmt.Fprintf(w, "%s\t%s %s\t%s\n", f.SkillID, mark, f.Status, f.Advice)
	}
	w.Flush()

	if problems > 0 {
		return fmt.Errorf("doctor encontró %d problema(s)", problems)
	}
	fmt.Fprintln(cmd.OutOrStdout(), "Todo sano ✓")
	return nil
}
```

En `internal/cli/root.go`:

```go
	root.AddCommand(newInitCmd(), newListCmd(), newDoctorCmd())
```

(reemplaza la línea de AddCommand anterior)

- [ ] **Step 4: Verificar que pasan**

Run: `go test ./internal/cli/ -v`
Expected: PASS todos.

- [ ] **Step 5: Commit**

```bash
git add internal/cli/
git commit -m "feat: andes doctor with drift report and nonzero exit on problems"
```

---

### Task 11: `andes init` interactivo (prompts `huh`)

**Files:**
- Create: `internal/cli/prompts.go`
- Modify: `internal/cli/init.go` (reemplazar los 3 branches marcados `// interactivo: Task 11`)

**Interfaces:**
- Consumes: `catalog.Catalog`, `huh`
- Produces:
  - `func promptCatalogPath() (string, error)`
  - `func promptProfiles(cat *catalog.Catalog) ([]string, error)`
  - `func confirmPlan() (bool, error)`
- Los prompts son capa fina SIN lógica — no se testean automáticamente en el MVP (verificación manual). Los tests existentes de `--yes` deben seguir pasando intactos.

- [ ] **Step 1: Agregar dependencia**

```bash
go get github.com/charmbracelet/huh@latest
```

- [ ] **Step 2: Implementar prompts**

`internal/cli/prompts.go`:

```go
package cli

import (
	"errors"
	"fmt"
	"sort"

	"github.com/charmbracelet/huh"

	"github.com/andespath/andes-ai/internal/catalog"
)

func promptCatalogPath() (string, error) {
	var path string
	err := huh.NewInput().
		Title("¿Dónde está el catálogo de skills?").
		Description("Ruta a la carpeta que contiene catalog.json").
		Value(&path).
		Run()
	if err != nil {
		return "", err
	}
	if path == "" {
		return "", errors.New("necesito la ruta del catálogo para continuar")
	}
	return path, nil
}

func promptProfiles(cat *catalog.Catalog) ([]string, error) {
	names := make([]string, 0, len(cat.Profiles))
	for name := range cat.Profiles {
		names = append(names, name)
	}
	sort.Strings(names)

	opts := make([]huh.Option[string], 0, len(names))
	for _, name := range names {
		label := fmt.Sprintf("%s — %s", name, cat.Profiles[name].Description)
		opts = append(opts, huh.NewOption(label, name))
	}

	var selected []string
	err := huh.NewMultiSelect[string]().
		Title("¿Qué perfiles querés instalar?").
		Options(opts...).
		Value(&selected).
		Run()
	if err != nil {
		return nil, err
	}
	if len(selected) == 0 {
		return nil, errors.New("no elegiste ningún perfil")
	}
	return selected, nil
}

func confirmPlan() (bool, error) {
	var ok bool
	err := huh.NewConfirm().
		Title("¿Aplicar estos cambios?").
		Affirmative("Sí, dale").
		Negative("No").
		Value(&ok).
		Run()
	return ok, err
}
```

- [ ] **Step 3: Conectar los prompts en `init.go`**

En `internal/cli/init.go`, reemplazar:

```go
	if catalogPath == "" {
		// interactivo: Task 11
		return errors.New("no sé dónde está el catálogo: pasá --catalog <ruta>")
	}
```

por:

```go
	if catalogPath == "" {
		if yes {
			return errors.New("no sé dónde está el catálogo: pasá --catalog <ruta>")
		}
		catalogPath, err = promptCatalogPath()
		if err != nil {
			return err
		}
	}
```

Reemplazar:

```go
	if len(profiles) == 0 {
		// interactivo: Task 11
		return errors.New("no sé qué perfiles instalar: pasá --profiles a,b (corré `andes list` para verlos)")
	}
```

por:

```go
	if len(profiles) == 0 {
		if yes {
			return errors.New("no sé qué perfiles instalar: pasá --profiles a,b (corré `andes list` para verlos)")
		}
		profiles, err = promptProfiles(cat)
		if err != nil {
			return err
		}
	}
```

Reemplazar:

```go
	if !yes {
		// interactivo: Task 11 (confirmación). Sin --yes hoy: abortar explícito.
		return errors.New("confirmación interactiva no disponible todavía: re-corré con --yes")
	}
```

por:

```go
	if !yes {
		ok, err := confirmPlan()
		if err != nil {
			return err
		}
		if !ok {
			fmt.Fprintln(cmd.OutOrStdout(), "Abortado — no se tocó nada.")
			return nil
		}
	}
```

- [ ] **Step 4: Verificar que los tests siguen pasando y probar a mano**

Run: `go test ./... && go build -o andes ./cmd/andes`
Expected: PASS todos (los tests usan `--yes`, no tocan prompts).

Verificación manual (en una terminal interactiva):

```bash
HOME=$(mktemp -d) ./andes init --catalog ./testdata/catalog
```

Expected: prompt de perfiles con checkboxes → plan → confirmación → `✓ N skills al día`.

- [ ] **Step 5: Commit**

```bash
git add go.mod go.sum internal/cli/
git commit -m "feat: interactive init with profile selection and confirm"
```

---

### Task 12: verificación E2E + README

**Files:**
- Create: `README.md`
- Test: `internal/cli/e2e_test.go`

**Interfaces:**
- Consumes: todo
- Produces: flujo completo verificado init → list → doctor → drift → doctor.

- [ ] **Step 1: Escribir el test E2E**

`internal/cli/e2e_test.go`:

```go
package cli_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestEndToEnd walks the full demo: init → list → doctor healthy →
// break something → doctor catches it.
func TestEndToEnd(t *testing.T) {
	home := t.TempDir()

	// 1. Onboarding: init with both profiles
	out, err := runAndes(t, home,
		"init", "--catalog", fixtureCatalog(t), "--profiles", "andespath-core,tri-fleet", "--yes")
	if err != nil {
		t.Fatalf("init: %v\n%s", err, out)
	}

	// 2. list shows everything installed
	out, err = runAndes(t, home, "list")
	if err != nil {
		t.Fatalf("list: %v\n%s", err, out)
	}
	if strings.Contains(out, "✗") {
		t.Errorf("tras init completo no debería haber skills sin instalar:\n%s", out)
	}

	// 3. doctor healthy
	if out, err = runAndes(t, home, "doctor"); err != nil {
		t.Fatalf("doctor sano: %v\n%s", err, out)
	}

	// 4. simulate local edit → doctor catches modified
	skillMD := filepath.Join(home, ".claude", "skills", "golang", "SKILL.md")
	os.WriteFile(skillMD, []byte("# tocado a mano"), 0o644)

	out, err = runAndes(t, home, "doctor")
	if err == nil {
		t.Errorf("doctor debería detectar la skill modificada:\n%s", out)
	}
	if !strings.Contains(out, "modificada") {
		t.Errorf("doctor no clasificó como modificada:\n%s", out)
	}

	// 5. re-init repairs
	if out, err = runAndes(t, home,
		"init", "--profiles", "andespath-core,tri-fleet", "--yes"); err != nil {
		t.Fatalf("re-init: %v\n%s", err, out)
	}
	if out, err = runAndes(t, home, "doctor"); err != nil {
		t.Fatalf("doctor tras re-init debería estar sano: %v\n%s", err, out)
	}
}
```

- [ ] **Step 2: Correr TODO**

Run: `go test ./... -v && go vet ./... && gofmt -l .`
Expected: PASS todos, vet limpio, gofmt sin output.

- [ ] **Step 3: Escribir README**

`README.md`:

```markdown
# andes-ai

Gestor de skills de agentes IA de andespath. Instala sets de skills
estandarizados (perfiles) desde un catálogo central hacia `~/.claude/skills/`,
con un manifiesto-recibo y diagnóstico de drift.

## Quickstart

```bash
go build -o andes ./cmd/andes

# Onboarding: elegir perfiles e instalar (interactivo)
./andes init --catalog ./testdata/catalog

# O scripteado (CI, dotfiles, onboarding automatizado)
./andes init --catalog ./testdata/catalog --profiles andespath-core,tri-fleet --yes

# Ver qué hay y qué tenés
./andes list

# Chequear drift (exit != 0 si hay problemas)
./andes doctor
```

## Conceptos

- **Catálogo**: carpeta (repo git en v2) con `catalog.json` + `skills/<id>/SKILL.md`.
- **Perfil**: bundle nombrado de skills (`andespath-core` para todos, uno por equipo/cliente).
- **Manifiesto** (`~/.claude/andes.json`): recibo de qué está instalado, con hash por skill.
- **Reparación**: siempre re-correr `andes init`. `doctor` diagnostica, no toca.

`andes` solo administra las skills que instaló (las del manifiesto) — jamás
toca skills personales en `~/.claude/skills/`.

## Diseño

Spec completo en `docs/superpowers/specs/2026-07-07-andes-ai-mvp-design.md`.
```

- [ ] **Step 4: Commit final**

```bash
git add README.md internal/cli/e2e_test.go
git commit -m "feat: end-to-end test and readme"
```

---

## Self-Review (ya aplicado)

- **Cobertura de spec:** init (idempotente, plan, atómico, scriptable, interactivo) → Tasks 8+11; list → Task 9; doctor (3 estados, exit code, nunca escribe) → Tasks 7+10; contrato de datos → Tasks 2+4; hash → Task 3; fixture real → Task 2; regla de propiedad (no tocar skills ajenas) → test en Task 8; validación temprana del catálogo → Task 2; errores en español accionables → transversal.
- **Placeholders:** ninguno — todo step tiene código o comando completo. Los branches `// interactivo: Task 11` de Task 8 no son placeholders: son código funcional (error accionable) que Task 11 reemplaza con edits exactos old→new.
- **Consistencia de tipos:** `ResolveSkills` retorna `map[string]string` (Task 2) y así lo consume `Plan` (Task 5); `Action.Hash` definido en Task 5, usado por `Apply` (Task 6); `manifest.Load` retorna `(nil, nil)` si no existe (Task 4) y así lo manejan Tasks 8-10; `runAndes`/`fixtureCatalog` definidos en Task 8 y reutilizados en Tasks 9, 10, 12.
