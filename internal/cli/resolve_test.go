package cli

import (
	"strings"
	"testing"
)

func TestSourceForRejectsDashPrefixedURL(t *testing.T) {
	malicious := "--upload-pack=touch /tmp/andes_pwned_cli #.git"
	_, _, err := sourceFor(malicious)
	if err == nil {
		t.Fatal("sourceFor() with '-'-prefixed URL should return an error")
	}
	if !strings.Contains(err.Error(), "must not start with '-'") {
		t.Errorf("error = %q, want message containing \"must not start with '-'\"", err)
	}
}

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
		{"/abs/path/ending.git", true}, // known edge: local bare repo — use file:// instead
		{"http://github.com/x/y", true},
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
