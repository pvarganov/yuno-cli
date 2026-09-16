package model

import (
	"encoding/json"
	"testing"
)

func TestSubscriptionView_FlattensTheNestedFields(t *testing.T) {
	raw := `{"id":"sub-1","name":"Gold","status":"ACTIVE","plan_id":"plan-1",
		"amount":{"currency":"USD","value":9.99},
		"frequency":{"type":"MONTH","value":1},
		"customer_payer":{"id":"cus-1"},
		"current_period_end":"2026-10-16T10:00:00Z",
		"created_at":"2026-09-16T10:00:00Z","unknown":{"kept":true}}`

	var subscription Subscription
	if err := json.Unmarshal([]byte(raw), &subscription); err != nil {
		t.Fatalf("unmarshal subscription: %v", err)
	}

	view := subscription.View()

	if view.ID != "sub-1" || view.Status != "ACTIVE" || view.PlanID != "plan-1" {
		t.Errorf("unexpected view: %+v", view)
	}

	if view.Amount != "USD 9.99" || view.Frequency != "1 MONTH" || view.CustomerID != "cus-1" {
		t.Errorf("unexpected labels: %+v", view)
	}
}

func TestSubscriptionMarshalJSON_ReplaysTheRawBody(t *testing.T) {
	raw := `{"id":"sub-1","phases":[{"name":"trial"}]}`

	var subscription Subscription
	if err := json.Unmarshal([]byte(raw), &subscription); err != nil {
		t.Fatalf("unmarshal subscription: %v", err)
	}

	encoded, err := json.Marshal(subscription)
	if err != nil {
		t.Fatalf("marshal subscription: %v", err)
	}

	if string(encoded) != raw {
		t.Errorf("expected the raw body back, got %s", encoded)
	}
}

func TestSubscriptionMarshalJSON_WithoutRawFallsBackToTheTypedFields(t *testing.T) {
	subscription := Subscription{ID: "sub-1", Status: "ACTIVE"}

	encoded, err := json.Marshal(subscription)
	if err != nil {
		t.Fatalf("marshal subscription: %v", err)
	}

	var decoded map[string]any
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		t.Fatalf("unmarshal encoded subscription: %v", err)
	}

	if decoded["id"] != "sub-1" || decoded["status"] != "ACTIVE" {
		t.Errorf("unexpected encoding: %s", encoded)
	}
}

func TestSubscriptionView_EmptyNestedFieldsRenderAsEmptyCells(t *testing.T) {
	subscription := Subscription{ID: "sub-1"}

	view := subscription.View()
	if view.Amount != "" || view.Frequency != "" || view.CustomerID != "" {
		t.Errorf("expected empty cells for the missing fields, got %+v", view)
	}
}

func TestSubscriptionViews_FlattensEveryRow(t *testing.T) {
	views := SubscriptionViews([]Subscription{{ID: "sub-1"}, {ID: "sub-2"}})
	if len(views) != 2 || views[1].ID != "sub-2" {
		t.Errorf("unexpected views: %+v", views)
	}
}

func TestSubscriptionPaymentView_RendersTheBillingCycle(t *testing.T) {
	cycle := 3
	payment := SubscriptionPayment{
		ID: "sp-1", PaymentID: "pay-1", Status: "SUCCEEDED",
		Currency: "USD", Amount: 9.9, BillingCycle: &cycle,
	}

	view := payment.View()
	if view.Amount != "9.9" || view.BillingCycle != "3" || view.PaymentID != "pay-1" {
		t.Errorf("unexpected view: %+v", view)
	}

	if missing := (&SubscriptionPayment{ID: "sp-2"}).View(); missing.BillingCycle != "" {
		t.Errorf("expected an empty billing cycle, got %q", missing.BillingCycle)
	}

	if views := SubscriptionPaymentViews([]SubscriptionPayment{payment}); len(views) != 1 {
		t.Errorf("unexpected views: %+v", views)
	}
}

func TestPlanView_FlattensTheNestedFields(t *testing.T) {
	raw := `{"id":"plan-1","name":"Gold","status":"ACTIVE",` +
		`"base_amount":{"currency":"USD","value":9.99},"frequency":{"type":"MONTH","value":1},` +
		`"subscribers_count":7,"merchant_reference":"gold-1","countries":["CO"]}`

	var plan Plan
	if err := json.Unmarshal([]byte(raw), &plan); err != nil {
		t.Fatalf("unmarshal plan: %v", err)
	}

	view := plan.View()
	if view.BaseAmount != "USD 9.99" || view.Frequency != "1 MONTH" || view.SubscribersCount != "7" {
		t.Errorf("unexpected view: %+v", view)
	}

	encoded, err := json.Marshal(plan)
	if err != nil {
		t.Fatalf("marshal plan: %v", err)
	}

	if string(encoded) != raw {
		t.Errorf("expected the raw body back, got %s", encoded)
	}
}

func TestPlanView_MissingSubscribersCountIsEmpty(t *testing.T) {
	plan := Plan{ID: "plan-1"}

	if view := plan.View(); view.SubscribersCount != "" || view.BaseAmount != "" {
		t.Errorf("expected empty cells, got %+v", view)
	}

	encoded, err := json.Marshal(plan)
	if err != nil {
		t.Fatalf("marshal plan: %v", err)
	}

	if len(encoded) == 0 {
		t.Error("expected the typed fields to be encoded")
	}

	if views := PlanViews([]Plan{plan}); len(views) != 1 {
		t.Errorf("unexpected views: %+v", views)
	}
}

func TestPlanStatusView_RendersTheAffectedSubscriptions(t *testing.T) {
	affected := 12
	status := PlanStatus{ID: "plan-1", Status: "CANCELED", AffectedSubscriptions: &affected}

	if view := status.View(); view.AffectedSubscriptions != "12" || view.Status != "CANCELED" {
		t.Errorf("unexpected view: %+v", view)
	}

	if view := (&PlanStatus{ID: "plan-1"}).View(); view.AffectedSubscriptions != "" {
		t.Errorf("expected an empty cell, got %q", view.AffectedSubscriptions)
	}
}

func TestFrequencyLabel_HandlesNilAndZero(t *testing.T) {
	var frequency *Frequency
	if frequency.Label() != "" {
		t.Error("expected an empty label for a nil frequency")
	}

	if (&Frequency{}).Label() != "" {
		t.Error("expected an empty label for a zero frequency")
	}
}
