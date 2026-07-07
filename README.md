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
