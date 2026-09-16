package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// newCompletionCommand generates the shell completion script of the CLI. The
// command is written by hand instead of relying on cobra's built-in one so the
// help text matches the rest of the tree.
func newCompletionCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "completion [bash|zsh|fish|powershell]",
		Short: "Generate shell completion scripts",
		Long: "Generate shell completion scripts.\n\n" +
			"Load the script from your shell profile, e.g.\n" +
			"  yuno-cli completion zsh > \"${fpath[1]}/_yuno-cli\"\n" +
			"  source <(yuno-cli completion bash)",
		Example: "  yuno-cli completion bash\n" +
			"  yuno-cli completion zsh > \"${fpath[1]}/_yuno-cli\"",
		ValidArgs: []string{"bash", "zsh", "fish", "powershell"},
		Args:      cobra.ExactArgs(1),
		RunE:      runCompletion,
	}
}

// runCompletion writes the completion script of one shell to stdout.
func runCompletion(cmd *cobra.Command, args []string) error {
	root := cmd.Root()
	out := cmd.OutOrStdout()

	var err error

	switch args[0] {
	case "bash":
		err = root.GenBashCompletion(out)
	case "zsh":
		err = root.GenZshCompletion(out)
	case "fish":
		err = root.GenFishCompletion(out, true)
	case "powershell":
		err = root.GenPowerShellCompletionWithDesc(out)
	default:
		return fmt.Errorf("unsupported shell: %s", args[0])
	}

	if err != nil {
		return fmt.Errorf("generate %s completion: %w", args[0], err)
	}

	return nil
}
