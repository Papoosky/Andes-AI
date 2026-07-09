# andes-ai

andespath AI agent skill manager. Installs standardized skill sets (profiles)
from a central catalog into `~/.claude/skills/`, with an install receipt manifest
and drift diagnostics.

## Install

From a clone (works for the private repo — `gh` handles auth):

```bash
gh repo clone Papoosky/Andes-AI && ./Andes-AI/install.sh
```

One-liner (once the repo is public or on an internal mirror):

```bash
curl -fsSL https://raw.githubusercontent.com/Papoosky/Andes-AI/main/install.sh | bash
# pin a version:
curl -fsSL https://raw.githubusercontent.com/Papoosky/Andes-AI/main/install.sh | bash -s -- --version v0.1.0
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

- **Catalog**: this repo's `catalog/` directory — `catalog.json` + `skills/<id>/SKILL.md`.
  The tool and the catalog live in the same repo; consumers read a managed git
  mirror of it at `~/.andes/catalog`.
- **Profile**: named bundle of skills (`andespath-core` for everyone, one per team/client).
- **Manifest** (`~/.claude/andes.json`): receipt of what is installed, with a hash per skill.
- **Repair**: always re-run `andes install`. `doctor` diagnoses, never touches.

`andes` only manages the skills it installed (those in the manifest) — it never
touches personal skills in `~/.claude/skills/`.

## Contributing

- Adding or editing a **skill**: see [CONTRIBUTING.md](CONTRIBUTING.md).
- Working on the **tool** (build, test, layout, conventions): see [AGENTS.md](AGENTS.md).
