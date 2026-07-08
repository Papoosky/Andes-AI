package catalog_test

import (
	"strings"
	"testing"

	"github.com/andespath/andes-ai/internal/catalog"
)

func twoProfileCatalog() *catalog.Catalog {
	return &catalog.Catalog{
		Name: "andespath",
		Profiles: map[string]catalog.Profile{
			"core": {Description: "base", Skills: []string{"git-conventions", "code-review"}},
			"tri":  {Description: "tri", Skills: []string{"golang", "git-conventions"}},
		},
	}
}

func TestResolveSkills(t *testing.T) {
	tests := []struct {
		name     string
		profiles []string
		want     map[string]string
		wantErr  string
	}{
		{
			name:     "one profile",
			profiles: []string{"core"},
			want:     map[string]string{"git-conventions": "core", "code-review": "core"},
		},
		{
			name:     "dedup: shared skill stays with first profile",
			profiles: []string{"core", "tri"},
			want: map[string]string{
				"git-conventions": "core",
				"code-review":     "core",
				"golang":          "tri",
			},
		},
		{
			name:     "non-existent profile",
			profiles: []string{"nope"},
			wantErr:  `profile "nope" does not exist`,
		},
		{
			name:     "no profiles returns empty map",
			profiles: nil,
			want:     map[string]string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := catalog.ResolveSkills(twoProfileCatalog(), tt.profiles)
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("error = %v, want contains %q", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("ResolveSkills() error = %v", err)
			}
			if len(got) != len(tt.want) {
				t.Fatalf("got %v, want %v", got, tt.want)
			}
			for id, profile := range tt.want {
				if got[id] != profile {
					t.Errorf("skill %q → %q, want %q", id, got[id], profile)
				}
			}
		})
	}
}
