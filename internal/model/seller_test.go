package model

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestSellerView_LabelsTheDocument(t *testing.T) {
	t.Parallel()

	var seller Seller
	if err := json.Unmarshal([]byte(`{"seller_id":"se-1","merchant_seller_id":"shop-1",
		"name":"Acme Shop","email":"shop@acme.com","country":"US",
		"document":{"document_type":"EIN","document_number":"123"},
		"created_at":"2026-09-16T10:00:00Z"}`), &seller); err != nil {
		t.Fatalf("decode seller: %v", err)
	}

	view := seller.View()
	if view.SellerID != "se-1" || view.MerchantSellerID != "shop-1" || view.Document != "EIN 123" {
		t.Errorf("unexpected view: %+v", view)
	}
}

func TestSellerView_HandlesAMissingDocument(t *testing.T) {
	t.Parallel()

	seller := Seller{SellerID: "se-1"}
	if got := seller.View().Document; got != "" {
		t.Errorf("expected no document, got %q", got)
	}
}

func TestSellerMarshal_ReplaysTheOriginalBody(t *testing.T) {
	t.Parallel()

	var seller Seller
	if err := json.Unmarshal([]byte(`{"seller_id":"se-1","unknown_field":"kept"}`), &seller); err != nil {
		t.Fatalf("decode seller: %v", err)
	}

	out, err := json.Marshal(seller)
	if err != nil {
		t.Fatalf("marshal seller: %v", err)
	}

	if !strings.Contains(string(out), "unknown_field") {
		t.Errorf("expected the original body replayed, got %s", out)
	}
}

func TestSellerMarshal_FallsBackToTheTypedFields(t *testing.T) {
	t.Parallel()

	out, err := json.Marshal(Seller{SellerID: "se-1"})
	if err != nil {
		t.Fatalf("marshal seller: %v", err)
	}

	if !strings.Contains(string(out), `"seller_id":"se-1"`) {
		t.Errorf("unexpected json: %s", out)
	}
}
