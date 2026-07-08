# andes-ai

andespath AI agent skill manager. Installs standardized skill sets (profiles)
from a central catalog into `~/.claude/skills/`, with an install receipt manifest
and drift diagnostics.

## Quickstart

```bash
go build -o andes ./cmd/andes

# Onboarding: choose profiles and install (interactive)
./andes init --catalog ./catalog

# Or scripted (CI, dotfiles, automated onboarding)
./andes init --catalog ./catalog --profiles andespath-core,tri-fleet --yes

# See what's available and what you have
./andes list

# Check for drift (exit != 0 if there are problems)
./andes doctor
```

## Concepts

- **Catalog**: folder (git repo in v2) with `catalog.json` + `skills/<id>/SKILL.md`.
- **Profile**: named bundle of skills (`andespath-core` for everyone, one per team/client).
- **Manifest** (`~/.claude/andes.json`): receipt of what is installed, with a hash per skill.
- **Repair**: always re-run `andes init`. `doctor` diagnoses, never touches.

`andes` only manages the skills it installed (those in the manifest) — it never
touches personal skills in `~/.claude/skills/`.

## Design

Full spec in `docs/superpowers/specs/2026-07-07-andes-ai-mvp-design.md`.
