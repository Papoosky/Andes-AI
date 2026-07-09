package catalog_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/andespath/andes-ai/internal/catalog"
)

// lintFixture builds a temp catalog dir with the given profiles and skills.
// skills maps skill-id → SKILL.md content (empty string means "omit the file").
func lintFixture(t *testing.T, profiles map[string]catalog.Profile, skills map[string]string) catalog.LocalDir {
	t.Helper()
	root := t.TempDir()
	for id, content := range skills {
		dir := filepath.Join(root, "skills", id)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		if content != "" {
			if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte(content), 0o644); err != nil {
				t.Fatal(err)
			}
		}
	}
	return catalog.LocalDir{Root: root}
}

const goodMD = "---\nname: x\ndescription: does a thing\n---\n# X\n"

func TestLintClean(t *testing.T) {
	src := lintFixture(t,
		map[string]catalog.Profile{"core": {Description: "d", Skills: []string{"a", "b"}}},
		map[string]string{"a": goodMD, "b": goodMD},
	)
	c := &catalog.Catalog{Name: "x", Profiles: map[string]catalog.Profile{
		"core": {Description: "d", Skills: []string{"a", "b"}},
	}}
	if got := catalog.Lint(src, c); len(got) != 0 {
		t.Errorf("Lint clean catalog = %v, want none", got)
	}
}

// TestLintSkipsUnreadableSkillMd covers the spec guarantee that a skill whose
// SKILL.md is unreadable (here: absent) is silently skipped rather than
// reported — existence is Load's responsibility, not Lint's.
func TestLintSkipsUnreadableSkillMd(t *testing.T) {
	src := lintFixture(t,
		map[string]catalog.Profile{"core": {Description: "d", Skills: []string{"a"}}},
		map[string]string{"a": ""}, // dir created, SKILL.md omitted
	)
	c := &catalog.Catalog{Name: "x", Profiles: map[string]catalog.Profile{
		"core": {Description: "d", Skills: []string{"a"}},
	}}
	if got := catalog.Lint(src, c); len(got) != 0 {
		t.Errorf("Lint with unreadable SKILL.md = %v, want none (skip, not report)", got)
	}
}

func TestLintProblems(t *testing.T) {
	tests := []struct {
		name     string
		profiles map[string]catalog.Profile
		skills   map[string]string
		wantSub  string
	}{
		{
			name:     "empty profile",
			profiles: map[string]catalog.Profile{"empty": {Description: "d", Skills: []string{}}},
			skills:   map[string]string{},
			wantSub:  "has no skills",
		},
		{
			name:     "duplicate skill in profile",
			profiles: map[string]catalog.Profile{"core": {Description: "d", Skills: []string{"a", "a"}}},
			skills:   map[string]string{"a": goodMD},
			wantSub:  "more than once",
		},
		{
			name:     "missing frontmatter",
			profiles: map[string]catalog.Profile{"core": {Description: "d", Skills: []string{"a"}}},
			skills:   map[string]string{"a": "# no frontmatter here\n"},
			wantSub:  "frontmatter",
		},
		{
			name:     "frontmatter present but empty description",
			profiles: map[string]catalog.Profile{"core": {Description: "d", Skills: []string{"a"}}},
			skills:   map[string]string{"a": "---\nname: x\ndescription:\n---\n"},
			wantSub:  "frontmatter",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			src := lintFixture(t, tt.profiles, tt.skills)
			c := &catalog.Catalog{Name: "x", Profiles: tt.profiles}
			got := catalog.Lint(src, c)
			joined := strings.Join(got, "\n")
			if !strings.Contains(joined, tt.wantSub) {
				t.Errorf("Lint = %q, want a problem containing %q", joined, tt.wantSub)
			}
		})
	}
}

func TestLintSorted(t *testing.T) {
	src := lintFixture(t,
		map[string]catalog.Profile{"z": {Description: "d", Skills: []string{}}, "a": {Description: "d", Skills: []string{}}},
		map[string]string{},
	)
	c := &catalog.Catalog{Name: "x", Profiles: src2profiles("z", "a")}
	got := catalog.Lint(src, c)
	if len(got) != 2 || got[0] > got[1] {
		t.Errorf("Lint problems not sorted: %v", got)
	}
}

func src2profiles(names ...string) map[string]catalog.Profile {
	m := map[string]catalog.Profile{}
	for _, n := range names {
		m[n] = catalog.Profile{Description: "d", Skills: []string{}}
	}
	return m
}
