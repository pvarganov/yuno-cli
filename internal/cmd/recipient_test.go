package cmd_test

import (
	"net/http"
	"strings"
	"testing"
)

// recipientResponse is the recipient payload the fake API returns.
const recipientResponse = `{"id":"rec-1","account_id":"acc-1","merchant_recipient_id":"mr-1",
	"national_entity":"INDIVIDUAL","entity_type":"INDIVIDUAL","first_name":"Ada","last_name":"Lovelace",
	"email":"ada@example.com","country":"BR","created_at":"2026-09-16T10:00:00Z",
	"onboardings":[{"id":"onb-1","status":"SUCCEEDED","provider":{"id":"NUVEI"}}]}`

func TestRecipientCreate_PostsTheBody(t *testing.T) {
	isolateConfig(t)
	seedCredentials(t)

	got := startAPI(t, http.StatusCreated, recipientResponse)

	out, err := runCLI(t, "", "recipient", "create", "--yes",
		"--account-id", "acc-1", "--merchant-recipient-id", "mr-1",
		"--national-entity", "INDIVIDUAL", "--first-name", "Ada", "--last-name", "Lovelace",
		"--country", "BR", "--email", "ada@example.com",
		"--document-number", "123", "--document-type", "CPF",
		"--phone-country-code", "55", "--phone-number", "999999999",
		"--address", `{"city":"Sao Paulo","country":"BR"}`)
	if err != nil {
		t.Fatalf("recipient create failed: %v (%s)", err, out)
	}

	if got.Method != http.MethodPost || got.Path != "/v1/recipients" {
		t.Errorf("expected POST /v1/recipients, got %s %s", got.Method, got.Path)
	}

	body := decodeBody(t, got.Body)

	document, ok := body["document"].(map[string]any)
	if !ok || document["document_number"] != "123" || document["document_type"] != "CPF" {
		t.Errorf("unexpected document in body: %s", got.Body)
	}

	phone, ok := body["phone"].(map[string]any)
	if !ok || phone["country_code"] != "55" {
		t.Errorf("unexpected phone in body: %s", got.Body)
	}

	address, ok := body["address"].(map[string]any)
	if !ok || address["city"] != "Sao Paulo" {
		t.Errorf("unexpected address in body: %s", got.Body)
	}

	for _, want := range []string{"rec-1", "Ada Lovelace", "NUVEI SUCCEEDED"} {
		if !strings.Contains(out, want) {
			t.Errorf("expected table output to contain %q, got:\n%s", want, out)
		}
	}
}

func TestRecipientCreate_RequiresABody(t *testing.T) {
	isolateConfig(t)
	seedCredentials(t)
	startAPI(t, http.StatusCreated, recipientResponse)

	if _, err := runCLI(t, "", "recipient", "create", "--yes"); err == nil ||
		!strings.Contains(err.Error(), "request body is required") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestRecipientList_SendsTheFiltersAndPaging(t *testing.T) {
	isolateConfig(t)
	seedCredentials(t)

	got := startAPI(t, http.StatusOK, `{"data":[`+recipientResponse+`],"pagination":{"has_next":false}}`)

	out, err := runCLI(t, "", "recipient", "list", "--country", "BR", "--national-entity", "INDIVIDUAL",
		"--page-size", "5")
	if err != nil {
		t.Fatalf("recipient list failed: %v (%s)", err, out)
	}

	if got.Method != http.MethodGet || got.Path != "/v1/recipients" {
		t.Errorf("expected GET /v1/recipients, got %s %s", got.Method, got.Path)
	}

	for _, want := range []string{"country=BR", "national_entity=INDIVIDUAL", "limit=5", "offset=0"} {
		if !strings.Contains(got.Query, want) {
			t.Errorf("expected the query to contain %q, got %s", want, got.Query)
		}
	}

	if !strings.Contains(out, "rec-1") {
		t.Errorf("expected the recipient in the table, got:\n%s", out)
	}
}

func TestRecipientGet_HitsTheIDPath(t *testing.T) {
	isolateConfig(t)
	seedCredentials(t)

	got := startAPI(t, http.StatusOK, recipientResponse)

	out, err := runCLI(t, "", "recipient", "get", "rec-1", "--json")
	if err != nil {
		t.Fatalf("recipient get failed: %v (%s)", err, out)
	}

	if got.Method != http.MethodGet || got.Path != "/v1/recipients/rec-1" {
		t.Errorf("expected GET /v1/recipients/rec-1, got %s %s", got.Method, got.Path)
	}

	if !strings.Contains(out, `"merchant_recipient_id"`) {
		t.Errorf("expected the whole payload as json, got:\n%s", out)
	}
}

func TestRecipientUpdate_PatchesTheIDPath(t *testing.T) {
	isolateConfig(t)
	seedCredentials(t)

	got := startAPI(t, http.StatusOK, recipientResponse)

	out, err := runCLI(t, "", "recipient", "update", "rec-1", "--yes",
		"--onboarding-id", "onb-1", "--email", "ada@example.com")
	if err != nil {
		t.Fatalf("recipient update failed: %v (%s)", err, out)
	}

	if got.Method != http.MethodPatch || got.Path != "/v1/recipients/rec-1" {
		t.Errorf("expected PATCH /v1/recipients/rec-1, got %s %s", got.Method, got.Path)
	}

	body := decodeBody(t, got.Body)
	if body["onboarding_id"] != "onb-1" || body["email"] != "ada@example.com" {
		t.Errorf("unexpected body: %s", got.Body)
	}

	if _, patched := body["national_entity"]; patched {
		t.Errorf("the national entity must not be patchable: %s", got.Body)
	}
}

func TestRecipientDelete_SendsDelete(t *testing.T) {
	isolateConfig(t)
	seedCredentials(t)

	got := startAPI(t, http.StatusNoContent, "")

	out, err := runCLI(t, "", "recipient", "delete", "rec-1", "--yes")
	if err != nil {
		t.Fatalf("recipient delete failed: %v (%s)", err, out)
	}

	if got.Method != http.MethodDelete || got.Path != "/v1/recipients/rec-1" {
		t.Errorf("expected DELETE /v1/recipients/rec-1, got %s %s", got.Method, got.Path)
	}
}

func TestRecipientDelete_AsksForConfirmation(t *testing.T) {
	isolateConfig(t)
	seedCredentials(t)
	startAPI(t, http.StatusNoContent, "")

	if _, err := runCLI(t, "n\n", "recipient", "delete", "rec-1"); err == nil {
		t.Error("expected the confirmation gate to abort the delete")
	}
}

func TestRecipientHintsAtTheScopeOnA403(t *testing.T) {
	isolateConfig(t)
	seedCredentials(t)
	startAPI(t, http.StatusForbidden, `{"code":"INSUFFICIENT_SCOPE","message":"nope"}`)

	_, err := runCLI(t, "", "recipient", "get", "rec-1")
	if err == nil || !strings.Contains(err.Error(), "recipients:read/recipients:write") {
		t.Errorf("expected the scope hint, got %v", err)
	}
}
