package model

import (
	"encoding/json"
	"testing"
)

func TestDecodePayouts_AcceptsAnArrayAndAnObject(t *testing.T) {
	tests := []struct {
		name  string
		body  string
		count int
		first string
	}{
		{name: "array", body: `[{"id":"po-1"},{"id":"po-2"}]`, count: 2, first: "po-1"},
		{name: "object", body: `{"id":"po-3"}`, count: 1, first: "po-3"},
		{name: "empty", body: ``, count: 0},
		{name: "null", body: `null`, count: 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			payouts, err := DecodePayouts([]byte(tt.body))
			if err != nil {
				t.Fatalf("DecodePayouts: %v", err)
			}

			if len(payouts) != tt.count {
				t.Fatalf("expected %d payouts, got %d", tt.count, len(payouts))
			}

			if tt.count > 0 && payouts[0].ID != tt.first {
				t.Errorf("unexpected first payout: %+v", payouts[0])
			}
		})
	}
}

func TestDecodePayouts_RejectsMalformedBodies(t *testing.T) {
	for _, body := range []string{`[{"id":`, `{"id":`} {
		if _, err := DecodePayouts([]byte(body)); err == nil {
			t.Errorf("expected an error for %q", body)
		}
	}
}

func TestPayoutView_FlattensTheBeneficiary(t *testing.T) {
	var payout Payout
	if err := json.Unmarshal([]byte(`{"id":"po-1","status":"SUCCEEDED","country":"US",
		"purpose":"SALARY","merchant_reference":"ref-1","amount":{"currency":"USD","value":100.5},
		"beneficiary":{"first_name":"Ada","last_name":"Lovelace"}}`), &payout); err != nil {
		t.Fatalf("unmarshal payout: %v", err)
	}

	view := payout.View()
	if view.Amount != "USD 100.5" || view.Beneficiary != "Ada Lovelace" {
		t.Errorf("unexpected view: %+v", view)
	}
}

func TestPayoutBeneficiaryLabel_FallsBack(t *testing.T) {
	tests := []struct {
		name        string
		beneficiary *PayoutBeneficiary
		want        string
	}{
		{name: "nil", beneficiary: nil, want: ""},
		{name: "person", beneficiary: &PayoutBeneficiary{FirstName: "Ada", LastName: "Lovelace"}, want: "Ada Lovelace"},
		{name: "company", beneficiary: &PayoutBeneficiary{LegalName: "Yuno Inc"}, want: "Yuno Inc"},
		{name: "reference", beneficiary: &PayoutBeneficiary{MerchantBeneficiaryID: "ben-1"}, want: "ben-1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.beneficiary.Label(); got != tt.want {
				t.Errorf("expected %q, got %q", tt.want, got)
			}
		})
	}
}

func TestPayoutMarshalJSON_ReplaysTheRawBody(t *testing.T) {
	body := `{"id":"po-1","unknown_field":"kept"}`

	var payout Payout
	if err := json.Unmarshal([]byte(body), &payout); err != nil {
		t.Fatalf("unmarshal payout: %v", err)
	}

	encoded, err := json.Marshal(payout)
	if err != nil {
		t.Fatalf("marshal payout: %v", err)
	}

	if string(encoded) != body {
		t.Errorf("expected the raw body back, got %s", encoded)
	}

	// A payout built in code and never decoded falls back to the typed fields.
	encoded, err = json.Marshal(Payout{ID: "po-2"})
	if err != nil {
		t.Fatalf("marshal payout: %v", err)
	}

	if string(encoded) != `{"id":"po-2"}` {
		t.Errorf("unexpected encoding: %s", encoded)
	}
}

func TestPayoutViews_FlattensAList(t *testing.T) {
	views := PayoutViews([]Payout{{ID: "po-1"}, {ID: "po-2"}})
	if len(views) != 2 || views[1].ID != "po-2" {
		t.Errorf("unexpected views: %+v", views)
	}

	if got := PayoutViews(nil); len(got) != 0 {
		t.Errorf("expected no views, got %+v", got)
	}
}
