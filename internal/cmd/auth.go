package cmd

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/pvarganov/yuno-cli/internal/config"
	"github.com/pvarganov/yuno-cli/internal/mask"
	"github.com/pvarganov/yuno-cli/internal/output"
)

// statusView is the flat shape rendered by `auth status`.
type statusView struct {
	Profile          string `json:"profile"`
	Environment      string `json:"environment"`
	Endpoint         string `json:"endpoint"`
	PublicAPIKey     string `json:"public_api_key"`
	PrivateSecretKey string `json:"private_secret_key"`
	AccountCode      string `json:"account_code"`
	OrganizationCode string `json:"organization_code"`
	AccountID        string `json:"account_id"`
}

// newAuthCommand builds `auth`, which stores credentials, and `auth status`,
// which reports the credentials currently in effect.
func newAuthCommand() *cobra.Command {
	authCmd := &cobra.Command{
		Use:   "auth",
		Short: "Store Yuno API credentials for a profile",
		Long: "Prompt for the public API key and the private secret key and store them in " +
			config.Path() + " with 0600 permissions.",
		Args: cobra.NoArgs,
		RunE: runAuthLogin,
	}

	flags := authCmd.Flags()
	flags.String("environment", "", "environment of the profile (sandbox, prod-us, prod-eu)")
	flags.String("account-code", "", "value sent as the X-Account-Code header")
	flags.String("organization-code", "", "value sent as the X-Organization-Code header")
	flags.String("account-id", "", "default account_id used by account-scoped commands")

	authCmd.AddCommand(&cobra.Command{
		Use:   "status",
		Short: "Show the resolved profile, endpoint and masked credentials",
		Args:  cobra.NoArgs,
		RunE:  runAuthStatus,
	})

	return authCmd
}

// runAuthLogin reads both keys from stdin and saves the profile.
func runAuthLogin(cmd *cobra.Command, _ []string) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	name := cfg.ProfileName(flagString(cmd, "profile"))
	profile := cfg.Profiles[name]

	if err := applyAuthFlags(cmd, &profile); err != nil {
		return err
	}

	if err := promptCredentials(cmd, &profile); err != nil {
		return err
	}

	if err := profile.Validate(); err != nil {
		return fmt.Errorf("profile %q: %w", name, err)
	}

	if cfg.Profiles == nil {
		cfg.Profiles = map[string]config.Profile{}
	}

	cfg.Profiles[name] = profile

	if cfg.DefaultProfile == "" {
		cfg.DefaultProfile = name
	}

	if err := cfg.Save(); err != nil {
		return err
	}

	_, err = fmt.Fprintf(cmd.OutOrStdout(), "saved profile %q to %s\n", name, config.Path())
	if err != nil {
		return fmt.Errorf("write auth confirmation: %w", err)
	}

	return nil
}

// applyAuthFlags copies the non-credential flags onto the profile, leaving the
// stored value in place for flags the user did not pass.
func applyAuthFlags(cmd *cobra.Command, profile *config.Profile) error {
	fields := []struct {
		flag   string
		target *string
	}{
		{"environment", &profile.Environment},
		{"account-code", &profile.AccountCode},
		{"organization-code", &profile.OrganizationCode},
		{"account-id", &profile.AccountID},
	}

	for _, f := range fields {
		if cmd.Flags().Changed(f.flag) {
			*f.target = flagString(cmd, f.flag)
		}
	}

	if profile.Environment == "" {
		profile.Environment = config.EnvironmentSandbox
	}

	return profile.ValidateEnvironment()
}

// promptCredentials asks for both keys, keeping an empty answer as "unchanged"
// when the profile already has that key.
func promptCredentials(cmd *cobra.Command, profile *config.Profile) error {
	prompter := newPrompter(cmd.OutOrStdout(), cmd.InOrStdin())

	public, err := prompter.readSecret("Public API key: ")
	if err != nil {
		return err
	}

	if public != "" {
		profile.PublicAPIKey = public
	}

	private, err := prompter.readSecret("Private secret key: ")
	if err != nil {
		return err
	}

	if private != "" {
		profile.PrivateSecretKey = private
	}

	return nil
}

// prompter reads secrets from one input, without echoing them when that input
// is an interactive terminal. It keeps a single buffered reader so that a
// second prompt does not lose the bytes the first one read ahead.
type prompter struct {
	out io.Writer
	buf *bufio.Reader
	fd  int
	tty bool
}

// newPrompter binds a prompter to the command's input and output.
func newPrompter(out io.Writer, in io.Reader) *prompter {
	p := &prompter{out: out}

	if f, ok := in.(interface{ Fd() uintptr }); ok {
		p.fd = int(f.Fd())
		p.tty = term.IsTerminal(p.fd)
	}

	if !p.tty {
		p.buf = bufio.NewReader(in)
	}

	return p
}

// readSecret prints the prompt and returns one trimmed answer.
func (p *prompter) readSecret(prompt string) (string, error) {
	if _, err := fmt.Fprint(p.out, prompt); err != nil {
		return "", fmt.Errorf("write prompt: %w", err)
	}

	if p.tty {
		line, err := term.ReadPassword(p.fd)
		if err != nil {
			return "", fmt.Errorf("read secret: %w", err)
		}

		_, _ = fmt.Fprintln(p.out)

		return strings.TrimSpace(string(line)), nil
	}

	line, err := p.buf.ReadString('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		return "", fmt.Errorf("read secret: %w", err)
	}

	return strings.TrimSpace(line), nil
}

// runAuthStatus prints the profile in effect, with the keys masked unless
// --unmask was given.
func runAuthStatus(cmd *cobra.Command, _ []string) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	explicit := flagString(cmd, "profile")

	profile, err := cfg.Profile(explicit)
	if err != nil {
		return err
	}

	endpoint, err := profile.Endpoint()
	if err != nil {
		return err
	}

	view := statusView{
		Profile:          cfg.ProfileName(explicit),
		Environment:      profile.Environment,
		Endpoint:         endpoint,
		PublicAPIKey:     hide(profile.PublicAPIKey, flagBool(cmd, "unmask")),
		PrivateSecretKey: hide(profile.PrivateSecretKey, flagBool(cmd, "unmask")),
		AccountCode:      profile.AccountCode,
		OrganizationCode: profile.OrganizationCode,
		AccountID:        profile.AccountID,
	}

	if view.Environment == "" {
		view.Environment = config.EnvironmentSandbox
	}

	formatter := output.NewFormatter(flagBool(cmd, "json"), output.WithUnmask(true))

	if err := formatter.Format(cmd.OutOrStdout(), []statusView{view}); err != nil {
		return fmt.Errorf("print auth status: %w", err)
	}

	return nil
}

// hide masks a credential unless the caller asked for the raw value.
func hide(secret string, unmask bool) string {
	if unmask {
		return secret
	}

	return mask.Secret(secret)
}

// flagString reads a string flag, treating a missing flag as empty.
func flagString(cmd *cobra.Command, name string) string {
	value, err := cmd.Flags().GetString(name)
	if err != nil {
		return ""
	}

	return value
}

// flagBool reads a bool flag, treating a missing flag as false.
func flagBool(cmd *cobra.Command, name string) bool {
	value, err := cmd.Flags().GetBool(name)
	if err != nil {
		return false
	}

	return value
}
