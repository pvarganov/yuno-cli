package cmd

import (
	"github.com/spf13/cobra"
)

// NewRootCommand creates the root cobra command for the Yuno CLI.
func NewRootCommand(version string) *cobra.Command {
	root := &cobra.Command{
		Use:           "yuno-cli",
		Short:         "Yuno CLI - manage your Yuno account from the terminal",
		Version:       version,
		SilenceErrors: true,
		SilenceUsage:  true,
		Args:          cobra.NoArgs,
		// show help when no subcommand is given
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}

	flags := root.PersistentFlags()
	flags.Bool("json", false, "output as JSON")
	flags.String("profile", "", "configuration profile to use")
	flags.Bool("verbose", false, "dump HTTP requests and responses to stderr")
	flags.Bool("yes", false, "skip confirmation prompts for write operations")
	flags.Bool("unmask", false, "show card data and credentials unmasked in the output")

	root.AddCommand(
		newAuthCommand(),
		newProfileCommand(),
		newRawCommand(),
		newPaymentCommand(),
		newRoutingCommand(),
		newConnectionCommand(),
	)

	return root
}
