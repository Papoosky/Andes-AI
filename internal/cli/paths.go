package cli

import (
	"fmt"
	"os"
	"path/filepath"
)

// skillsDir returns ~/.claude/skills (respects $HOME, overridable in tests).
func skillsDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("could not resolve home directory: %w", err)
	}
	return filepath.Join(home, ".claude", "skills"), nil
}
