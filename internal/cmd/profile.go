package cmd

import (
	"fmt"
	"sort"

	"github.com/spf13/cobra"

	"github.com/pvarganov/yuno-cli/internal/config"
	"github.com/pvarganov/yuno-cli/internal/mask"
	"github.com/pvarganov/yuno-cli/internal/output"
)

// profileView is the flat shape rendered by `profile list`.
type profileView struct {
	Name         string `json:"name"`
	Environment  string `json:"environment"`
	Default      bool   `json:"default"`
	PublicAPIKey string `json:"public_api_key"`
}

// newProfileCommand builds the `profile` command group.
func newProfileCommand() *cobra.Command {
	profileCmd := &cobra.Command{
		Use:   "profile",
		Short: "Manage the configured profiles",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return cmd.Help()
		},
	}

	profileCmd.AddCommand(
		&cobra.Command{
			Use:   "list",
			Short: "List the configured profiles",
			Args:  cobra.NoArgs,
			RunE:  runProfileList,
		},
		&cobra.Command{
			Use:   "use <name>",
			Short: "Make a profile the default one",
			Args:  cobra.ExactArgs(1),
			RunE:  runProfileUse,
		},
		&cobra.Command{
			Use:   "delete <name>",
			Short: "Remove a profile from the configuration",
			Args:  cobra.ExactArgs(1),
			RunE:  runProfileDelete,
		},
	)

	return profileCmd
}

// runProfileList prints every configured profile with its public key masked.
func runProfileList(cmd *cobra.Command, _ []string) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	if len(cfg.Profiles) == 0 {
		return fmt.Errorf("no profiles configured: run `yuno-cli auth` first")
	}

	names := make([]string, 0, len(cfg.Profiles))
	for name := range cfg.Profiles {
		names = append(names, name)
	}

	sort.Strings(names)

	views := make([]profileView, 0, len(names))

	for _, name := range names {
		profile := cfg.Profiles[name]

		environment := profile.Environment
		if environment == "" {
			environment = config.EnvironmentSandbox
		}

		views = append(views, profileView{
			Name:         name,
			Environment:  environment,
			Default:      name == cfg.DefaultProfile,
			PublicAPIKey: mask.Secret(profile.PublicAPIKey),
		})
	}

	formatter := output.NewFormatter(flagBool(cmd, "json"), output.WithUnmask(true))

	if err := formatter.Format(cmd.OutOrStdout(), views); err != nil {
		return fmt.Errorf("print profiles: %w", err)
	}

	return nil
}

// runProfileUse switches the default profile.
func runProfileUse(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	name := args[0]
	if _, ok := cfg.Profiles[name]; !ok {
		return fmt.Errorf("%q: %w", name, config.ErrProfileNotFound)
	}

	cfg.DefaultProfile = name

	if err := cfg.Save(); err != nil {
		return err
	}

	if _, err := fmt.Fprintf(cmd.OutOrStdout(), "default profile is now %q\n", name); err != nil {
		return fmt.Errorf("write profile confirmation: %w", err)
	}

	return nil
}

// runProfileDelete removes a profile, clearing the default when it pointed at it.
func runProfileDelete(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	name := args[0]
	if _, ok := cfg.Profiles[name]; !ok {
		return fmt.Errorf("%q: %w", name, config.ErrProfileNotFound)
	}

	delete(cfg.Profiles, name)

	if cfg.DefaultProfile == name {
		cfg.DefaultProfile = ""
	}

	if err := cfg.Save(); err != nil {
		return err
	}

	if _, err := fmt.Fprintf(cmd.OutOrStdout(), "deleted profile %q\n", name); err != nil {
		return fmt.Errorf("write profile confirmation: %w", err)
	}

	return nil
}
