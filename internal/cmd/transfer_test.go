package cmd_test

import (
	"net/http"
	"strings"
	"testing"
)

// transferResponse is the transfer payload the fake API returns.
const transferResponse = `{"id":"tr-1","recipient_id":"rec-1","status":"SUCCEEDED",
	"amount":{"currency":"USD","value":25},"merchant_reference":"ref-1","provider":"NUVEI",
	"provider_data":{"id":"NUVEI","response_code":"00","response_message":"OK"},
	"created_at":"2026-09-16T10:00:00Z"}`

func TestTransferCreate_PostsTheBody(t *testing.T) {
	isolateConfig(t)
	seedCredentials(t)

	got := startAPI(t, http.StatusOK, transferResponse)

	out, err := runCLI(t, "", "transfer", "create", "--yes",
		"--account-id", "acc-1", "--recipient-id", "rec-1", "--provider-id", "NUVEI",
		"--currency", "USD", "--amount", "25", "--merchant-reference", "ref-1")
	if err != nil {
		t.Fatalf("transfer create failed: %v (%s)", err, out)
	}

	if got.Method != http.MethodPost || got.Path != "/v1/split-marketplace/transfers" {
		t.Errorf("expected POST /v1/split-marketplace/transfers, got %s %s", got.Method, got.Path)
	}

	body := decodeBody(t, got.Body)
	if body["recipient_id"] != "rec-1" || body["provider_id"] != "NUVEI" {
		t.Errorf("unexpected body: %s", got.Body)
	}

	amount, ok := body["amount"].(map[string]any)
	if !ok || amount["value"] != 25.0 {
		t.Errorf("unexpected amount in body: %s", got.Body)
	}

	for _, want := range []string{"tr-1", "SUCCEEDED", "USD 25", "NUVEI"} {
		if !strings.Contains(out, want) {
			t.Errorf("expected table output to contain %q, got:\n%s", want, out)
		}
	}
}

func TestTransferCreate_RequiresABody(t *testing.T) {
	isolateConfig(t)
	seedCredentials(t)
	startAPI(t, http.StatusOK, transferResponse)

	if _, err := runCLI(t, "", "transfer", "create", "--yes"); err == nil ||
		!strings.Contains(err.Error(), "request body is required") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestTransferGet_HitsTheIDPath(t *testing.T) {
	isolateConfig(t)
	seedCredentials(t)

	got := startAPI(t, http.StatusOK, transferResponse)

	out, err := runCLI(t, "", "transfer", "get", "tr-1", "--json")
	if err != nil {
		t.Fatalf("transfer get failed: %v (%s)", err, out)
	}

	if got.Method != http.MethodGet || got.Path != "/v1/split-marketplace/transfers/tr-1" {
		t.Errorf("expected GET /v1/split-marketplace/transfers/tr-1, got %s %s", got.Method, got.Path)
	}

	if !strings.Contains(out, "provider_data") {
		t.Errorf("expected the whole response with --json, got:\n%s", out)
	}
}

func TestTransferReverse_PostsToTheReversePath(t *testing.T) {
	isolateConfig(t)
	seedCredentials(t)

	got := startAPI(t, http.StatusOK, `{"id":"tr-1","status":"REVERSED"}`)

	out, err := runCLI(t, "", "transfer", "reverse", "tr-1", "--yes",
		"--account-id", "acc-1", "--currency", "USD", "--amount", "10", "--reason", "DUPLICATE")
	if err != nil {
		t.Fatalf("transfer reverse failed: %v (%s)", err, out)
	}

	if got.Method != http.MethodPost || got.Path != "/v1/split-marketplace/transfers/tr-1/reverse" {
		t.Errorf("expected POST /v1/split-marketplace/transfers/tr-1/reverse, got %s %s", got.Method, got.Path)
	}

	body := decodeBody(t, got.Body)
	if body["reason"] != "DUPLICATE" || body["account_id"] != "acc-1" {
		t.Errorf("unexpected body: %s", got.Body)
	}

	if !strings.Contains(out, "REVERSED") {
		t.Errorf("expected the reversed status in the table, got:\n%s", out)
	}
}

func TestTransferReverse_SendsNoBodyWhenNoFlagIsSet(t *testing.T) {
	isolateConfig(t)
	seedCredentials(t)

	got := startAPI(t, http.StatusOK, `{"id":"tr-1","status":"REVERSED"}`)

	if out, err := runCLI(t, "", "transfer", "reverse", "tr-1", "--yes"); err != nil {
		t.Fatalf("transfer reverse failed: %v (%s)", err, out)
	}

	if strings.TrimSpace(got.Body) != "" {
		t.Errorf("expected an empty body for a full reversal, got %q", got.Body)
	}
}

func TestTransferReverse_ReportsTheScopeHint(t *testing.T) {
	isolateConfig(t)
	seedCredentials(t)
	startAPI(t, http.StatusForbidden, `{"code":"INSUFFICIENT_SCOPE","message":"nope"}`)

	_, err := runCLI(t, "", "transfer", "reverse", "tr-1", "--yes")
	if err == nil || !strings.Contains(err.Error(), "split-marketplace") {
		t.Errorf("expected the scope hint, got %v", err)
	}
}
