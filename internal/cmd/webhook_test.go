package cmd_test

import (
	"net/http"
	"strings"
	"testing"
)

// webhookResponse is the webhook payload the fake API returns.
const webhookResponse = `{"id":"wh-1","account_id":"acc-1","name":"listener","state":"ACTIVE",
	"url":"https://api.acme.com/yuno","api_key":"***","payment_triggers":["AUTHORIZE","REFUND"],
	"created_at":"2026-09-16T10:00:00Z"}`

func TestWebhookCreate_PostsTheBody(t *testing.T) {
	isolateConfig(t)
	seedCredentials(t)

	got := startAPI(t, http.StatusCreated, webhookResponse)

	out, err := runCLI(t, "", "webhook", "create", "--yes",
		"--account-id", "acc-1", "--name", "listener", "--url", "https://api.acme.com/yuno",
		"--api-key", "wh-key", "--payment-trigger", "AUTHORIZE", "--payment-trigger", "REFUND",
		"--renewal-days", "5")
	if err != nil {
		t.Fatalf("webhook create failed: %v (%s)", err, out)
	}

	if got.Method != http.MethodPost || got.Path != "/v1/webhooks" {
		t.Errorf("expected POST /v1/webhooks, got %s %s", got.Method, got.Path)
	}

	body := decodeBody(t, got.Body)
	if body["account_id"] != "acc-1" || body["name"] != "listener" || body["api_key"] != "wh-key" {
		t.Errorf("unexpected body: %s", got.Body)
	}

	triggers, ok := body["payment_triggers"].([]any)
	if !ok || len(triggers) != 2 || triggers[0] != "AUTHORIZE" {
		t.Errorf("unexpected payment triggers: %s", got.Body)
	}

	if body["renewal_days"] != 5.0 {
		t.Errorf("unexpected renewal days: %s", got.Body)
	}

	for _, want := range []string{"wh-1", "ACTIVE", "payment:AUTHORIZE|REFUND"} {
		if !strings.Contains(out, want) {
			t.Errorf("expected table output to contain %q, got:\n%s", want, out)
		}
	}
}

func TestWebhookCreate_RequiresABody(t *testing.T) {
	isolateConfig(t)
	seedCredentials(t)
	startAPI(t, http.StatusCreated, webhookResponse)

	if _, err := runCLI(t, "", "webhook", "create", "--yes"); err == nil ||
		!strings.Contains(err.Error(), "request body is required") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestWebhookList_FiltersByAccountAndState(t *testing.T) {
	isolateConfig(t)
	seedCredentials(t)

	got := startAPI(t, http.StatusOK, "["+webhookResponse+"]")

	out, err := runCLI(t, "", "webhook", "list", "--account-id", "acc-1", "--state", "ACTIVE")
	if err != nil {
		t.Fatalf("webhook list failed: %v (%s)", err, out)
	}

	if got.Method != http.MethodGet || got.Path != "/v1/webhooks" {
		t.Errorf("expected GET /v1/webhooks, got %s %s", got.Method, got.Path)
	}

	if got.Query != "account_id=acc-1&state=ACTIVE" {
		t.Errorf("unexpected query: %s", got.Query)
	}

	if !strings.Contains(out, "wh-1") {
		t.Errorf("expected the webhook in the table, got:\n%s", out)
	}
}

func TestWebhookList_RequiresTheAccountID(t *testing.T) {
	isolateConfig(t)
	seedCredentials(t)
	startAPI(t, http.StatusOK, "[]")

	if _, err := runCLI(t, "", "webhook", "list"); err == nil ||
		!strings.Contains(err.Error(), "account-id") {
		t.Errorf("expected the required flag error, got %v", err)
	}
}

func TestWebhookList_EmptyResponsePrintsNothing(t *testing.T) {
	isolateConfig(t)
	seedCredentials(t)
	startAPI(t, http.StatusOK, "[]")

	out, err := runCLI(t, "", "webhook", "list", "--account-id", "acc-1")
	if err != nil {
		t.Fatalf("webhook list failed: %v (%s)", err, out)
	}

	if strings.TrimSpace(out) != "" {
		t.Errorf("expected no output for an empty list, got:\n%s", out)
	}
}

func TestWebhookGet_HitsTheIDPath(t *testing.T) {
	isolateConfig(t)
	seedCredentials(t)

	got := startAPI(t, http.StatusOK, webhookResponse)

	out, err := runCLI(t, "", "webhook", "get", "wh-1", "--account-id", "acc-1", "--json")
	if err != nil {
		t.Fatalf("webhook get failed: %v (%s)", err, out)
	}

	if got.Method != http.MethodGet || got.Path != "/v1/webhooks/wh-1" || got.Query != "account_id=acc-1" {
		t.Errorf("unexpected request: %s %s?%s", got.Method, got.Path, got.Query)
	}

	if !strings.Contains(out, `"id": "wh-1"`) {
		t.Errorf("expected the json body, got:\n%s", out)
	}
}

func TestWebhookUpdate_PatchesOnlyTheChangedFields(t *testing.T) {
	isolateConfig(t)
	seedCredentials(t)

	got := startAPI(t, http.StatusOK, webhookResponse)

	out, err := runCLI(t, "", "webhook", "update", "wh-1", "--yes",
		"--account-id", "acc-1", "--state", "INACTIVE")
	if err != nil {
		t.Fatalf("webhook update failed: %v (%s)", err, out)
	}

	if got.Method != http.MethodPatch || got.Path != "/v1/webhooks/wh-1" {
		t.Errorf("expected PATCH /v1/webhooks/wh-1, got %s %s", got.Method, got.Path)
	}

	body := decodeBody(t, got.Body)
	if len(body) != 2 || body["state"] != "INACTIVE" || body["account_id"] != "acc-1" {
		t.Errorf("expected only the changed fields, got: %s", got.Body)
	}
}

func TestWebhookDelete_SendsTheAccountID(t *testing.T) {
	isolateConfig(t)
	seedCredentials(t)

	got := startAPI(t, http.StatusOK, `{"deleted":true}`)

	out, err := runCLI(t, "", "webhook", "delete", "wh-1", "--account-id", "acc-1", "--yes")
	if err != nil {
		t.Fatalf("webhook delete failed: %v (%s)", err, out)
	}

	if got.Method != http.MethodDelete || got.Path != "/v1/webhooks/wh-1" || got.Query != "account_id=acc-1" {
		t.Errorf("unexpected request: %s %s?%s", got.Method, got.Path, got.Query)
	}

	if !strings.Contains(out, "deleted") {
		t.Errorf("expected the confirmation body, got:\n%s", out)
	}
}

func TestWebhookDelete_IsConfirmed(t *testing.T) {
	isolateConfig(t)
	seedCredentials(t)
	startAPI(t, http.StatusOK, `{"deleted":true}`)

	if _, err := runCLI(t, "no\n", "webhook", "delete", "wh-1", "--account-id", "acc-1"); err == nil {
		t.Error("expected the declined confirmation to fail the command")
	}
}
