# andes-ai MVP — Design Doc

**Fecha:** 2026-07-07
**Estado:** Aprobado (diseño validado sección por sección en sesión de brainstorming)

## Contexto y problema

andespath es una software factory donde cada equipo trabaja como proyecto independiente para un cliente externo (ej. TRI fleet manager). Hoy cada dev usa un setup de IA distinto (skills, plugins, convenciones), el onboarding es lento y el conocimiento no se comparte.

**andes-ai es el gestor de paquetes de skills de agentes IA de la empresa**: un catálogo central de skills agrupadas en perfiles, y un CLI que las instala a nivel dev de forma replicable e idempotente.

Decisiones de modelo ya tomadas en las conversaciones previas:

- **Instalación a nivel DEV** (`~/.claude/skills/`), no por repo. Un dev de TRI con 20 microservicios instala una vez y todos sus repos heredan las skills. Install per-repo queda como excepción futura.
- **Perfiles** = bundles nombrados de skills (`andespath-core` para todos, `tri-fleet` por equipo/cliente). Es lo que da "mismo pero no idéntico".
- **Modelo COPY + manifiesto-recibo** (no symlinks, no wrapper de plugins nativos de Claude Code): archivos planos + un manifiesto declarativo que la herramienta controla.
- **Sin versionado semver por skill**: hash de contenido por skill + ref global del catálogo. Suficiente para detectar drift y desactualización.
- **Sin MCPs en v1** (los equipos están separados; el problema de aislamiento de MCP por cliente no aplica todavía).

## Objetivo del MVP (momento demo)

Un dev nuevo corre `andes init`, elige su(s) perfil(es), y termina con las skills instaladas en `~/.claude/skills/` funcionando en Claude Code. Onboarding de punta a punta con un comando.

## Alcance

**Incluye:** comandos `init`, `list`, `doctor`; catálogo local (carpeta en disco); catálogo fixture con contenido real mínimo.

**Excluye (explícito):** `update` (la reparación es re-correr `init`), catálogo git remoto, TUI, skills a nivel repo/proyecto, perfiles anidados, MCPs, managed settings.

## Decisiones de stack

- **Go + Cobra**: binario único sin runtime — ideal para una herramienta de onboarding (descargás y corre). Camino directo a TUI futura con Bubbletea.
- **`huh` (charmbracelet)** para prompts interactivos: misma familia que Bubbletea, el salto a TUI es continuidad.

## Arquitectura

```
andes-ai/
├── cmd/andes/main.go        # entry point
├── internal/
│   ├── cli/                 # comandos Cobra (init, list, doctor) — SOLO orquestan
│   ├── catalog/             # leer catálogo: interface Source + impl LocalDir
│   ├── manifest/            # leer/escribir ~/.claude/andes.json
│   ├── installer/           # copiar skills al destino (~/.claude/skills/)
│   └── doctor/              # motor de diff: manifiesto vs disco vs catálogo
└── testdata/catalog/        # catálogo fixture para tests y demo
```

Decisiones estructurales:

1. **`catalog.Source` es una interface** (`Profiles()`, `Skills()`, `FetchSkill(id)`). Única implementación v1: `LocalDir`. Cuando llegue el catálogo git remoto (v2), se agrega `GitRepo` sin tocar comandos, installer ni doctor.
2. **Los comandos Cobra no contienen lógica**: parsean flags, llaman a módulos internos, formatean salida. La TUI futura es otra capa de presentación sobre los mismos módulos.

## Contrato de datos

### Catálogo (`catalog.json` en la raíz de la carpeta catálogo)

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
      "skills": ["golang", "microservice-patterns"]
    }
  }
}
```

Estructura de carpeta:

```
catalog/
├── catalog.json
└── skills/
    ├── git-conventions/
    │   └── SKILL.md          # formato estándar de skills de Claude Code
    ├── code-review/
    └── golang/
```

Reglas:
- Una skill = una carpeta bajo `skills/` con `SKILL.md` (+ archivos auxiliares).
- Los perfiles solo REFERENCIAN skills por id; una skill puede estar en varios perfiles sin duplicarse.
- Perfil que referencia una skill inexistente → catálogo inválido → el CLI aborta al cargar con mensaje claro (validación temprana, nunca falla a mitad de install).

### Manifiesto (`~/.claude/andes.json`) — el recibo

```json
{
  "version": 1,
  "catalog": { "type": "local", "path": "/Users/pablo/andes-catalog" },
  "profiles": ["andespath-core", "tri-fleet"],
  "installed": {
    "git-conventions": { "hash": "sha256:ab12...", "profile": "andespath-core" },
    "golang": { "hash": "sha256:cd34...", "profile": "tri-fleet" }
  }
}
```

- **`hash` por skill** = sha256 del contenido de la carpeta (archivos ordenados por path, contenido concatenado). Permite a `doctor` distinguir "editado localmente" (disco ≠ manifiesto) de "hay versión nueva" (manifiesto ≠ catálogo) sin git ni semver.
- **`catalog.type`**: siempre `"local"` en v1; el campo existe desde el día 1 para que los manifiestos sigan siendo válidos cuando llegue `"git"`.
- **`version: 1`**: versión de schema del manifiesto, seguro de migración futura.

**Regla de propiedad:** `andes` es dueño SOLO de las skills listadas en `installed`. Jamás toca skills que el dev puso a mano en `~/.claude/skills/` (protege el tier personal).

## Comandos

### `andes init [--catalog <path>] [--profiles a,b] [--yes]`

1. Localiza el catálogo: flag `--catalog` → path guardado en manifiesto previo → pregunta interactiva.
2. Carga y valida el catálogo. Inválido = abort, no instala a medias.
3. Prompt interactivo (`huh`): checkbox de perfiles con descripciones. Con `--profiles` + `--yes` se saltea (scriptable desde el día 1).
4. Resuelve perfiles → set de skills (dedup si dos perfiles comparten una skill).
5. **Plan antes de tocar** (estilo `terraform plan`): muestra instalar/actualizar/sin cambio y pide confirmación. Skills con hash igual se saltean → **idempotente**: correr `init` 2× = correr 1×.
6. Copia cada skill a `~/.claude/skills/<id>/`, calcula hash, escribe el manifiesto **al final, atómico** (temp + rename). Si falla a mitad: manifiesto viejo intacto; re-correr `init` repara. No hay rollback complejo — esa es toda la estrategia de recovery.

### `andes list`

Una vista, dos fuentes (catálogo + manifiesto):

```
PERFIL           SKILL                ESTADO
andespath-core   git-conventions      ✓ instalada
andespath-core   code-review          ✗ no instalada
tri-fleet        golang               ⚠ desactualizada
```

Sin manifiesto (nunca corriste `init`) → lista solo el catálogo y sugiere `andes init`.

### `andes doctor`

Compara los TRES estados — manifiesto (declarado), disco (real), catálogo (fuente) — y clasifica:

| Hallazgo | Detección | Consejo |
|---|---|---|
| Skill falta en disco | en manifiesto, no en `~/.claude/skills/` | re-corré `andes init` |
| Modificada localmente | hash disco ≠ manifiesto | re-init pisa tus cambios — decidí |
| Desactualizada | hash manifiesto ≠ catálogo | re-corré `andes init` |
| Catálogo inaccesible | path no existe | corregí `--catalog` |
| Todo sano | — | ✓ |

`doctor` **jamás modifica nada**. Exit code ≠ 0 si hay problemas (usable en CI/scripts). La reparación es siempre re-`init`.

### Manejo de errores (general)

Mensajes en español, accionables (qué pasó + qué hacer). Nunca stack traces al usuario.

## Testing

TDD (test primero). Estrategia por capa:

- **Unit, table-driven** para lógica pura: parseo/validación de `catalog.json`, resolución perfiles→skills con dedup, hashing, motor de diff de `doctor`. Aquí vive el 80% de los tests.
- **Integración con `t.TempDir()`**: catálogo + destino en dirs temporales, flujo real de `init` (copia real, manifiesto real). Cubre filesystem, atomicidad e idempotencia.
- **Lo interactivo no se testea vía UI en el MVP**: los tests usan `--profiles --yes`. El prompt `huh` es capa fina sin lógica. `teatest` entra recién con la TUI.
- **Mocks mínimos**: `catalog.Source` es la única interface; los tests usan `LocalDir` con fixtures reales.

## Catálogo fixture (`testdata/catalog/`)

Doble función: tests + demo. Contenido REAL aunque corto (no lorem ipsum) — la demo termina abriendo Claude Code y mostrando que la skill instalada funciona.

```
testdata/catalog/
├── catalog.json                 # andespath-core: [git-conventions, code-review]
│                                # tri-fleet: [golang]  ← solo skills que existen en el fixture
└── skills/
    ├── git-conventions/SKILL.md # conventional commits, mínimo real
    ├── code-review/SKILL.md
    └── golang/SKILL.md
```

Es además el template de estructura que el encargado de skills llena con contenido real.

## Approaches consideradas y descartadas

- **Symlink farm** (skills symlinkeadas desde un clone del catálogo): updates atómicos, pero frágil — mover/borrar el catálogo mata todas las skills silenciosamente; el estado instalado deja de ser autocontenido.
- **Wrapper sobre plugins nativos de Claude Code** (`claude plugin marketplace add` + `install`): auto-update gratis, pero casa la herramienta 100% con Claude Code, la mecánica no-interactiva tiene costuras, y la TUI futura quedaría acoplada a un CLI ajeno.

## Roadmap post-MVP (no comprometido)

`update` real, `catalog.Source` tipo `GitRepo` (catálogo remoto), TUI Bubbletea, skills a nivel repo/proyecto, distribución del scaffold de la LLM wiki (`knowledge/` + agents.md) como template.
