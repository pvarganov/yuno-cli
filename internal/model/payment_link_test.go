package model

import (
	"encoding/json"
	"testing"
)

func TestPaymentLinkView_FlattensTheLink(t *testing.T) {
	t.Parallel()

	var link PaymentLink
	if err := json.Unmarshal([]byte(`{"code":"pl-1","country":"US","status":"CREATED",
		"amount":{"currency":"USD","value":50},"payment_method_types":["CARD","PSE"],
		"merchant_order_id":"order-1","checkout_url":"https://pay.y.uno/pl-1",
		"availability":{"finish_at":"2026-10-29T14:00:12Z"}}`), &link); err != nil {
		t.Fatalf("decode link: %v", err)
	}

	view := link.View()

	if view.Code != "pl-1" || view.Amount != "USD 50" || view.PaymentMethods != "CARD,PSE" {
		t.Errorf("unexpected view: %+v", view)
	}

	if view.ExpiresAt != "2026-10-29T14:00:12Z" {
		t.Errorf("unexpected expiry: %q", view.ExpiresAt)
	}
}

func TestPaymentLinkReference_FallsBackToTheID(t *testing.T) {
	t.Parallel()

	link := PaymentLink{ID: "pl-created"}
	if got := link.Reference(); got != "pl-created" {
		t.Errorf("expected the id, got %q", got)
	}

	link.Code = "pl-1"
	if got := link.Reference(); got != "pl-1" {
		t.Errorf("expected the code to win, got %q", got)
	}
}

func TestPaymentLinkMarshal_ReplaysTheRawResponse(t *testing.T) {
	t.Parallel()

	raw := `{"code":"pl-1","unknown_field":"kept"}`

	var link PaymentLink
	if err := json.Unmarshal([]byte(raw), &link); err != nil {
		t.Fatalf("decode link: %v", err)
	}

	encoded, err := json.Marshal(link)
	if err != nil {
		t.Fatalf("encode link: %v", err)
	}

	if string(encoded) != raw {
		t.Errorf("expected the raw response back, got %s", encoded)
	}
}

func TestPaymentLinkViews_FlattensTheList(t *testing.T) {
	t.Parallel()

	views := PaymentLinkViews([]PaymentLink{{Code: "pl-1"}, {ID: "pl-2"}})
	if len(views) != 2 || views[1].Code != "pl-2" {
		t.Errorf("unexpected views: %+v", views)
	}
}

func TestAvailabilityFinishLabel_HandlesNil(t *testing.T) {
	t.Parallel()

	var availability *Availability
	if got := availability.FinishLabel(); got != "" {
		t.Errorf("expected an empty label, got %q", got)
	}
}

func TestConversionRateView_FlattensTheQuote(t *testing.T) {
	t.Parallel()

	var rate ConversionRate
	if err := json.Unmarshal([]byte(`{"id":"cr-1","amount":{"value":10000,"currency":"COP",
		"currency_conversion":{"cardholder_currency":"USD","cardholder_amount":2.58,"rate":3879.81,
		"provider_data":{"id":"CIBC","transaction_id":"tx-1","response_code":"2000",
		"response_message":"Successful"}}}}`), &rate); err != nil {
		t.Fatalf("decode rate: %v", err)
	}

	view := rate.View()

	if view.Amount != "COP 10000" || view.CardholderAmount != "USD 2.58" {
		t.Errorf("unexpected amounts: %+v", view)
	}

	if view.Rate != "3879.81" || view.Provider != "CIBC" || view.ProviderResponse != "2000 Successful" {
		t.Errorf("unexpected conversion: %+v", view)
	}
}

func TestConversionRateView_HandlesAPartialQuote(t *testing.T) {
	t.Parallel()

	empty := ConversionRate{ID: "cr-1"}
	if view := empty.View(); view.ID != "cr-1" || view.Amount != "" {
		t.Errorf("unexpected view for an amountless quote: %+v", view)
	}

	noConversion := ConversionRate{ID: "cr-2", Amount: &ConversionAmount{Currency: "COP", Value: 10}}
	if view := noConversion.View(); view.Amount != "COP 10" || view.Rate != "" {
		t.Errorf("unexpected view without a conversion: %+v", view)
	}
}

func TestConversionRateMarshal_ReplaysTheRawResponse(t *testing.T) {
	t.Parallel()

	raw := `{"id":"cr-1","unknown_field":"kept"}`

	var rate ConversionRate
	if err := json.Unmarshal([]byte(raw), &rate); err != nil {
		t.Fatalf("decode rate: %v", err)
	}

	encoded, err := json.Marshal(rate)
	if err != nil {
		t.Fatalf("encode rate: %v", err)
	}

	if string(encoded) != raw {
		t.Errorf("expected the raw response back, got %s", encoded)
	}
}
