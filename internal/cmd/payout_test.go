package cmd_test

import (
	"net/http"
	"strings"
	"testing"
)

// payoutResponse is the payout payload the fake API returns.
const payoutResponse = `{"id":"po-1","account_id":"acc-1","status":"SUCCEEDED",
	"merchant_reference":"ref-1","purpose":"SALARY","country":"US",
	"amount":{"currency":"USD","value":100},
	"beneficiary":{"merchant_beneficiary_id":"ben-1","first_name":"Ada","last_name":"Lovelace"},
	"created_at":"2026-09-16T10:00:00Z"}`

func TestPayoutCreate_PostsTheBody(t *testing.T) {
	isolateConfig(t)
	seedCredentials(t)

	got := startAPI(t, http.StatusCreated, payoutResponse)

	out, err := runCLI(t, "", "payout", "create", "--yes",
		"--account-id", "acc-1", "--merchant-reference", "ref-1", "--country", "US",
		"--purpose", "SALARY", "--currency", "USD", "--amount", "100",
		"--beneficiary-id", "ben-1", "--beneficiary-country", "US",
		"--beneficiary-first-name", "Ada", "--beneficiary-last-name", "Lovelace",
		"--withdrawal-type", "BANK_TRANSFER", "--provider-id", "NUVEI")
	if err != nil {
		t.Fatalf("payout create failed: %v (%s)", err, out)
	}

	if got.Method != http.MethodPost || got.Path != "/v1/payouts" {
		t.Errorf("expected POST /v1/payouts, got %s %s", got.Method, got.Path)
	}

	body := decodeBody(t, got.Body)

	amount, ok := body["amount"].(map[string]any)
	if !ok || amount["currency"] != "USD" || amount["value"] != 100.0 {
		t.Errorf("unexpected amount in body: %s", got.Body)
	}

	beneficiary, ok := body["beneficiary"].(map[string]any)
	if !ok || beneficiary["merchant_beneficiary_id"] != "ben-1" || beneficiary["first_name"] != "Ada" {
		t.Errorf("unexpected beneficiary in body: %s", got.Body)
	}

	withdrawal, ok := body["withdrawal_method"].(map[string]any)
	if !ok || withdrawal["type"] != "BANK_TRANSFER" || withdrawal["provider_id"] != "NUVEI" {
		t.Errorf("unexpected withdrawal method in body: %s", got.Body)
	}

	for _, want := range []string{"po-1", "SUCCEEDED", "USD 100", "Ada Lovelace"} {
		if !strings.Contains(out, want) {
			t.Errorf("expected table output to contain %q, got:\n%s", want, out)
		}
	}
}

func TestPayoutCreate_RequiresABody(t *testing.T) {
	isolateConfig(t)
	seedCredentials(t)
	startAPI(t, http.StatusCreated, payoutResponse)

	if _, err := runCLI(t, "", "payout", "create", "--yes"); err == nil ||
		!strings.Contains(err.Error(), "request body is required") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestPayoutList_FiltersByMerchantReference(t *testing.T) {
	isolateConfig(t)
	seedCredentials(t)

	got := startAPI(t, http.StatusOK, "["+payoutResponse+"]")

	out, err := runCLI(t, "", "payout", "list", "--merchant-reference", "ref-1")
	if err != nil {
		t.Fatalf("payout list failed: %v (%s)", err, out)
	}

	if got.Method != http.MethodGet || got.Path != "/v1/payouts" {
		t.Errorf("expected GET /v1/payouts, got %s %s", got.Method, got.Path)
	}

	if got.Query != "merchant_reference=ref-1" {
		t.Errorf("unexpected query: %s", got.Query)
	}

	if !strings.Contains(out, "po-1") {
		t.Errorf("expected the payout in the table, got:\n%s", out)
	}
}

func TestPayoutList_RequiresTheMerchantReference(t *testing.T) {
	isolateConfig(t)
	seedCredentials(t)
	startAPI(t, http.StatusOK, "[]")

	if _, err := runCLI(t, "", "payout", "list"); err == nil ||
		!strings.Contains(err.Error(), "merchant-reference") {
		t.Errorf("expected the required flag error, got %v", err)
	}
}

func TestPayoutGet_HitsTheIDPath(t *testing.T) {
	isolateConfig(t)
	seedCredentials(t)

	got := startAPI(t, http.StatusOK, payoutResponse)

	out, err := runCLI(t, "", "payout", "get", "po-1", "--json")
	if err != nil {
		t.Fatalf("payout get failed: %v (%s)", err, out)
	}

	if got.Method != http.MethodGet || got.Path != "/v1/payouts/po-1" {
		t.Errorf("expected GET /v1/payouts/po-1, got %s %s", got.Method, got.Path)
	}

	if !strings.Contains(out, "merchant_reference") {
		t.Errorf("expected the whole response with --json, got:\n%s", out)
	}
}

func TestPayoutGet_ReportsTheScopeHint(t *testing.T) {
	isolateConfig(t)
	seedCredentials(t)
	startAPI(t, http.StatusForbidden, `{"code":"INSUFFICIENT_SCOPE","message":"nope"}`)

	_, err := runCLI(t, "", "payout", "get", "po-1")
	if err == nil || !strings.Contains(err.Error(), "payouts:read") {
		t.Errorf("expected the scope hint, got %v", err)
	}
}
