package cli

import (
	"fmt"
	"os"

	"github.com/mattn/go-isatty"
	"github.com/spf13/cobra"

	"github.com/andespath/andes-ai/internal/tui"
)

// NewRootCmd builds the andes root command. Subcommands attach here.
func NewRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:           "andes",
		Short:         "andespath AI agent skill manager",
		SilenceUsage:  true,
		SilenceErrors: true,
		// Run is invoked when andes is called with no subcommand.
		// cobra handles -h/--help before reaching Run, so --help still works.
		RunE: func(cmd *cobra.Command, args []string) error {
			// On a real TTY → interactive TUI. Otherwise (CI, pipes, tests) →
			// static banner so existing tests keep passing unchanged.
			if isatty.IsTerminal(os.Stdout.Fd()) || isatty.IsCygwinTerminal(os.Stdout.Fd()) {
				return tui.Run(NewRootCmd)
			}
			fmt.Fprintln(cmd.OutOrStdout(), renderBanner())
			return nil
		},
	}
	root.AddCommand(newInitCmd(), newListCmd(), newDoctorCmd(), newUpdateCmd())
	return root
}
