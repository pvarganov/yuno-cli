package cmd_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/pvarganov/yuno-cli/internal/config"
)

func seedProfiles(t *testing.T) {
	t.Helper()

	writeConfig(t, &config.Config{
		DefaultProfile: "sandbox",
		Profiles: map[string]config.Profile{
			"sandbox": {Environment: config.EnvironmentSandbox, PublicAPIKey: "pub-sandbox"},
			"prod-us": {Environment: config.EnvironmentProdUS, PublicAPIKey: "pub-prod"},
		},
	})
}

func TestProfileList(t *testing.T) {
	isolateConfig(t)
	seedProfiles(t)

	out, err := runCLI(t, "", "profile", "list")
	if err != nil {
		t.Fatalf("profile list failed: %v (%s)", err, out)
	}

	for _, want := range []string{"sandbox", "prod-us", "NAME", "DEFAULT"} {
		if !strings.Contains(out, want) {
			t.Errorf("expected %q in the output, got: %s", want, out)
		}
	}

	if strings.Contains(out, "pub-sandbox") {
		t.Errorf("profile list must mask the public key, got: %s", out)
	}
}

func TestProfileList_Empty(t *testing.T) {
	isolateConfig(t)

	_, err := runCLI(t, "", "profile", "list")
	if err == nil || !strings.Contains(err.Error(), "no profiles configured") {
		t.Fatalf("expected a 'no profiles configured' error, got %v", err)
	}
}

func TestProfileUse(t *testing.T) {
	isolateConfig(t)
	seedProfiles(t)

	if _, err := runCLI(t, "", "profile", "use", "prod-us"); err != nil {
		t.Fatalf("profile use failed: %v", err)
	}

	cfg, _ := config.Load()
	if cfg.DefaultProfile != "prod-us" {
		t.Errorf("expected the default profile to be prod-us, got %q", cfg.DefaultProfile)
	}
}

func TestProfileUse_Unknown(t *testing.T) {
	isolateConfig(t)
	seedProfiles(t)

	_, err := runCLI(t, "", "profile", "use", "nope")
	if !errors.Is(err, config.ErrProfileNotFound) {
		t.Fatalf("expected a profile-not-found error, got %v", err)
	}
}

func TestProfileDelete(t *testing.T) {
	isolateConfig(t)
	seedProfiles(t)

	if _, err := runCLI(t, "", "profile", "delete", "sandbox"); err != nil {
		t.Fatalf("profile delete failed: %v", err)
	}

	cfg, _ := config.Load()
	if _, ok := cfg.Profiles["sandbox"]; ok {
		t.Error("expected the sandbox profile to be removed")
	}

	if cfg.DefaultProfile == "sandbox" {
		t.Error("deleting the default profile must clear default_profile")
	}
}

func TestProfileDelete_Unknown(t *testing.T) {
	isolateConfig(t)
	seedProfiles(t)

	_, err := runCLI(t, "", "profile", "delete", "nope")
	if !errors.Is(err, config.ErrProfileNotFound) {
		t.Fatalf("expected a profile-not-found error, got %v", err)
	}
}

func TestProfileList_JSON(t *testing.T) {
	isolateConfig(t)
	seedProfiles(t)

	out, err := runCLI(t, "", "profile", "list", "--json")
	if err != nil {
		t.Fatalf("profile list --json failed: %v", err)
	}

	if !strings.Contains(out, `"name": "prod-us"`) {
		t.Errorf("expected json rows, got: %s", out)
	}

	if strings.Contains(out, "pub-prod") {
		t.Errorf("json output must mask the public key, got: %s", out)
	}
}
