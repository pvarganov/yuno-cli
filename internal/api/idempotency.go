package api

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
)

// mutatingMethods are the methods Yuno treats as state changing, and therefore
// the only ones that carry an idempotency key.
var mutatingMethods = map[string]struct{}{
	http.MethodPost:  {},
	http.MethodPatch: {},
	http.MethodPut:   {},
}

// isMutating reports whether requests with this method must carry an
// idempotency key.
func isMutating(method string) bool {
	_, ok := mutatingMethods[method]

	return ok
}

// newIdempotencyKey returns a random RFC 4122 version 4 UUID.
func newIdempotencyKey() string {
	var b [16]byte

	// rand.Read never fails on the platforms we build for.
	_, _ = rand.Read(b[:])

	b[6] = (b[6] & 0x0f) | 0x40 // version 4
	b[8] = (b[8] & 0x3f) | 0x80 // variant 10

	var out [36]byte

	hex.Encode(out[0:8], b[0:4])
	out[8] = '-'
	hex.Encode(out[9:13], b[4:6])
	out[13] = '-'
	hex.Encode(out[14:18], b[6:8])
	out[18] = '-'
	hex.Encode(out[19:23], b[8:10])
	out[23] = '-'
	hex.Encode(out[24:36], b[10:16])

	return string(out[:])
}

// idempotencyKeyFor returns the key a request of this method should send, or an
// empty string when the method must not carry one. Retries of the same call
// reuse the returned value, which is what makes them safe.
func (c *Client) idempotencyKeyFor(method string) string {
	if !isMutating(method) {
		return ""
	}

	if c.idempotencyKey != "" {
		return c.idempotencyKey
	}

	return newIdempotencyKey()
}
