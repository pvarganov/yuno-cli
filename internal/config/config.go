// Package config loads and stores the yuno-cli configuration: named profiles
// holding Yuno API credentials and the environment each profile points at.
package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// Supported Yuno environments, mapped to the servers declared in the OpenAPI spec.
const (
	EnvironmentSandbox = "sandbox"
	EnvironmentProdUS  = "prod-us"
	EnvironmentProdEU  = "prod-eu"
)

// DefaultProfileName is used when neither a flag, an env var nor the config
// file names a profile.
const DefaultProfileName = "default"

const (
	dirPerm  = 0o700
	filePerm = 0o600
)

// Environment variables recognised by the CLI.
const (
	EnvConfigDir        = "YUNO_CONFIG_DIR"
	EnvProfile          = "YUNO_PROFILE"
	EnvPublicAPIKey     = "YUNO_PUBLIC_API_KEY"     //nolint:gosec // env var name, not a credential
	EnvPrivateSecretKey = "YUNO_PRIVATE_SECRET_KEY" //nolint:gosec // env var name, not a credential
	EnvAccountCode      = "YUNO_ACCOUNT_CODE"
	EnvOrganizationCode = "YUNO_ORGANIZATION_CODE"
	EnvAccountID        = "YUNO_ACCOUNT_ID"
	EnvAPIEndpoint      = "YUNO_API_ENDPOINT"
)

// Sentinel errors, comparable with errors.Is.
var (
	ErrProfileNotFound    = errors.New("profile not found")
	ErrMissingCredentials = errors.New("missing api credentials")
	ErrUnknownEnvironment = errors.New("unknown environment")
)

var endpoints = map[string]string{
	EnvironmentSandbox: "https://api-sandbox.y.uno/v1",
	EnvironmentProdUS:  "https://api.y.uno/v1",
	EnvironmentProdEU:  "https://api.eu.y.uno/v1",
}

// Profile holds the credentials and defaults for one Yuno environment.
type Profile struct {
	Environment      string `yaml:"environment,omitempty"`
	PublicAPIKey     string `yaml:"public_api_key,omitempty"`
	PrivateSecretKey string `yaml:"private_secret_key,omitempty"`
	AccountCode      string `yaml:"account_code,omitempty"`
	OrganizationCode string `yaml:"organization_code,omitempty"`
	AccountID        string `yaml:"account_id,omitempty"`
}

// Config is the on-disk configuration file.
type Config struct {
	DefaultProfile string             `yaml:"default_profile,omitempty"`
	Profiles       map[string]Profile `yaml:"profiles,omitempty"`
}

// Dir returns the configuration directory, honouring YUNO_CONFIG_DIR.
func Dir() string {
	if dir := os.Getenv(EnvConfigDir); dir != "" {
		return dir
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join(".config", "yuno-cli")
	}

	return filepath.Join(home, ".config", "yuno-cli")
}

// Path returns the full path of the configuration file.
func Path() string {
	return filepath.Join(Dir(), "config.yaml")
}

// Load reads the configuration file. A missing file is not an error: it yields
// an empty configuration.
func Load() (*Config, error) {
	cfg := &Config{Profiles: map[string]Profile{}}

	data, err := os.ReadFile(Path())
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return cfg, nil
		}

		return nil, fmt.Errorf("read config %s: %w", Path(), err)
	}

	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("parse config %s: %w", Path(), err)
	}

	if cfg.Profiles == nil {
		cfg.Profiles = map[string]Profile{}
	}

	return cfg, nil
}

// Save writes the configuration with 0600 permissions inside a 0700 directory.
func (c *Config) Save() error {
	dir := Dir()
	if err := os.MkdirAll(dir, dirPerm); err != nil {
		return fmt.Errorf("create config dir %s: %w", dir, err)
	}

	// MkdirAll respects umask, so tighten the permissions explicitly.
	if err := os.Chmod(dir, dirPerm); err != nil {
		return fmt.Errorf("chmod config dir %s: %w", dir, err)
	}

	data, err := yaml.Marshal(c)
	if err != nil {
		return fmt.Errorf("encode config: %w", err)
	}

	path := Path()
	if err := os.WriteFile(path, data, filePerm); err != nil {
		return fmt.Errorf("write config %s: %w", path, err)
	}

	if err := os.Chmod(path, filePerm); err != nil {
		return fmt.Errorf("chmod config %s: %w", path, err)
	}

	return nil
}

// ProfileName resolves which profile to use: the explicit flag value, then
// YUNO_PROFILE, then the configured default, then "default".
func (c *Config) ProfileName(explicit string) string {
	if explicit != "" {
		return explicit
	}

	if name := os.Getenv(EnvProfile); name != "" {
		return name
	}

	if c.DefaultProfile != "" {
		return c.DefaultProfile
	}

	return DefaultProfileName
}

// Profile returns the named profile with environment overrides applied. A
// profile missing from the file is still usable when the environment supplies
// both API keys.
func (c *Config) Profile(explicit string) (Profile, error) {
	name := c.ProfileName(explicit)
	p, ok := c.Profiles[name]

	applyEnvOverrides(&p)

	if !ok && p.PublicAPIKey == "" && p.PrivateSecretKey == "" {
		return Profile{}, fmt.Errorf("%q: %w (run `yuno-cli auth --profile %s`)", name, ErrProfileNotFound, name)
	}

	return p, nil
}

func applyEnvOverrides(p *Profile) {
	overrides := []struct {
		env    string
		target *string
	}{
		{EnvPublicAPIKey, &p.PublicAPIKey},
		{EnvPrivateSecretKey, &p.PrivateSecretKey},
		{EnvAccountCode, &p.AccountCode},
		{EnvOrganizationCode, &p.OrganizationCode},
		{EnvAccountID, &p.AccountID},
	}

	for _, o := range overrides {
		if v := os.Getenv(o.env); v != "" {
			*o.target = v
		}
	}
}

// Endpoint resolves the API base URL for the profile. YUNO_API_ENDPOINT wins
// over the profile environment.
func (p *Profile) Endpoint() (string, error) {
	if override := os.Getenv(EnvAPIEndpoint); override != "" {
		return strings.TrimRight(override, "/"), nil
	}

	environment := p.Environment
	if environment == "" {
		environment = EnvironmentSandbox
	}

	endpoint, ok := endpoints[environment]
	if !ok {
		return "", fmt.Errorf("%q: %w (want %s, %s or %s)",
			environment, ErrUnknownEnvironment, EnvironmentSandbox, EnvironmentProdUS, EnvironmentProdEU)
	}

	return endpoint, nil
}

// Validate reports whether the profile can be used to call the API.
func (p *Profile) Validate() error {
	var missing []string

	if p.PublicAPIKey == "" {
		missing = append(missing, "public_api_key")
	}

	if p.PrivateSecretKey == "" {
		missing = append(missing, "private_secret_key")
	}

	if len(missing) > 0 {
		return fmt.Errorf("%w: %s", ErrMissingCredentials, strings.Join(missing, ", "))
	}

	if p.Environment != "" {
		if _, ok := endpoints[p.Environment]; !ok {
			return fmt.Errorf("%q: %w (want %s, %s or %s)",
				p.Environment, ErrUnknownEnvironment, EnvironmentSandbox, EnvironmentProdUS, EnvironmentProdEU)
		}
	}

	return nil
}
