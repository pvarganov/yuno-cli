package model_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/pvarganov/yuno-cli/internal/model"
)

func TestCustomerView_FlattensNameAndDocument(t *testing.T) {
	var customer model.Customer

	raw := `{"id":"cus-1","merchant_customer_id":"user-42","first_name":"Ada","last_name":"Lovelace",
		"email":"ada@example.com","country":"CO",
		"document":{"document_number":"123456","document_type":"CC"},
		"created_at":"2026-09-16T10:00:00Z"}`

	if err := json.Unmarshal([]byte(raw), &customer); err != nil {
		t.Fatalf("unmarshal customer: %v", err)
	}

	view := customer.View()

	if view.Name != "Ada Lovelace" {
		t.Errorf("unexpected name: %q", view.Name)
	}

	if view.Document != "CC 123456" {
		t.Errorf("unexpected document: %q", view.Document)
	}

	if view.ID != "cus-1" || view.MerchantCustomerID != "user-42" || view.Country != "CO" {
		t.Errorf("unexpected view: %+v", view)
	}
}

func TestCustomerView_HandlesMissingParts(t *testing.T) {
	customer := model.Customer{ID: "cus-1", FirstName: "Ada"}

	view := customer.View()

	if view.Name != "Ada" {
		t.Errorf("unexpected name: %q", view.Name)
	}

	if view.Document != "" {
		t.Errorf("expected an empty document for a customer without one, got %q", view.Document)
	}
}

func TestCustomerViews_FlattensAList(t *testing.T) {
	views := model.CustomerViews([]model.Customer{{ID: "cus-1"}, {ID: "cus-2"}})

	if len(views) != 2 || views[0].ID != "cus-1" || views[1].ID != "cus-2" {
		t.Errorf("unexpected views: %+v", views)
	}
}

func TestCustomerMarshalJSON_ReplaysTheRawResponse(t *testing.T) {
	var customer model.Customer

	raw := `{"id":"cus-1","metadata":[{"key":"tier","value":"gold"}]}`

	if err := json.Unmarshal([]byte(raw), &customer); err != nil {
		t.Fatalf("unmarshal customer: %v", err)
	}

	out, err := json.Marshal(customer)
	if err != nil {
		t.Fatalf("marshal customer: %v", err)
	}

	if !strings.Contains(string(out), "gold") {
		t.Errorf("expected the untyped fields to survive, got %s", out)
	}
}

func TestCustomerMarshalJSON_FallsBackToTheTypedFields(t *testing.T) {
	customer := model.Customer{ID: "cus-1", Email: "ada@example.com"}

	out, err := json.Marshal(customer)
	if err != nil {
		t.Fatalf("marshal customer: %v", err)
	}

	if !strings.Contains(string(out), `"id":"cus-1"`) {
		t.Errorf("unexpected json: %s", out)
	}
}

func TestCustomerSessionView_FlattensTheSession(t *testing.T) {
	session := model.CustomerSession{
		CustomerSession: "sess-1",
		CustomerID:      "cus-1",
		Country:         "CO",
		CheckoutID:      "chk-1",
		CreatedAt:       "2026-09-16T10:00:00Z",
	}

	view := session.View()

	if view.CustomerSession != "sess-1" || view.CheckoutID != "chk-1" {
		t.Errorf("unexpected view: %+v", view)
	}
}

func TestNetworkTokenCryptogramView_FlattensTheCryptogram(t *testing.T) {
	cryptogram := model.NetworkTokenCryptogram{
		VaultedToken: "vt-1",
		NetworkToken: "nt-1",
		Cryptogram:   "AgAAAA",
		ECI:          "05",
	}

	view := cryptogram.View()

	if view.NetworkToken != "nt-1" || view.Cryptogram != "AgAAAA" || view.ECI != "05" {
		t.Errorf("unexpected view: %+v", view)
	}
}
