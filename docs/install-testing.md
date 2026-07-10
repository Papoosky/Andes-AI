# Install script testing guide

`install.sh` is the first touchpoint for a new developer. If it fails, the TUI
and the rest of the CLI do not matter. Keep this script boring, predictable, and
covered by tests.

The installer supports two real-world entry points:

1. **No-clone install through `gh api | bash`** for the private repository.
2. **Clone-and-run install** with `./install.sh`, falling back to a source build
   only when release download is unavailable and Go is installed.

## Automated tests

The repository contains Go tests for `install.sh` in `install_script_test.go`.
They run as part of the normal test suite:

```bash
go test ./...
```

Inside Codex's sandbox, use a writable Go cache:

```bash
GOCACHE=/private/tmp/andes-go-build-cache go test ./...
```

To run only the install script tests:

```bash
GOCACHE=/private/tmp/andes-go-build-cache go test . -run TestInstallScript -count=1 -v
```

These tests intentionally avoid network and GitHub. They use a fake `gh` binary
and a temporary `HOME`, then assert that:

- `install.sh` is valid Bash (`bash -n`).
- `--help` prints usage.
- unknown flags fail.
- latest release download calls `gh release download` with the expected repo,
  asset pattern, output path, and `--clobber`.
- pinned release download passes the requested version tag to `gh`.
- the installed binary is written to `$HOME/.local/bin/andes` and marked
  executable.

The tests do **not** exercise the source-build fallback because that path runs
`go build`, and this project rule says not to build after changes. Validate that
path manually only when specifically working on installer distribution.

## Manual no-clone test

Use this when validating the real private-repo install path with GitHub auth:

```bash
export ANDES_TEST_HOME="$(mktemp -d)"

HOME="$ANDES_TEST_HOME" \
  gh api repos/Papoosky/Andes-AI/contents/install.sh \
    -H "Accept: application/vnd.github.raw" \
  | HOME="$ANDES_TEST_HOME" bash
```

Then verify:

```bash
test -x "$ANDES_TEST_HOME/.local/bin/andes" && echo "andes installed"
HOME="$ANDES_TEST_HOME" "$ANDES_TEST_HOME/.local/bin/andes" --help
```

Pin a release version with:

```bash
HOME="$ANDES_TEST_HOME" \
  gh api repos/Papoosky/Andes-AI/contents/install.sh \
    -H "Accept: application/vnd.github.raw" \
  | HOME="$ANDES_TEST_HOME" bash -s -- --version vX.Y.Z
```

## Manual clone-and-run test

Use a temporary home so the test does not touch your real install:

```bash
export ANDES_TEST_HOME="$(mktemp -d)"
HOME="$ANDES_TEST_HOME" ./install.sh
```

Then verify:

```bash
test -x "$ANDES_TEST_HOME/.local/bin/andes" && echo "andes installed"
HOME="$ANDES_TEST_HOME" "$ANDES_TEST_HOME/.local/bin/andes" --help
```

## Source-build fallback

The source-build fallback exists for cloned checkouts when `gh release download`
is unavailable. It bakes a catalog URL into the binary using this precedence:

1. `ANDES_CATALOG_URL`, when set.
2. The URL derived from `REPO`, probing SSH first and HTTPS second.
3. HTTPS fallback if neither probe succeeds, so the CLI can surface the auth
   error later.

Only test this manually when intentionally changing fallback behavior:

```bash
export ANDES_TEST_HOME="$(mktemp -d)"
export ANDES_CATALOG_URL="git@github.com:Papoosky/Andes-AI.git"
PATH="/usr/bin:/bin" HOME="$ANDES_TEST_HOME" ./install.sh
```

Do not use this fallback as the normal validation path for unrelated changes.
It compiles the binary and is slower than the fake-`gh` automated tests.

## Expected release asset names

The script maps platform to release assets using:

```text
andes-${os}-${arch}
```

Examples:

```text
andes-darwin-arm64
andes-darwin-amd64
andes-linux-arm64
andes-linux-amd64
```

If release asset naming changes, update both `install.sh` and
`install_script_test.go` together.
