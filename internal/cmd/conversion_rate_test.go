package cmd_test

import (
	"net/http"
	"strings"
	"testing"
)

// conversionRateResponse is the currency conversion payload the fake API returns.
const conversionRateResponse = `{"id":"cr-1","amount":{"value":10000,"currency":"COP",
	"currency_conversion":{"cardholder_currency":"USD","cardholder_amount":2.58,"rate":3879.81,
	"provider_data":{"id":"CIBC","transaction_id":"tx-1","response_code":"2000",
	"response_message":"Successful"}}}}`

func TestConversionRateGet_PostsTheBody(t *testing.T) {
	isolateConfig(t)
	seedCredentials(t)

	got := startAPI(t, http.StatusOK, conversionRateResponse)

	out, err := runCLI(t, "", "conversion-rate", "get", "--yes",
		"--account-id", "acc-1", "--currency", "COP", "--amount", "10000",
		"--cardholder-currency", "USD", "--provider", "CIBC", "--vaulted-token", "vt-1")
	if err != nil {
		t.Fatalf("conversion-rate get failed: %v (%s)", err, out)
	}

	if got.Method != http.MethodPost || got.Path != "/v1/currency-conversion" {
		t.Errorf("expected POST /v1/currency-conversion, got %s %s", got.Method, got.Path)
	}

	body := decodeBody(t, got.Body)

	amount, ok := body["amount"].(map[string]any)
	if !ok || amount["currency"] != "COP" || amount["value"] != 10000.0 {
		t.Errorf("unexpected amount in body: %s", got.Body)
	}

	conversion, ok := amount["currency_conversion"].(map[string]any)
	if !ok || conversion["cardholder_currency"] != "USD" {
		t.Errorf("unexpected conversion in body: %s", got.Body)
	}

	provider, ok := body["provider_data"].(map[string]any)
	if !ok || provider["id"] != "CIBC" {
		t.Errorf("unexpected provider in body: %s", got.Body)
	}

	method, ok := body["payment_method"].(map[string]any)
	if !ok || method["vaulted_token"] != "vt-1" {
		t.Errorf("unexpected payment method in body: %s", got.Body)
	}

	for _, want := range []string{"cr-1", "COP 10000", "USD 2.58", "CIBC"} {
		if !strings.Contains(out, want) {
			t.Errorf("expected table output to contain %q, got:\n%s", want, out)
		}
	}
}

func TestConversionRateGet_RequiresABody(t *testing.T) {
	isolateConfig(t)
	seedCredentials(t)
	startAPI(t, http.StatusOK, conversionRateResponse)

	if _, err := runCLI(t, "", "conversion-rate", "get", "--yes"); err == nil ||
		!strings.Contains(err.Error(), "request body is required") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestConversionRateGet_ReportsTheAPIError(t *testing.T) {
	isolateConfig(t)
	seedCredentials(t)
	startAPI(t, http.StatusForbidden, `{"code":"INSUFFICIENT_SCOPE","message":"nope"}`)

	_, err := runCLI(t, "", "conversion-rate", "get", "--yes", "--account-id", "acc-1")
	if err == nil || !strings.Contains(err.Error(), "payments:read") {
		t.Errorf("expected the scope hint, got %v", err)
	}
}
