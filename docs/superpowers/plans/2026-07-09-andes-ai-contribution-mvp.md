# andes-ai Contribution MVP (Part B) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax.

**Goal:** A minimal `andes validate` command plus contribution docs/CI so a broken catalog PR can't merge and contributors have a clear workflow.

**Architecture:** `andes validate` locates a local catalog (cwd walk-up or `--catalog`), reuses `catalog.LocalDir.Load()` (JSON + skill-existence + id validation) and adds `catalog.Lint()` (empty profiles, duplicate skill in a profile, SKILL.md frontmatter present). A GitHub Actions workflow runs it on every PR. Convention lives in CONTRIBUTING.md + a PR template.

**Tech Stack:** Go 1.23, Cobra. TDD, table-driven + `t.TempDir()` fixtures. No new Go dependencies.

**Spec:** `docs/superpowers/specs/2026-07-09-andes-ai-contribution-mvp-design.md`

## Global Constraints

- Everything user-facing in ENGLISH, actionable. Conventional Commits, no AI attribution, no Co-Authored-By. TDD.
- `validate` is contributor-mode: operates on a LOCAL catalog checkout, NOT the managed mirror. Location: `--catalog <path>` → else walk up from cwd for `catalog.json` → else actionable error `no catalog.json found — run this inside a catalog repo, or pass --catalog <path>`.
- Reuse `catalog.LocalDir.Load()` for JSON/skill-existence/id checks; `catalog.Lint()` adds the rest. Problems are accumulated and sorted, not fail-on-first (matches existing `LocalDir.validate`).
- No new Go dependency for frontmatter (lightweight line parse, not a YAML lib).
- No `andes new`, no `andes profile` — out of scope.
- The production `catalog/` at repo root must pass `andes validate`.

---

### Task 1: `catalog.Lint` — the extra content checks

**Files:**
- Create: `internal/catalog/lint.go`
- Test: `internal/catalog/lint_test.go`

**Interfaces:**
- Consumes: `Source` (for `SkillPath`), `Catalog` (existing types).
- Produces:
  - `func Lint(src Source, c *Catalog) []string` — returns sorted problem strings (empty slice = clean). Checks: empty profile, duplicate skill id within a profile, SKILL.md missing non-empty `name`/`description` frontmatter. Assumes base validation (skills exist) already passed; silently skips a skill whose SKILL.md is unreadable.

- [ ] **Step 1: Write the failing tests**

`internal/catalog/lint_test.go`:

```go
package catalog_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/andespath/andes-ai/internal/catalog"
)

// lintFixture builds a temp catalog dir with the given profiles and skills.
// skills maps skill-id → SKILL.md content (empty string means "omit the file").
func lintFixture(t *testing.T, profiles map[string]catalog.Profile, skills map[string]string) catalog.LocalDir {
	t.Helper()
	root := t.TempDir()
	for id, content := range skills {
		dir := filepath.Join(root, "skills", id)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		if content != "" {
			if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte(content), 0o644); err != nil {
				t.Fatal(err)
			}
		}
	}
	return catalog.LocalDir{Root: root}
}

const goodMD = "---\nname: x\ndescription: does a thing\n---\n# X\n"

func TestLintClean(t *testing.T) {
	src := lintFixture(t,
		map[string]catalog.Profile{"core": {Description: "d", Skills: []string{"a", "b"}}},
		map[string]string{"a": goodMD, "b": goodMD},
	)
	c := &catalog.Catalog{Name: "x", Profiles: map[string]catalog.Profile{
		"core": {Description: "d", Skills: []string{"a", "b"}},
	}}
	if got := catalog.Lint(src, c); len(got) != 0 {
		t.Errorf("Lint clean catalog = %v, want none", got)
	}
}

func TestLintProblems(t *testing.T) {
	tests := []struct {
		name     string
		profiles map[string]catalog.Profile
		skills   map[string]string
		wantSub  string
	}{
		{
			name:     "empty profile",
			profiles: map[string]catalog.Profile{"empty": {Description: "d", Skills: []string{}}},
			skills:   map[string]string{},
			wantSub:  "has no skills",
		},
		{
			name:     "duplicate skill in profile",
			profiles: map[string]catalog.Profile{"core": {Description: "d", Skills: []string{"a", "a"}}},
			skills:   map[string]string{"a": goodMD},
			wantSub:  "more than once",
		},
		{
			name:     "missing frontmatter",
			profiles: map[string]catalog.Profile{"core": {Description: "d", Skills: []string{"a"}}},
			skills:   map[string]string{"a": "# no frontmatter here\n"},
			wantSub:  "frontmatter",
		},
		{
			name:     "frontmatter present but empty description",
			profiles: map[string]catalog.Profile{"core": {Description: "d", Skills: []string{"a"}}},
			skills:   map[string]string{"a": "---\nname: x\ndescription:\n---\n"},
			wantSub:  "frontmatter",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			src := lintFixture(t, tt.profiles, tt.skills)
			c := &catalog.Catalog{Name: "x", Profiles: tt.profiles}
			got := catalog.Lint(src, c)
			joined := strings.Join(got, "\n")
			if !strings.Contains(joined, tt.wantSub) {
				t.Errorf("Lint = %q, want a problem containing %q", joined, tt.wantSub)
			}
		})
	}
}

func TestLintSorted(t *testing.T) {
	src := lintFixture(t,
		map[string]catalog.Profile{"z": {Description: "d", Skills: []string{}}, "a": {Description: "d", Skills: []string{}}},
		map[string]string{},
	)
	c := &catalog.Catalog{Name: "x", Profiles: src2profiles("z", "a")}
	got := catalog.Lint(src, c)
	if len(got) != 2 || got[0] > got[1] {
		t.Errorf("Lint problems not sorted: %v", got)
	}
}

func src2profiles(names ...string) map[string]catalog.Profile {
	m := map[string]catalog.Profile{}
	for _, n := range names {
		m[n] = catalog.Profile{Description: "d", Skills: []string{}}
	}
	return m
}
```

- [ ] **Step 2: Run tests, verify they fail**

Run: `go test ./internal/catalog/ -run TestLint`
Expected: FAIL — `undefined: catalog.Lint`.

- [ ] **Step 3: Implement `internal/catalog/lint.go`**

```go
package catalog

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Lint performs content checks BEYOND Load's structural validation
// (Load already covers JSON validity, skill existence, and id safety).
// It returns a sorted list of problems; an empty slice means the catalog
// is clean. It assumes Load has already passed, so a skill whose SKILL.md
// is unreadable here is skipped rather than reported twice.
func Lint(src Source, c *Catalog) []string {
	var problems []string

	for name, p := range c.Profiles {
		if len(p.Skills) == 0 {
			problems = append(problems, fmt.Sprintf("profile %q has no skills", name))
		}
		seen := map[string]bool{}
		for _, id := range p.Skills {
			if seen[id] {
				problems = append(problems, fmt.Sprintf("profile %q lists skill %q more than once", name, id))
				continue
			}
			seen[id] = true
		}
	}

	// Check each unique referenced skill's frontmatter once.
	checked := map[string]bool{}
	for _, p := range c.Profiles {
		for _, id := range p.Skills {
			if checked[id] {
				continue
			}
			checked[id] = true
			mdPath := filepath.Join(src.SkillPath(id), "SKILL.md")
			data, err := os.ReadFile(mdPath)
			if err != nil {
				continue // existence is Load's job; skip unreadable here
			}
			name, desc := frontmatterFields(data)
			if name == "" || desc == "" {
				problems = append(problems,
					fmt.Sprintf("skill %q: SKILL.md is missing frontmatter name/description", id))
			}
		}
	}

	sort.Strings(problems)
	return problems
}

// frontmatterFields extracts name/description from a leading YAML frontmatter
// block delimited by --- lines. Lightweight line parse — no YAML dependency.
// Returns empty strings if the block or a field is absent.
func frontmatterFields(md []byte) (name, desc string) {
	s := string(md)
	if !strings.HasPrefix(s, "---") {
		return "", ""
	}
	rest := s[len("---"):]
	end := strings.Index(rest, "\n---")
	if end < 0 {
		return "", ""
	}
	block := rest[:end]
	for _, line := range strings.Split(block, "\n") {
		line = strings.TrimSpace(line)
		if v, ok := strings.CutPrefix(line, "name:"); ok {
			name = strings.TrimSpace(v)
		}
		if v, ok := strings.CutPrefix(line, "description:"); ok {
			desc = strings.TrimSpace(v)
		}
	}
	return name, desc
}
```

- [ ] **Step 4: Run tests, verify pass**

Run: `go test ./internal/catalog/ -v -run TestLint`
Expected: PASS all.

- [ ] **Step 5: Commit**

```bash
git add internal/catalog/lint.go internal/catalog/lint_test.go
git commit -m "feat: catalog Lint for empty profiles, dup skills, frontmatter"
```

---

### Task 2: `andes validate` command

**Files:**
- Create: `internal/cli/validate.go`
- Modify: `internal/cli/root.go` (register subcommand)
- Test: `internal/cli/validate_test.go`

**Interfaces:**
- Consumes: `catalog.LocalDir.Load` (base checks), `catalog.Lint` (Task 1).
- Produces:
  - `func newValidateCmd() *cobra.Command` with `--catalog string`.
  - `func findCatalogRoot() (string, error)` — walks up from cwd for `catalog.json`.
  - Success output `✓ catalog valid: N profiles, M skills` (M = unique skill ids across profiles), exit 0. Any problem → returned error, exit 1.

- [ ] **Step 1: Write the failing tests**

`internal/cli/validate_test.go`:

```go
package cli_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// writeCatalog builds a temp catalog dir. profilesJSON is the "profiles" object
// body; skills maps id → SKILL.md content ("" = create dir without SKILL.md).
func writeCatalog(t *testing.T, profilesJSON string, skills map[string]string) string {
	t.Helper()
	root := t.TempDir()
	cat := `{"name":"test","profiles":` + profilesJSON + `}`
	if err := os.WriteFile(filepath.Join(root, "catalog.json"), []byte(cat), 0o644); err != nil {
		t.Fatal(err)
	}
	for id, content := range skills {
		dir := filepath.Join(root, "skills", id)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		if content != "" {
			if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte(content), 0o644); err != nil {
				t.Fatal(err)
			}
		}
	}
	return root
}

const skillMD = "---\nname: x\ndescription: d\n---\n# X\n"

func TestValidateValid(t *testing.T) {
	home := t.TempDir()
	root := writeCatalog(t, `{"core":{"description":"d","skills":["a"]}}`, map[string]string{"a": skillMD})
	out, err := runAndes(t, home, "validate", "--catalog", root)
	if err != nil {
		t.Fatalf("validate valid catalog: %v\n%s", err, out)
	}
	if !strings.Contains(out, "catalog valid") || !strings.Contains(out, "1 profiles") || !strings.Contains(out, "1 skills") {
		t.Errorf("unexpected success output:\n%s", out)
	}
}

func TestValidateFailures(t *testing.T) {
	tests := []struct {
		name        string
		profilesJSON string
		skills      map[string]string
		wantSub     string
	}{
		{"missing skill", `{"core":{"description":"d","skills":["ghost"]}}`, map[string]string{}, "ghost"},
		{"empty profile", `{"empty":{"description":"d","skills":[]}}`, map[string]string{}, "has no skills"},
		{"dup skill", `{"core":{"description":"d","skills":["a","a"]}}`, map[string]string{"a": skillMD}, "more than once"},
		{"no frontmatter", `{"core":{"description":"d","skills":["a"]}}`, map[string]string{"a": "# nope\n"}, "frontmatter"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			home := t.TempDir()
			root := writeCatalog(t, tt.profilesJSON, tt.skills)
			out, err := runAndes(t, home, "validate", "--catalog", root)
			if err == nil {
				t.Fatalf("expected validation failure, got success:\n%s", out)
			}
			if !strings.Contains(err.Error()+out, tt.wantSub) {
				t.Errorf("error = %v\n%s\nwant substring %q", err, out, tt.wantSub)
			}
		})
	}
}

func TestValidateBrokenJSON(t *testing.T) {
	home := t.TempDir()
	root := t.TempDir()
	os.WriteFile(filepath.Join(root, "catalog.json"), []byte("{not json"), 0o644)
	if _, err := runAndes(t, home, "validate", "--catalog", root); err == nil {
		t.Error("broken JSON should fail validation")
	}
}

// chdir switches to dir and restores the previous cwd after the test.
// Used instead of t.Chdir (Go 1.24+) because the module targets Go 1.23.
// These tests do NOT call t.Parallel, so process-wide cwd mutation is safe.
func chdir(t *testing.T, dir string) {
	t.Helper()
	prev, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(prev) })
}

func TestValidateCwdDetection(t *testing.T) {
	home := t.TempDir()
	root := writeCatalog(t, `{"core":{"description":"d","skills":["a"]}}`, map[string]string{"a": skillMD})
	sub := filepath.Join(root, "skills", "a")
	chdir(t, sub) // run from a nested dir; validate must walk up to catalog.json
	out, err := runAndes(t, home, "validate")
	if err != nil {
		t.Fatalf("cwd detection: %v\n%s", err, out)
	}
	if !strings.Contains(out, "catalog valid") {
		t.Errorf("cwd detection did not validate:\n%s", out)
	}
}

func TestValidateNoCatalogAnywhere(t *testing.T) {
	home := t.TempDir()
	chdir(t, t.TempDir()) // empty dir; no catalog.json up the tree within temp
	_, err := runAndes(t, home, "validate")
	if err == nil {
		t.Error("validate with no catalog should fail with an actionable error")
	}
}

func TestProductionCatalogIsValid(t *testing.T) {
	home := t.TempDir()
	abs, err := filepath.Abs("../../catalog")
	if err != nil {
		t.Fatal(err)
	}
	out, err := runAndes(t, home, "validate", "--catalog", abs)
	if err != nil {
		t.Fatalf("the production catalog/ must pass validate: %v\n%s", err, out)
	}
}
```

- [ ] **Step 2: Run tests, verify they fail**

Run: `go test ./internal/cli/ -run TestValidate`
Expected: FAIL — `unknown command "validate"`.

- [ ] **Step 3: Implement `internal/cli/validate.go`**

```go
package cli

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/andespath/andes-ai/internal/catalog"
)

func newValidateCmd() *cobra.Command {
	var catalogFlag string
	cmd := &cobra.Command{
		Use:   "validate",
		Short: "Validate a catalog before opening a PR (also run by CI)",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runValidate(cmd, catalogFlag)
		},
	}
	cmd.Flags().StringVar(&catalogFlag, "catalog", "", "path to the catalog folder (default: search up from the current directory)")
	return cmd
}

func runValidate(cmd *cobra.Command, catalogFlag string) error {
	root := catalogFlag
	if root == "" {
		found, err := findCatalogRoot()
		if err != nil {
			return err
		}
		root = found
	}

	src := catalog.LocalDir{Root: root}
	c, err := src.Load() // JSON, skill existence, id safety — accumulated + sorted
	if err != nil {
		return err
	}

	if problems := catalog.Lint(src, c); len(problems) > 0 {
		return fmt.Errorf("invalid catalog:\n  %s", strings.Join(problems, "\n  "))
	}

	unique := map[string]bool{}
	for _, p := range c.Profiles {
		for _, id := range p.Skills {
			unique[id] = true
		}
	}
	fmt.Fprintf(cmd.OutOrStdout(), "✓ catalog valid: %d profiles, %d skills\n", len(c.Profiles), len(unique))
	return nil
}

// findCatalogRoot walks up from the current directory looking for catalog.json.
func findCatalogRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("could not resolve the current directory: %w", err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "catalog.json")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", errors.New("no catalog.json found — run this inside a catalog repo, or pass --catalog <path>")
		}
		dir = parent
	}
}
```

In `internal/cli/root.go`, add `newValidateCmd()` to the existing `AddCommand(...)` call (append it to the current argument list).

- [ ] **Step 4: Run tests, verify pass**

Run: `go test ./internal/cli/ -v -run TestValidate` then `go test ./...`
Expected: PASS all, including `TestProductionCatalogIsValid`.

- [ ] **Step 5: Commit**

```bash
git add internal/cli/validate.go internal/cli/validate_test.go internal/cli/root.go
git commit -m "feat: andes validate command for catalog checks"
```

---

### Task 3: CI gate + CONTRIBUTING.md + PR template

**Files:**
- Create: `.github/workflows/validate.yml`
- Create: `CONTRIBUTING.md`
- Create: `.github/PULL_REQUEST_TEMPLATE.md`

**Interfaces:**
- Consumes: the `andes validate` command (Task 2).
- Produces: a required PR check + contributor docs.

- [ ] **Step 1: Write the CI workflow**

`.github/workflows/validate.yml`:

```yaml
name: validate
on:
  pull_request:
    branches: [main]
jobs:
  validate:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: "1.23"
      - name: Build andes
        run: go build -o andes ./cmd/andes
      - name: Validate catalog
        run: ./andes validate --catalog catalog
```

- [ ] **Step 2: Verify the workflow validates the real catalog locally**

Run: `go build -o andes ./cmd/andes && ./andes validate --catalog catalog`
Expected: `✓ catalog valid: 2 profiles, 3 skills` (exit 0). This is exactly what the CI step runs.

- [ ] **Step 3: Write CONTRIBUTING.md**

`CONTRIBUTING.md`:

```markdown
# Contributing skills to the andespath catalog

The catalog lives in `catalog/`. A skill is a folder `catalog/skills/<id>/`
containing a `SKILL.md`. Profiles in `catalog/catalog.json` group skills for
who should install them. Profiles are organizing buckets, not ranks:

- `andespath-core` — company-wide standards everyone installs (git conventions,
  code review, microservice practices, …).
- team profiles (e.g. `tri-fleet`) — skills specific to one team's stack.

A skill can belong to several profiles. There is no promotion ladder: open a PR
that adds your skill to whichever profile(s) make sense — a team profile,
`andespath-core`, or both.

## Add a skill

1. Create `catalog/skills/<id>/SKILL.md` with frontmatter:

   ```markdown
   ---
   name: <id>
   description: One line — what it is and when the agent should use it.
   ---

   # <Title>

   The guidance itself.
   ```

2. Reference the skill in one or more profiles in `catalog/catalog.json`:

   ```json
   "andespath-core": { "description": "...", "skills": ["git-conventions", "<id>"] }
   ```

   One PR may add multiple skills.

3. Validate locally before pushing:

   ```bash
   go build -o andes ./cmd/andes
   ./andes validate --catalog catalog
   ```

4. Open a PR. CI runs `andes validate` — a PR that breaks the catalog cannot
   merge. Reviewers discuss the skill's content and fit.
```

- [ ] **Step 4: Write the PR template**

`.github/PULL_REQUEST_TEMPLATE.md`:

```markdown
## What skill(s) does this add or change?

<!-- id(s) and a one-line summary each -->

## Which profile(s) and why?

<!-- andespath-core (company-wide) / a team profile / both — and the reasoning -->

## Did you test it?

<!-- How you exercised the skill with an agent, or why it's low-risk -->

## Checklist

- [ ] `andes validate --catalog catalog` passes locally
- [ ] Each skill has `SKILL.md` frontmatter (name + description)
```

- [ ] **Step 5: Commit**

```bash
git add .github/workflows/validate.yml CONTRIBUTING.md .github/PULL_REQUEST_TEMPLATE.md
git commit -m "docs: contribution guide, PR template, and catalog CI gate"
```

---

## Self-Review (applied)

- **Spec coverage:** `andes validate` (cwd detection + `--catalog`, reuse Load, extra checks, accumulate+sort, exit codes) → Tasks 1+2; CI gate → Task 3; CONTRIBUTING.md → Task 3; PR template → Task 3; production catalog passes → `TestProductionCatalogIsValid` in Task 2; two-mode note honored (validate never touches the mirror). Out-of-scope items (`new`, `profile`) correctly absent.
- **Placeholders:** none — every step has complete code/commands. The `<id>`/`<Title>` tokens in CONTRIBUTING.md are intentional doc placeholders for the reader, not plan gaps.
- **Type consistency:** `catalog.Lint(src Source, c *Catalog) []string` defined in Task 1, consumed verbatim in Task 2's `runValidate`; `frontmatterFields` is internal to `lint.go`; `findCatalogRoot`/`newValidateCmd` names consistent; `runAndes`/`writeCatalog`/`chdir` helpers used consistently across Task 2 tests (`runAndes` already exists in the cli test package; `writeCatalog`/`chdir` are defined in `validate_test.go`). The cwd tests use a local `chdir` helper (os.Chdir + t.Cleanup) rather than `t.Chdir`, since `t.Chdir` needs Go 1.24 and the module targets 1.23 — this compiles cleanly and the validate tests are non-parallel.
```
