package cmd_test

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pvarganov/yuno-cli/internal/api"
	"github.com/pvarganov/yuno-cli/internal/config"
)

// capturedRequest records what the fake Yuno server received.
type capturedRequest struct {
	Method string
	Path   string
	Query  string
	Header http.Header
	Body   string
}

// startAPI spins up a fake Yuno API, points the CLI at it and records the last
// request it served.
func startAPI(t *testing.T, status int, response string) *capturedRequest {
	t.Helper()

	got := &capturedRequest{}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)

		got.Method = r.Method
		got.Path = r.URL.Path
		got.Query = r.URL.RawQuery
		got.Header = r.Header.Clone()
		got.Body = string(body)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_, _ = io.WriteString(w, response)
	}))
	t.Cleanup(server.Close)

	t.Setenv(config.EnvAPIEndpoint, server.URL+"/v1")

	return got
}

// seedCredentials stores a usable sandbox profile.
func seedCredentials(t *testing.T) {
	t.Helper()

	writeConfig(t, &config.Config{
		DefaultProfile: "sandbox",
		Profiles: map[string]config.Profile{
			"sandbox": {
				Environment:      config.EnvironmentSandbox,
				PublicAPIKey:     "pub-key",
				PrivateSecretKey: "sec-key",
			},
		},
	})
}

func TestRawGet_HitsTheRightURL(t *testing.T) {
	isolateConfig(t)
	seedCredentials(t)

	got := startAPI(t, http.StatusOK, `{"items":[{"id":"r-1"}]}`)

	out, err := runCLI(t, "", "raw", "GET", "/routing",
		"--query", "account_id=11111111-2222-3333-4444-555555555555",
		"--query", "payment_method=CARD")
	if err != nil {
		t.Fatalf("raw get failed: %v (%s)", err, out)
	}

	if got.Method != http.MethodGet {
		t.Errorf("expected GET, got %s", got.Method)
	}

	if got.Path != "/v1/routing" {
		t.Errorf("expected path /v1/routing, got %s", got.Path)
	}

	wantQuery := "account_id=11111111-2222-3333-4444-555555555555&payment_method=CARD"
	if got.Query != wantQuery {
		t.Errorf("expected query %q, got %q", wantQuery, got.Query)
	}

	if got.Header.Get(api.HeaderPublicAPIKey) != "pub-key" {
		t.Errorf("expected the public key header, got %q", got.Header.Get(api.HeaderPublicAPIKey))
	}

	if !strings.Contains(out, `"r-1"`) {
		t.Errorf("expected the response body in the output, got: %s", out)
	}
}

func TestRawGet_FormatsAsIndentedJSON(t *testing.T) {
	isolateConfig(t)
	seedCredentials(t)
	startAPI(t, http.StatusOK, `{"id":"r-1","enabled":true}`)

	out, err := runCLI(t, "", "raw", "GET", "routing/r-1")
	if err != nil {
		t.Fatalf("raw get failed: %v (%s)", err, out)
	}

	if !strings.Contains(out, "\n  \"id\": \"r-1\"") {
		t.Errorf("expected indented JSON, got: %s", out)
	}

	var decoded map[string]any
	if err := json.Unmarshal([]byte(out), &decoded); err != nil {
		t.Fatalf("output should be valid JSON: %v (%s)", err, out)
	}
}

func TestRawGet_NeverPrompts(t *testing.T) {
	isolateConfig(t)
	seedCredentials(t)
	startAPI(t, http.StatusOK, `{}`)

	// stdin is not a terminal: a prompt would fail instead of hanging.
	out, err := runCLI(t, "", "raw", "GET", "/routing")
	if err != nil {
		t.Fatalf("a read-only raw call must not be gated: %v (%s)", err, out)
	}

	if strings.Contains(out, "About to send") {
		t.Errorf("GET must not print a confirmation preview, got: %s", out)
	}
}

func TestRawPost_SendsFileBody(t *testing.T) {
	isolateConfig(t)
	seedCredentials(t)

	got := startAPI(t, http.StatusCreated, `{"id":"cus-1"}`)

	path := filepath.Join(t.TempDir(), "body.json")
	if err := os.WriteFile(path, []byte(`{"email":"a@b.c"}`), 0o600); err != nil {
		t.Fatalf("write body file: %v", err)
	}

	out, err := runCLI(t, "", "raw", "POST", "/customers", "--file", path, "--yes")
	if err != nil {
		t.Fatalf("raw post failed: %v (%s)", err, out)
	}

	if got.Method != http.MethodPost || got.Path != "/v1/customers" {
		t.Errorf("expected POST /v1/customers, got %s %s", got.Method, got.Path)
	}

	if got.Body != `{"email":"a@b.c"}` {
		t.Errorf("unexpected body sent: %s", got.Body)
	}

	if got.Header.Get("Content-Type") != "application/json" {
		t.Errorf("expected a JSON content type, got %q", got.Header.Get("Content-Type"))
	}

	if got.Header.Get(api.HeaderIdempotencyKey) == "" {
		t.Error("expected a generated idempotency key on POST")
	}
}

func TestRawPost_SendsStdinBody(t *testing.T) {
	isolateConfig(t)
	seedCredentials(t)

	got := startAPI(t, http.StatusCreated, `{"id":"cus-2"}`)

	out, err := runCLI(t, `{"email":"stdin@example.com"}`,
		"raw", "POST", "/customers", "--data", "@-", "--yes")
	if err != nil {
		t.Fatalf("raw post from stdin failed: %v (%s)", err, out)
	}

	if got.Body != `{"email":"stdin@example.com"}` {
		t.Errorf("unexpected body sent: %s", got.Body)
	}
}

func TestRawPost_SendsInlineBodyAndHeaders(t *testing.T) {
	isolateConfig(t)
	seedCredentials(t)

	got := startAPI(t, http.StatusOK, `{}`)

	_, err := runCLI(t, "", "raw", "POST", "/customers",
		"--data", `{"email":"inline@example.com"}`,
		"--header", "X-Custom: one",
		"--header", "X-Other:two",
		"--yes")
	if err != nil {
		t.Fatalf("raw post failed: %v", err)
	}

	if got.Header.Get("X-Custom") != "one" || got.Header.Get("X-Other") != "two" {
		t.Errorf("expected both custom headers, got %v", got.Header)
	}
}

func TestRawPost_GoesThroughTheConfirmationGate(t *testing.T) {
	isolateConfig(t)
	seedCredentials(t)

	got := startAPI(t, http.StatusOK, `{}`)

	// No --yes and a non-TTY stdin: the gate must refuse instead of sending.
	_, err := runCLI(t, "", "raw", "POST", "/customers", "--data", `{"email":"a@b.c"}`)
	if err == nil {
		t.Fatal("expected the confirmation gate to block the request")
	}

	if !strings.Contains(err.Error(), "--yes") {
		t.Errorf("expected the error to mention --yes, got: %v", err)
	}

	if got.Method != "" {
		t.Errorf("no request should have been sent, got %s %s", got.Method, got.Path)
	}
}

func TestRawDelete_NoBodyIsAllowed(t *testing.T) {
	isolateConfig(t)
	seedCredentials(t)

	got := startAPI(t, http.StatusNoContent, "")

	out, err := runCLI(t, "", "raw", "DELETE", "/customers/cus-1", "--yes")
	if err != nil {
		t.Fatalf("raw delete failed: %v (%s)", err, out)
	}

	if got.Method != http.MethodDelete {
		t.Errorf("expected DELETE, got %s", got.Method)
	}

	if got.Header.Get(api.HeaderIdempotencyKey) != "" {
		t.Errorf("DELETE must not carry an idempotency key, got %q", got.Header.Get(api.HeaderIdempotencyKey))
	}

	if strings.TrimSpace(out) != "" {
		t.Errorf("an empty response should print nothing, got: %s", out)
	}
}

func TestRaw_InvalidMethod(t *testing.T) {
	isolateConfig(t)
	seedCredentials(t)

	_, err := runCLI(t, "", "raw", "FETCH", "/routing")
	if err == nil || !strings.Contains(err.Error(), "unsupported method") {
		t.Fatalf("expected an unsupported method error, got %v", err)
	}
}

func TestRaw_MissingBody(t *testing.T) {
	isolateConfig(t)
	seedCredentials(t)

	_, err := runCLI(t, "", "raw", "POST", "/customers", "--yes")
	if err == nil || !strings.Contains(err.Error(), "request body is required") {
		t.Fatalf("expected a missing body error, got %v", err)
	}
}

func TestRaw_MalformedJSONBody(t *testing.T) {
	isolateConfig(t)
	seedCredentials(t)

	_, err := runCLI(t, "", "raw", "POST", "/customers", "--data", "{oops", "--yes")
	if err == nil || !strings.Contains(err.Error(), "not valid json") {
		t.Fatalf("expected a malformed json error, got %v", err)
	}
}

func TestRaw_MalformedQueryAndHeader(t *testing.T) {
	isolateConfig(t)
	seedCredentials(t)

	if _, err := runCLI(t, "", "raw", "GET", "/routing", "--query", "nope"); err == nil ||
		!strings.Contains(err.Error(), "invalid query") {
		t.Fatalf("expected an invalid query error, got %v", err)
	}

	if _, err := runCLI(t, "", "raw", "GET", "/routing", "--header", "nope"); err == nil ||
		!strings.Contains(err.Error(), "invalid header") {
		t.Fatalf("expected an invalid header error, got %v", err)
	}
}

func TestRaw_MissingFile(t *testing.T) {
	isolateConfig(t)
	seedCredentials(t)

	_, err := runCLI(t, "", "raw", "POST", "/customers", "--file", "/no/such/body.json", "--yes")
	if err == nil || !strings.Contains(err.Error(), "read body file") {
		t.Fatalf("expected a read body file error, got %v", err)
	}
}

func TestRaw_Unauthenticated(t *testing.T) {
	isolateConfig(t)

	_, err := runCLI(t, "", "raw", "GET", "/routing")
	if err == nil || !strings.Contains(err.Error(), "profile not found") {
		t.Fatalf("expected a profile not found error, got %v", err)
	}
}

func TestRaw_APIError(t *testing.T) {
	isolateConfig(t)
	seedCredentials(t)
	startAPI(t, http.StatusForbidden, `{"code":"INSUFFICIENT_SCOPE","message":"missing routing:read"}`)

	_, err := runCLI(t, "", "raw", "GET", "/routing")
	if err == nil || !strings.Contains(err.Error(), "INSUFFICIENT_SCOPE") {
		t.Fatalf("expected the api error to surface, got %v", err)
	}
}
