package cli

import "github.com/spf13/cobra"

// NewRootCmd builds the andes root command. Subcommands attach here.
func NewRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:           "andes",
		Short:         "Gestor de skills de agentes IA de andespath",
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	root.AddCommand(newInitCmd(), newListCmd(), newDoctorCmd())
	return root
}
