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
	"number":           {},
	"securitycode":     {},
	"cvv":              {},
	"token":            {},
	"vaultedtoken":     {},
	"networktoken":     {},
	"cryptogram":       {},
	"privatesecretkey": {},
	"publicapikey":     {},
	"accesstoken":      {},
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

// JSON walks a decoded JSON tree and returns a copy with the value of every
// sensitive field replaced by Secret. Non-string values and unknown fields are
// copied through untouched; the input is never modified.
func JSON(v any) any {
	switch value := v.(type) {
	case map[string]any:
		out := make(map[string]any, len(value))
		for key, item := range value {
			if s, ok := item.(string); ok && IsSensitive(key) {
				out[key] = Secret(s)

				continue
			}

			out[key] = JSON(item)
		}

		return out
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
