package cli

import (
	"errors"
	"fmt"
	"sort"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/andespath/andes-ai/internal/catalog"
	"github.com/andespath/andes-ai/internal/hashdir"
	"github.com/andespath/andes-ai/internal/manifest"
)

func newListCmd() *cobra.Command {
	var catalogPath string

	cmd := &cobra.Command{
		Use:   "list",
		Short: "Muestra perfiles y skills del catálogo con su estado",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runList(cmd, catalogPath)
		},
	}
	cmd.Flags().StringVar(&catalogPath, "catalog", "", "ruta a la carpeta del catálogo")
	return cmd
}

func runList(cmd *cobra.Command, catalogPath string) error {
	mPath, err := manifest.DefaultPath()
	if err != nil {
		return err
	}
	m, err := manifest.Load(mPath)
	if err != nil {
		return err
	}

	if catalogPath == "" && m != nil {
		catalogPath = m.Catalog.Path
	}
	if catalogPath == "" {
		return errors.New("no sé dónde está el catálogo: pasá --catalog <ruta> o corré `andes init` primero")
	}

	src := catalog.LocalDir{Root: catalogPath}
	cat, err := src.Load()
	if err != nil {
		return err
	}

	profileNames := make([]string, 0, len(cat.Profiles))
	for name := range cat.Profiles {
		profileNames = append(profileNames, name)
	}
	sort.Strings(profileNames)

	w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 4, 2, ' ', 0)
	fmt.Fprintln(w, "PERFIL\tSKILL\tESTADO")
	for _, pname := range profileNames {
		for _, id := range cat.Profiles[pname].Skills {
			status, err := skillStatus(src, m, id)
			if err != nil {
				return err
			}
			fmt.Fprintf(w, "%s\t%s\t%s\n", pname, id, status)
		}
	}
	w.Flush()

	if m == nil {
		fmt.Fprintln(cmd.OutOrStdout(), "\nTodavía no corriste `andes init` — corrélo para instalar un perfil.")
	}
	return nil
}

// skillStatus compares manifest hash vs catalog hash. Disk state is
// doctor's job, not list's.
func skillStatus(src catalog.Source, m *manifest.Manifest, id string) (string, error) {
	if m == nil {
		return "✗ no instalada", nil
	}
	inst, ok := m.Installed[id]
	if !ok {
		return "✗ no instalada", nil
	}
	catHash, err := hashdir.Hash(src.SkillPath(id))
	if err != nil {
		return "", fmt.Errorf("no pude leer la skill %q del catálogo: %w", id, err)
	}
	if catHash != inst.Hash {
		return "⚠ desactualizada", nil
	}
	return "✓ instalada", nil
}
