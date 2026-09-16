package model

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestCustomerPaymentMethodCardLabel(t *testing.T) {
	tests := []struct {
		name string
		card *EnrolledCard
		want string
	}{
		{name: "no card data", card: nil, want: ""},
		{name: "brand and last four", card: &EnrolledCard{Brand: "VISA", LFD: "1111"}, want: "VISA ****1111"},
		{name: "brand only", card: &EnrolledCard{Brand: "VISA"}, want: "VISA"},
		{name: "last four only", card: &EnrolledCard{LFD: "1111"}, want: "****1111"},
		{name: "empty card data", card: &EnrolledCard{}, want: ""},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			method := CustomerPaymentMethod{CardData: tc.card}
			if got := method.CardLabel(); got != tc.want {
				t.Errorf("CardLabel() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestCustomerPaymentMethodViewNeverCarriesThePAN(t *testing.T) {
	body := `{"id":"pm-1","type":"CARD","status":"ENROLLED",
		"card_data":{"brand":"VISA","lfd":"1111","iin":"41111111","number":"4111111111111111"}}`

	var method CustomerPaymentMethod
	if err := json.Unmarshal([]byte(body), &method); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}

	view := method.View()
	if view.Card != "VISA ****1111" {
		t.Errorf("Card = %q, want VISA ****1111", view.Card)
	}

	row, err := json.Marshal(view)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	for _, leak := range []string{"4111111111111111", "41111111"} {
		if strings.Contains(string(row), leak) {
			t.Errorf("view leaks card digits %q: %s", leak, row)
		}
	}
}

func TestCustomerPaymentMethodListDecodesTheEnvelope(t *testing.T) {
	var list CustomerPaymentMethodList
	if err := json.Unmarshal([]byte(`{"payment_methods":[{"id":"pm-1"},{"id":"pm-2"}]}`), &list); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}

	if len(list.PaymentMethods) != 2 || list.PaymentMethods[1].ID != "pm-2" {
		t.Errorf("unexpected list: %+v", list.PaymentMethods)
	}
}

func TestAccountUpdaterResultView(t *testing.T) {
	result := AccountUpdaterResult{Accepted: 3}
	if got := result.View(); got.Accepted != 3 {
		t.Errorf("View().Accepted = %d, want 3", got.Accepted)
	}
}
