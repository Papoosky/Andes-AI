package cli

import (
	"errors"
	"fmt"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/andespath/andes-ai/internal/catalog"
	"github.com/andespath/andes-ai/internal/doctor"
	"github.com/andespath/andes-ai/internal/manifest"
)

func newDoctorCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "doctor",
		Short: "Diagnoses drift between manifest, disk and catalog (read-only)",
		RunE:  runDoctor,
	}
}

func runDoctor(cmd *cobra.Command, args []string) error {
	mPath, err := manifest.DefaultPath()
	if err != nil {
		return err
	}
	m, err := manifest.Load(mPath)
	if err != nil {
		return err
	}
	if m == nil {
		return errors.New("no manifest found: you have not run `andes init` yet")
	}

	var src catalog.Source
	if m.Catalog.Type == "git" {
		dir, err := mirrorDir()
		if err != nil {
			return err
		}
		src = catalog.GitRepo{URL: m.Catalog.URL, Dir: dir}
	} else {
		src = catalog.LocalDir{Root: m.Catalog.Path}
	}
	if _, err := src.Load(); err != nil {
		loc := m.Catalog.Path
		if m.Catalog.Type == "git" {
			loc = m.Catalog.URL
		}
		return fmt.Errorf("catalog inaccessible at %s: fix the path and re-run `andes init --catalog <path>` (%w)",
			loc, err)
	}

	sDir, err := skillsDir()
	if err != nil {
		return err
	}
	findings, err := doctor.Check(src, m, sDir)
	if err != nil {
		return err
	}

	problems := 0
	w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 4, 2, ' ', 0)
	fmt.Fprintln(w, "SKILL\tSTATUS\tADVICE")
	for _, f := range findings {
		mark := "✓"
		if f.Status != doctor.StatusOK {
			mark = "✗"
			problems++
		}
		fmt.Fprintf(w, "%s\t%s %s\t%s\n", f.SkillID, mark, f.Status, f.Advice)
	}
	w.Flush()

	if problems > 0 {
		return fmt.Errorf("doctor found %d problem(s)", problems)
	}
	fmt.Fprintln(cmd.OutOrStdout(), "All healthy ✓")
	return nil
}
