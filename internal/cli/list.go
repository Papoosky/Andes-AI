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
		Short: "Shows catalog profiles and skills with their status",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runList(cmd, catalogPath)
		},
	}
	cmd.Flags().StringVar(&catalogPath, "catalog", "", "path or git URL of the catalog")
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

	var src catalog.Source
	if catalogPath != "" {
		// Flag wins: explicit --catalog path or git URL.
		s, _, err := sourceFor(catalogPath)
		if err != nil {
			return err
		}
		src = s
	} else if m != nil {
		// Resolve from manifest type.
		dir, err := mirrorDir()
		if err != nil {
			return err
		}
		src = catalog.SourceFromManifest(m, dir)
	} else {
		return errors.New("catalog location unknown: pass --catalog <path> or run `andes init` first")
	}

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
	fmt.Fprintln(w, "PROFILE\tSKILL\tSTATUS")
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
		fmt.Fprintln(cmd.OutOrStdout(), "\nYou haven't run `andes init` yet — run it to install a profile.")
	}
	return nil
}

// skillStatus compares manifest hash vs catalog hash. Disk state is
// doctor's job, not list's.
func skillStatus(src catalog.Source, m *manifest.Manifest, id string) (string, error) {
	if m == nil {
		return "✗ not installed", nil
	}
	inst, ok := m.Installed[id]
	if !ok {
		return "✗ not installed", nil
	}
	catHash, err := hashdir.Hash(src.SkillPath(id))
	if err != nil {
		return "", fmt.Errorf("could not read skill %q from catalog: %w", id, err)
	}
	if catHash != inst.Hash {
		return "⚠ outdated", nil
	}
	return "✓ installed", nil
}
