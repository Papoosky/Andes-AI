package catalog

import (
	"reflect"
	"testing"
)

func TestURLVariants(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want []string
	}{
		{
			name: "ssh github with .git",
			in:   "git@github.com:Papoosky/Andes-AI.git",
			want: []string{"git@github.com:Papoosky/Andes-AI.git", "https://github.com/Papoosky/Andes-AI.git"},
		},
		{
			name: "https github with .git",
			in:   "https://github.com/Papoosky/Andes-AI.git",
			want: []string{"https://github.com/Papoosky/Andes-AI.git", "git@github.com:Papoosky/Andes-AI.git"},
		},
		{
			name: "https without .git",
			in:   "https://github.com/o/r",
			want: []string{"https://github.com/o/r", "git@github.com:o/r"},
		},
		{
			name: "enterprise host ssh",
			in:   "git@github.example.com:team/repo.git",
			want: []string{"git@github.example.com:team/repo.git", "https://github.example.com/team/repo.git"},
		},
		{
			name: "file url is not expanded",
			in:   "file:///tmp/some-repo",
			want: []string{"file:///tmp/some-repo"},
		},
		{
			name: "local path is not expanded",
			in:   "/abs/path/catalog",
			want: []string{"/abs/path/catalog"},
		},
		{
			name: "malformed ssh without path",
			in:   "git@github.com",
			want: []string{"git@github.com"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := URLVariants(tt.in)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("URLVariants(%q) = %v, want %v", tt.in, got, tt.want)
			}
		})
	}
}
