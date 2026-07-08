package cli

import (
	"errors"
	"fmt"
	"sort"

	"github.com/charmbracelet/huh"

	"github.com/andespath/andes-ai/internal/catalog"
)

func promptCatalogPath() (string, error) {
	var path string
	err := huh.NewInput().
		Title("Where is the skills catalog?").
		Description("Path to the folder containing catalog.json").
		Value(&path).
		Run()
	if err != nil {
		return "", err
	}
	if path == "" {
		return "", errors.New("catalog path is required to continue")
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
