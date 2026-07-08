package cli

import (
	"errors"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/andespath/andes-ai/internal/catalog"
	"github.com/andespath/andes-ai/internal/manifest"
)

func newUpdateCmd() *cobra.Command {
	var yes bool
	cmd := &cobra.Command{
		Use:   "update",
		Short: "Syncs the catalog mirror and refreshes outdated skills",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runUpdate(cmd, yes)
		},
	}
	cmd.Flags().BoolVar(&yes, "yes", false, "apply without confirmation prompt")
	return cmd
}

func runUpdate(cmd *cobra.Command, yes bool) error {
	mPath, err := manifest.DefaultPath()
	if err != nil {
		return err
	}
	prev, err := manifest.Load(mPath)
	if err != nil {
		return err
	}
	if prev == nil {
		return errors.New("no manifest found: you haven't run `andes init` yet")
	}
	if prev.Catalog.Type != "git" {
		return errors.New("nothing to update: local catalog (re-run `andes init` to refresh from a local folder)")
	}

	dir, err := mirrorDir()
	if err != nil {
		return err
	}
	g := catalog.GitRepo{URL: prev.Catalog.URL, Dir: dir}
	if err := g.Ensure(); err != nil {
		return err
	}
	newHead, err := g.Sync()
	if err != nil {
		return err
	}
	if newHead == prev.Catalog.Ref {
		fmt.Fprintln(cmd.OutOrStdout(), "Already up to date")
		return nil
	}

	cat, err := g.Load()
	if err != nil {
		return err
	}
	catRef := manifest.CatalogRef{Type: "git", URL: prev.Catalog.URL}
	return installAndSave(cmd, g, cat, prev, prev.Profiles, catRef, yes)
}
