# andes-ai v2 — Git Catalog Design Doc

**Fecha:** 2026-07-08
**Estado:** Aprobado (diseño validado sección por sección)
**Alcance:** Parte A de v2 + install story (Parte D esencial). Partes B (tiers/promoción) y C (`andes try`) tienen sus propios ciclos futuros.

## Contexto

El MVP funciona con catálogo local (`--catalog <path>`). v2 lo convierte en herramienta desplegable: el catálogo vive en el repo git de la empresa (GitHub privado), andes lo gestiona solo, detecta desactualización y actualiza con una tecla. Flujo objetivo del dev nuevo: `gh repo clone` + `./install.sh` → `andes` → TUI funcionando con el baseline de andespath, **cero preguntas**.

## Decisiones tomadas (con el usuario)

1. **Shell out a git** (no go-git, no API de GitHub): la auth es la que cada dev ya tiene para su trabajo diario. Requisito: git instalado y autenticado contra el GitHub de la empresa.
2. **Clone gestionado**: andes mantiene SU propio mirror del catálogo en `~/.andes/catalog`. No existe dependencia de ningún clone del dev (la instalación no requiere conservar el clone).
3. **Update UX**: la TUI avisa con banner ("⚠ catalog updated — press u to update") y el dev decide. Nada de auto-update silencioso.
4. **Versionado**: hashes sha256 por skill + SHA de commit del catálogo. **Sin bumps manuales de versión** (se olvidan y la detección miente). "Qué cambió" = `git log -- catalog/skills/<id>/`.
5. **Default horneado**: la URL del repo de la empresa va compilada en el binario (`-ldflags -X`). Resolución de catálogo: flag `--catalog` (path o URL) → manifiesto → default horneado → recién ahí prompt. **Nota de transición**: el repo aún no está en el GitHub de la empresa; hasta entonces los builds no hornean URL (variable vacía) y el fallback es el prompt actual — el día que se suba, se agrega el ldflag al install.sh/CI y el prompt desaparece para los devs. Nada del diseño depende de la fecha de subida.
6. **El catálogo de producción vive en ESTE repo**: `catalog/` en la raíz (se mueve `testdata/catalog` → `catalog/`; los tests apuntan ahí).
7. **Approach elegido**: GitRepo como wrapper fino que delega en LocalDir (descartados: source sobre git plumbing/bare, y API de GitHub sin git).

## Sección 1 — Arquitectura: GitRepo source

```go
// internal/catalog/gitrepo.go — única pieza que conoce git
type GitRepo struct {
    URL string // repo de la empresa
    Dir string // mirror gestionado: ~/.andes/catalog
}
```

`GitRepo` implementa `Source` delegando en `LocalDir{Root: filepath.Join(Dir, "catalog")}` una vez asegurado el mirror. Installer, doctor, list y hashdir NO CAMBIAN — ven un path local.

Operaciones git (exec.Command, tres):

| Operación | Comando | Cuándo |
|---|---|---|
| `Ensure()` | `git clone <url> <dir>` si falta; si existe, `git -C <dir> reset --hard` (dir privado de andes, siempre limpio) | antes de cualquier lectura |
| `RemoteHead()` | `git ls-remote <url> HEAD` → SHA remoto (~200ms, no baja contenido) | check de outdated (TUI al abrir) |
| `Sync()` | `git -C <dir> fetch origin` + `reset --hard origin/HEAD` → SHA nuevo | `andes update` / tecla `u` |

Mirror corrupto (no es repo git válido) → `Ensure()` lo borra y re-clona. Autocurable.

`LocalHead()` (SHA del mirror) via `git -C <dir> rev-parse HEAD` — es lo que se guarda como `ref` en el manifiesto.

## Sección 2 — Manifiesto v2 y detección

Manifiesto con catálogo git (el campo `catalog.type` del MVP paga hoy):

```json
{
  "version": 1,
  "catalog": {
    "type": "git",
    "url": "git@github.com:andespath/andes-ai.git",
    "ref": "a1b2c3d..."
  },
  "profiles": ["andespath-core", "tri-fleet"],
  "installed": { "golang": { "hash": "sha256:...", "profile": "tri-fleet" } }
}
```

- `type: "local"` sigue funcionando sin cambios (tests, desarrollo, override). Manifiestos v1 existentes siguen válidos.
- `CatalogRef` gana campos `URL` y `Ref` (omitempty para local).

**Detección en dos niveles (separados a propósito, robustez offline):**

1. **Mirror vs remoto**: `RemoteHead() != manifest.Catalog.Ref` → hay update. Corre al abrir la TUI, async (tea.Cmd, no bloquea el render), timeout 2s. Sin red → se omite en silencio, footer discreto "offline".
2. **Skills vs mirror**: el `doctor` existente (hashes vs clone local). Cero red, cero cambios.

Flujo: banner "⚠ catalog updated — press u to update" → `u` → `Sync()` + re-init idempotente (los hashes deciden qué se recopia) → manifiesto guarda el `ref` nuevo.

No se construye: registry, protocolo de versiones, estado extra. Git SHA = versión global; hash por skill = diff granular.

## Sección 3 — Flujos de comandos y errores

### `andes init` (dev nuevo, cero preguntas)
1. Sin `--catalog` ni manifiesto → default horneado (URL git).
2. `Ensure()` con mensaje "Fetching the andespath catalog…" (el primer clone tarda segundos, que se sepa).
3. Init normal: perfiles (interactivo o flags) → plan → apply → manifiesto `type: "git"` + `ref = LocalHead()`.
4. `--catalog` acepta path local (comportamiento actual) o URL git (se detecta por prefijo `git@`/`https://`/sufijo `.git`).

### `andes update` (comando nuevo; la tecla `u` de la TUI invoca lo mismo)
1. Manifiesto requerido (sin manifiesto → "you haven't run `andes init` yet"). Solo para `type: "git"` (local → "nothing to update: local catalog").
2. `Sync()`. SHA sin cambios → "Already up to date", exit 0.
3. Re-ejecuta lógica de init con perfiles del manifiesto → plan visible (`update golang`, resto `unchanged`) → apply → `ref` nuevo. `--yes` disponible; desde la TUI corre con confirmación implícita de la tecla.
4. En TUI: output al viewport existente; banner se limpia al volver al menú.

### `andes list` / `doctor`
Sin cambios de lógica: resuelven Source desde el manifiesto (git → `Ensure()` + LocalDir del mirror; local → como hoy).

### Errores (accionables, en inglés, nunca stack traces)

| Falla | Comportamiento |
|---|---|
| git no instalado | init/update: "git is required — install it and retry". Check de TUI omitido |
| Sin auth (clone/fetch falla) | "could not reach the catalog repo — check your GitHub access (SSH key or token)" + resumen del stderr de git |
| Sin red | TUI: check omitido + "offline" en footer. `update` explícito: error accionable |
| Mirror corrupto | re-clone automático |
| ls-remote lento | timeout 2s, banner best-effort — JAMÁS degrada el arranque |

## Sección 4 — Testing (sin red)

Git funciona contra repos locales: fixture = repo git real creado en `t.TempDir()` (`git init` + copiar catálogo + commit), `GitRepo{URL: <ese path>}`.

- **Unit `gitrepo_test.go`**: Ensure clona; Ensure repara mirror sucio/corrupto; RemoteHead devuelve SHA; Sync trae commit nuevo; LocalHead correcto.
- **Integración CLI**: init con URL git local → skills + manifiesto con ref; update tras commit nuevo → plan muestra solo la skill tocada como `update`; update sin cambios → "Already up to date"; update con catálogo local → mensaje "nothing to update".
- **TUI (Update directo, patrón existente)**: msg outdated → banner en View; tecla `u` → dispara update cmd; msg offline → footer.
- **E2E**: ciclo completo — init desde fixture git → commit nuevo en fixture → RemoteHead detecta → update → doctor sano → manifest.ref == nuevo SHA.
- Helper de test para crear el fixture git (shell out a git en TempDir) — requiere git en el entorno de CI (aceptado: es requisito del producto).

## Sección 5 — Instalación: `install.sh`

README (dos líneas):

```bash
gh repo clone andespath/andes-ai && ./andes-ai/install.sh
```

`gh` y no `curl | bash`: repo privado — gh ya está autenticado; curl requeriría manejar tokens.

`install.sh` (idempotente):
1. **Consigue el binario**: (a) si hay release publicada → `gh release download` del binario para OS/arch — no requiere Go (clave: hay devs frontend sin Go); (b) fallback con `go` instalado → `go build` del clone con `-ldflags` (URL default del catálogo + versión); (c) ninguno → "install Go or wait for the first release".
2. Instala en `~/.local/bin/andes`; verifica `$PATH` (si falta, imprime la línea exacta para el shell rc).
3. Cierra: "Done — run 'andes' to get started." El clone es descartable.

**CI de release**: workflow que en cada tag compila multi-OS/arch (goreleaser o script go build) y publica release en GitHub. La URL del catálogo y la versión se hornean en el build.

## Fuera de alcance (explícito)

- Parte B: estructura de tiers (personal → equipo → empresa) y flujo de promoción por PRs — diseño de repo/proceso, ciclo propio.
- Parte C: `andes try <branch|pr>` — instalar skills de un PR/branch para probarlas. Se apoya en el mirror de esta parte (git fetch de refs).
- Auto-update del BINARIO (re-correr installer; aviso de "new version" es v3).
- `andes remove`, migración de wiki/knowledge, MCPs.

## Estructura de archivos (resumen)

```
catalog/                          ← movido desde testdata/catalog (producción + tests)
internal/catalog/gitrepo.go       ← nuevo: GitRepo source
internal/catalog/gitrepo_test.go
internal/cli/update.go            ← nuevo comando
internal/cli/resolve.go           ← resolución catálogo: flag → manifiesto → default horneado
internal/tui/                     ← check async al abrir + banner + tecla u + footer offline
internal/manifest/                ← CatalogRef gana URL/Ref
install.sh                        ← raíz
.github/workflows/release.yml    ← CI de releases
```
