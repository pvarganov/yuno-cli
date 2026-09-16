package model

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestWebhookView_SummarisesTheTriggers(t *testing.T) {
	t.Parallel()

	var webhook Webhook
	if err := json.Unmarshal([]byte(`{"id":"wh-1","name":"listener","state":"ACTIVE",
		"url":"https://api.acme.com/yuno","payment_triggers":["AUTHORIZE","REFUND"],
		"report_triggers":["UPDATE"],"created_at":"2026-09-16T10:00:00Z"}`), &webhook); err != nil {
		t.Fatalf("decode webhook: %v", err)
	}

	view := webhook.View()
	if view.ID != "wh-1" || view.State != "ACTIVE" || view.URL != "https://api.acme.com/yuno" {
		t.Errorf("unexpected view: %+v", view)
	}

	if view.Triggers != "payment:AUTHORIZE|REFUND report:UPDATE" {
		t.Errorf("unexpected triggers: %q", view.Triggers)
	}
}

func TestWebhookView_EmptyTriggersRenderAsEmpty(t *testing.T) {
	t.Parallel()

	webhook := Webhook{ID: "wh-1"}
	if got := webhook.View().Triggers; got != "" {
		t.Errorf("expected no triggers, got %q", got)
	}
}

func TestWebhookMarshal_ReplaysTheOriginalBody(t *testing.T) {
	t.Parallel()

	raw := `{"id":"wh-1","unknown_field":"kept"}`

	var webhook Webhook
	if err := json.Unmarshal([]byte(raw), &webhook); err != nil {
		t.Fatalf("decode webhook: %v", err)
	}

	out, err := json.Marshal(webhook)
	if err != nil {
		t.Fatalf("marshal webhook: %v", err)
	}

	if !strings.Contains(string(out), "unknown_field") {
		t.Errorf("expected the original body replayed, got %s", out)
	}
}

func TestWebhookMarshal_FallsBackToTheTypedFields(t *testing.T) {
	t.Parallel()

	out, err := json.Marshal(Webhook{ID: "wh-1", Name: "listener"})
	if err != nil {
		t.Fatalf("marshal webhook: %v", err)
	}

	if !strings.Contains(string(out), `"id":"wh-1"`) {
		t.Errorf("unexpected json: %s", out)
	}
}

func TestWebhookViews_FlattensTheList(t *testing.T) {
	t.Parallel()

	views := WebhookViews([]Webhook{{ID: "wh-1"}, {ID: "wh-2"}})
	if len(views) != 2 || views[1].ID != "wh-2" {
		t.Errorf("unexpected views: %+v", views)
	}
}
