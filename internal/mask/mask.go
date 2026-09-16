// Package mask hides credentials and card data before they reach a terminal,
// a log line or an HTTP dump.
package mask

import (
	"strings"
)

// prefixLen is the number of leading runes kept visible by Secret, enough to
// tell two keys apart without disclosing either of them.
const prefixLen = 4

// maskChars replaces everything past the visible prefix. Its length is fixed so
// that the masked value never hints at the length of the original.
const maskChars = "****"

// sensitiveFields lists the JSON field names whose values are secrets. Lookup is
// done on the normalised name, so `security_code`, `securityCode` and
// `security-code` all match the same entry.
var sensitiveFields = map[string]struct{}{
	"number":                       {},
	"securitycode":                 {},
	"cvv":                          {},
	"cardpin":                      {},
	"token":                        {},
	"vaultedtoken":                 {},
	"networktoken":                 {},
	"paymenttoken":                 {},
	"providertoken":                {},
	"paymentmethodtoken":           {},
	"cryptogram":                   {},
	"privatesecretkey":             {},
	"publicapikey":                 {},
	"accesstoken":                  {},
	"apikey":                       {},
	"secret":                       {},
	"hmacclientsecret":             {},
	"clientsecret":                 {},
	"oauth2clientsecret":           {},
	"accountnumber":                {},
	"routingnumber":                {},
	"iban":                         {},
	"yunoproxyauthsecretkey":       {},
	"paymentprocessingkey":         {},
	"paymentprocessingcertificate": {},
	"merchantidentitykey":          {},
	"merchantidentitycertificate":  {},
	"merchantidentitypassword":     {},
}

// secretParamMarkers mark a param_id as naming a credential. Connection
// parameters carry the credential in a neutrally named "value" field, so the
// pair can only be judged by its param_id.
var secretParamMarkers = []string{
	"secret",
	"password",
	"passphrase",
	"token",
	"key",
	"certificate",
	"cryptogram",
	"credential",
	"signature",
}

// Secret keeps a short prefix of s and masks the rest. Values that are empty or
// no longer than the prefix are masked entirely.
func Secret(s string) string {
	if s == "" {
		return ""
	}

	runes := []rune(s)
	if len(runes) <= prefixLen {
		return maskChars
	}

	return string(runes[:prefixLen]) + maskChars
}

// IsSensitive reports whether a JSON field of this name carries a secret.
func IsSensitive(field string) bool {
	_, ok := sensitiveFields[normalise(field)]

	return ok
}

// IsSensitiveParam reports whether a param_id names a credential.
func IsSensitiveParam(name string) bool {
	normalised := normalise(name)

	if IsSensitive(normalised) {
		return true
	}

	for _, marker := range secretParamMarkers {
		if strings.Contains(normalised, marker) {
			return true
		}
	}

	return false
}

// JSON walks a decoded JSON tree and returns a copy with the value of every
// sensitive field replaced by Secret. Non-string values and unknown fields are
// copied through untouched; the input is never modified.
func JSON(v any) any {
	switch value := v.(type) {
	case map[string]any:
		return jsonMap(value)
	case []any:
		out := make([]any, len(value))
		for i, item := range value {
			out[i] = JSON(item)
		}

		return out
	default:
		return v
	}
}

// jsonMap masks the sensitive fields of one object. A {param_id, value} pair
// whose param_id names a credential has its value masked too.
func jsonMap(value map[string]any) map[string]any {
	maskValue := isSecretParamPair(value)
	out := make(map[string]any, len(value))

	for key, item := range value {
		s, isString := item.(string)
		if isString && (IsSensitive(key) || (maskValue && normalise(key) == "value")) {
			out[key] = Secret(s)

			continue
		}

		out[key] = JSON(item)
	}

	return out
}

// isSecretParamPair reports whether an object is a connection parameter whose
// param_id names a credential.
func isSecretParamPair(value map[string]any) bool {
	name, ok := value["param_id"].(string)

	return ok && IsSensitiveParam(name)
}

// normalise lowercases a field name and drops the separators, so that snake,
// kebab and camel spellings of the same field collapse to one key.
func normalise(field string) string {
	var b strings.Builder

	b.Grow(len(field))

	for _, r := range strings.ToLower(field) {
		if r == '_' || r == '-' {
			continue
		}

		b.WriteRune(r)
	}

	return b.String()
}
