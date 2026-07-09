package cli

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/andespath/andes-ai/internal/catalog"
)

func plural(n int, word string) string {
	if n == 1 {
		return fmt.Sprintf("%d %s", n, word)
	}
	return fmt.Sprintf("%d %ss", n, word)
}

func newValidateCmd() *cobra.Command {
	var catalogFlag string
	cmd := &cobra.Command{
		Use:   "validate",
		Short: "Validate a catalog before opening a PR (also run by CI)",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runValidate(cmd, catalogFlag)
		},
	}
	cmd.Flags().StringVar(&catalogFlag, "catalog", "", "path to the catalog folder (default: search up from the current directory)")
	return cmd
}

func runValidate(cmd *cobra.Command, catalogFlag string) error {
	root := catalogFlag
	if root == "" {
		found, err := findCatalogRoot()
		if err != nil {
			return err
		}
		root = found
	}

	src := catalog.LocalDir{Root: root}
	c, err := src.Load() // JSON, skill existence, id safety — accumulated + sorted
	if err != nil {
		return err
	}

	if problems := catalog.Lint(src, c); len(problems) > 0 {
		return fmt.Errorf("invalid catalog:\n  %s", strings.Join(problems, "\n  "))
	}

	unique := map[string]bool{}
	for _, p := range c.Profiles {
		for _, id := range p.Skills {
			unique[id] = true
		}
	}
	fmt.Fprintf(cmd.OutOrStdout(), "✓ catalog valid: %s, %s\n", plural(len(c.Profiles), "profile"), plural(len(unique), "skill"))
	return nil
}

// findCatalogRoot walks up from the current directory looking for catalog.json.
func findCatalogRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("could not resolve the current directory: %w", err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "catalog.json")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", errors.New("no catalog.json found — run this inside a catalog repo, or pass --catalog <path>")
		}
		dir = parent
	}
}
