# andes-ai Contribution MVP (Part B) — Design Doc

**Date:** 2026-07-09
**Status:** Approved (validated section by section)
**Scope:** Part B of v2 — the skill contribution workflow. Deliberately minimal: convention + one protective command, no scaffolding/editing commands.

## Context

The catalog lives in the company git repo (`catalog/` with `catalog.json` declaring profiles + `skills/<id>/SKILL.md`). Profiles (`andespath-core`, `tri-fleet`, …) are the organizing buckets, NOT a promotion ladder. A contribution is a PR that edits `catalog.json` to add one or more skills to one or more profiles. GitHub PRs provide the discussion; human reviewers judge content and fit.

## Decisions (with the user)

- **Tiers = profiles, not a ladder.** A PR may target a team profile (`tri-fleet`), the company profile (`andespath-core`), or both. One PR may add several skills. There is NO forced personal→team→company path.
- **No `andes new`, no `andes profile add/rm`.** Creating a skill folder and editing `catalog.json` by hand is easy; a `CONTRIBUTING.md` explains it and human review corrects content. Building those commands is premature for the MVP.
- **Keep one machine check.** Humans miss what machines catch: a mistyped skill id in `catalog.json` (e.g. `"golang-testng"`) reads past human review but breaks EVERY consumer's install (the catalog fails to load — maximum blast radius). A minimal `andes validate` catches exactly this. It is a thin wrapper over the existing `catalog.LocalDir.Load()` validation.
- **Two modes of andes.** Consumer mode (`install`/`list`/`doctor`/`update`) reads the managed mirror (`~/.andes/catalog`, read-only). Contributor mode (`validate`) operates on a LOCAL catalog checkout (the cloned repo), never the mirror.

## Deliverables

### 1. `andes validate [--catalog <path>]`

Contributor-mode command.

- **Catalog location:** if `--catalog` is given, use it. Otherwise detect `catalog.json` by walking up from the current working directory (run it inside the cloned repo). If none found → actionable error: `no catalog.json found — run this inside a catalog repo, or pass --catalog <path>`.
- **Checks:**
  1. `catalog.json` is valid JSON (reuses `LocalDir.Load`).
  2. Every skill referenced by a profile exists as `skills/<id>/SKILL.md` (reuses `LocalDir.Load`'s existing validation).
  3. No empty profiles (a profile with zero skills).
  4. No duplicate skill id within a single profile.
  5. Each referenced skill's `SKILL.md` has YAML frontmatter with non-empty `name` and `description`.
- **Output:** on success `✓ catalog valid: N profiles, M skills` (exit 0). On failure, list every problem (accumulated, sorted — not fail-on-first) with actionable messages, exit 1.
- **Reuse:** checks 1–2 come from `LocalDir.Load` (already implemented and tested). Checks 3–5 are new, in the validator.

### 2. `.github/workflows/validate.yml`

Runs on every pull request targeting `main`:
- Checks out the repo, sets up Go, builds `andes`.
- Runs `andes validate --catalog .`.
- A PR that breaks the catalog cannot be merged (required check). This is the machine gate that catches the human-invisible typo.

### 3. `CONTRIBUTING.md`

Documents the workflow in English:
- What a skill is (`skills/<id>/SKILL.md`, frontmatter with `name` + `description`).
- How to add one: create the folder + `SKILL.md`, add the id to the chosen profile(s) in `catalog.json`.
- Profiles are organizing buckets, not ranks: pick `tri-fleet` for team-specific skills, `andespath-core` for company-wide standards (git conventions, microservice practices), or both. A single PR may add multiple skills.
- Run `andes validate` locally before pushing.
- Open a PR; reviewers discuss content and fit.

### 4. `.github/PULL_REQUEST_TEMPLATE.md`

Guides the PR discussion: which skill(s)? which profile(s) and why? did you test it? Keeps reviews consistent.

## Out of scope (explicit)

- `andes new` (skill scaffolding), `andes profile add/rm` (catalog editing) — hand-edit + review instead.
- `andes try <pr>` (installing from an open PR) — that is Part C, separate work.
- Any promotion "engine" — git + profiles + PRs already do it.

## Testing

- `andes validate` unit/integration tests (table-driven + `t.TempDir()` catalogs): valid catalog → OK exit 0; broken JSON → error; profile references missing skill → error; empty profile → error; duplicate skill in profile → error; SKILL.md missing frontmatter → error; cwd detection finds catalog in an ancestor dir; `--catalog` override works; no catalog anywhere → actionable error.
- The production `catalog/` must pass `andes validate` (a test asserts it).

## File structure

```
internal/cli/validate.go        ← new: newValidateCmd + runValidate + the extra checks
internal/cli/validate_test.go
internal/catalog/                ← reuse LocalDir.Load; add a frontmatter check helper if cleaner here
.github/workflows/validate.yml   ← new: PR gate
.github/PULL_REQUEST_TEMPLATE.md ← new
CONTRIBUTING.md                  ← new
```
