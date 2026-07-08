package cli

import (
	"errors"
	"fmt"
	"os"

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
		Short: "Installs skills from the catalog according to profiles",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runInit(cmd, catalogPath, profiles, yes)
		},
	}
	cmd.Flags().StringVar(&catalogPath, "catalog", "", "path or git URL of the catalog")
	cmd.Flags().StringSliceVar(&profiles, "profiles", nil, "profiles to install (e.g.: andespath-core,tri-fleet)")
	cmd.Flags().BoolVar(&yes, "yes", false, "apply without confirmation prompt")
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

	src, catRef, err := resolveSource(catalogPath, prev, yes)
	if err != nil {
		return err
	}
	if g, ok := src.(catalog.GitRepo); ok {
		if _, statErr := os.Stat(g.Dir); statErr != nil {
			fmt.Fprintln(cmd.OutOrStdout(), "Fetching the andespath catalog…")
		}
		if err := g.Ensure(); err != nil {
			return err
		}
	}
	cat, err := src.Load()
	if err != nil {
		return err
	}

	// Resolve profiles: flag → previous manifest → prompt (Task 11).
	if len(profiles) == 0 && prev != nil {
		profiles = prev.Profiles
	}
	if len(profiles) == 0 {
		if yes {
			return errors.New("no profiles specified: pass --profiles a,b (run `andes list` to see available ones)")
		}
		profiles, err = promptProfiles(cat)
		if err != nil {
			return err
		}
	}

	return installAndSave(cmd, src, cat, prev, profiles, catRef, yes)
}

// installAndSave runs the shared plan→confirm→apply→save pipeline used by
// both init and update.
func installAndSave(cmd *cobra.Command, src catalog.Source, cat *catalog.Catalog, prev *manifest.Manifest, profiles []string, catRef manifest.CatalogRef, yes bool) error {
	actions, err := installer.Plan(src, cat, prev, profiles)
	if err != nil {
		return err
	}

	fmt.Fprintln(cmd.OutOrStdout(), "Plan:")
	for _, a := range actions {
		fmt.Fprintf(cmd.OutOrStdout(), "  %-12s %s\n", a.Type, a.SkillID)
	}

	if !yes {
		ok, err := confirmPlan()
		if err != nil {
			return err
		}
		if !ok {
			fmt.Fprintln(cmd.OutOrStdout(), "Aborted — nothing was touched.")
			return nil
		}
	}

	sDir, err := skillsDir()
	if err != nil {
		return err
	}
	installed, err := installer.Apply(src, actions, sDir)
	if err != nil {
		return err
	}

	catRef, err = finalizeRef(src, catRef)
	if err != nil {
		return err
	}
	next := &manifest.Manifest{
		Version:   1,
		Catalog:   catRef,
		Profiles:  profiles,
		Installed: installed,
	}
	mPath, err := manifest.DefaultPath()
	if err != nil {
		return err
	}
	if err := next.Save(mPath); err != nil {
		return err
	}

	fmt.Fprintf(cmd.OutOrStdout(), "✓ %d skills up to date in %s\n", len(installed), sDir)
	return nil
}
