package cmd

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/pvarganov/yuno-cli/internal/config"
	"github.com/pvarganov/yuno-cli/internal/confirm"
)

// isolate points the config at a temporary directory and clears every override
// the ambient environment might carry.
func isolate(t *testing.T) {
	t.Helper()

	t.Setenv(config.EnvConfigDir, t.TempDir())

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
}

// seedProfile stores a usable sandbox profile.
func seedProfile(t *testing.T) {
	t.Helper()

	cfg := &config.Config{
		DefaultProfile: "sandbox",
		Profiles: map[string]config.Profile{
			"sandbox": {
				Environment:      config.EnvironmentSandbox,
				PublicAPIKey:     "pub-key",
				PrivateSecretKey: "sec-key",
			},
		},
	}

	if err := cfg.Save(); err != nil {
		t.Fatalf("save config: %v", err)
	}
}

// bodyTestCommand builds a throw-away command carrying the body flags.
func bodyTestCommand(t *testing.T, fields []FieldFlag, stdin string, args ...string) *cobra.Command {
	t.Helper()

	command := &cobra.Command{
		Use:  "test",
		RunE: func(*cobra.Command, []string) error { return nil },
	}

	registerBodyFlags(command.Flags())
	registerFieldFlags(command.Flags(), fields)

	command.SetIn(strings.NewReader(stdin))
	command.SetOut(io.Discard)
	command.SetErr(io.Discard)
	command.SetArgs(args)

	if err := command.Execute(); err != nil {
		t.Fatalf("parse flags: %v", err)
	}

	return command
}

// decodeBodyMap renders a built body back into a map for assertions.
func decodeBodyMap(t *testing.T, body any) map[string]any {
	t.Helper()

	payload, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("marshal body: %v", err)
	}

	var decoded map[string]any
	if err := json.Unmarshal(payload, &decoded); err != nil {
		t.Fatalf("unmarshal body: %v (%s)", err, payload)
	}

	return decoded
}

func TestBodyFromFlags_ReadsFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "body.json")
	if err := os.WriteFile(path, []byte(`{"email":"a@b.c"}`), 0o600); err != nil {
		t.Fatalf("write body file: %v", err)
	}

	command := bodyTestCommand(t, nil, "", "--file", path)

	body, err := bodyFromFlags(command, nil)
	if err != nil {
		t.Fatalf("bodyFromFlags failed: %v", err)
	}

	if got := decodeBodyMap(t, body)["email"]; got != "a@b.c" {
		t.Errorf("expected the file body, got %v", got)
	}
}

func TestBodyFromFlags_ReadsDataAndStdin(t *testing.T) {
	tests := []struct {
		name  string
		stdin string
		args  []string
	}{
		{name: "inline data", args: []string{"--data", `{"email":"a@b.c"}`}},
		{name: "stdin", stdin: `{"email":"a@b.c"}`, args: []string{"--data", stdinMarker}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			command := bodyTestCommand(t, nil, tt.stdin, tt.args...)

			body, err := bodyFromFlags(command, nil)
			if err != nil {
				t.Fatalf("bodyFromFlags failed: %v", err)
			}

			if got := decodeBodyMap(t, body)["email"]; got != "a@b.c" {
				t.Errorf("expected the decoded body, got %v", got)
			}
		})
	}
}

func TestBodyFromFlags_NoSourceYieldsNil(t *testing.T) {
	command := bodyTestCommand(t, []FieldFlag{{Flag: "email", Field: "email"}}, "")

	body, err := bodyFromFlags(command, []FieldFlag{{Flag: "email", Field: "email"}})
	if err != nil {
		t.Fatalf("bodyFromFlags failed: %v", err)
	}

	if body != nil {
		t.Errorf("expected no body, got %#v", body)
	}
}

func TestBodyFromFlags_MergesFieldFlagsIntoFileBody(t *testing.T) {
	fields := []FieldFlag{
		{Flag: "email", Field: "email"},
		{Flag: "country", Field: "country"},
		{Flag: "city", Field: "billing_address.city"},
	}

	command := bodyTestCommand(t, fields, "",
		"--data", `{"email":"old@b.c","merchant_customer_id":"m-1"}`,
		"--email", "new@b.c",
		"--city", "Lisbon")

	body, err := bodyFromFlags(command, fields)
	if err != nil {
		t.Fatalf("bodyFromFlags failed: %v", err)
	}

	decoded := decodeBodyMap(t, body)

	if decoded["email"] != "new@b.c" {
		t.Errorf("expected the flag to win over the file, got %v", decoded["email"])
	}

	if decoded["merchant_customer_id"] != "m-1" {
		t.Errorf("expected the untouched field to survive, got %v", decoded["merchant_customer_id"])
	}

	if _, ok := decoded["country"]; ok {
		t.Error("expected an unchanged flag to stay out of the body")
	}

	address, ok := decoded["billing_address"].(map[string]any)
	if !ok {
		t.Fatalf("expected a nested object, got %#v", decoded["billing_address"])
	}

	if address["city"] != "Lisbon" {
		t.Errorf("expected the nested field to be set, got %v", address["city"])
	}
}

func TestBodyFromFlags_KeepsExplicitEmptyValue(t *testing.T) {
	fields := []FieldFlag{{Flag: "email", Field: "email"}}

	command := bodyTestCommand(t, fields, "", "--email", "")

	body, err := bodyFromFlags(command, fields)
	if err != nil {
		t.Fatalf("bodyFromFlags failed: %v", err)
	}

	decoded := decodeBodyMap(t, body)

	value, ok := decoded["email"]
	if !ok {
		t.Fatalf("expected an explicit empty string to land in the body, got %#v", decoded)
	}

	if value != "" {
		t.Errorf("expected an empty string, got %v", value)
	}
}

func TestBodyFromFlags_TypedFields(t *testing.T) {
	fields := []FieldFlag{
		{Flag: "count", Field: "count", Kind: FieldInt},
		{Flag: "enabled", Field: "enabled", Kind: FieldBool},
		{Flag: "amount", Field: "amount.value", Kind: FieldNumber},
		{Flag: "tag", Field: "tags", Kind: FieldStringSlice},
		{Flag: "metadata", Field: "metadata", Kind: FieldJSON},
	}

	command := bodyTestCommand(t, fields, "",
		"--count", "3",
		"--enabled",
		"--amount", "10.5",
		"--tag", "a", "--tag", "b",
		"--metadata", `{"k":"v"}`)

	body, err := bodyFromFlags(command, fields)
	if err != nil {
		t.Fatalf("bodyFromFlags failed: %v", err)
	}

	decoded := decodeBodyMap(t, body)

	if decoded["count"] != float64(3) {
		t.Errorf("expected count 3, got %v", decoded["count"])
	}

	if decoded["enabled"] != true {
		t.Errorf("expected enabled true, got %v", decoded["enabled"])
	}

	amount, ok := decoded["amount"].(map[string]any)
	if !ok || amount["value"] != 10.5 {
		t.Errorf("expected amount.value 10.5, got %#v", decoded["amount"])
	}

	tags, ok := decoded["tags"].([]any)
	if !ok || len(tags) != 2 || tags[0] != "a" || tags[1] != "b" {
		t.Errorf("expected the tags array, got %#v", decoded["tags"])
	}

	metadata, ok := decoded["metadata"].(map[string]any)
	if !ok || metadata["k"] != "v" {
		t.Errorf("expected the decoded metadata object, got %#v", decoded["metadata"])
	}
}

func TestBodyFromFlags_Errors(t *testing.T) {
	fields := []FieldFlag{
		{Flag: "email", Field: "email"},
		{Flag: "metadata", Field: "metadata", Kind: FieldJSON},
		{Flag: "city", Field: "email.city"},
	}

	tests := []struct {
		name string
		args []string
		want string
	}{
		{name: "malformed body", args: []string{"--data", "{not json"}, want: "not valid json"},
		{
			name: "malformed json flag",
			args: []string{"--metadata", "{nope", "--data", "{}"},
			want: "--metadata is not valid json",
		},
		{
			name: "body is not an object",
			args: []string{"--data", `["a"]`, "--email", "a@b.c"},
			want: "not a json object",
		},
		{
			name: "nested path collides with a scalar",
			args: []string{"--data", `{"email":"a@b.c"}`, "--city", "Lisbon"},
			want: "is not a json object",
		},
		{name: "missing file", args: []string{"--file", "/nope/body.json"}, want: "read body file"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			command := bodyTestCommand(t, fields, "", tt.args...)

			_, err := bodyFromFlags(command, fields)
			if err == nil {
				t.Fatal("expected an error")
			}

			if !strings.Contains(err.Error(), tt.want) {
				t.Errorf("expected error containing %q, got %v", tt.want, err)
			}
		})
	}
}

func TestNewClientFromFlags_ResolvesProfileAndEndpoint(t *testing.T) {
	isolate(t)
	seedProfile(t)
	t.Setenv(config.EnvAPIEndpoint, "https://example.test/v1")

	root := NewRootCommand("test")

	client, err := newClientFromFlags(root)
	if err != nil {
		t.Fatalf("newClientFromFlags failed: %v", err)
	}

	if client.Endpoint() != "https://example.test/v1" {
		t.Errorf("expected the endpoint override to win, got %s", client.Endpoint())
	}
}

func TestNewClientFromFlags_Unauthenticated(t *testing.T) {
	isolate(t)

	root := NewRootCommand("test")

	_, err := newClientFromFlags(root)
	if err == nil {
		t.Fatal("expected an error when no profile is configured")
	}

	if !strings.Contains(err.Error(), "yuno-cli auth") {
		t.Errorf("expected a hint on how to authenticate, got %v", err)
	}
}

func TestNewClientFromFlags_UnknownProfile(t *testing.T) {
	isolate(t)
	seedProfile(t)

	var buf bytes.Buffer

	root := NewRootCommand("test")
	root.SetOut(&buf)
	root.SetErr(&buf)
	root.SetIn(strings.NewReader(""))
	root.SetArgs([]string{"raw", "GET", "/widgets", "--profile", "nope"})

	err := root.Execute()
	if err == nil {
		t.Fatal("expected an error for an unknown profile")
	}

	if !strings.Contains(err.Error(), "nope") {
		t.Errorf("expected the profile name in the error, got %v", err)
	}
}
