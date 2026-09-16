package cmd_test

import (
	"net/http"
	"strings"
	"testing"
)

// enrolledMethodResponse is the enrolled payment method the fake API returns.
const enrolledMethodResponse = `{"id":"pm-1","account_id":"acc-1","customer_id":"cus-1",
	"name":"VISA ****1111","type":"CARD","category":"CARD","country":"US","status":"ENROLLED",
	"vaulted_token":"46adcbc0-aa26-4867-a4e7-28a5ad9100ae",
	"card_data":{"iin":"41111111","lfd":"1111","brand":"VISA","type":"CREDIT",
	"number":"4111111111111111","holder_name":"JOHN DOE"},
	"created_at":"2026-09-16T10:00:00Z"}`

func TestPaymentMethodList_HitsTheCustomerPath(t *testing.T) {
	isolateConfig(t)
	seedCredentials(t)

	got := startAPI(t, http.StatusOK, `{"payment_methods":[`+enrolledMethodResponse+`]}`)

	out, err := runCLI(t, "", "payment-method", "list", "cus-1")
	if err != nil {
		t.Fatalf("payment-method list failed: %v (%s)", err, out)
	}

	if got.Method != http.MethodGet || got.Path != "/v1/customers/cus-1/payment-methods" {
		t.Errorf("expected GET /v1/customers/cus-1/payment-methods, got %s %s", got.Method, got.Path)
	}

	for _, want := range []string{"VAULTED_TOKEN", "CARD", "VISA ****1111", "ENROLLED"} {
		if !strings.Contains(out, want) {
			t.Errorf("expected table output to contain %q, got:\n%s", want, out)
		}
	}
}

func TestPaymentMethodList_MasksCardDataInTheTable(t *testing.T) {
	isolateConfig(t)
	seedCredentials(t)
	startAPI(t, http.StatusOK, `{"payment_methods":[`+enrolledMethodResponse+`]}`)

	out, err := runCLI(t, "", "payment-method", "list", "cus-1")
	if err != nil {
		t.Fatalf("payment-method list failed: %v (%s)", err, out)
	}

	if strings.Contains(out, "4111111111111111") {
		t.Errorf("table output leaks the card number:\n%s", out)
	}

	if strings.Contains(out, "46adcbc0-aa26-4867-a4e7-28a5ad9100ae") {
		t.Errorf("table output leaks the vaulted token:\n%s", out)
	}

	if !strings.Contains(out, "46ad****") {
		t.Errorf("expected the masked vaulted token, got:\n%s", out)
	}
}

func TestPaymentMethodList_JSONKeepsTheRawResponse(t *testing.T) {
	isolateConfig(t)
	seedCredentials(t)
	startAPI(t, http.StatusOK, `{"payment_methods":[`+enrolledMethodResponse+`]}`)

	out, err := runCLI(t, "", "payment-method", "list", "cus-1", "--json")
	if err != nil {
		t.Fatalf("payment-method list failed: %v (%s)", err, out)
	}

	if !strings.Contains(out, "holder_name") || !strings.Contains(out, "JOHN DOE") {
		t.Errorf("expected the raw response to survive --json, got:\n%s", out)
	}
}

func TestPaymentMethodGet_HitsTheCustomerScopedPath(t *testing.T) {
	isolateConfig(t)
	seedCredentials(t)

	got := startAPI(t, http.StatusOK, enrolledMethodResponse)

	out, err := runCLI(t, "", "payment-method", "get", "cus-1", "pm-1")
	if err != nil {
		t.Fatalf("payment-method get failed: %v (%s)", err, out)
	}

	if got.Method != http.MethodGet || got.Path != "/v1/customers/cus-1/payment-methods/pm-1" {
		t.Errorf("expected GET /v1/customers/cus-1/payment-methods/pm-1, got %s %s", got.Method, got.Path)
	}

	if strings.Contains(out, "4111111111111111") {
		t.Errorf("table output leaks the card number:\n%s", out)
	}
}

func TestPaymentMethodGet_WantsBothIDs(t *testing.T) {
	isolateConfig(t)
	seedCredentials(t)
	startAPI(t, http.StatusOK, enrolledMethodResponse)

	if _, err := runCLI(t, "", "payment-method", "get", "cus-1"); err == nil {
		t.Fatal("expected an error when the payment method id is missing")
	}
}

func TestPaymentMethodEnroll_PostsTheBody(t *testing.T) {
	isolateConfig(t)
	seedCredentials(t)

	got := startAPI(t, http.StatusOK, enrolledMethodResponse)

	out, err := runCLI(t, "", "payment-method", "enroll", "cus-1", "--yes",
		"--account-id", "acc-1", "--type", "CARD", "--country", "US",
		"--workflow", "DIRECT", "--account-updater", "--vault-on-success")
	if err != nil {
		t.Fatalf("payment-method enroll failed: %v (%s)", err, out)
	}

	if got.Method != http.MethodPost || got.Path != "/v1/customers/cus-1/payment-methods" {
		t.Errorf("expected POST /v1/customers/cus-1/payment-methods, got %s %s", got.Method, got.Path)
	}

	body := decodeBody(t, got.Body)

	if body["account_id"] != "acc-1" || body["type"] != "CARD" || body["country"] != "US" {
		t.Errorf("unexpected body: %s", got.Body)
	}

	if body["workflow"] != "DIRECT" || body["account_updater"] != true {
		t.Errorf("unexpected body: %s", got.Body)
	}

	verify, ok := body["verify"].(map[string]any)
	if !ok || verify["vault_on_success"] != true {
		t.Errorf("unexpected verify block in body: %s", got.Body)
	}
}

func TestPaymentMethodEnroll_RequiresABody(t *testing.T) {
	isolateConfig(t)
	seedCredentials(t)
	startAPI(t, http.StatusOK, enrolledMethodResponse)

	_, err := runCLI(t, "", "payment-method", "enroll", "cus-1", "--yes")
	if err == nil {
		t.Fatal("expected an error when no body is given")
	}

	if !strings.Contains(err.Error(), "request body is required") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestPaymentMethodUnenroll_HitsTheUnenrollPath(t *testing.T) {
	isolateConfig(t)
	seedCredentials(t)

	got := startAPI(t, http.StatusOK, `{"id":"pm-1","status":"UNENROLLED"}`)

	out, err := runCLI(t, "", "payment-method", "unenroll", "cus-1", "pm-1", "--yes")
	if err != nil {
		t.Fatalf("payment-method unenroll failed: %v (%s)", err, out)
	}

	if got.Method != http.MethodPost || got.Path != "/v1/customers/cus-1/payment-methods/pm-1/unenroll" {
		t.Errorf("unexpected request %s %s", got.Method, got.Path)
	}

	if !strings.Contains(out, "UNENROLLED") {
		t.Errorf("expected the new status in the table, got:\n%s", out)
	}
}

func TestPaymentMethodUpdate_PatchesAndSendsTheAccountCode(t *testing.T) {
	isolateConfig(t)
	seedCredentials(t)

	got := startAPI(t, http.StatusOK, enrolledMethodResponse)

	out, err := runCLI(t, "", "payment-method", "update", "pm-1", "--yes",
		"--customer-id", "cus-2", "--account-code", "acc-code")
	if err != nil {
		t.Fatalf("payment-method update failed: %v (%s)", err, out)
	}

	if got.Method != http.MethodPatch || got.Path != "/v1/payment-methods/pm-1" {
		t.Errorf("expected PATCH /v1/payment-methods/pm-1, got %s %s", got.Method, got.Path)
	}

	if code := got.Header.Get("X-Account-Code"); code != "acc-code" {
		t.Errorf("X-Account-Code = %q, want acc-code", code)
	}

	if body := decodeBody(t, got.Body); body["customer_id"] != "cus-2" {
		t.Errorf("unexpected body: %s", got.Body)
	}
}

func TestPaymentMethodUpdate_RequiresABody(t *testing.T) {
	isolateConfig(t)
	seedCredentials(t)
	startAPI(t, http.StatusOK, enrolledMethodResponse)

	_, err := runCLI(t, "", "payment-method", "update", "pm-1", "--yes")
	if err == nil {
		t.Fatal("expected an error when no body is given")
	}

	if !strings.Contains(err.Error(), "request body is required") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestPaymentMethodAccountUpdater_PostsTheIDs(t *testing.T) {
	isolateConfig(t)
	seedCredentials(t)

	got := startAPI(t, http.StatusOK, `{"accepted":2}`)

	out, err := runCLI(t, "", "payment-method", "account-updater", "--yes",
		"--payment-method-id", "pm-1", "--payment-method-id", "pm-2")
	if err != nil {
		t.Fatalf("payment-method account-updater failed: %v (%s)", err, out)
	}

	if got.Method != http.MethodPost || got.Path != "/v1/payment-methods/account-updater" {
		t.Errorf("expected POST /v1/payment-methods/account-updater, got %s %s", got.Method, got.Path)
	}

	ids, ok := decodeBody(t, got.Body)["payment_method_ids"].([]any)
	if !ok || len(ids) != 2 || ids[0] != "pm-1" || ids[1] != "pm-2" {
		t.Errorf("unexpected body: %s", got.Body)
	}

	if !strings.Contains(out, "ACCEPTED") || !strings.Contains(out, "2") {
		t.Errorf("expected the accepted count in the table, got:\n%s", out)
	}
}

func TestPaymentMethodAccountUpdater_ForbiddenHintsTheScope(t *testing.T) {
	isolateConfig(t)
	seedCredentials(t)
	startAPI(t, http.StatusForbidden, `{"code":"AUTHORIZATION_REQUIRED","message":"no"}`)

	_, err := runCLI(t, "", "payment-method", "account-updater", "--yes", "--payment-method-id", "pm-1")
	if err == nil {
		t.Fatal("expected an error on 403")
	}

	if !strings.Contains(err.Error(), "payment-method:write") {
		t.Errorf("expected the scope hint, got: %v", err)
	}
}
