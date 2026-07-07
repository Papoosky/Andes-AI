package cli

import (
	"errors"
	"fmt"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/andespath/andes-ai/internal/catalog"
	"github.com/andespath/andes-ai/internal/installer"
	"github.com/andespath/andes-ai/internal/manifest"
)

func newInitCmd() *cobra.Command {
	var catalogPath string
	var profiles []string
	var yes bool

	cmd := &cobra.Command{
		Use:   "init",
		Short: "Instala skills desde el catálogo según perfiles",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runInit(cmd, catalogPath, profiles, yes)
		},
	}
	cmd.Flags().StringVar(&catalogPath, "catalog", "", "ruta a la carpeta del catálogo")
	cmd.Flags().StringSliceVar(&profiles, "profiles", nil, "perfiles a instalar (ej: andespath-core,tri-fleet)")
	cmd.Flags().BoolVar(&yes, "yes", false, "aplicar sin pedir confirmación")
	return cmd
}

func runInit(cmd *cobra.Command, catalogPath string, profiles []string, yes bool) error {
	mPath, err := manifest.DefaultPath()
	if err != nil {
		return err
	}
	prev, err := manifest.Load(mPath)
	if err != nil {
		return err
	}

	// Resolve catalog path: flag → previous manifest → prompt (Task 11).
	if catalogPath == "" && prev != nil {
		catalogPath = prev.Catalog.Path
	}
	if catalogPath == "" {
		// interactivo: Task 11
		return errors.New("no sé dónde está el catálogo: pasá --catalog <ruta>")
	}

	src := catalog.LocalDir{Root: catalogPath}
	cat, err := src.Load()
	if err != nil {
		return err
	}

	// Resolve profiles: flag → previous manifest → prompt (Task 11).
	if len(profiles) == 0 && prev != nil {
		profiles = prev.Profiles
	}
	if len(profiles) == 0 {
		// interactivo: Task 11
		return errors.New("no sé qué perfiles instalar: pasá --profiles a,b (corré `andes list` para verlos)")
	}

	actions, err := installer.Plan(src, cat, prev, profiles)
	if err != nil {
		return err
	}

	fmt.Fprintln(cmd.OutOrStdout(), "Plan:")
	for _, a := range actions {
		fmt.Fprintf(cmd.OutOrStdout(), "  %-12s %s\n", a.Type, a.SkillID)
	}

	if !yes {
		// interactivo: Task 11 (confirmación). Sin --yes hoy: abortar explícito.
		return errors.New("confirmación interactiva no disponible todavía: re-corré con --yes")
	}

	sDir, err := skillsDir()
	if err != nil {
		return err
	}
	installed, err := installer.Apply(src, actions, sDir)
	if err != nil {
		return err
	}

	absCatalog, err := filepath.Abs(catalogPath)
	if err != nil {
		return err
	}
	next := &manifest.Manifest{
		Version:   1,
		Catalog:   manifest.CatalogRef{Type: "local", Path: absCatalog},
		Profiles:  profiles,
		Installed: installed,
	}
	if err := next.Save(mPath); err != nil {
		return err
	}

	fmt.Fprintf(cmd.OutOrStdout(), "✓ %d skills al día en %s\n", len(installed), sDir)
	return nil
}
