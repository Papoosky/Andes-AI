package cli

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/charmbracelet/huh"

	"github.com/andespath/andes-ai/internal/catalog"
)

func promptCatalogPath() (string, error) {
	var path string
	err := huh.NewInput().
		Title("Where is the skills catalog?").
		Description("The catalog is the source folder your skills are installed from — it contains a catalog.json (profiles) and a skills/ directory.\nTip: for the demo catalog use ./catalog").
		Value(&path).
		Validate(func(s string) error {
			if strings.TrimSpace(s) == "" {
				return errors.New("I need the catalog path to continue")
			}

			// Expand ~ to home directory
			expanded := s
			if strings.HasPrefix(s, "~/") {
				home, err := os.UserHomeDir()
				if err != nil {
					return fmt.Errorf("could not expand home directory: %w", err)
				}
				expanded = filepath.Join(home, s[2:])
			}

			// Check if catalog.json exists
			catalogFile := filepath.Join(expanded, "catalog.json")
			if _, err := os.Stat(catalogFile); err != nil {
				return fmt.Errorf("no catalog.json found in %q — point me at the folder that contains it", s)
			}

			return nil
		}).
		Run()
	if err != nil {
		return "", err
	}

	// Expand ~ in the final path before returning
	if strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("could not expand home directory: %w", err)
		}
		path = filepath.Join(home, path[2:])
	}

	return path, nil
}

func promptProfiles(cat *catalog.Catalog) ([]string, error) {
	names := make([]string, 0, len(cat.Profiles))
	for name := range cat.Profiles {
		names = append(names, name)
	}
	sort.Strings(names)

	opts := make([]huh.Option[string], 0, len(names))
	for _, name := range names {
		label := fmt.Sprintf("%s — %s", name, cat.Profiles[name].Description)
		opts = append(opts, huh.NewOption(label, name))
	}

	var selected []string
	err := huh.NewMultiSelect[string]().
		Title("Which profiles do you want to install?").
		Options(opts...).
		Value(&selected).
		Run()
	if err != nil {
		return nil, err
	}
	if len(selected) == 0 {
		return nil, errors.New("no profile selected")
	}
	return selected, nil
}

func confirmPlan() (bool, error) {
	var ok bool
	err := huh.NewConfirm().
		Title("Apply these changes?").
		Affirmative("Yes, go ahead").
		Negative("No").
		Value(&ok).
		Run()
	return ok, err
}
