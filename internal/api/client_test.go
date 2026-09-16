package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/pvarganov/yuno-cli/internal/config"
)

func testProfile() *config.Profile {
	return &config.Profile{
		Environment:      config.EnvironmentSandbox,
		PublicAPIKey:     "pub-key",
		PrivateSecretKey: "secret-key",
	}
}

// newTestClient builds a client pointed at srv with sleeping disabled.
func newTestClient(t *testing.T, srv *httptest.Server, profile *config.Profile, opts ...Option) (*Client, *[]time.Duration) {
	t.Helper()

	var slept []time.Duration

	all := append([]Option{WithEndpoint(srv.URL)}, opts...)

	c, err := NewClient(profile, all...)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	c.sleep = func(ctx context.Context, d time.Duration) error {
		slept = append(slept, d)

		return ctx.Err()
	}

	return c, &slept
}

func TestClientSendsAuthHeaders(t *testing.T) {
	var got http.Header

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = r.Header.Clone()
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()

	profile := testProfile()
	profile.AccountCode = "acc-code"
	profile.OrganizationCode = "org-code"

	c, _ := newTestClient(t, srv, profile)

	if _, err := c.DoRaw(t.Context(), Request{Method: http.MethodPost, Path: "/payments", Body: map[string]any{"a": 1}}); err != nil {
		t.Fatalf("DoRaw: %v", err)
	}

	checks := map[string]string{
		HeaderPublicAPIKey:     "pub-key",
		HeaderPrivateSecretKey: "secret-key",
		HeaderAccountCode:      "acc-code",
		HeaderOrganizationCode: "org-code",
		"Content-Type":         "application/json",
		"Accept":               "application/json",
	}

	for header, want := range checks {
		if v := got.Get(header); v != want {
			t.Errorf("header %s = %q, want %q", header, v, want)
		}
	}
}

func TestClientOmitsOptionalHeaders(t *testing.T) {
	var got http.Header

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = r.Header.Clone()
		_, _ = w.Write([]byte(`[]`))
	}))
	defer srv.Close()

	c, _ := newTestClient(t, srv, testProfile())

	if _, err := c.DoRaw(t.Context(), Request{Method: http.MethodGet, Path: "/routing"}); err != nil {
		t.Fatalf("DoRaw: %v", err)
	}

	for _, header := range []string{HeaderAccountCode, HeaderOrganizationCode, "Content-Type"} {
		if v := got.Get(header); v != "" {
			t.Errorf("header %s = %q, want empty", header, v)
		}
	}
}

func TestClientExtraHeadersAndQuery(t *testing.T) {
	var (
		gotPath  string
		gotQuery url.Values
		gotExtra string
	)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotQuery = r.URL.Query()
		gotExtra = r.Header.Get("X-Custom")
		_, _ = w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	c, _ := newTestClient(t, srv, testProfile())

	req := Request{
		Method:  http.MethodGet,
		Path:    "routing",
		Query:   url.Values{"account_id": {"uuid-1"}},
		Headers: map[string]string{"X-Custom": "yes"},
	}

	if _, err := c.DoRaw(t.Context(), req); err != nil {
		t.Fatalf("DoRaw: %v", err)
	}

	if gotPath != "/routing" {
		t.Errorf("path = %q, want /routing", gotPath)
	}

	if gotQuery.Get("account_id") != "uuid-1" {
		t.Errorf("query account_id = %q", gotQuery.Get("account_id"))
	}

	if gotExtra != "yes" {
		t.Errorf("X-Custom = %q, want yes", gotExtra)
	}
}

func TestNewClientErrors(t *testing.T) {
	t.Setenv(config.EnvAPIEndpoint, "")

	if _, err := NewClient(&config.Profile{}); !errors.Is(err, config.ErrMissingCredentials) {
		t.Errorf("err = %v, want ErrMissingCredentials", err)
	}

	bad := testProfile()
	bad.Environment = "mars"

	if _, err := NewClient(bad); !errors.Is(err, config.ErrUnknownEnvironment) {
		t.Errorf("err = %v, want ErrUnknownEnvironment", err)
	}
}

func TestDoDecodesJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"id":"pay_1"}`))
	}))
	defer srv.Close()

	c, _ := newTestClient(t, srv, testProfile())

	type payment struct {
		ID string `json:"id"`
	}

	got, err := Do[payment](t.Context(), c, Request{Method: http.MethodGet, Path: "/payments/pay_1"})
	if err != nil {
		t.Fatalf("Do: %v", err)
	}

	if got.ID != "pay_1" {
		t.Errorf("id = %q, want pay_1", got.ID)
	}
}

func TestDoEmptyBodyAndBadJSON(t *testing.T) {
	t.Run("empty body", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusNoContent)
		}))
		defer srv.Close()

		c, _ := newTestClient(t, srv, testProfile())

		got, err := Do[map[string]any](t.Context(), c, Request{Method: http.MethodDelete, Path: "/customers/1"})
		if err != nil {
			t.Fatalf("Do: %v", err)
		}

		if got != nil {
			t.Errorf("got = %v, want nil", got)
		}
	})

	t.Run("malformed body", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte(`not json`))
		}))
		defer srv.Close()

		c, _ := newTestClient(t, srv, testProfile())

		if _, err := Do[map[string]any](t.Context(), c, Request{Method: http.MethodGet, Path: "/x"}); err == nil {
			t.Fatal("want decode error, got nil")
		}
	})
}

func TestEncodeBody(t *testing.T) {
	tests := []struct {
		name string
		body any
		want string
	}{
		{"nil", nil, ""},
		{"bytes", []byte(`{"raw":1}`), `{"raw":1}`},
		{"raw message", json.RawMessage(`{"raw":2}`), `{"raw":2}`},
		{"struct", map[string]string{"a": "b"}, `{"a":"b"}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := encodeBody(tt.body)
			if err != nil {
				t.Fatalf("encodeBody: %v", err)
			}

			if string(got) != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}

	if _, err := encodeBody(make(chan int)); err == nil {
		t.Error("want error for unmarshalable body")
	}
}

func TestClientSendsBodyOnEveryRetry(t *testing.T) {
	var bodies []string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		data := make([]byte, r.ContentLength)
		_, _ = r.Body.Read(data)
		bodies = append(bodies, string(data))

		if len(bodies) < 2 {
			w.WriteHeader(http.StatusInternalServerError)

			return
		}

		_, _ = w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	c, _ := newTestClient(t, srv, testProfile())

	if _, err := c.DoRaw(t.Context(), Request{Method: http.MethodPost, Path: "/p", Body: map[string]int{"n": 1}}); err != nil {
		t.Fatalf("DoRaw: %v", err)
	}

	if len(bodies) != 2 {
		t.Fatalf("attempts = %d, want 2", len(bodies))
	}

	for i, b := range bodies {
		if b != `{"n":1}` {
			t.Errorf("attempt %d body = %q", i, b)
		}
	}
}

func TestClientTransportErrorRetried(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {}))
	srv.Close() // nothing is listening, every attempt fails at the transport level

	c, slept := newTestClient(t, srv, testProfile())

	_, err := c.DoRaw(t.Context(), Request{Method: http.MethodGet, Path: "/routing"})
	if err == nil {
		t.Fatal("want transport error, got nil")
	}

	if !strings.Contains(err.Error(), "giving up after 4 attempts") {
		t.Errorf("err = %v, want exhausted retries", err)
	}

	if len(*slept) != 3 {
		t.Errorf("sleeps = %d, want 3", len(*slept))
	}
}

func TestClientContextCancelledNotRetried(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	c, slept := newTestClient(t, srv, testProfile())

	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	_, err := c.DoRaw(ctx, Request{Method: http.MethodGet, Path: "/routing"})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("err = %v, want context.Canceled", err)
	}

	if len(*slept) != 0 {
		t.Errorf("sleeps = %d, want 0", len(*slept))
	}
}
