# Contributing skills to the andespath catalog

The catalog lives in `catalog/`. A skill is a folder `catalog/skills/<id>/`
containing a `SKILL.md`. Profiles in `catalog/catalog.json` group skills for
who should install them. Profiles are organizing buckets, not ranks:

- `andespath-core` — company-wide standards everyone installs (git conventions,
  code review, microservice practices, …).
- team profiles (e.g. `tri-fleet`) — skills specific to one team's stack.

A skill can belong to several profiles. There is no promotion ladder: open a PR
that adds your skill to whichever profile(s) make sense — a team profile,
`andespath-core`, or both.

## Add a skill

1. Create `catalog/skills/<id>/SKILL.md` with frontmatter:

   ```markdown
   ---
   name: <id>
   description: One line — what it is and when the agent should use it.
   ---

   # <Title>

   The guidance itself.
   ```

2. Reference the skill in one or more profiles in `catalog/catalog.json`:

   ```json
   "andespath-core": { "description": "...", "skills": ["git-conventions", "<id>"] }
   ```

   One PR may add multiple skills.

3. Validate locally before pushing:

   ```bash
   go build -o andes ./cmd/andes
   ./andes validate --catalog catalog
   ```

4. Open a PR. The `ci` workflow (build + tests + `andes validate`) must pass —
   a PR that breaks the catalog cannot merge. Reviewers discuss the skill's
   content and fit.
