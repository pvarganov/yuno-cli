package api

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestVerboseDumpsRequestAndResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"pay-1","status":"SUCCEEDED"}`))
	}))
	defer srv.Close()

	var out bytes.Buffer

	c, _ := newTestClient(t, srv, testProfile(), WithVerbose(&out))

	if _, err := c.DoRaw(t.Context(), Request{Method: http.MethodPost, Path: "/payments", Body: map[string]any{"amount": 10}}); err != nil {
		t.Fatalf("DoRaw: %v", err)
	}

	dump := out.String()

	for _, want := range []string{"> POST /payments", "> Public-Api-Key", "< HTTP/1.1 200 OK", `"status":"SUCCEEDED"`} {
		if !strings.Contains(dump, want) {
			t.Errorf("dump missing %q:\n%s", want, dump)
		}
	}
}

func TestVerboseMasksCredentialsAndCardData(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"card":{"number":"4111111111111111"},"vaulted_token":"vt-super-secret"}`))
	}))
	defer srv.Close()

	var out bytes.Buffer

	profile := testProfile()
	profile.PublicAPIKey = "public-key-value"
	profile.PrivateSecretKey = "private-key-value"

	c, _ := newTestClient(t, srv, profile, WithVerbose(&out))

	body := map[string]any{"card": map[string]any{"number": "4242424242424242", "security_code": "123"}}
	if _, err := c.DoRaw(t.Context(), Request{Method: http.MethodPost, Path: "/payments", Body: body}); err != nil {
		t.Fatalf("DoRaw: %v", err)
	}

	dump := out.String()

	secrets := []string{
		"public-key-value", "private-key-value",
		"4242424242424242", "4111111111111111", "vt-super-secret", `"123"`,
	}
	for _, secret := range secrets {
		if strings.Contains(dump, secret) {
			t.Errorf("dump leaked %q:\n%s", secret, dump)
		}
	}

	if !strings.Contains(dump, "publ****") {
		t.Errorf("dump missing masked public key:\n%s", dump)
	}
}

func TestVerboseOffByDefault(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	var out bytes.Buffer

	c, _ := newTestClient(t, srv, testProfile())
	c.verboseOut = &out

	if _, err := c.DoRaw(t.Context(), Request{Method: http.MethodGet, Path: "/payments/pay-1"}); err != nil {
		t.Fatalf("DoRaw: %v", err)
	}

	if out.Len() != 0 {
		t.Errorf("non-verbose client wrote %q", out.String())
	}
}

func TestWithVerboseNilWriterDisablesDump(t *testing.T) {
	c, err := NewClient(testProfile(), WithVerbose(nil))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	if c.verbose {
		t.Error("WithVerbose(nil) enabled verbose mode")
	}

	// A disabled dump must stay a no-op rather than panic on a nil writer.
	c.dumpRequest(nil)
}

func TestVerboseDumpsErrorResponses(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"code":"INVALID","message":"bad request"}`))
	}))
	defer srv.Close()

	var out bytes.Buffer

	c, _ := newTestClient(t, srv, testProfile(), WithVerbose(&out))

	if _, err := c.DoRaw(t.Context(), Request{Method: http.MethodGet, Path: "/payments/pay-1"}); err == nil {
		t.Fatal("DoRaw: want error")
	}

	if !strings.Contains(out.String(), "400 Bad Request") {
		t.Errorf("dump missing the failed status:\n%s", out.String())
	}
}

func TestSanitizeDumpLeavesPlainBodies(t *testing.T) {
	tests := []struct {
		name string
		dump string
		want string
	}{
		{
			name: "headers only",
			dump: "GET /payments HTTP/1.1\r\nAccept: application/json\r\n",
			want: "GET /payments HTTP/1.1\r\nAccept: application/json\r\n",
		},
		{
			name: "non json body",
			dump: "POST /payments HTTP/1.1\r\n\r\nnot json",
			want: "POST /payments HTTP/1.1\r\n\r\nnot json",
		},
		{
			name: "empty body",
			dump: "GET /payments HTTP/1.1\r\n\r\n",
			want: "GET /payments HTTP/1.1\r\n\r\n",
		},
		{
			name: "credential header",
			dump: "GET /payments HTTP/1.1\r\nprivate-secret-key: abcdefgh\r\n\r\n",
			want: "GET /payments HTTP/1.1\r\nprivate-secret-key: abcd****\r\n\r\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := string(sanitizeDump([]byte(tt.dump))); got != tt.want {
				t.Errorf("sanitizeDump(%q) = %q, want %q", tt.dump, got, tt.want)
			}
		})
	}
}
