# andes-ai

andespath AI agent skill manager. Installs standardized skill sets (profiles)
from a central catalog into `~/.claude/skills/`, with an install receipt manifest
and drift diagnostics.

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
./andes install --catalog ./catalog --profiles andespath-core --yes   # local catalog
```

## Concepts

- **Catalog**: folder (git repo in v2) with `catalog.json` + `skills/<id>/SKILL.md`.
- **Profile**: named bundle of skills (`andespath-core` for everyone, one per team/client).
- **Manifest** (`~/.claude/andes.json`): receipt of what is installed, with a hash per skill.
- **Repair**: always re-run `andes install`. `doctor` diagnoses, never touches.

`andes` only manages the skills it installed (those in the manifest) — it never
touches personal skills in `~/.claude/skills/`.

## Design

Full spec in `docs/superpowers/specs/2026-07-07-andes-ai-mvp-design.md`.
