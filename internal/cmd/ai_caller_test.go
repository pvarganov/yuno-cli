package cmd_test

import (
	"net/http"
	"strings"
	"testing"
)

func TestAICallerDeclinedPayments_PostsToTheRecoverPath(t *testing.T) {
	isolateConfig(t)
	seedCredentials(t)

	got := startAPI(t, http.StatusOK, `{"id":"call-1","message":"outreach scheduled"}`)

	out, err := runCLI(t, "", "ai-caller", "declined-payments", "--yes",
		"--settings", `{"contact_details":{"channel":"PHONE_CALL"}}`,
		"--additional-information", `{"payment":{"id":"pay-1","declined_reason":"INSUFFICIENT_FUNDS"}}`)
	if err != nil {
		t.Fatalf("ai-caller declined-payments failed: %v (%s)", err, out)
	}

	if got.Method != http.MethodPost || got.Path != "/v1/smart-support/external/payments/recover" {
		t.Errorf("expected POST /v1/smart-support/external/payments/recover, got %s %s", got.Method, got.Path)
	}

	body := decodeBody(t, got.Body)

	settings, ok := body["settings"].(map[string]any)
	if !ok || settings["contact_details"] == nil {
		t.Errorf("unexpected settings: %s", got.Body)
	}

	if _, ok := body["additional_information"].(map[string]any); !ok {
		t.Errorf("unexpected additional_information: %s", got.Body)
	}

	for _, want := range []string{"call-1", "outreach scheduled"} {
		if !strings.Contains(out, want) {
			t.Errorf("expected table output to contain %q, got:\n%s", want, out)
		}
	}
}

func TestAICallerDeclinedPayments_RejectsInvalidJSONFlags(t *testing.T) {
	isolateConfig(t)
	seedCredentials(t)
	startAPI(t, http.StatusOK, `{"id":"call-1"}`)

	if _, err := runCLI(t, "", "ai-caller", "declined-payments", "--yes", "--settings", "{"); err == nil ||
		!strings.Contains(err.Error(), "not valid json") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestAICallerRecover_PostsToThePaymentsPath(t *testing.T) {
	isolateConfig(t)
	seedCredentials(t)

	got := startAPI(t, http.StatusOK, "")

	out, err := runCLI(t, "", "ai-caller", "recover", "--yes",
		"--customer", `{"full_name":"Ana","phone_number":"+573001112233"}`,
		"--session", `{"flow_stage":"checkout","platform":"web"}`,
		"--cart", `{"total_value":120.5,"currency":"COP"}`,
		"--engagement", `{"preferred_channel":"whatsapp"}`)
	if err != nil {
		t.Fatalf("ai-caller recover failed: %v (%s)", err, out)
	}

	if got.Method != http.MethodPost || got.Path != "/v1/smart-support/external/payments" {
		t.Errorf("expected POST /v1/smart-support/external/payments, got %s %s", got.Method, got.Path)
	}

	body := decodeBody(t, got.Body)
	for _, key := range []string{"customer", "session", "cart", "engagement"} {
		if _, ok := body[key].(map[string]any); !ok {
			t.Errorf("expected %q in the body, got: %s", key, got.Body)
		}
	}

	if strings.TrimSpace(out) != "" {
		t.Errorf("expected no output for an empty response, got:\n%s", out)
	}
}

func TestAICallerRecover_AsksBeforeWriting(t *testing.T) {
	isolateConfig(t)
	seedCredentials(t)
	startAPI(t, http.StatusOK, "")

	if _, err := runCLI(t, "n\n", "ai-caller", "recover", "--customer", `{"email":"a@b.c"}`); err == nil {
		t.Error("expected a declined confirmation to fail the command")
	}
}

func TestAICallerRecover_RefusesAnEmptyBody(t *testing.T) {
	isolateConfig(t)
	seedCredentials(t)
	startAPI(t, http.StatusOK, "")

	if _, err := runCLI(t, "", "ai-caller", "recover", "--yes"); err == nil ||
		!strings.Contains(err.Error(), "request body is required") {
		t.Errorf("unexpected error: %v", err)
	}
}
