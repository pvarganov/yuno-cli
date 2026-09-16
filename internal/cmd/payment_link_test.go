package cmd_test

import (
	"net/http"
	"strings"
	"testing"
)

// paymentLinkResponse is the payment link payload the fake API returns.
const paymentLinkResponse = `{"code":"pl-1","country":"US","status":"CREATED",
	"description":"Test link","merchant_order_id":"order-1",
	"amount":{"currency":"USD","value":50},"payment_method_types":["CARD","PSE"],
	"checkout_url":"https://pay.y.uno/pl-1","one_time_use":true,
	"availability":{"finish_at":"2026-10-29T14:00:12Z"}}`

func TestPaymentLinkCreate_PostsTheBody(t *testing.T) {
	isolateConfig(t)
	seedCredentials(t)

	got := startAPI(t, http.StatusOK, paymentLinkResponse)

	out, err := runCLI(t, "", "payment-link", "create", "--yes",
		"--account-id", "acc-1", "--country", "US", "--description", "Test link",
		"--merchant-order-id", "order-1", "--currency", "USD", "--amount", "50",
		"--payment-method-type", "CARD", "--payment-method-type", "PSE",
		"--one-time-use", "--capture=false", "--callback-url", "https://example.com/return")
	if err != nil {
		t.Fatalf("payment-link create failed: %v (%s)", err, out)
	}

	if got.Method != http.MethodPost || got.Path != "/v1/payment-links" {
		t.Errorf("expected POST /v1/payment-links, got %s %s", got.Method, got.Path)
	}

	body := decodeBody(t, got.Body)

	amount, ok := body["amount"].(map[string]any)
	if !ok || amount["currency"] != "USD" || amount["value"] != 50.0 {
		t.Errorf("unexpected amount in body: %s", got.Body)
	}

	methods, ok := body["payment_method_types"].([]any)
	if !ok || len(methods) != 2 || methods[1] != "PSE" {
		t.Errorf("unexpected payment methods in body: %s", got.Body)
	}

	if body["one_time_use"] != true || body["capture"] != false {
		t.Errorf("unexpected booleans in body: %s", got.Body)
	}

	for _, want := range []string{"pl-1", "CREATED", "USD 50", "CARD,PSE"} {
		if !strings.Contains(out, want) {
			t.Errorf("expected table output to contain %q, got:\n%s", want, out)
		}
	}
}

func TestPaymentLinkCreate_RequiresABody(t *testing.T) {
	isolateConfig(t)
	seedCredentials(t)
	startAPI(t, http.StatusOK, paymentLinkResponse)

	if _, err := runCLI(t, "", "payment-link", "create", "--yes"); err == nil ||
		!strings.Contains(err.Error(), "request body is required") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestPaymentLinkGet_HitsTheCodePath(t *testing.T) {
	isolateConfig(t)
	seedCredentials(t)

	got := startAPI(t, http.StatusOK, paymentLinkResponse)

	out, err := runCLI(t, "", "payment-link", "get", "pl-1", "--json")
	if err != nil {
		t.Fatalf("payment-link get failed: %v (%s)", err, out)
	}

	if got.Method != http.MethodGet || got.Path != "/v1/payment-links/pl-1" {
		t.Errorf("expected GET /v1/payment-links/pl-1, got %s %s", got.Method, got.Path)
	}

	if !strings.Contains(out, "checkout_url") {
		t.Errorf("expected the whole response with --json, got:\n%s", out)
	}
}

func TestPaymentLinkCancel_PostsToTheCancelPath(t *testing.T) {
	isolateConfig(t)
	seedCredentials(t)

	got := startAPI(t, http.StatusOK, `{"code":"pl-1","status":"CANCELED"}`)

	out, err := runCLI(t, "", "payment-link", "cancel", "pl-1", "--yes")
	if err != nil {
		t.Fatalf("payment-link cancel failed: %v (%s)", err, out)
	}

	if got.Method != http.MethodPost || got.Path != "/v1/payment-links/pl-1/cancel" {
		t.Errorf("expected POST /v1/payment-links/pl-1/cancel, got %s %s", got.Method, got.Path)
	}

	if !strings.Contains(out, "CANCELED") {
		t.Errorf("expected the canceled status in the table, got:\n%s", out)
	}
}

func TestPaymentLinkCancel_ReportsTheAPIError(t *testing.T) {
	isolateConfig(t)
	seedCredentials(t)
	startAPI(t, http.StatusForbidden, `{"code":"INSUFFICIENT_SCOPE","message":"nope"}`)

	_, err := runCLI(t, "", "payment-link", "cancel", "pl-1", "--yes")
	if err == nil || !strings.Contains(err.Error(), "payment-links") {
		t.Errorf("expected the scope hint, got %v", err)
	}
}
