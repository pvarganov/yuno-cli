package model

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestBankingEntity_KeepsTheRawBody(t *testing.T) {
	t.Parallel()

	const body = `{"id":"be-1","merchant_entity_id":"me-1","unknown_field":"kept"}`

	var entity BankingEntity
	if err := json.Unmarshal([]byte(body), &entity); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	replayed, err := json.Marshal(entity)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	if !strings.Contains(string(replayed), "unknown_field") {
		t.Errorf("expected the raw body to survive, got %s", replayed)
	}
}

func TestBankingDetail_Label(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		detail *BankingDetail
		want   string
	}{
		{name: "nil", detail: nil, want: ""},
		{name: "legal name", detail: &BankingDetail{LegalName: "Acme Inc"}, want: "Acme Inc"},
		{name: "trading name", detail: &BankingDetail{TradingName: "Acme"}, want: "Acme"},
		{name: "person", detail: &BankingDetail{FirstName: "Ada", LastName: "Lovelace"}, want: "Ada Lovelace"},
		{name: "first name only", detail: &BankingDetail{FirstName: "Ada"}, want: "Ada"},
		{name: "empty", detail: &BankingDetail{EntityType: "BUSINESS"}, want: ""},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if got := tc.detail.Label(); got != tc.want {
				t.Errorf("expected %q, got %q", tc.want, got)
			}
		})
	}
}

func TestBankingViews_FlattenTheResources(t *testing.T) {
	t.Parallel()

	entity := BankingEntity{
		ID: "be-1", MerchantEntityID: "me-1", NationalEntity: "US",
		EntityDetail: &BankingDetail{LegalName: "Acme Inc"},
	}
	if view := entity.View(); view.Name != "Acme Inc" || view.NationalEntity != "US" {
		t.Errorf("unexpected entity view: %+v", view)
	}

	onboarding := BankingOnboarding{ID: "bo-1", EntityID: "be-1", Status: "PENDING"}
	if view := onboarding.View(); view.ID != "bo-1" || view.Status != "PENDING" {
		t.Errorf("unexpected onboarding view: %+v", view)
	}

	account := BankingAccount{ID: "ba-1", Currency: "USD", Balance: &Amount{Currency: "USD", Value: 12.5}}
	if view := account.View(); view.Balance != "USD 12.5" {
		t.Errorf("unexpected account view: %+v", view)
	}

	transfer := BankingTransfer{ID: "bt-1", Direction: "OUTBOUND", Amount: &Amount{Currency: "USD", Value: 250}}
	if view := transfer.View(); view.Amount != "USD 250" || view.Direction != "OUTBOUND" {
		t.Errorf("unexpected transfer view: %+v", view)
	}
}

func TestBankingResources_MarshalWithoutARawBody(t *testing.T) {
	t.Parallel()

	payloads := []any{
		BankingEntity{ID: "be-1"},
		BankingOnboarding{ID: "bo-1"},
		BankingAccount{ID: "ba-1"},
		BankingTransfer{ID: "bt-1"},
	}

	for _, payload := range payloads {
		data, err := json.Marshal(payload)
		if err != nil {
			t.Fatalf("marshal %T: %v", payload, err)
		}

		if !strings.Contains(string(data), `"id"`) {
			t.Errorf("expected %T to marshal its typed fields, got %s", payload, data)
		}
	}
}
