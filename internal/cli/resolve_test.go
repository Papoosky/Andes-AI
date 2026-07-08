package cli

import "testing"

func TestIsGitURL(t *testing.T) {
	tests := []struct {
		in   string
		want bool
	}{
		{"git@github.com:andespath/andes-ai.git", true},
		{"https://github.com/andespath/andes-ai.git", true},
		{"https://github.com/andespath/andes-ai", true},
		{"ssh://git@github.com/x/y", true},
		{"file:///tmp/some-repo", true},
		{"./catalog", false},
		{"/abs/path/catalog", false},
		{"", false},
	}
	for _, tt := range tests {
		if got := isGitURL(tt.in); got != tt.want {
			t.Errorf("isGitURL(%q) = %v, want %v", tt.in, got, tt.want)
		}
	}
}
