package api

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"
	"time"
)

var uuidV4Re = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)

// captureHeaders serves a fixed body and records the headers of every request.
func captureHeaders(t *testing.T, seen *[]http.Header) *httptest.Server {
	t.Helper()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		*seen = append(*seen, r.Header.Clone())
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	t.Cleanup(srv.Close)

	return srv
}

func TestIdempotencyKeyPerMethod(t *testing.T) {
	tests := []struct {
		method string
		body   any
		want   bool
	}{
		{http.MethodPost, map[string]any{"a": 1}, true},
		{http.MethodPatch, map[string]any{"a": 1}, true},
		{http.MethodPut, map[string]any{"a": 1}, true},
		{http.MethodGet, nil, false},
		{http.MethodDelete, nil, false},
	}

	for _, tt := range tests {
		t.Run(tt.method, func(t *testing.T) {
			var seen []http.Header

			srv := captureHeaders(t, &seen)
			c, _ := newTestClient(t, srv, testProfile())

			if _, err := c.DoRaw(t.Context(), Request{Method: tt.method, Path: "/payments", Body: tt.body}); err != nil {
				t.Fatalf("DoRaw: %v", err)
			}

			got := seen[0].Get(HeaderIdempotencyKey)

			switch {
			case tt.want && !uuidV4Re.MatchString(got):
				t.Errorf("%s idempotency key = %q, want a v4 uuid", tt.method, got)
			case !tt.want && got != "":
				t.Errorf("%s carried idempotency key %q, want none", tt.method, got)
			}
		})
	}
}

func TestIdempotencyKeyIsUniquePerCall(t *testing.T) {
	var seen []http.Header

	srv := captureHeaders(t, &seen)
	c, _ := newTestClient(t, srv, testProfile())

	for range 2 {
		if _, err := c.DoRaw(t.Context(), Request{Method: http.MethodPost, Path: "/payments", Body: map[string]any{}}); err != nil {
			t.Fatalf("DoRaw: %v", err)
		}
	}

	if a, b := seen[0].Get(HeaderIdempotencyKey), seen[1].Get(HeaderIdempotencyKey); a == b {
		t.Errorf("two calls shared idempotency key %q", a)
	}
}

func TestIdempotencyKeyStableAcrossRetries(t *testing.T) {
	var seen []http.Header

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen = append(seen, r.Header.Clone())

		if len(seen) == 1 {
			w.WriteHeader(http.StatusInternalServerError)

			return
		}

		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()

	c, _ := newTestClient(t, srv, testProfile())
	c.sleep = func(_ context.Context, _ time.Duration) error { return nil }

	if _, err := c.DoRaw(t.Context(), Request{Method: http.MethodPost, Path: "/payments", Body: map[string]any{}}); err != nil {
		t.Fatalf("DoRaw: %v", err)
	}

	if len(seen) != 2 {
		t.Fatalf("attempts = %d, want 2", len(seen))
	}

	if a, b := seen[0].Get(HeaderIdempotencyKey), seen[1].Get(HeaderIdempotencyKey); a != b {
		t.Errorf("retry changed idempotency key: %q then %q", a, b)
	}
}

func TestWithIdempotencyKeyOverrides(t *testing.T) {
	var seen []http.Header

	srv := captureHeaders(t, &seen)
	c, _ := newTestClient(t, srv, testProfile(), WithIdempotencyKey("fixed-key"))

	if _, err := c.DoRaw(t.Context(), Request{Method: http.MethodPost, Path: "/payments", Body: map[string]any{}}); err != nil {
		t.Fatalf("DoRaw: %v", err)
	}

	if got := seen[0].Get(HeaderIdempotencyKey); got != "fixed-key" {
		t.Errorf("idempotency key = %q, want %q", got, "fixed-key")
	}
}

func TestRequestHeaderOverridesIdempotencyKey(t *testing.T) {
	var seen []http.Header

	srv := captureHeaders(t, &seen)
	c, _ := newTestClient(t, srv, testProfile())

	req := Request{
		Method:  http.MethodPost,
		Path:    "/payments",
		Body:    map[string]any{},
		Headers: map[string]string{HeaderIdempotencyKey: "per-request"},
	}

	if _, err := c.DoRaw(t.Context(), req); err != nil {
		t.Fatalf("DoRaw: %v", err)
	}

	if got := seen[0].Get(HeaderIdempotencyKey); got != "per-request" {
		t.Errorf("idempotency key = %q, want %q", got, "per-request")
	}
}

func TestNewIdempotencyKeyFormat(t *testing.T) {
	key := newIdempotencyKey()

	if !uuidV4Re.MatchString(key) {
		t.Fatalf("newIdempotencyKey() = %q, want a v4 uuid", key)
	}

	if strings.ToLower(key) != key {
		t.Errorf("key %q is not lowercase", key)
	}

	if bytes.Contains([]byte(key), []byte(" ")) {
		t.Errorf("key %q contains a space", key)
	}
}
