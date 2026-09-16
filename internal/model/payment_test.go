package model

import (
	"encoding/json"
	"testing"
)

func TestPaymentUnmarshalKeepsTheRawBody(t *testing.T) {
	raw := `{"id":"pay-1","status":"SUCCEEDED","unknown_field":{"a":1}}`

	var payment Payment
	if err := json.Unmarshal([]byte(raw), &payment); err != nil {
		t.Fatalf("unmarshal payment: %v", err)
	}

	if payment.ID != "pay-1" || payment.Status != "SUCCEEDED" {
		t.Errorf("unexpected typed fields: %+v", payment)
	}

	out, err := json.Marshal(payment)
	if err != nil {
		t.Fatalf("marshal payment: %v", err)
	}

	var decoded map[string]any
	if err := json.Unmarshal(out, &decoded); err != nil {
		t.Fatalf("marshalled payment is not json: %v", err)
	}

	if _, ok := decoded["unknown_field"]; !ok {
		t.Errorf("expected unknown_field to survive, got %s", out)
	}
}

func TestPaymentMarshalWithoutRawUsesTheTypedFields(t *testing.T) {
	payment := Payment{ID: "pay-2", Amount: Amount{Currency: "USD", Value: 12.5}}

	out, err := json.Marshal(payment)
	if err != nil {
		t.Fatalf("marshal payment: %v", err)
	}

	var decoded map[string]any
	if err := json.Unmarshal(out, &decoded); err != nil {
		t.Fatalf("marshalled payment is not json: %v", err)
	}

	if decoded["id"] != "pay-2" {
		t.Errorf("unexpected marshalled payment: %s", out)
	}
}

func TestPaymentUnmarshalRejectsBrokenJSON(t *testing.T) {
	var payment Payment
	if err := json.Unmarshal([]byte(`{"amount":"not an object"}`), &payment); err == nil {
		t.Fatal("expected an error for a payment with a wrong amount type")
	}
}

func TestPaymentViewFlattensTheAmountAndMethod(t *testing.T) {
	payment := Payment{
		ID:              "pay-1",
		MerchantOrderID: "order-42",
		Status:          "SUCCEEDED",
		SubStatus:       "APPROVED",
		Amount:          Amount{Currency: "USD", Value: 150},
		PaymentMethod: PaymentMethod{
			Type:   "CARD",
			Detail: &PaymentMethodDetail{Card: &CardDetail{Brand: "VISA", LFD: "1234"}},
		},
	}

	view := payment.View()

	if view.Amount != "150" || view.Currency != "USD" {
		t.Errorf("unexpected amount cells: %+v", view)
	}

	if view.PaymentMethod != "CARD VISA 1234" {
		t.Errorf("unexpected payment method label: %q", view.PaymentMethod)
	}
}

func TestPaymentMethodLabelWithoutCardDetail(t *testing.T) {
	method := PaymentMethod{Type: "PSE"}

	if got := method.Label(); got != "PSE" {
		t.Errorf("expected PSE, got %q", got)
	}
}

func TestPaymentViewsAndTransactionViews(t *testing.T) {
	payments := []Payment{{
		ID: "pay-1",
		Transactions: []Transaction{{
			ID: "t-1", Type: "PURCHASE", Status: "SUCCEEDED",
			ProviderID: "STRIPE", Amount: Amount{Currency: "USD", Value: 10.5},
		}},
	}}

	views := PaymentViews(payments)
	if len(views) != 1 || views[0].ID != "pay-1" {
		t.Fatalf("unexpected payment views: %+v", views)
	}

	rows := payments[0].TransactionViews()
	if len(rows) != 1 || rows[0].Amount != "10.5" || rows[0].ProviderID != "STRIPE" {
		t.Errorf("unexpected transaction views: %+v", rows)
	}

	if empty := (&Payment{}).TransactionViews(); len(empty) != 0 {
		t.Errorf("expected no rows for a payment without transactions, got %+v", empty)
	}
}

func TestIssuerListViews(t *testing.T) {
	list := IssuerList{Issuers: []Issuer{{ID: "1001", Name: "BANCO DE BOGOTA"}}}

	views := list.Views()
	if len(views) != 1 || views[0].Name != "BANCO DE BOGOTA" {
		t.Errorf("unexpected issuer views: %+v", views)
	}

	if empty := (&IssuerList{}).Views(); len(empty) != 0 {
		t.Errorf("expected no rows for an empty issuer list, got %+v", empty)
	}
}
