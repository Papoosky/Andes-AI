#!/usr/bin/env bash
# andes installer — downloads a release binary via gh, or builds from source.
set -euo pipefail

REPO="Papoosky/Andes-AI"
BIN_DIR="${HOME}/.local/bin"
# derive_catalog_url resolves the catalog git URL from REPO by probing which
# protocol THIS dev can actually reach: SSH first (stable for private repos —
# no token expiry), then HTTPS. ANDES_CATALOG_URL overrides everything. This
# runs on the dev's machine, so each dev bakes the URL that matches their auth.
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

OS="$(uname -s | tr '[:upper:]' '[:lower:]')"
ARCH="$(uname -m)"
case "$ARCH" in
  x86_64) ARCH="amd64" ;;
  aarch64 | arm64) ARCH="arm64" ;;
esac

mkdir -p "$BIN_DIR"

install_from_release() {
  command -v gh >/dev/null 2>&1 || return 1
  gh release download --repo "$REPO" --pattern "andes-${OS}-${ARCH}" \
    --output "$BIN_DIR/andes" --clobber 2>/dev/null || return 1
  chmod +x "$BIN_DIR/andes"
  echo "installed release binary → $BIN_DIR/andes"
}

install_from_source() {
  command -v go >/dev/null 2>&1 || return 1
  local src_dir
  src_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
  local catalog_url
  catalog_url="${ANDES_CATALOG_URL:-$(derive_catalog_url)}"
  local ldflags=""
  if [ -n "$catalog_url" ]; then
    ldflags="-X github.com/andespath/andes-ai/internal/cli.defaultCatalogURL=${catalog_url}"
    echo "baking catalog URL → ${catalog_url}"
  fi
  (cd "$src_dir" && go build -ldflags "$ldflags" -o "$BIN_DIR/andes" ./cmd/andes)
  echo "built from source → $BIN_DIR/andes"
}

if ! install_from_release; then
  echo "no release available (or gh missing) — trying to build from source…"
  if ! install_from_source; then
    echo "error: need either the gh CLI (for release download) or Go (to build)." >&2
    echo "install one of them and re-run ./install.sh" >&2
    exit 1
  fi
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
