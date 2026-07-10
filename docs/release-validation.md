# Release validation guide

Use this checklist when validating a real GitHub Release, not just the local
unit tests. Always run it with a temporary `HOME`; never test installers against
your real `~/.claude`, `~/.andes`, or `~/.local/bin` state.

## What this validates

A real release validation proves the full distribution path works:

1. `release-please` created a GitHub Release.
2. The build job uploaded the expected OS/architecture binaries.
3. `install.sh` can download the right asset through `gh release download`.
4. The downloaded binary has a baked catalog URL.
5. First-run install can clone the catalog mirror and write a manifest.
6. `doctor` can read the manifest, disk state, and catalog mirror.

This is intentionally stronger than the automated `install.sh` tests, which use
a fake `gh` binary and avoid network access.

## Preconditions

- You have the GitHub CLI installed and authenticated:

  ```bash
  gh auth status
  ```

- You can access the private repository and its releases.
- You can clone the catalog repo over either SSH or HTTPS. Release binaries bake
  the repo-derived catalog URL, and the CLI probes SSH then HTTPS when needed.

## Check release metadata and assets

```bash
gh release view --repo Papoosky/Andes-AI \
  --json tagName,assets,url,createdAt,publishedAt,isDraft,isPrerelease,targetCommitish
```

Expected assets:

```text
andes-darwin-amd64
andes-darwin-arm64
andes-linux-amd64
andes-linux-arm64
```

Each asset should be uploaded, non-empty, and attached to the expected release
tag. A missing asset means `install.sh` will fail on that platform even if the
release exists.

## Check release-please status

After normal commits land on `main`, release-please usually opens or updates a
Release PR instead of publishing immediately:

```bash
gh run list --repo Papoosky/Andes-AI --workflow release-please.yml --limit 5

gh pr list --repo Papoosky/Andes-AI \
  --state open \
  --search 'release-please' \
  --json number,title,url,headRefName,baseRefName
```

A successful release-please run with no new release is normal when it only
updates the Release PR. The binary build job runs only when merging the Release
PR causes `release_created=true`.

## Validate install from the latest release

Run from a checkout of this repo so you use the current `install.sh`, but isolate
all install state in a temporary home:

```bash
export ANDES_TEST_HOME="$(mktemp -d)"

HOME="$ANDES_TEST_HOME" bash install.sh
```

Verify the binary exists and is executable:

```bash
test -x "$ANDES_TEST_HOME/.local/bin/andes"
HOME="$ANDES_TEST_HOME" "$ANDES_TEST_HOME/.local/bin/andes" --help
```

## Verify the baked catalog URL

Use `strings` as a lightweight smoke test that the release binary was built with
the intended `defaultCatalogURL` linker flag:

```bash
strings "$ANDES_TEST_HOME/.local/bin/andes" \
  | grep -E 'github.com[:/].*Andes-AI|Papoosky/Andes-AI'
```

You should see a repository URL such as:

```text
git@github.com:Papoosky/Andes-AI.git
```

If no repo URL appears, the release binary may not have a baked catalog default,
and first-run install will require `--catalog` instead of working through the
TUI/default flow.

## Validate first-run install from the baked default

Use a non-interactive profile install to prove the binary can clone the managed
catalog mirror and write a manifest:

```bash
HOME="$ANDES_TEST_HOME" \
  "$ANDES_TEST_HOME/.local/bin/andes" install --profiles tri-fleet --yes
```

Then inspect the manifest:

```bash
cat "$ANDES_TEST_HOME/.claude/andes.json"
```

Expected properties:

- `catalog.type` is `git`.
- `catalog.url` is the baked repository URL variant that worked for this user.
- `catalog.ref` is the current catalog mirror HEAD.
- The requested profile's skills are present under `installed`.

Important: the release binary version and `catalog.ref` do not have to point to
the same commit. Andes uses a moving catalog mirror; `catalog.ref` records the
catalog HEAD that was applied during install/update, not the binary release tag.

## Validate doctor

```bash
HOME="$ANDES_TEST_HOME" "$ANDES_TEST_HOME/.local/bin/andes" doctor
```

Expected result:

```text
All healthy ✓
```

A catalog ref drift warning is not automatically a failure. It means the local
catalog mirror HEAD differs from the manifest's last applied `catalog.ref`, so
you should decide whether to run `andes update` or reinstall.

## Validate a pinned release version

Use this when checking an older tag or a newly published tag explicitly:

```bash
export ANDES_TEST_HOME="$(mktemp -d)"
HOME="$ANDES_TEST_HOME" bash install.sh --version vX.Y.Z
```

Then repeat the baked URL, first-run install, manifest, and doctor checks above.

## Cleanup

The temporary home can be deleted after validation:

```bash
rm -rf "$ANDES_TEST_HOME"
```

Only remove the temporary directory you created for the validation run.
