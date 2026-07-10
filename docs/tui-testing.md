# TUI-first testing guide

This project is meant to be used primarily through the `andes` TUI. The CLI flags
exist for automation, tests, and CI, but humans should normally validate behavior
through the interactive flow.

Use a temporary `HOME` when testing manually so you do not touch your real
`~/.claude/skills` or `~/.claude/andes.json`:

```bash
export ANDES_TEST_HOME="$(mktemp -d)"
```

When running from a source checkout during development, use:

```bash
HOME="$ANDES_TEST_HOME" go run ./cmd/andes
```

When testing an installed binary, use:

```bash
HOME="$ANDES_TEST_HOME" andes
```

## First install through the TUI

1. Start the TUI:

   ```bash
   HOME="$ANDES_TEST_HOME" go run ./cmd/andes
   ```

2. Choose `install`.
3. If prompted for a catalog path, enter:

   ```text
   ./catalog
   ```

4. Select the profiles you want, for example:
   - `andespath-core`
   - `tri-fleet`

5. Continue to the Review screen.
6. Verify that the Review screen lists the individual skills, not only profile
   counts. It should show action, skill id, and profile.
7. Press Enter to apply.
8. Confirm installed files exist under the temporary home:

   ```bash
   find "$ANDES_TEST_HOME/.claude/skills" -maxdepth 2 -type f | sort
   ```

## Deselecting a profile removes Andes-managed skills

Andes only removes skills it previously managed. It must not scan and delete
personal skills that happen to live in `~/.claude/skills`.

1. Install both profiles through the TUI:
   - `andespath-core`
   - `tri-fleet`

2. Verify a `tri-fleet` skill exists, for example:

   ```bash
   test -e "$ANDES_TEST_HOME/.claude/skills/golang/SKILL.md" && echo "golang installed"
   ```

3. Start the TUI again:

   ```bash
   HOME="$ANDES_TEST_HOME" go run ./cmd/andes
   ```

4. Choose `install`.
5. Leave only `andespath-core` selected.
6. Continue to the Review screen.
7. Verify the Review screen shows a remove action similar to:

   ```text
   remove    golang  (tri-fleet)
   ```

8. Verify the summary counts removals separately from unchanged skills:

   ```text
   0 to install, 0 to update, 1 to remove, N unchanged
   ```

9. Apply the plan.
10. Verify the deselected managed skill was removed:

    ```bash
    test ! -e "$ANDES_TEST_HOME/.claude/skills/golang" && echo "golang removed"
    ```

## Personal skills must not be touched

This verifies the ownership boundary: Andes removes only skills listed in the
previous manifest, not every folder under `~/.claude/skills`.

1. Create a personal skill in the temporary home:

   ```bash
   mkdir -p "$ANDES_TEST_HOME/.claude/skills/my-personal-skill"
   printf '# mine\n' > "$ANDES_TEST_HOME/.claude/skills/my-personal-skill/SKILL.md"
   ```

2. Use the TUI to install and later deselect catalog profiles.
3. Verify the personal skill remains:

   ```bash
   test -e "$ANDES_TEST_HOME/.claude/skills/my-personal-skill/SKILL.md" && echo "personal skill preserved"
   ```

## Permission drift is detected and repaired

The directory hash includes file permissions. If a managed executable loses its
execute bit, Andes should treat it as drift and repair it on reinstall.

The stock catalog may not always contain an executable skill file. To test this
manually, use a temporary catalog fixture with a skill that contains `run.sh`.

1. Create a temporary catalog:

   ```bash
   export ANDES_EXEC_CAT="$(mktemp -d)"
   mkdir -p "$ANDES_EXEC_CAT/skills/execskill"
   cat > "$ANDES_EXEC_CAT/catalog.json" <<'JSON'
   {
     "name": "exec-test",
     "profiles": {
       "exec": {"description": "Executable skill", "skills": ["execskill"]}
     }
   }
   JSON
   printf '# execskill\n' > "$ANDES_EXEC_CAT/skills/execskill/SKILL.md"
   printf '#!/bin/sh\n' > "$ANDES_EXEC_CAT/skills/execskill/run.sh"
   chmod 755 "$ANDES_EXEC_CAT/skills/execskill/run.sh"
   ```

2. Start the TUI with the temporary home:

   ```bash
   HOME="$ANDES_TEST_HOME" go run ./cmd/andes
   ```

3. Choose `install` and use the temporary catalog path from `ANDES_EXEC_CAT`.
4. Select the `exec` profile and apply.
5. Break the installed executable permission:

   ```bash
   chmod 644 "$ANDES_TEST_HOME/.claude/skills/execskill/run.sh"
   ```

6. Run `doctor` from the TUI. It should report the skill as modified.
7. Run `install` again from the TUI with the same profile selected.
8. Verify the execute bit was repaired:

   ```bash
   test -x "$ANDES_TEST_HOME/.claude/skills/execskill/run.sh" && echo "exec bit repaired"
   ```

## Canceling a plan must not apply changes

1. Start with a clean temporary home:

   ```bash
   export ANDES_TEST_HOME="$(mktemp -d)"
   ```

2. Start the TUI and begin an install.
3. Continue until the Review screen.
4. Press `esc` to go back or decline from the command-line confirmation flow.
5. Verify no manifest or skill was written:

   ```bash
   test ! -e "$ANDES_TEST_HOME/.claude/andes.json" && echo "no manifest written"
   test ! -d "$ANDES_TEST_HOME/.claude/skills" && echo "no skills installed"
   ```

## Automated verification

CI runs:

```bash
go test ./...
```

Inside Codex's sandbox, use a writable Go cache:

```bash
GOCACHE=/private/tmp/andes-go-build-cache go test ./...
```

The CI workflow also validates the catalog after tests:

```bash
go build -o andes ./cmd/andes
./andes validate --catalog catalog
```

Do not consider a change ready if `go test ./...` is red.
