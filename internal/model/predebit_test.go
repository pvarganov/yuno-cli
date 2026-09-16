package model

import (
	"encoding/json"
	"testing"
)

func TestLooseAmount_AcceptsStringsAndNumbers(t *testing.T) {
	tests := []struct {
		name string
		body string
		want string
	}{
		{name: "string", body: `{"currency":"INR","value":"1000"}`, want: "INR 1000"},
		{name: "number", body: `{"currency":"USD","value":10.5}`, want: "USD 10.5"},
		{name: "null value", body: `{"currency":"USD","value":null}`, want: "USD"},
		{name: "missing value", body: `{"currency":"USD"}`, want: "USD"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var amount LooseAmount
			if err := json.Unmarshal([]byte(tt.body), &amount); err != nil {
				t.Fatalf("unmarshal amount: %v", err)
			}

			if got := amount.Label(); got != tt.want {
				t.Errorf("expected %q, got %q", tt.want, got)
			}
		})
	}
}

func TestLooseAmount_LabelHandlesNil(t *testing.T) {
	var amount *LooseAmount
	if got := amount.Label(); got != "" {
		t.Errorf("expected an empty label, got %q", got)
	}
}

func TestLooseAmount_MarshalsTheValueAsAString(t *testing.T) {
	encoded, err := json.Marshal(LooseAmount{Currency: "INR", Value: "1000"})
	if err != nil {
		t.Fatalf("marshal amount: %v", err)
	}

	if string(encoded) != `{"currency":"INR","value":"1000"}` {
		t.Errorf("unexpected encoding: %s", encoded)
	}
}

func TestLooseAmount_RejectsAMalformedBody(t *testing.T) {
	var amount LooseAmount
	if err := json.Unmarshal([]byte(`{"currency":1}`), &amount); err == nil {
		t.Error("expected a decode error")
	}
}

func TestPreDebitNotificationView_FlattensTheProvider(t *testing.T) {
	var notification PreDebitNotification
	if err := json.Unmarshal([]byte(`{"id":"pdn-1","status":"NOTIFIED","merchant_reference":"ref-1",
		"amount":{"currency":"INR","value":"1000"},"billing_date":"2026-10-01",
		"origin_payment_id":"pay-1","provider_data":{"id":"RAZORPAY"}}`), &notification); err != nil {
		t.Fatalf("unmarshal notification: %v", err)
	}

	view := notification.View()
	if view.Amount != "INR 1000" || view.Provider != "RAZORPAY" || view.BillingDate != "2026-10-01" {
		t.Errorf("unexpected view: %+v", view)
	}
}

func TestPreDebitNotificationView_HandlesAnEmptyNotification(t *testing.T) {
	notification := PreDebitNotification{}
	if view := notification.View(); view.Amount != "" || view.Provider != "" {
		t.Errorf("unexpected view: %+v", view)
	}
}

func TestPreDebitNotificationMarshalJSON_ReplaysTheRawBody(t *testing.T) {
	body := `{"id":"pdn-1","unknown_field":"kept"}`

	var notification PreDebitNotification
	if err := json.Unmarshal([]byte(body), &notification); err != nil {
		t.Fatalf("unmarshal notification: %v", err)
	}

	encoded, err := json.Marshal(notification)
	if err != nil {
		t.Fatalf("marshal notification: %v", err)
	}

	if string(encoded) != body {
		t.Errorf("expected the raw body back, got %s", encoded)
	}
}

func TestPreDebitNotificationViews_FlattensAList(t *testing.T) {
	views := PreDebitNotificationViews([]PreDebitNotification{{ID: "pdn-1"}, {ID: "pdn-2"}})
	if len(views) != 2 || views[1].ID != "pdn-2" {
		t.Errorf("unexpected views: %+v", views)
	}
}
