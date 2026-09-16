package cmd_test

import (
	"net/http"
	"strings"
	"testing"
)

func TestThreeDSecureSetup_PostsTheBody(t *testing.T) {
	isolateConfig(t)
	seedCredentials(t)

	got := startAPI(t, http.StatusCreated, `{"three_d_secure_setup_id":"tds-1","account_id":"acc-1",
		"type":"BROWSER","browser_info":{"screen_width":"1920","screen_height":"1080","language":"en-US"},
		"device_fingerprints":[{"provider_id":"CYBERSOURCE","id":"fp-1"}]}`)

	out, err := runCLI(t, "", "three-d-secure", "setup", "--yes",
		"--account-id", "acc-1", "--type", "BROWSER", "--user-agent", "Mozilla/5.0",
		"--accept-header", "*/*", "--language", "en-US", "--screen-width", "1920",
		"--screen-height", "1080", "--color-depth", "24", "--javascript-enabled")
	if err != nil {
		t.Fatalf("three-d-secure setup failed: %v (%s)", err, out)
	}

	if got.Method != http.MethodPost || got.Path != "/v1/three-d-secure/setups" {
		t.Errorf("expected POST /v1/three-d-secure/setups, got %s %s", got.Method, got.Path)
	}

	body := decodeBody(t, got.Body)

	info, ok := body["browser_info"].(map[string]any)
	if !ok || info["language"] != "en-US" || info["javascript_enabled"] != true {
		t.Errorf("unexpected browser info in body: %s", got.Body)
	}

	for _, want := range []string{"tds-1", "BROWSER", "1920x1080", "CYBERSOURCE=fp-1"} {
		if !strings.Contains(out, want) {
			t.Errorf("expected table output to contain %q, got:\n%s", want, out)
		}
	}
}

func TestThreeDSecureSetup_RequiresABody(t *testing.T) {
	isolateConfig(t)
	seedCredentials(t)
	startAPI(t, http.StatusCreated, `{"three_d_secure_setup_id":"tds-1"}`)

	if _, err := runCLI(t, "", "three-d-secure", "setup", "--yes"); err == nil ||
		!strings.Contains(err.Error(), "request body is required") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestThreeDSecureSetup_RejectsMalformedJSONFlags(t *testing.T) {
	isolateConfig(t)
	seedCredentials(t)
	startAPI(t, http.StatusCreated, `{"three_d_secure_setup_id":"tds-1"}`)

	_, err := runCLI(t, "", "three-d-secure", "setup", "--yes", "--device-fingerprints", "[oops")
	if err == nil || !strings.Contains(err.Error(), "not valid json") {
		t.Errorf("expected a json error, got %v", err)
	}
}

func TestThreeDSecureSetup_ReportsTheAPIError(t *testing.T) {
	isolateConfig(t)
	seedCredentials(t)
	startAPI(t, http.StatusBadRequest, `{"code":"VALIDATION_ERROR","message":"bad browser info"}`)

	_, err := runCLI(t, "", "three-d-secure", "setup", "--yes", "--type", "BROWSER")
	if err == nil || !strings.Contains(err.Error(), "bad browser info") {
		t.Errorf("expected the api error, got %v", err)
	}
}

func TestDryRunProviderEvent_PostsTheBody(t *testing.T) {
	isolateConfig(t)
	seedCredentials(t)

	got := startAPI(t, http.StatusCreated, `{"id":"dr-1","status":"PROCESSED","merchant_reference":"ref-1",
		"provider_id":"NUVEI","payment_method_type":"CARD","operation_type":"CREATE_PAYMENT","payment_id":null}`)

	out, err := runCLI(t, "", "dry-run", "provider-event", "--yes",
		"--merchant-reference", "ref-1", "--provider-id", "NUVEI", "--payment-method-type", "CARD",
		"--events", `[{"type":"WEBHOOK","http_method":"POST","operation_type":"CREATE_PAYMENT","headers":"{}"}]`)
	if err != nil {
		t.Fatalf("dry-run provider-event failed: %v (%s)", err, out)
	}

	if got.Method != http.MethodPost || got.Path != "/v1/dry-run/provider-events" {
		t.Errorf("expected POST /v1/dry-run/provider-events, got %s %s", got.Method, got.Path)
	}

	body := decodeBody(t, got.Body)

	events, ok := body["events"].([]any)
	if !ok || len(events) != 1 {
		t.Errorf("unexpected events in body: %s", got.Body)
	}

	for _, want := range []string{"dr-1", "PROCESSED", "CREATE_PAYMENT"} {
		if !strings.Contains(out, want) {
			t.Errorf("expected table output to contain %q, got:\n%s", want, out)
		}
	}
}

func TestDryRunProviderEvent_RequiresABody(t *testing.T) {
	isolateConfig(t)
	seedCredentials(t)
	startAPI(t, http.StatusCreated, `{"id":"dr-1"}`)

	if _, err := runCLI(t, "", "dry-run", "provider-event", "--yes"); err == nil ||
		!strings.Contains(err.Error(), "request body is required") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestDryRunProviderEvent_ReportsTheScopeHint(t *testing.T) {
	isolateConfig(t)
	seedCredentials(t)
	startAPI(t, http.StatusForbidden, `{"code":"INSUFFICIENT_SCOPE","message":"nope"}`)

	_, err := runCLI(t, "", "dry-run", "provider-event", "--yes", "--merchant-reference", "ref-1")
	if err == nil || !strings.Contains(err.Error(), "payments:write") {
		t.Errorf("expected the scope hint, got %v", err)
	}
}
