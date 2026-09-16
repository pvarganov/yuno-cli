package config_test

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pvarganov/yuno-cli/internal/config"
)

// isolate points the config package at a temporary directory for the test.
func isolate(t *testing.T) string {
	t.Helper()

	dir := t.TempDir()
	t.Setenv("YUNO_CONFIG_DIR", dir)

	for _, name := range []string{
		"YUNO_PROFILE",
		"YUNO_PUBLIC_API_KEY",
		"YUNO_PRIVATE_SECRET_KEY",
		"YUNO_ACCOUNT_CODE",
		"YUNO_ORGANIZATION_CODE",
		"YUNO_ACCOUNT_ID",
		"YUNO_API_ENDPOINT",
	} {
		// an empty value is treated as unset by the package
		t.Setenv(name, "")
	}

	return dir
}

func TestLoadMissingFile(t *testing.T) {
	isolate(t)

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Load() error = %v, want nil", err)
	}

	if cfg.DefaultProfile != "" {
		t.Errorf("DefaultProfile = %q, want empty", cfg.DefaultProfile)
	}

	if len(cfg.Profiles) != 0 {
		t.Errorf("Profiles = %v, want empty", cfg.Profiles)
	}
}

func TestPathUsesConfigDir(t *testing.T) {
	dir := isolate(t)

	want := filepath.Join(dir, "config.yaml")
	if got := config.Path(); got != want {
		t.Errorf("Path() = %q, want %q", got, want)
	}
}

func TestSaveAndLoadRoundTrip(t *testing.T) {
	dir := isolate(t)

	cfg := &config.Config{
		DefaultProfile: "sandbox",
		Profiles: map[string]config.Profile{
			"sandbox": {
				Environment:      config.EnvironmentSandbox,
				PublicAPIKey:     "pub-1",
				PrivateSecretKey: "sec-1",
				AccountCode:      "acc-code",
				OrganizationCode: "org-code",
				AccountID:        "acc-id",
			},
		},
	}

	if err := cfg.Save(); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	loaded, err := config.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	got, err := loaded.Profile("sandbox")
	if err != nil {
		t.Fatalf("Profile() error = %v", err)
	}

	want := cfg.Profiles["sandbox"]
	if got != want {
		t.Errorf("Profile() = %+v, want %+v", got, want)
	}

	if loaded.DefaultProfile != "sandbox" {
		t.Errorf("DefaultProfile = %q, want %q", loaded.DefaultProfile, "sandbox")
	}

	fi, err := os.Stat(filepath.Join(dir, "config.yaml"))
	if err != nil {
		t.Fatalf("stat config file: %v", err)
	}

	if perm := fi.Mode().Perm(); perm != 0o600 {
		t.Errorf("file mode = %o, want %o", perm, 0o600)
	}
}

func TestSaveCreatesDirWith0700(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "nested", "yuno-cli")
	t.Setenv("YUNO_CONFIG_DIR", dir)

	cfg := &config.Config{
		Profiles: map[string]config.Profile{
			"default": {PublicAPIKey: "pub", PrivateSecretKey: "sec"},
		},
	}

	if err := cfg.Save(); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	fi, err := os.Stat(dir)
	if err != nil {
		t.Fatalf("stat config dir: %v", err)
	}

	if perm := fi.Mode().Perm(); perm != 0o700 {
		t.Errorf("dir mode = %o, want %o", perm, 0o700)
	}
}

func TestLoadMalformedFile(t *testing.T) {
	dir := isolate(t)

	if err := os.WriteFile(filepath.Join(dir, "config.yaml"), []byte("::not yaml::\n\t- x"), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	if _, err := config.Load(); err == nil {
		t.Fatal("Load() error = nil, want a parse error")
	}
}

func TestProfileSelection(t *testing.T) {
	isolate(t)

	cfg := &config.Config{
		DefaultProfile: "prod",
		Profiles: map[string]config.Profile{
			"sandbox": {Environment: config.EnvironmentSandbox, PublicAPIKey: "pub-s", PrivateSecretKey: "sec-s"},
			"prod":    {Environment: config.EnvironmentProdUS, PublicAPIKey: "pub-p", PrivateSecretKey: "sec-p"},
		},
	}

	tests := []struct {
		name     string
		explicit string
		env      string
		wantKey  string
	}{
		{name: "explicit wins", explicit: "sandbox", env: "prod", wantKey: "pub-s"},
		{name: "env over default", env: "sandbox", wantKey: "pub-s"},
		{name: "default profile", wantKey: "pub-p"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.env != "" {
				t.Setenv("YUNO_PROFILE", tt.env)
			}

			got, err := cfg.Profile(tt.explicit)
			if err != nil {
				t.Fatalf("Profile() error = %v", err)
			}

			if got.PublicAPIKey != tt.wantKey {
				t.Errorf("PublicAPIKey = %q, want %q", got.PublicAPIKey, tt.wantKey)
			}
		})
	}
}

func TestProfileFallsBackToDefaultName(t *testing.T) {
	isolate(t)

	cfg := &config.Config{
		Profiles: map[string]config.Profile{
			"default": {PublicAPIKey: "pub-d", PrivateSecretKey: "sec-d"},
		},
	}

	got, err := cfg.Profile("")
	if err != nil {
		t.Fatalf("Profile() error = %v", err)
	}

	if got.PublicAPIKey != "pub-d" {
		t.Errorf("PublicAPIKey = %q, want %q", got.PublicAPIKey, "pub-d")
	}
}

func TestProfileNotFound(t *testing.T) {
	isolate(t)

	cfg := &config.Config{Profiles: map[string]config.Profile{}}

	_, err := cfg.Profile("nope")
	if !errors.Is(err, config.ErrProfileNotFound) {
		t.Fatalf("Profile() error = %v, want ErrProfileNotFound", err)
	}

	if !strings.Contains(err.Error(), "nope") {
		t.Errorf("error %q does not name the missing profile", err)
	}
}

func TestProfileEnvOverrides(t *testing.T) {
	isolate(t)

	cfg := &config.Config{
		Profiles: map[string]config.Profile{
			"sandbox": {
				Environment:      config.EnvironmentSandbox,
				PublicAPIKey:     "pub-file",
				PrivateSecretKey: "sec-file",
				AccountCode:      "code-file",
				OrganizationCode: "org-file",
				AccountID:        "id-file",
			},
		},
	}

	t.Setenv("YUNO_PUBLIC_API_KEY", "pub-env")
	t.Setenv("YUNO_PRIVATE_SECRET_KEY", "sec-env")
	t.Setenv("YUNO_ACCOUNT_CODE", "code-env")
	t.Setenv("YUNO_ORGANIZATION_CODE", "org-env")
	t.Setenv("YUNO_ACCOUNT_ID", "id-env")

	got, err := cfg.Profile("sandbox")
	if err != nil {
		t.Fatalf("Profile() error = %v", err)
	}

	want := config.Profile{
		Environment:      config.EnvironmentSandbox,
		PublicAPIKey:     "pub-env",
		PrivateSecretKey: "sec-env",
		AccountCode:      "code-env",
		OrganizationCode: "org-env",
		AccountID:        "id-env",
	}

	if got != want {
		t.Errorf("Profile() = %+v, want %+v", got, want)
	}
}

func TestProfileFromEnvOnly(t *testing.T) {
	isolate(t)

	cfg := &config.Config{}

	t.Setenv("YUNO_PUBLIC_API_KEY", "pub-env")
	t.Setenv("YUNO_PRIVATE_SECRET_KEY", "sec-env")

	got, err := cfg.Profile("")
	if err != nil {
		t.Fatalf("Profile() error = %v", err)
	}

	if got.PublicAPIKey != "pub-env" || got.PrivateSecretKey != "sec-env" {
		t.Errorf("Profile() = %+v, want env-provided keys", got)
	}
}

func TestEndpoint(t *testing.T) {
	isolate(t)

	tests := []struct {
		name        string
		environment string
		want        string
	}{
		{name: "sandbox", environment: config.EnvironmentSandbox, want: "https://api-sandbox.y.uno/v1"},
		{name: "prod-us", environment: config.EnvironmentProdUS, want: "https://api.y.uno/v1"},
		{name: "prod-eu", environment: config.EnvironmentProdEU, want: "https://api.eu.y.uno/v1"},
		{name: "empty defaults to sandbox", environment: "", want: "https://api-sandbox.y.uno/v1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := config.Profile{Environment: tt.environment}

			got, err := p.Endpoint()
			if err != nil {
				t.Fatalf("Endpoint() error = %v", err)
			}

			if got != tt.want {
				t.Errorf("Endpoint() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestEndpointUnknownEnvironment(t *testing.T) {
	isolate(t)

	p := config.Profile{Environment: "mars"}

	_, err := p.Endpoint()
	if !errors.Is(err, config.ErrUnknownEnvironment) {
		t.Fatalf("Endpoint() error = %v, want ErrUnknownEnvironment", err)
	}
}

func TestEndpointEnvOverride(t *testing.T) {
	isolate(t)

	t.Setenv("YUNO_API_ENDPOINT", "http://127.0.0.1:8080/v1/")

	for _, env := range []string{config.EnvironmentProdUS, "mars", ""} {
		p := config.Profile{Environment: env}

		got, err := p.Endpoint()
		if err != nil {
			t.Fatalf("Endpoint() error = %v", err)
		}

		if got != "http://127.0.0.1:8080/v1" {
			t.Errorf("Endpoint() = %q, want the YUNO_API_ENDPOINT override without a trailing slash", got)
		}
	}
}

func TestProfileValidate(t *testing.T) {
	isolate(t)

	tests := []struct {
		name    string
		profile config.Profile
		wantErr error
	}{
		{
			name:    "complete",
			profile: config.Profile{Environment: config.EnvironmentSandbox, PublicAPIKey: "pub", PrivateSecretKey: "sec"},
		},
		{
			name:    "missing public key",
			profile: config.Profile{PrivateSecretKey: "sec"},
			wantErr: config.ErrMissingCredentials,
		},
		{
			name:    "missing private key",
			profile: config.Profile{PublicAPIKey: "pub"},
			wantErr: config.ErrMissingCredentials,
		},
		{
			name:    "both empty",
			profile: config.Profile{},
			wantErr: config.ErrMissingCredentials,
		},
		{
			name:    "unknown environment",
			profile: config.Profile{Environment: "mars", PublicAPIKey: "pub", PrivateSecretKey: "sec"},
			wantErr: config.ErrUnknownEnvironment,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.profile.Validate()
			if tt.wantErr == nil {
				if err != nil {
					t.Fatalf("Validate() error = %v, want nil", err)
				}

				return
			}

			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("Validate() error = %v, want %v", err, tt.wantErr)
			}
		})
	}
}

func TestProfileValidateEnvironment(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		environment string
		wantErr     bool
	}{
		{name: "empty defaults to sandbox", environment: ""},
		{name: "sandbox", environment: config.EnvironmentSandbox},
		{name: "prod-us", environment: config.EnvironmentProdUS},
		{name: "prod-eu", environment: config.EnvironmentProdEU},
		{name: "unknown", environment: "moon", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			p := config.Profile{Environment: tt.environment}

			err := p.ValidateEnvironment()
			if tt.wantErr {
				if !errors.Is(err, config.ErrUnknownEnvironment) {
					t.Fatalf("expected config.ErrUnknownEnvironment, got %v", err)
				}

				return
			}

			if err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
		})
	}
}
