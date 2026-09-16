package cmd_test

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/pvarganov/yuno-cli/internal/api"
	"github.com/pvarganov/yuno-cli/internal/cmd"
	"github.com/pvarganov/yuno-cli/internal/config"
	"github.com/pvarganov/yuno-cli/internal/confirm"
)

// startCountingAPI serves the same answer to every request and counts how many
// requests it served, so retry behaviour is observable end to end.
func startCountingAPI(t *testing.T, status int, response string) *atomic.Int64 {
	t.Helper()

	var calls atomic.Int64

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)

		_, _ = io.Copy(io.Discard, r.Body)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_, _ = io.WriteString(w, response)
	}))
	t.Cleanup(server.Close)

	t.Setenv(config.EnvAPIEndpoint, server.URL+"/v1")

	return &calls
}

func TestAcceptance_InvalidKeysReportUnauthorized(t *testing.T) {
	isolateConfig(t)
	seedCredentials(t)

	calls := startCountingAPI(t, http.StatusUnauthorized,
		`{"code":"UNAUTHORIZED","message":"invalid api key","type":"AUTHENTICATION"}`)

	out, err := runCLI(t, "", "customer", "get", "cus-1")
	if err == nil {
		t.Fatalf("expected an error for invalid keys, got output: %s", out)
	}

	if !errors.Is(err, api.ErrUnauthorized) {
		t.Errorf("expected api.ErrUnauthorized, got %v", err)
	}

	if !strings.Contains(err.Error(), "check your api keys") {
		t.Errorf("error should tell the user to check the keys, got: %v", err)
	}

	if got := calls.Load(); got != 1 {
		t.Errorf("a 401 must not be retried, server saw %d calls", got)
	}

	if code := cmd.ExitCode(err); code != cmd.ExitAPIError {
		t.Errorf("ExitCode = %d, want %d", code, cmd.ExitAPIError)
	}
}

func TestAcceptance_RateLimitStormIsRetriedThenReported(t *testing.T) {
	isolateConfig(t)
	seedCredentials(t)

	calls := startCountingAPI(t, http.StatusTooManyRequests, `{"code":"RATE_LIMIT","message":"slow down"}`)

	out, err := runCLI(t, "", "customer", "get", "cus-1")
	if err == nil {
		t.Fatalf("expected an error under a 429 storm, got output: %s", out)
	}

	if !errors.Is(err, api.ErrRateLimited) {
		t.Errorf("expected api.ErrRateLimited, got %v", err)
	}

	// One initial attempt plus the three retries of the default client.
	if got := calls.Load(); got != 4 {
		t.Errorf("expected 4 attempts under a 429 storm, server saw %d", got)
	}

	if code := cmd.ExitCode(err); code != cmd.ExitAPIError {
		t.Errorf("ExitCode = %d, want %d", code, cmd.ExitAPIError)
	}
}

func TestAcceptance_EmptyListPrintsNothing(t *testing.T) {
	isolateConfig(t)
	seedCredentials(t)
	startAPI(t, http.StatusOK, `[]`)

	out, err := runCLI(t, "", "routing", "list", "--account-id", "acc-1")
	if err != nil {
		t.Fatalf("routing list failed: %v (%s)", err, out)
	}

	if strings.TrimSpace(out) != "" {
		t.Errorf("an empty list must print nothing, got: %q", out)
	}
}

func TestAcceptance_EmptyJSONListPrintsAnEmptyArray(t *testing.T) {
	isolateConfig(t)
	seedCredentials(t)
	startAPI(t, http.StatusOK, `[]`)

	out, err := runCLI(t, "", "routing", "list", "--account-id", "acc-1", "--json")
	if err != nil {
		t.Fatalf("routing list failed: %v (%s)", err, out)
	}

	if strings.TrimSpace(out) != "[]" {
		t.Errorf("expected an empty json array, got: %q", out)
	}
}

func TestAcceptance_NonTTYStdinRefusesAWrite(t *testing.T) {
	isolateConfig(t)
	seedCredentials(t)

	calls := startCountingAPI(t, http.StatusOK, `{"id":"cus-1"}`)

	_, err := runCLI(t, "", "customer", "create", "--email", "user@example.com")
	if err == nil {
		t.Fatal("expected a write without --yes on a non-TTY stdin to fail")
	}

	if !errors.Is(err, confirm.ErrNotATerminal) {
		t.Errorf("expected confirm.ErrNotATerminal, got %v", err)
	}

	if got := calls.Load(); got != 0 {
		t.Errorf("the request must not be sent when the gate refuses, server saw %d calls", got)
	}

	if code := cmd.ExitCode(err); code != cmd.ExitDeclined {
		t.Errorf("ExitCode = %d, want %d", code, cmd.ExitDeclined)
	}
}

func TestAcceptance_YesBypassesTheConfirmationGate(t *testing.T) {
	isolateConfig(t)
	seedCredentials(t)

	got := startAPI(t, http.StatusOK, `{"id":"cus-1","email":"user@example.com"}`)

	out, err := runCLI(t, "", "customer", "create", "--email", "user@example.com", "--yes")
	if err != nil {
		t.Fatalf("customer create --yes failed: %v (%s)", err, out)
	}

	if got.Method != http.MethodPost || got.Path != "/v1/customers" {
		t.Errorf("expected POST /v1/customers, got %s %s", got.Method, got.Path)
	}

	if !strings.Contains(got.Body, "user@example.com") {
		t.Errorf("unexpected request body: %s", got.Body)
	}
}

func TestAcceptance_AssumeYesEnvBypassesTheGate(t *testing.T) {
	isolateConfig(t)
	seedCredentials(t)
	t.Setenv(confirm.EnvAssumeYes, "1")

	got := startAPI(t, http.StatusOK, `{"id":"cus-1"}`)

	if out, err := runCLI(t, "", "customer", "create", "--email", "user@example.com"); err != nil {
		t.Fatalf("customer create failed: %v (%s)", err, out)
	}

	if got.Method != http.MethodPost {
		t.Errorf("expected the request to be sent, got method %q", got.Method)
	}
}

func TestAcceptance_MissingCredentialsExitThree(t *testing.T) {
	isolateConfig(t)

	_, err := runCLI(t, "", "customer", "get", "cus-1")
	if err == nil {
		t.Fatal("expected an error without any configured profile")
	}

	if code := cmd.ExitCode(err); code != cmd.ExitConfigError {
		t.Errorf("ExitCode = %d, want %d (err: %v)", code, cmd.ExitConfigError, err)
	}
}

func TestAcceptance_TimeoutFlagAbortsASlowRequest(t *testing.T) {
	isolateConfig(t)
	seedCredentials(t)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		time.Sleep(200 * time.Millisecond)
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"id":"cus-1"}`)
	}))
	t.Cleanup(server.Close)

	t.Setenv(config.EnvAPIEndpoint, server.URL+"/v1")

	out, err := runCLI(t, "", "customer", "get", "cus-1", "--timeout", "20ms")
	if err == nil {
		t.Fatalf("expected the request to time out, got output: %s", out)
	}

	if !strings.Contains(err.Error(), "deadline exceeded") && !strings.Contains(err.Error(), "timeout") {
		t.Errorf("expected a timeout error, got: %v", err)
	}
}

func TestExitCode(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		err  error
		want int
	}{
		{name: "success", err: nil, want: cmd.ExitOK},
		{name: "api error", err: &api.Error{StatusCode: http.StatusNotFound}, want: cmd.ExitAPIError},
		{name: "wrapped api sentinel", err: errors.Join(api.ErrForbidden), want: cmd.ExitAPIError},
		{name: "declined", err: confirm.ErrDeclined, want: cmd.ExitDeclined},
		{name: "not a terminal", err: confirm.ErrNotATerminal, want: cmd.ExitDeclined},
		{name: "profile not found", err: config.ErrProfileNotFound, want: cmd.ExitConfigError},
		{name: "missing credentials", err: config.ErrMissingCredentials, want: cmd.ExitConfigError},
		{name: "unknown environment", err: config.ErrUnknownEnvironment, want: cmd.ExitConfigError},
		{name: "input error", err: errors.New("unknown flag: --nope"), want: cmd.ExitConfigError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := cmd.ExitCode(tt.err); got != tt.want {
				t.Errorf("ExitCode(%v) = %d, want %d", tt.err, got, tt.want)
			}
		})
	}
}
