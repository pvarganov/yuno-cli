package cmd_test

import (
	"net/http"
	"strings"
	"testing"
)

// sellerResponse is the seller payload the fake API returns.
const sellerResponse = `{"seller_id":"se-1","merchant_seller_id":"shop-1","name":"Acme Shop",
	"email":"shop@acme.com","country":"US","document":{"document_type":"EIN","document_number":"123"},
	"created_at":"2026-09-16T10:00:00Z"}`

func TestSellerCreate_PostsTheBody(t *testing.T) {
	isolateConfig(t)
	seedCredentials(t)

	got := startAPI(t, http.StatusCreated, sellerResponse)

	out, err := runCLI(t, "", "seller", "create", "--yes",
		"--account-id", "acc-1", "--merchant-seller-id", "shop-1", "--name", "Acme Shop",
		"--email", "shop@acme.com", "--country", "US",
		"--document", `{"document_type":"EIN","document_number":"123"}`)
	if err != nil {
		t.Fatalf("seller create failed: %v (%s)", err, out)
	}

	if got.Method != http.MethodPost || got.Path != "/v1/sellers" {
		t.Errorf("expected POST /v1/sellers, got %s %s", got.Method, got.Path)
	}

	body := decodeBody(t, got.Body)
	if body["merchant_seller_id"] != "shop-1" || body["country"] != "US" {
		t.Errorf("unexpected body: %s", got.Body)
	}

	document, ok := body["document"].(map[string]any)
	if !ok || document["document_type"] != "EIN" {
		t.Errorf("unexpected document: %s", got.Body)
	}

	for _, want := range []string{"se-1", "shop-1", "EIN 123"} {
		if !strings.Contains(out, want) {
			t.Errorf("expected table output to contain %q, got:\n%s", want, out)
		}
	}
}

func TestSellerCreate_RejectsInvalidJSONFlags(t *testing.T) {
	isolateConfig(t)
	seedCredentials(t)
	startAPI(t, http.StatusCreated, sellerResponse)

	if _, err := runCLI(t, "", "seller", "create", "--yes", "--document", "{"); err == nil ||
		!strings.Contains(err.Error(), "not valid json") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestSellerGet_HitsTheIDPath(t *testing.T) {
	isolateConfig(t)
	seedCredentials(t)

	got := startAPI(t, http.StatusOK, sellerResponse)

	out, err := runCLI(t, "", "seller", "get", "shop-1", "--json")
	if err != nil {
		t.Fatalf("seller get failed: %v (%s)", err, out)
	}

	if got.Method != http.MethodGet || got.Path != "/v1/sellers/shop-1" {
		t.Errorf("expected GET /v1/sellers/shop-1, got %s %s", got.Method, got.Path)
	}

	if !strings.Contains(out, `"seller_id": "se-1"`) {
		t.Errorf("expected the json body, got:\n%s", out)
	}
}

func TestSellerUpdate_UsesPut(t *testing.T) {
	isolateConfig(t)
	seedCredentials(t)

	got := startAPI(t, http.StatusOK, sellerResponse)

	out, err := runCLI(t, "", "seller", "update", "shop-1", "--yes",
		"--account-id", "acc-1", "--name", "Acme Shop EU", "--country", "DE")
	if err != nil {
		t.Fatalf("seller update failed: %v (%s)", err, out)
	}

	if got.Method != http.MethodPut || got.Path != "/v1/sellers/shop-1" {
		t.Errorf("expected PUT /v1/sellers/shop-1, got %s %s", got.Method, got.Path)
	}

	body := decodeBody(t, got.Body)
	if body["name"] != "Acme Shop EU" || body["country"] != "DE" {
		t.Errorf("unexpected body: %s", got.Body)
	}
}

func TestSellerDelete_HitsTheIDPath(t *testing.T) {
	isolateConfig(t)
	seedCredentials(t)

	got := startAPI(t, http.StatusNoContent, "")

	out, err := runCLI(t, "", "seller", "delete", "shop-1", "--yes")
	if err != nil {
		t.Fatalf("seller delete failed: %v (%s)", err, out)
	}

	if got.Method != http.MethodDelete || got.Path != "/v1/sellers/shop-1" {
		t.Errorf("expected DELETE /v1/sellers/shop-1, got %s %s", got.Method, got.Path)
	}

	if strings.TrimSpace(out) != "" {
		t.Errorf("expected no output for a 204, got:\n%s", out)
	}
}

func TestSellerDelete_IsConfirmed(t *testing.T) {
	isolateConfig(t)
	seedCredentials(t)
	startAPI(t, http.StatusNoContent, "")

	if _, err := runCLI(t, "no\n", "seller", "delete", "shop-1"); err == nil {
		t.Error("expected the declined confirmation to fail the command")
	}
}
