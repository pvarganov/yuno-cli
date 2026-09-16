package cmd_test

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pvarganov/yuno-cli/internal/cmd"
	"github.com/pvarganov/yuno-cli/internal/config"
	"github.com/pvarganov/yuno-cli/internal/confirm"
)

// isolateConfig points the CLI at a throwaway config dir and clears every
// credential env var, so a developer's real environment cannot leak in.
func isolateConfig(t *testing.T) string {
	t.Helper()

	dir := t.TempDir()
	t.Setenv(config.EnvConfigDir, dir)

	for _, env := range []string{
		config.EnvProfile,
		config.EnvPublicAPIKey,
		config.EnvPrivateSecretKey,
		config.EnvAccountCode,
		config.EnvOrganizationCode,
		config.EnvAccountID,
		config.EnvAPIEndpoint,
		confirm.EnvAssumeYes,
	} {
		t.Setenv(env, "")
	}

	return dir
}

// runCLI executes the root command with the given args and stdin.
func runCLI(t *testing.T, stdin string, args ...string) (string, error) {
	t.Helper()

	var buf bytes.Buffer

	root := cmd.NewRootCommand("test")
	root.SetOut(&buf)
	root.SetErr(&buf)
	root.SetIn(strings.NewReader(stdin))
	root.SetArgs(args)

	err := root.Execute()

	return buf.String(), err
}

// writeConfig stores a config file for the tests to read back.
func writeConfig(t *testing.T, cfg *config.Config) {
	t.Helper()

	if err := cfg.Save(); err != nil {
		t.Fatalf("save config: %v", err)
	}
}

func TestAuthCommand_WritesProfile(t *testing.T) {
	dir := isolateConfig(t)

	out, err := runCLI(t, "pub-alpha\nsec-alpha\n",
		"auth", "--profile", "prod-us", "--environment", "prod-us")
	if err != nil {
		t.Fatalf("auth failed: %v (output: %s)", err, out)
	}

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	profile, ok := cfg.Profiles["prod-us"]
	if !ok {
		t.Fatalf("expected profile prod-us to be saved, got %+v", cfg.Profiles)
	}

	if profile.PublicAPIKey != "pub-alpha" || profile.PrivateSecretKey != "sec-alpha" {
		t.Errorf("unexpected credentials saved: %+v", profile)
	}

	if profile.Environment != config.EnvironmentProdUS {
		t.Errorf("expected environment prod-us, got %q", profile.Environment)
	}

	if strings.Contains(out, "sec-alpha") {
		t.Errorf("the private key must never be echoed, got: %s", out)
	}

	assertFileMode(t, filepath.Join(dir, "config.yaml"), 0o600)
	assertFileMode(t, dir, 0o700|os.ModeDir)
}

func assertFileMode(t *testing.T, path string, want os.FileMode) {
	t.Helper()

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat %s: %v", path, err)
	}

	if info.Mode().Perm() != want.Perm() {
		t.Errorf("%s: expected mode %v, got %v", path, want.Perm(), info.Mode().Perm())
	}
}

func TestAuthCommand_FirstProfileBecomesDefault(t *testing.T) {
	isolateConfig(t)

	if _, err := runCLI(t, "pub\npriv\n", "auth", "--profile", "sandbox"); err != nil {
		t.Fatalf("auth failed: %v", err)
	}

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	if cfg.DefaultProfile != "sandbox" {
		t.Errorf("expected the first profile to become the default, got %q", cfg.DefaultProfile)
	}
}

func TestAuthCommand_OptionalFields(t *testing.T) {
	isolateConfig(t)

	_, err := runCLI(t, "pub\npriv\n", "auth",
		"--profile", "eu",
		"--environment", "prod-eu",
		"--account-code", "acct-code",
		"--organization-code", "org-code",
		"--account-id", "acct-id")
	if err != nil {
		t.Fatalf("auth failed: %v", err)
	}

	cfg, _ := config.Load()
	got := cfg.Profiles["eu"]

	if got.AccountCode != "acct-code" || got.OrganizationCode != "org-code" || got.AccountID != "acct-id" {
		t.Errorf("optional fields not stored: %+v", got)
	}
}

func TestAuthCommand_UnknownEnvironment(t *testing.T) {
	isolateConfig(t)

	_, err := runCLI(t, "pub\npriv\n", "auth", "--environment", "moon")
	if !errors.Is(err, config.ErrUnknownEnvironment) {
		t.Fatalf("expected an unknown environment error, got %v", err)
	}
}

func TestAuthCommand_EmptyKeyIsRejected(t *testing.T) {
	isolateConfig(t)

	_, err := runCLI(t, "\n\n", "auth")
	if !errors.Is(err, config.ErrMissingCredentials) {
		t.Fatalf("expected missing credentials error, got %v", err)
	}
}

func TestAuthStatus_MasksKeys(t *testing.T) {
	isolateConfig(t)
	writeConfig(t, &config.Config{
		DefaultProfile: "sandbox",
		Profiles: map[string]config.Profile{
			"sandbox": {
				Environment:      config.EnvironmentSandbox,
				PublicAPIKey:     "pub-alpha",
				PrivateSecretKey: "sec-alpha",
			},
		},
	})

	out, err := runCLI(t, "", "auth", "status")
	if err != nil {
		t.Fatalf("auth status failed: %v (%s)", err, out)
	}

	if strings.Contains(out, "pub-alpha") || strings.Contains(out, "sec-alpha") {
		t.Errorf("auth status must mask the keys, got: %s", out)
	}

	if !strings.Contains(out, "https://api-sandbox.y.uno/v1") {
		t.Errorf("auth status should print the resolved endpoint, got: %s", out)
	}

	if !strings.Contains(out, "sandbox") {
		t.Errorf("auth status should print the profile name, got: %s", out)
	}
}

func TestAuthStatus_JSONIsMaskedToo(t *testing.T) {
	isolateConfig(t)
	writeConfig(t, &config.Config{
		Profiles: map[string]config.Profile{
			"default": {PublicAPIKey: "pub-alpha", PrivateSecretKey: "sec-alpha"},
		},
	})

	out, err := runCLI(t, "", "auth", "status", "--json")
	if err != nil {
		t.Fatalf("auth status --json failed: %v (%s)", err, out)
	}

	if strings.Contains(out, "sec-alpha") {
		t.Errorf("json output must be masked too, got: %s", out)
	}
}

func TestAuthStatus_Unmask(t *testing.T) {
	isolateConfig(t)
	writeConfig(t, &config.Config{
		Profiles: map[string]config.Profile{
			"default": {PublicAPIKey: "pub-alpha", PrivateSecretKey: "sec-alpha"},
		},
	})

	out, err := runCLI(t, "", "auth", "status", "--json", "--unmask")
	if err != nil {
		t.Fatalf("auth status --unmask failed: %v (%s)", err, out)
	}

	if !strings.Contains(out, "sec-alpha") {
		t.Errorf("--unmask should print the raw key, got: %s", out)
	}
}

func TestAuthStatus_NoProfile(t *testing.T) {
	isolateConfig(t)

	_, err := runCLI(t, "", "auth", "status")
	if !errors.Is(err, config.ErrProfileNotFound) {
		t.Fatalf("expected a profile-not-found error, got %v", err)
	}
}
