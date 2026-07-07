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
		Title("¿Dónde está el catálogo de skills?").
		Description("Ruta a la carpeta que contiene catalog.json").
		Value(&path).
		Run()
	if err != nil {
		return "", err
	}
	if path == "" {
		return "", errors.New("necesito la ruta del catálogo para continuar")
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
		Title("¿Qué perfiles querés instalar?").
		Options(opts...).
		Value(&selected).
		Run()
	if err != nil {
		return nil, err
	}
	if len(selected) == 0 {
		return nil, errors.New("no elegiste ningún perfil")
	}
	return selected, nil
}

func confirmPlan() (bool, error) {
	var ok bool
	err := huh.NewConfirm().
		Title("¿Aplicar estos cambios?").
		Affirmative("Sí, dale").
		Negative("No").
		Value(&ok).
		Run()
	return ok, err
}
