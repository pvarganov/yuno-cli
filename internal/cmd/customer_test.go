package cmd_test

import (
	"net/http"
	"strings"
	"testing"
)

// customerResponse is the customer payload the fake API returns.
const customerResponse = `{"id":"cus-1","merchant_customer_id":"user-42","first_name":"Ada",
	"last_name":"Lovelace","email":"ada@example.com","country":"CO",
	"document":{"document_number":"123456","document_type":"CC"},
	"phone":{"country_code":"57","number":"3000000000"},
	"metadata":[{"key":"tier","value":"gold"}],
	"created_at":"2026-09-16T10:00:00Z"}`

func TestCustomerCreate_PostsTheBody(t *testing.T) {
	isolateConfig(t)
	seedCredentials(t)

	got := startAPI(t, http.StatusCreated, customerResponse)

	out, err := runCLI(t, "", "customer", "create", "--yes",
		"--merchant-customer-id", "user-42", "--first-name", "Ada",
		"--last-name", "Lovelace", "--email", "ada@example.com",
		"--country", "CO", "--document-number", "123456", "--document-type", "CC",
		"--phone-number", "3000000000", "--phone-country-code", "57",
		"--metadata", `[{"key":"tier","value":"gold"}]`)
	if err != nil {
		t.Fatalf("customer create failed: %v (%s)", err, out)
	}

	if got.Method != http.MethodPost || got.Path != "/v1/customers" {
		t.Errorf("expected POST /v1/customers, got %s %s", got.Method, got.Path)
	}

	body := decodeBody(t, got.Body)

	if body["merchant_customer_id"] != "user-42" || body["email"] != "ada@example.com" {
		t.Errorf("unexpected body: %s", got.Body)
	}

	document, ok := body["document"].(map[string]any)
	if !ok || document["document_number"] != "123456" || document["document_type"] != "CC" {
		t.Errorf("unexpected document in body: %s", got.Body)
	}

	phone, ok := body["phone"].(map[string]any)
	if !ok || phone["number"] != "3000000000" || phone["country_code"] != "57" {
		t.Errorf("unexpected phone in body: %s", got.Body)
	}

	if metadata, ok := body["metadata"].([]any); !ok || len(metadata) != 1 {
		t.Errorf("unexpected metadata in body: %s", got.Body)
	}

	for _, want := range []string{"MERCHANT_CUSTOMER_ID", "Ada Lovelace", "CC 123456"} {
		if !strings.Contains(out, want) {
			t.Errorf("expected table output to contain %q, got:\n%s", want, out)
		}
	}
}

func TestCustomerCreate_RequiresABody(t *testing.T) {
	isolateConfig(t)
	seedCredentials(t)
	startAPI(t, http.StatusCreated, customerResponse)

	_, err := runCLI(t, "", "customer", "create", "--yes")
	if err == nil {
		t.Fatal("expected an error when no body is given")
	}

	if !strings.Contains(err.Error(), "request body is required") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestCustomerGet_HitsTheCustomerPath(t *testing.T) {
	isolateConfig(t)
	seedCredentials(t)

	got := startAPI(t, http.StatusOK, customerResponse)

	out, err := runCLI(t, "", "customer", "get", "cus-1")
	if err != nil {
		t.Fatalf("customer get failed: %v (%s)", err, out)
	}

	if got.Method != http.MethodGet || got.Path != "/v1/customers/cus-1" {
		t.Errorf("expected GET /v1/customers/cus-1, got %s %s", got.Method, got.Path)
	}
}

func TestCustomerGet_JSONKeepsUntypedFields(t *testing.T) {
	isolateConfig(t)
	seedCredentials(t)
	startAPI(t, http.StatusOK, customerResponse)

	out, err := runCLI(t, "", "customer", "get", "cus-1", "--json")
	if err != nil {
		t.Fatalf("customer get failed: %v (%s)", err, out)
	}

	if !strings.Contains(out, "metadata") || !strings.Contains(out, "gold") {
		t.Errorf("expected the raw response to survive --json, got:\n%s", out)
	}
}

func TestCustomerList_LooksUpByMerchantCustomerID(t *testing.T) {
	isolateConfig(t)
	seedCredentials(t)

	got := startAPI(t, http.StatusOK, customerResponse)

	out, err := runCLI(t, "", "customer", "list", "--merchant-customer-id", "user-42")
	if err != nil {
		t.Fatalf("customer list failed: %v (%s)", err, out)
	}

	if got.Method != http.MethodGet || got.Path != "/v1/customers" {
		t.Errorf("expected GET /v1/customers, got %s %s", got.Method, got.Path)
	}

	if got.Query != "merchant_customer_id=user-42" {
		t.Errorf("unexpected query: %s", got.Query)
	}
}

func TestCustomerList_RequiresTheMerchantCustomerID(t *testing.T) {
	isolateConfig(t)
	seedCredentials(t)
	startAPI(t, http.StatusOK, customerResponse)

	if _, err := runCLI(t, "", "customer", "list"); err == nil {
		t.Fatal("expected an error when the merchant customer id is missing")
	}
}

func TestCustomerGetByMerchantID_UsesTheSameOperation(t *testing.T) {
	isolateConfig(t)
	seedCredentials(t)

	got := startAPI(t, http.StatusOK, customerResponse)

	out, err := runCLI(t, "", "customer", "get-by-merchant-id", "user-42")
	if err != nil {
		t.Fatalf("customer get-by-merchant-id failed: %v (%s)", err, out)
	}

	if got.Path != "/v1/customers" || got.Query != "merchant_customer_id=user-42" {
		t.Errorf("expected GET /v1/customers?merchant_customer_id=user-42, got %s?%s", got.Path, got.Query)
	}
}

func TestCustomerUpdate_PatchesOnlyTheChangedFields(t *testing.T) {
	isolateConfig(t)
	seedCredentials(t)

	got := startAPI(t, http.StatusOK, customerResponse)

	out, err := runCLI(t, "", "customer", "update", "cus-1", "--yes", "--email", "")
	if err != nil {
		t.Fatalf("customer update failed: %v (%s)", err, out)
	}

	if got.Method != http.MethodPatch || got.Path != "/v1/customers/cus-1" {
		t.Errorf("expected PATCH /v1/customers/cus-1, got %s %s", got.Method, got.Path)
	}

	body := decodeBody(t, got.Body)

	if len(body) != 1 {
		t.Errorf("expected only the changed field in the body, got %s", got.Body)
	}

	if email, ok := body["email"]; !ok || email != "" {
		t.Errorf("expected an explicit empty email in the body, got %s", got.Body)
	}
}

func TestCustomerDelete_DeletesAndPrintsNothing(t *testing.T) {
	isolateConfig(t)
	seedCredentials(t)

	got := startAPI(t, http.StatusAccepted, "")

	out, err := runCLI(t, "", "customer", "delete", "cus-1", "--yes")
	if err != nil {
		t.Fatalf("customer delete failed: %v (%s)", err, out)
	}

	if got.Method != http.MethodDelete || got.Path != "/v1/customers/cus-1" {
		t.Errorf("expected DELETE /v1/customers/cus-1, got %s %s", got.Method, got.Path)
	}

	if strings.TrimSpace(out) != "" {
		t.Errorf("expected no output for an empty response, got:\n%s", out)
	}
}

func TestCustomerDelete_PromptsWithoutYes(t *testing.T) {
	isolateConfig(t)
	seedCredentials(t)

	got := startAPI(t, http.StatusAccepted, "")

	out, err := runCLI(t, "n\n", "customer", "delete", "cus-1")
	if err == nil {
		t.Fatalf("expected the declined confirmation to fail, got:\n%s", out)
	}

	if got.Method != "" {
		t.Errorf("expected no request to be sent, got %s %s", got.Method, got.Path)
	}
}

func TestCustomerSessionCreate_PostsToTheSessionsPath(t *testing.T) {
	isolateConfig(t)
	seedCredentials(t)

	got := startAPI(t, http.StatusCreated,
		`{"customer_session":"sess-1","customer_id":"cus-1","country":"CO","checkout_id":"chk-1"}`)

	out, err := runCLI(t, "", "customer", "session", "create", "--yes",
		"--account-id", "acc-1", "--country", "CO", "--customer-id", "cus-1",
		"--callback-url", "https://example.com/return", "--checkout-id", "chk-1")
	if err != nil {
		t.Fatalf("customer session create failed: %v (%s)", err, out)
	}

	if got.Method != http.MethodPost || got.Path != "/v1/customers/sessions" {
		t.Errorf("expected POST /v1/customers/sessions, got %s %s", got.Method, got.Path)
	}

	body := decodeBody(t, got.Body)

	if body["account_id"] != "acc-1" || body["country"] != "CO" || body["customer_id"] != "cus-1" {
		t.Errorf("unexpected body: %s", got.Body)
	}

	if body["callback_url"] != "https://example.com/return" || body["checkout_id"] != "chk-1" {
		t.Errorf("unexpected body: %s", got.Body)
	}

	if !strings.Contains(out, "sess-1") {
		t.Errorf("expected the session in the output, got:\n%s", out)
	}
}

func TestNetworkTokenCryptogram_PostsToTheCryptogramsPath(t *testing.T) {
	isolateConfig(t)
	seedCredentials(t)

	got := startAPI(t, http.StatusCreated,
		`{"vaulted_token":"vt-1","network_token":"4111111111111111","cryptogram":"AgAAAA","eci":"05"}`)

	out, err := runCLI(t, "", "network-token", "cryptogram", "--yes",
		"--vaulted-token", "vt-1", "--country", "US",
		"--currency", "USD", "--amount", "49.99", "--include-network-token")
	if err != nil {
		t.Fatalf("network-token cryptogram failed: %v (%s)", err, out)
	}

	if got.Method != http.MethodPost || got.Path != "/v1/network-tokens/cryptograms" {
		t.Errorf("expected POST /v1/network-tokens/cryptograms, got %s %s", got.Method, got.Path)
	}

	body := decodeBody(t, got.Body)

	if body["vaulted_token"] != "vt-1" || body["country"] != "US" || body["include_network_token"] != true {
		t.Errorf("unexpected body: %s", got.Body)
	}

	amount, ok := body["amount"].(map[string]any)
	if !ok || amount["currency"] != "USD" || amount["value"] != 49.99 {
		t.Errorf("unexpected amount in body: %s", got.Body)
	}

	if strings.Contains(out, "4111111111111111") {
		t.Errorf("expected the network token to be masked in the table output, got:\n%s", out)
	}
}

func TestCustomerGet_ForbiddenSuggestsTheScope(t *testing.T) {
	isolateConfig(t)
	seedCredentials(t)

	startAPI(t, http.StatusForbidden, `{"code":"INSUFFICIENT_SCOPE","message":"missing scope"}`)

	_, err := runCLI(t, "", "customer", "get", "cus-1")
	if err == nil {
		t.Fatal("expected an error on 403")
	}

	if !strings.Contains(err.Error(), "customers:read") {
		t.Errorf("expected the error to name the scope, got %v", err)
	}
}
