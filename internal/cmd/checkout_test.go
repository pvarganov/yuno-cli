package cmd_test

import (
	"net/http"
	"strings"
	"testing"
)

// checkoutSessionResponse is the checkout session payload the fake API returns.
const checkoutSessionResponse = `{"checkout_session":"chk-1","merchant_order_id":"order-42",
	"payment_description":"Test Cards","country":"CO","customer_id":"cus-1",
	"amount":{"currency":"USD","value":520},"workflow":"SDK_CHECKOUT",
	"metadata":[{"key":"tier","value":"gold"}],"created_at":"2026-09-16T10:00:00Z"}`

// paymentMethodResponse is the enrolled payment method the fake API returns.
const paymentMethodResponse = `{"id":"pm-1","account_id":"acc-1","name":"VISA ****1111",
	"type":"CARD","category":"CARD","country":"US","status":"ENROLLED",
	"vaulted_token":"vt-0123456789","enrollment":{"session":"cs-1"},
	"provider":{"id":"YUNO","type":"YUNO"},"created_at":"2026-09-16T10:00:00Z"}`

func TestCheckoutSessionCreate_PostsTheBody(t *testing.T) {
	isolateConfig(t)
	seedCredentials(t)

	got := startAPI(t, http.StatusOK, checkoutSessionResponse)

	out, err := runCLI(t, "", "checkout-session", "create", "--yes",
		"--account-id", "acc-1", "--customer-id", "cus-1",
		"--merchant-order-id", "order-42", "--payment-description", "Test Cards",
		"--country", "CO", "--currency", "USD", "--amount", "520",
		"--callback-url", "https://example.com/callback",
		"--metadata", `[{"key":"tier","value":"gold"}]`)
	if err != nil {
		t.Fatalf("checkout-session create failed: %v (%s)", err, out)
	}

	if got.Method != http.MethodPost || got.Path != "/v1/checkout/sessions" {
		t.Errorf("expected POST /v1/checkout/sessions, got %s %s", got.Method, got.Path)
	}

	body := decodeBody(t, got.Body)

	if body["account_id"] != "acc-1" || body["merchant_order_id"] != "order-42" ||
		body["callback_url"] != "https://example.com/callback" {
		t.Errorf("unexpected body: %s", got.Body)
	}

	amount, ok := body["amount"].(map[string]any)
	if !ok || amount["currency"] != "USD" || amount["value"] != float64(520) {
		t.Errorf("unexpected amount in body: %s", got.Body)
	}

	for _, want := range []string{"SESSION", "chk-1", "USD 520", "SDK_CHECKOUT"} {
		if !strings.Contains(out, want) {
			t.Errorf("expected table output to contain %q, got:\n%s", want, out)
		}
	}
}

func TestCheckoutSessionCreate_RequiresABody(t *testing.T) {
	isolateConfig(t)
	seedCredentials(t)
	startAPI(t, http.StatusOK, checkoutSessionResponse)

	_, err := runCLI(t, "", "checkout-session", "create", "--yes")
	if err == nil {
		t.Fatal("expected an error when no body is given")
	}

	if !strings.Contains(err.Error(), "request body is required") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestCheckoutSessionGet_HitsTheSessionPath(t *testing.T) {
	isolateConfig(t)
	seedCredentials(t)

	got := startAPI(t, http.StatusOK, checkoutSessionResponse)

	out, err := runCLI(t, "", "checkout-session", "get", "chk-1")
	if err != nil {
		t.Fatalf("checkout-session get failed: %v (%s)", err, out)
	}

	if got.Method != http.MethodGet || got.Path != "/v1/checkout/sessions/chk-1" {
		t.Errorf("expected GET /v1/checkout/sessions/chk-1, got %s %s", got.Method, got.Path)
	}
}

func TestCheckoutSessionGet_JSONKeepsUntypedFields(t *testing.T) {
	isolateConfig(t)
	seedCredentials(t)
	startAPI(t, http.StatusOK, checkoutSessionResponse)

	out, err := runCLI(t, "", "checkout-session", "get", "chk-1", "--json")
	if err != nil {
		t.Fatalf("checkout-session get failed: %v (%s)", err, out)
	}

	if !strings.Contains(out, `"metadata"`) {
		t.Errorf("expected --json to keep the untyped metadata field, got:\n%s", out)
	}
}

func TestCheckoutSessionUpdate_PatchesOnlyTheChangedFields(t *testing.T) {
	isolateConfig(t)
	seedCredentials(t)

	got := startAPI(t, http.StatusOK, checkoutSessionResponse)

	out, err := runCLI(t, "", "checkout-session", "update", "chk-1", "--yes", "--country", "BR")
	if err != nil {
		t.Fatalf("checkout-session update failed: %v (%s)", err, out)
	}

	if got.Method != http.MethodPatch || got.Path != "/v1/checkout/sessions/chk-1" {
		t.Errorf("expected PATCH /v1/checkout/sessions/chk-1, got %s %s", got.Method, got.Path)
	}

	body := decodeBody(t, got.Body)
	if len(body) != 1 || body["country"] != "BR" {
		t.Errorf("expected only the changed field in the body, got: %s", got.Body)
	}
}

func TestCheckoutSessionPaymentMethods_ListsTheMethods(t *testing.T) {
	isolateConfig(t)
	seedCredentials(t)

	got := startAPI(t, http.StatusOK, `[{"name":"VISA ****1111","type":"CARD","category":"CARD",
		"vaulted_token":"vt-0123456789","checkout":{"session":"chk-1"},
		"conditions":{"enabled":true}}]`)

	out, err := runCLI(t, "", "checkout-session", "payment-methods", "chk-1", "--category", "CARD")
	if err != nil {
		t.Fatalf("checkout-session payment-methods failed: %v (%s)", err, out)
	}

	if got.Method != http.MethodGet || got.Path != "/v1/checkout/sessions/chk-1/payment-methods" {
		t.Errorf("expected GET /v1/checkout/sessions/chk-1/payment-methods, got %s %s", got.Method, got.Path)
	}

	if got.Query != "category=CARD" {
		t.Errorf("expected the category query, got %q", got.Query)
	}

	for _, want := range []string{"VAULTED_TOKEN", "CATEGORY", "VISA ****1111", "chk-1"} {
		if !strings.Contains(out, want) {
			t.Errorf("expected table output to contain %q, got:\n%s", want, out)
		}
	}

	if strings.Contains(out, "vt-0123456789") {
		t.Errorf("expected the vaulted token to be masked in the table, got:\n%s", out)
	}
}

func TestCheckoutPaymentMethodList_UnwrapsThePayload(t *testing.T) {
	isolateConfig(t)
	seedCredentials(t)

	got := startAPI(t, http.StatusOK, `{"payment_methods":[{"name":"Visa Credit Card",
		"type":"VISA","category":"CARD","enrollment":{"session":"cs-1"}}]}`)

	out, err := runCLI(t, "", "checkout", "payment-method", "list", "cs-1")
	if err != nil {
		t.Fatalf("checkout payment-method list failed: %v (%s)", err, out)
	}

	if got.Method != http.MethodGet || got.Path != "/v1/checkout/customers/sessions/cs-1/payment-methods" {
		t.Errorf("expected GET /v1/checkout/customers/sessions/cs-1/payment-methods, got %s %s", got.Method, got.Path)
	}

	for _, want := range []string{"Visa Credit Card", "VISA", "cs-1"} {
		if !strings.Contains(out, want) {
			t.Errorf("expected table output to contain %q, got:\n%s", want, out)
		}
	}
}

func TestCheckoutPaymentMethodEnroll_PostsTheBody(t *testing.T) {
	isolateConfig(t)
	seedCredentials(t)

	got := startAPI(t, http.StatusCreated, paymentMethodResponse)

	out, err := runCLI(t, "", "checkout", "payment-method", "enroll", "cs-1", "--yes",
		"--account-id", "acc-1", "--payment-method-type", "CARD", "--country", "US",
		"--vault-on-success", "--account-updater=false")
	if err != nil {
		t.Fatalf("checkout payment-method enroll failed: %v (%s)", err, out)
	}

	if got.Method != http.MethodPost || got.Path != "/v1/customers/sessions/cs-1/payment-methods" {
		t.Errorf("expected POST /v1/customers/sessions/cs-1/payment-methods, got %s %s", got.Method, got.Path)
	}

	body := decodeBody(t, got.Body)

	if body["account_id"] != "acc-1" || body["payment_method_type"] != "CARD" ||
		body["country"] != "US" || body["account_updater"] != false {
		t.Errorf("unexpected body: %s", got.Body)
	}

	verify, ok := body["verify"].(map[string]any)
	if !ok || verify["vault_on_success"] != true {
		t.Errorf("unexpected verify block in body: %s", got.Body)
	}

	if !strings.Contains(out, "ENROLLED") {
		t.Errorf("expected the table to show the status, got:\n%s", out)
	}
}

func TestCheckoutPaymentMethodEnroll_RequiresABody(t *testing.T) {
	isolateConfig(t)
	seedCredentials(t)
	startAPI(t, http.StatusCreated, paymentMethodResponse)

	_, err := runCLI(t, "", "checkout", "payment-method", "enroll", "cs-1", "--yes")
	if err == nil {
		t.Fatal("expected an error when no body is given")
	}

	if !strings.Contains(err.Error(), "request body is required") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestCheckoutPaymentMethodGet_HitsThePaymentMethodPath(t *testing.T) {
	isolateConfig(t)
	seedCredentials(t)

	got := startAPI(t, http.StatusOK, paymentMethodResponse)

	out, err := runCLI(t, "", "checkout", "payment-method", "get", "pm-1")
	if err != nil {
		t.Fatalf("checkout payment-method get failed: %v (%s)", err, out)
	}

	if got.Method != http.MethodGet || got.Path != "/v1/payment-methods/pm-1" {
		t.Errorf("expected GET /v1/payment-methods/pm-1, got %s %s", got.Method, got.Path)
	}

	if strings.Contains(out, "vt-0123456789") {
		t.Errorf("expected the vaulted token to be masked in the table, got:\n%s", out)
	}
}

func TestCheckoutPaymentMethodGet_JSONKeepsUntypedFields(t *testing.T) {
	isolateConfig(t)
	seedCredentials(t)
	startAPI(t, http.StatusOK, paymentMethodResponse)

	out, err := runCLI(t, "", "checkout", "payment-method", "get", "pm-1", "--json")
	if err != nil {
		t.Fatalf("checkout payment-method get failed: %v (%s)", err, out)
	}

	if !strings.Contains(out, `"provider"`) {
		t.Errorf("expected --json to keep the untyped provider field, got:\n%s", out)
	}
}

func TestCheckoutPaymentMethodUnenroll_PostsToTheUnenrollPath(t *testing.T) {
	isolateConfig(t)
	seedCredentials(t)

	got := startAPI(t, http.StatusOK, `{"id":"pm-1","type":"CARD","status":"UNENROLLED"}`)

	out, err := runCLI(t, "", "checkout", "payment-method", "unenroll", "pm-1", "--yes")
	if err != nil {
		t.Fatalf("checkout payment-method unenroll failed: %v (%s)", err, out)
	}

	if got.Method != http.MethodPost || got.Path != "/v1/customers/payment-methods/pm-1/unenroll" {
		t.Errorf("expected POST /v1/customers/payment-methods/pm-1/unenroll, got %s %s", got.Method, got.Path)
	}

	if !strings.Contains(out, "UNENROLLED") {
		t.Errorf("expected the table to show the status, got:\n%s", out)
	}
}

func TestCheckoutPaymentMethodUnenroll_PromptsWithoutYes(t *testing.T) {
	isolateConfig(t)
	seedCredentials(t)

	startAPI(t, http.StatusOK, `{"id":"pm-1","status":"UNENROLLED"}`)

	out, err := runCLI(t, "n\n", "checkout", "payment-method", "unenroll", "pm-1")
	if err == nil {
		t.Fatalf("expected the declined confirmation to fail, got:\n%s", out)
	}
}

func TestCheckoutSessionPaymentMethods_ForbiddenMentionsTheScope(t *testing.T) {
	isolateConfig(t)
	seedCredentials(t)

	startAPI(t, http.StatusForbidden, `{"code":"INSUFFICIENT_SCOPE","message":"forbidden"}`)

	_, err := runCLI(t, "", "checkout-session", "payment-methods", "chk-1")
	if err == nil {
		t.Fatal("expected an error on 403")
	}

	if !strings.Contains(err.Error(), "checkout:read") {
		t.Errorf("expected the error to name the missing scope, got %v", err)
	}
}
