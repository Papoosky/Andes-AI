package cli

import (
	"fmt"

	"github.com/spf13/cobra"
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
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Fprintln(cmd.OutOrStdout(), renderBanner())
		},
	}
	root.AddCommand(newInitCmd(), newListCmd(), newDoctorCmd())
	return root
}
