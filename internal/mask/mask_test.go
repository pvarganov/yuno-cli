package mask_test

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"github.com/pvarganov/yuno-cli/internal/mask"
)

func TestSecret(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"empty", "", ""},
		{"shorter than prefix", "abc", "****"},
		{"exactly the prefix", "abcd", "****"},
		{"long key", "sandbox_gAAAAABm_secret_tail", "sand****"},
		{"unicode prefix", "ключ_секретный", "ключ****"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := mask.Secret(tt.in); got != tt.want {
				t.Errorf("Secret(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestSecretNeverLeaksTail(t *testing.T) {
	const key = "prod_api_key_0123456789abcdef"

	got := mask.Secret(key)
	if strings.Contains(got, "0123456789") {
		t.Fatalf("Secret(%q) = %q, leaks the secret tail", key, got)
	}
}

func TestJSONMasksSensitiveFields(t *testing.T) {
	in := map[string]any{
		"card": map[string]any{
			"number":        "4111111111111111",
			"security_code": "123",
			"holder_name":   "JOHN DOE",
		},
		"cvv":                "999",
		"token":              "tok_1234567890",
		"vaulted_token":      "vtok_1234567890",
		"private-secret-key": "sand_secret_value",
		"public-api-key":     "sand_public_value",
		"amount":             map[string]any{"value": 10.5, "currency": "USD"},
	}

	want := map[string]any{
		"card": map[string]any{
			"number":        "4111****",
			"security_code": "****",
			"holder_name":   "JOHN DOE",
		},
		"cvv":                "****",
		"token":              "tok_****",
		"vaulted_token":      "vtok****",
		"private-secret-key": "sand****",
		"public-api-key":     "sand****",
		"amount":             map[string]any{"value": 10.5, "currency": "USD"},
	}

	got := mask.JSON(in)
	if !reflect.DeepEqual(got, want) {
		t.Errorf("JSON() = %#v, want %#v", got, want)
	}
}

func TestJSONFieldNamesAreCaseAndSeparatorInsensitive(t *testing.T) {
	in := map[string]any{
		"Number":           "4111111111111111",
		"SecurityCode":     "123",
		"PrivateSecretKey": "sand_secret_value",
	}

	got, ok := mask.JSON(in).(map[string]any)
	if !ok {
		t.Fatalf("JSON() returned %T, want map[string]any", mask.JSON(in))
	}

	for key, want := range map[string]string{
		"Number":           "4111****",
		"SecurityCode":     "****",
		"PrivateSecretKey": "sand****",
	} {
		if got[key] != want {
			t.Errorf("JSON()[%q] = %v, want %q", key, got[key], want)
		}
	}
}

func TestJSONDoesNotMutateInput(t *testing.T) {
	in := map[string]any{"number": "4111111111111111"}

	_ = mask.JSON(in)

	if in["number"] != "4111111111111111" {
		t.Errorf("input mutated: %v", in["number"])
	}
}

func TestJSONEdgeCases(t *testing.T) {
	tests := []struct {
		name string
		in   any
		want any
	}{
		{"nil", nil, nil},
		{"plain string", "number", "number"},
		{"number value stays untouched", map[string]any{"number": 4111.0}, map[string]any{"number": 4111.0}},
		{"nil value", map[string]any{"token": nil}, map[string]any{"token": nil}},
		{
			"nested arrays",
			[]any{
				map[string]any{"token": "tok_abcdefgh"},
				[]any{map[string]any{"cvv": "1234"}},
			},
			[]any{
				map[string]any{"token": "tok_****"},
				[]any{map[string]any{"cvv": "****"}},
			},
		},
		{
			"already masked input",
			map[string]any{"token": "tok_****"},
			map[string]any{"token": "tok_****"},
		},
		{"empty secret", map[string]any{"token": ""}, map[string]any{"token": ""}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := mask.JSON(tt.in); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("JSON(%#v) = %#v, want %#v", tt.in, got, tt.want)
			}
		})
	}
}

func TestJSONOverDecodedPayload(t *testing.T) {
	const payload = `{"payment":{"card":{"number":"4111111111111111","cvv":"737"}},"id":"pay_1"}`

	var decoded any
	if err := json.Unmarshal([]byte(payload), &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	out, err := json.Marshal(mask.JSON(decoded))
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	got := string(out)
	if strings.Contains(got, "4111111111111111") || strings.Contains(got, "737") {
		t.Fatalf("masked payload still contains secrets: %s", got)
	}

	if !strings.Contains(got, `"id":"pay_1"`) {
		t.Fatalf("masked payload lost non-sensitive fields: %s", got)
	}
}

func TestIsSensitive(t *testing.T) {
	tests := []struct {
		field string
		want  bool
	}{
		{"number", true},
		{"security_code", true},
		{"cvv", true},
		{"token", true},
		{"vaulted_token", true},
		{"private-secret-key", true},
		{"public-api-key", true},
		{"phone_number", false},
		{"document_number", false},
		{"id", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.field, func(t *testing.T) {
			if got := mask.IsSensitive(tt.field); got != tt.want {
				t.Errorf("IsSensitive(%q) = %v, want %v", tt.field, got, tt.want)
			}
		})
	}
}
