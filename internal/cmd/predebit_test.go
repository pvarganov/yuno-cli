package cmd_test

import (
	"net/http"
	"strings"
	"testing"
)

// preDebitResponse is the pre-debit notification payload the fake API returns.
const preDebitResponse = `{"id":"pdn-1","account_id":"acc-1","status":"NOTIFIED",
	"merchant_reference":"ref-1","amount":{"currency":"INR","value":"1000"},
	"billing_date":"2026-10-01","origin_payment_id":"pay-1",
	"provider_data":{"id":"RAZORPAY"},"created_at":"2026-09-16T10:00:00Z"}`

func TestPreDebitCreate_PostsTheBody(t *testing.T) {
	isolateConfig(t)
	seedCredentials(t)

	got := startAPI(t, http.StatusCreated, preDebitResponse)

	out, err := runCLI(t, "", "pre-debit", "create", "--yes",
		"--account-id", "acc-1", "--merchant-reference", "ref-1",
		"--currency", "INR", "--amount", "1000", "--billing-date", "2026-10-01",
		"--merchant-customer-id", "cus-1", "--payment-method-type", "BANK_TRANSFER")
	if err != nil {
		t.Fatalf("pre-debit create failed: %v (%s)", err, out)
	}

	if got.Method != http.MethodPost || got.Path != "/v1/predebit-notify" {
		t.Errorf("expected POST /v1/predebit-notify, got %s %s", got.Method, got.Path)
	}

	body := decodeBody(t, got.Body)

	amount, ok := body["amount"].(map[string]any)
	if !ok || amount["value"] != "1000" {
		t.Errorf("expected the amount value to stay a string, got: %s", got.Body)
	}

	payer, ok := body["customer_payer"].(map[string]any)
	if !ok || payer["merchant_customer_id"] != "cus-1" {
		t.Errorf("unexpected customer payer in body: %s", got.Body)
	}

	for _, want := range []string{"pdn-1", "NOTIFIED", "INR 1000", "RAZORPAY"} {
		if !strings.Contains(out, want) {
			t.Errorf("expected table output to contain %q, got:\n%s", want, out)
		}
	}
}

func TestPreDebitCreate_RequiresABody(t *testing.T) {
	isolateConfig(t)
	seedCredentials(t)
	startAPI(t, http.StatusCreated, preDebitResponse)

	if _, err := runCLI(t, "", "pre-debit", "create", "--yes"); err == nil ||
		!strings.Contains(err.Error(), "request body is required") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestPreDebitList_FiltersByMerchantReference(t *testing.T) {
	isolateConfig(t)
	seedCredentials(t)

	got := startAPI(t, http.StatusOK, preDebitResponse)

	out, err := runCLI(t, "", "pre-debit", "list", "--merchant-reference", "ref-1")
	if err != nil {
		t.Fatalf("pre-debit list failed: %v (%s)", err, out)
	}

	if got.Method != http.MethodGet || got.Path != "/v1/predebit-notify" {
		t.Errorf("expected GET /v1/predebit-notify, got %s %s", got.Method, got.Path)
	}

	if got.Query != "merchant_reference=ref-1" {
		t.Errorf("unexpected query: %s", got.Query)
	}

	if !strings.Contains(out, "pdn-1") {
		t.Errorf("expected the notification in the table, got:\n%s", out)
	}
}

func TestPreDebitList_RequiresTheMerchantReference(t *testing.T) {
	isolateConfig(t)
	seedCredentials(t)
	startAPI(t, http.StatusOK, preDebitResponse)

	if _, err := runCLI(t, "", "pre-debit", "list"); err == nil ||
		!strings.Contains(err.Error(), "merchant-reference") {
		t.Errorf("expected the required flag error, got %v", err)
	}
}

func TestPreDebitGet_HitsTheIDPath(t *testing.T) {
	isolateConfig(t)
	seedCredentials(t)

	got := startAPI(t, http.StatusOK, preDebitResponse)

	out, err := runCLI(t, "", "pre-debit", "get", "pdn-1", "--json")
	if err != nil {
		t.Fatalf("pre-debit get failed: %v (%s)", err, out)
	}

	if got.Method != http.MethodGet || got.Path != "/v1/predebit-notify/pdn-1" {
		t.Errorf("expected GET /v1/predebit-notify/pdn-1, got %s %s", got.Method, got.Path)
	}

	if !strings.Contains(out, "origin_payment_id") {
		t.Errorf("expected the whole response with --json, got:\n%s", out)
	}
}

func TestPreDebitGet_ReportsTheScopeHint(t *testing.T) {
	isolateConfig(t)
	seedCredentials(t)
	startAPI(t, http.StatusForbidden, `{"code":"INSUFFICIENT_SCOPE","message":"nope"}`)

	_, err := runCLI(t, "", "pre-debit", "get", "pdn-1")
	if err == nil || !strings.Contains(err.Error(), "predebit-notify:read") {
		t.Errorf("expected the scope hint, got %v", err)
	}
}
