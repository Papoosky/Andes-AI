# AGENTS.md

Guide for anyone — human or AI agent — working on the **andes** tool itself.
For adding or editing skills, see [CONTRIBUTING.md](CONTRIBUTING.md) instead.

## What this is

`andes` is andespath's AI agent skill manager: a single Go binary that installs
standardized skill sets (profiles) from a central catalog into
`~/.claude/skills/`, tracks what it installed in a manifest, and diagnoses drift.

The **tool and the catalog live in the same repo**. Consumers read a managed git
mirror of `catalog/` at `~/.andes/catalog`; contributors edit `catalog/` directly.

## Run, build, test

```bash
# Build the binary
go build -o andes ./cmd/andes

# Run the full test suite (do this before every commit)
go test ./...

# Run the interactive TUI
./andes

# Run a command directly against the local catalog (no git mirror)
./andes install --catalog ./catalog --profiles andespath-core --yes
./andes list
./andes doctor
./andes validate --catalog catalog
```

Requires Go 1.23+. No other runtime dependencies — `git` is used at runtime for
the catalog mirror.

### Install script

`./install.sh` downloads a release binary via `gh`, or builds from source as a
fallback. When building from source it bakes the catalog git URL into the binary
(derived from `REPO`, probing SSH then HTTPS) so `andes install` needs no
`--catalog` flag. Override with `ANDES_CATALOG_URL`.

Downloaded release binaries bake a single URL (the `ANDES_CATALOG_URL` repo
variable). At runtime `andes` probes SSH then HTTPS for that catalog
(`pickWorkingGitURL` in `internal/cli/resolve.go`), so one binary serves both
SSH and HTTPS devs; the URL that works is persisted in the manifest.

## Commands

| Command | Mode | What it does |
|---------|------|--------------|
| `install` | consumer | Install/repair skills for the chosen profiles |
| `list` | consumer | Show catalog and install status |
| `doctor` | consumer | Diagnose drift (never modifies anything) |
| `update` | consumer | Sync the catalog mirror and re-apply |
| `validate` | contributor | Check a local catalog checkout (JSON, skill refs, frontmatter) |

Consumer commands read the managed mirror (`~/.andes/catalog`, read-only).
`validate` operates on a local catalog checkout — never the mirror.

## Project layout

```
cmd/andes/           Entry point (main.go)
internal/
  catalog/           Catalog loading + validation. Source interface:
                       LocalDir (local checkout) and GitRepo (managed mirror).
                       lint.go adds extra content checks (empty profiles, dups,
                       frontmatter).
  installer/         Plan (diff desired vs manifest) + Apply (copy skill folders).
  manifest/          The install receipt at ~/.claude/andes.json (atomic save).
  hashdir/           sha256 per skill folder — drift detection.
  doctor/            Drift diagnostics.
  cli/               Cobra commands + TUI wiring (callbacks injected into tui).
  tui/               Bubbletea TUI (menu, install flow, output screens).
  logo/, theme/      Company branding (braille logo, color palette).
```

Key architectural seams:
- **`catalog.Source`** decouples the installer/doctor/list from where the catalog
  lives (local dir vs git mirror).
- **`applyActions`** (in `cli/apply_core.go`) is the single manifest-writing path
  shared by CLI and TUI — do not add a second one.
- **`cli` imports `tui`, never the reverse.** The TUI receives all its
  dependencies as injected function values (see `cli/tuiwire.go`).

## CI

The `ci` workflow (`.github/workflows/ci.yml`) runs on every PR to `main`:
`go build` → `go test ./...` → `andes validate --catalog catalog`. It covers
both code PRs and skill PRs — a code-only PR passes `validate` trivially because
the catalog is unchanged. All three steps must pass to merge.

Releases are automated with **release-please** (`.github/workflows/release-please.yml`).
It reads the Conventional Commits on `main` and keeps a **Release PR** open with
the next version bump + `CHANGELOG.md`. Merging that PR creates the git tag and
GitHub Release; the workflow then builds the multi-OS/arch binaries and uploads
them as release assets. Version state lives in `.release-please-manifest.json`;
config in `release-please-config.json`. You never tag by hand — you merge the
Release PR when you want to ship.

## Conventions

- **Commits**: Conventional Commits (`feat`, `fix`, `chore`, `docs`, `refactor`,
  `test`), English, imperative, no trailing period. One commit = one logical change.
- **Branches**: `type/short-description` (e.g. `feat/user-auth`).
- **Everything user-facing is in English.**
- **Tests**: table-driven, `t.TempDir()` for fixtures. Add a test with every
  behavior change; the suite must be green before committing.

## Contributing a change to the tool

1. Branch: `git checkout -b feat/my-change`.
2. Make the change with a test. Run `go test ./...`.
3. Commit following the conventions above.
4. Open a PR to `main` and let CI run.
