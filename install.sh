#!/usr/bin/env bash
# andes installer. Works two ways:
#   - piped:   curl -fsSL <raw-url>/install.sh | bash [-s -- --version vX.Y.Z]
#   - cloned:  ./install.sh
# It downloads a release binary (via gh for private repos, or curl for public),
# and falls back to building from source only when run inside a checkout.
set -euo pipefail

REPO="Papoosky/Andes-AI"
BIN_DIR="${HOME}/.local/bin"
VERSION=""  # empty = latest

# ── Args ─────────────────────────────────────────────────────────────────────
while [ $# -gt 0 ]; do
  case "$1" in
    --version) VERSION="${2:-}"; shift 2 ;;
    --version=*) VERSION="${1#*=}"; shift ;;
    -h | --help)
      echo "usage: install.sh [--version vX.Y.Z]"
      exit 0
      ;;
    *) echo "unknown option: $1" >&2; exit 1 ;;
  esac
done

OS="$(uname -s | tr '[:upper:]' '[:lower:]')"
ARCH="$(uname -m)"
case "$ARCH" in
  x86_64) ARCH="amd64" ;;
  aarch64 | arm64) ARCH="arm64" ;;
esac
ASSET="andes-${OS}-${ARCH}"

mkdir -p "$BIN_DIR"

# Detect whether we're running inside a checkout (so build-from-source is
# possible). When piped via curl, BASH_SOURCE is empty and this stays 0.
SRC_DIR=""
SCRIPT_SRC="${BASH_SOURCE[0]:-}"
if [ -n "$SCRIPT_SRC" ] && [ -f "$SCRIPT_SRC" ]; then
  maybe="$(cd "$(dirname "$SCRIPT_SRC")" && pwd)"
  [ -f "$maybe/go.mod" ] && SRC_DIR="$maybe"
fi

# derive_catalog_url resolves the catalog git URL from REPO by probing which
# protocol THIS dev can reach: SSH first, then HTTPS. ANDES_CATALOG_URL wins.
# Only used by build-from-source (release binaries carry a URL baked by CI).
derive_catalog_url() {
  local ssh_url="git@github.com:${REPO}.git"
  local https_url="https://github.com/${REPO}.git"
  if git ls-remote "$ssh_url" >/dev/null 2>&1; then
    echo "$ssh_url"
  elif git ls-remote "$https_url" >/dev/null 2>&1; then
    echo "$https_url"
  else
    echo "$https_url"  # fallback; andes surfaces the auth error at install time
  fi
}

# ── Download strategies ──────────────────────────────────────────────────────

install_via_gh() {
  command -v gh >/dev/null 2>&1 || return 1
  if [ -n "$VERSION" ]; then
    gh release download "$VERSION" --repo "$REPO" --pattern "$ASSET" \
      --output "$BIN_DIR/andes" --clobber 2>/dev/null || return 1
  else
    gh release download --repo "$REPO" --pattern "$ASSET" \
      --output "$BIN_DIR/andes" --clobber 2>/dev/null || return 1
  fi
  chmod +x "$BIN_DIR/andes"
  echo "installed release binary (${VERSION:-latest}) via gh → $BIN_DIR/andes"
}

# Public-repo path. For private repos use gh (handles auth); a raw curl of a
# private release asset needs a signed redirect this keeps simple on purpose.
install_via_curl() {
  command -v curl >/dev/null 2>&1 || return 1
  local auth=()
  [ -n "${GH_TOKEN:-}" ] && auth=(-H "Authorization: Bearer ${GH_TOKEN}")
  local tag="$VERSION"
  if [ -z "$tag" ]; then
    tag="$(curl -fsSL "${auth[@]}" "https://api.github.com/repos/${REPO}/releases/latest" \
      | grep '"tag_name"' | head -1 | sed -E 's/.*"tag_name":[[:space:]]*"([^"]+)".*/\1/')" || return 1
    [ -n "$tag" ] || return 1
  fi
  curl -fsSL "${auth[@]}" -o "$BIN_DIR/andes" \
    "https://github.com/${REPO}/releases/download/${tag}/${ASSET}" || return 1
  chmod +x "$BIN_DIR/andes"
  echo "installed release binary (${tag}) via curl → $BIN_DIR/andes"
}

install_from_source() {
  [ -n "$SRC_DIR" ] || return 1
  command -v go >/dev/null 2>&1 || return 1
  local catalog_url ldflags=""
  catalog_url="${ANDES_CATALOG_URL:-$(derive_catalog_url)}"
  if [ -n "$catalog_url" ]; then
    ldflags="-X github.com/andespath/andes-ai/internal/cli.defaultCatalogURL=${catalog_url}"
    echo "baking catalog URL → ${catalog_url}"
  fi
  (cd "$SRC_DIR" && go build -ldflags "$ldflags" -o "$BIN_DIR/andes" ./cmd/andes)
  echo "built from source → $BIN_DIR/andes"
}

# ── Run ──────────────────────────────────────────────────────────────────────
if ! install_via_gh && ! install_via_curl && ! install_from_source; then
  echo "error: could not install andes." >&2
  echo "  - private repo? install the gh CLI and authenticate, then re-run." >&2
  echo "  - public repo?  ensure curl can reach the release, or pass --version." >&2
  echo "  - from a clone? install Go to build from source." >&2
  exit 1
fi

case ":$PATH:" in
  *":$BIN_DIR:"*) ;;
  *)
    echo ""
    echo "⚠ $BIN_DIR is not in your PATH. Add this to your shell rc:"
    echo "    export PATH=\"\$PATH:$BIN_DIR\""
    ;;
esac

echo ""
echo "Done — run 'andes' to get started."
