package cli_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// writeCatalog builds a temp catalog dir. profilesJSON is the "profiles" object
// body; skills maps id → SKILL.md content ("" = create dir without SKILL.md).
func writeCatalog(t *testing.T, profilesJSON string, skills map[string]string) string {
	t.Helper()
	root := t.TempDir()
	cat := `{"name":"test","profiles":` + profilesJSON + `}`
	if err := os.WriteFile(filepath.Join(root, "catalog.json"), []byte(cat), 0o644); err != nil {
		t.Fatal(err)
	}
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
	return root
}

const skillMD = "---\nname: x\ndescription: d\n---\n# X\n"

func TestValidateValid(t *testing.T) {
	home := t.TempDir()
	root := writeCatalog(t, `{"core":{"description":"d","skills":["a"]}}`, map[string]string{"a": skillMD})
	out, err := runAndes(t, home, "validate", "--catalog", root)
	if err != nil {
		t.Fatalf("validate valid catalog: %v\n%s", err, out)
	}
	if !strings.Contains(out, "catalog valid") || !strings.Contains(out, "1 profiles") || !strings.Contains(out, "1 skills") {
		t.Errorf("unexpected success output:\n%s", out)
	}
}

func TestValidateFailures(t *testing.T) {
	tests := []struct {
		name         string
		profilesJSON string
		skills       map[string]string
		wantSub      string
	}{
		{"missing skill", `{"core":{"description":"d","skills":["ghost"]}}`, map[string]string{}, "ghost"},
		{"empty profile", `{"empty":{"description":"d","skills":[]}}`, map[string]string{}, "has no skills"},
		{"dup skill", `{"core":{"description":"d","skills":["a","a"]}}`, map[string]string{"a": skillMD}, "more than once"},
		{"no frontmatter", `{"core":{"description":"d","skills":["a"]}}`, map[string]string{"a": "# nope\n"}, "frontmatter"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			home := t.TempDir()
			root := writeCatalog(t, tt.profilesJSON, tt.skills)
			out, err := runAndes(t, home, "validate", "--catalog", root)
			if err == nil {
				t.Fatalf("expected validation failure, got success:\n%s", out)
			}
			if !strings.Contains(err.Error()+out, tt.wantSub) {
				t.Errorf("error = %v\n%s\nwant substring %q", err, out, tt.wantSub)
			}
		})
	}
}

func TestValidateBrokenJSON(t *testing.T) {
	home := t.TempDir()
	root := t.TempDir()
	os.WriteFile(filepath.Join(root, "catalog.json"), []byte("{not json"), 0o644)
	if _, err := runAndes(t, home, "validate", "--catalog", root); err == nil {
		t.Error("broken JSON should fail validation")
	}
}

// chdir switches to dir and restores the previous cwd after the test.
// Used instead of t.Chdir (Go 1.24+) because the module targets Go 1.23.
// These tests do NOT call t.Parallel, so process-wide cwd mutation is safe.
func chdir(t *testing.T, dir string) {
	t.Helper()
	prev, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(prev) })
}

func TestValidateCwdDetection(t *testing.T) {
	home := t.TempDir()
	root := writeCatalog(t, `{"core":{"description":"d","skills":["a"]}}`, map[string]string{"a": skillMD})
	sub := filepath.Join(root, "skills", "a")
	chdir(t, sub) // run from a nested dir; validate must walk up to catalog.json
	out, err := runAndes(t, home, "validate")
	if err != nil {
		t.Fatalf("cwd detection: %v\n%s", err, out)
	}
	if !strings.Contains(out, "catalog valid") {
		t.Errorf("cwd detection did not validate:\n%s", out)
	}
}

func TestValidateNoCatalogAnywhere(t *testing.T) {
	home := t.TempDir()
	chdir(t, t.TempDir()) // empty dir; no catalog.json up the tree within temp
	_, err := runAndes(t, home, "validate")
	if err == nil {
		t.Error("validate with no catalog should fail with an actionable error")
	}
}

func TestProductionCatalogIsValid(t *testing.T) {
	home := t.TempDir()
	abs, err := filepath.Abs("../../catalog")
	if err != nil {
		t.Fatal(err)
	}
	out, err := runAndes(t, home, "validate", "--catalog", abs)
	if err != nil {
		t.Fatalf("the production catalog/ must pass validate: %v\n%s", err, out)
	}
}
