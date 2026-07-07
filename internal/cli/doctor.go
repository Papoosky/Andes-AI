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
		Short: "Diagnostica drift entre manifiesto, disco y catálogo (no modifica nada)",
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
		return errors.New("no hay manifiesto: nunca corriste `andes init`")
	}

	src := catalog.LocalDir{Root: m.Catalog.Path}
	if _, err := src.Load(); err != nil {
		return fmt.Errorf("catálogo inaccesible en %s: corregí la ruta y re-corré `andes init --catalog <ruta>` (%w)",
			m.Catalog.Path, err)
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
	fmt.Fprintln(w, "SKILL\tESTADO\tCONSEJO")
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
		return fmt.Errorf("doctor encontró %d problema(s)", problems)
	}
	fmt.Fprintln(cmd.OutOrStdout(), "Todo sano ✓")
	return nil
}
