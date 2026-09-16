package model

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestCampaignUnmarshal_UnwrapsTheDataEnvelope(t *testing.T) {
	t.Parallel()

	var campaign Campaign
	if err := json.Unmarshal([]byte(`{"data":{"id":"cp-1","name":"Recovery CO","country":"CO",
		"channel":"PHONE_CALL","status":"ACTIVE",
		"duration":{"start_at":"2026-10-01","end_at":"2026-12-31"}}}`), &campaign); err != nil {
		t.Fatalf("decode campaign: %v", err)
	}

	view := campaign.View()
	if view.ID != "cp-1" || view.StartAt != "2026-10-01" || view.EndAt != "2026-12-31" {
		t.Errorf("unexpected view: %+v", view)
	}
}

func TestCampaignUnmarshal_AcceptsABareObject(t *testing.T) {
	t.Parallel()

	var campaign Campaign
	if err := json.Unmarshal([]byte(`{"id":"cp-1","name":"Recovery CO"}`), &campaign); err != nil {
		t.Fatalf("decode campaign: %v", err)
	}

	if campaign.ID != "cp-1" || campaign.Name != "Recovery CO" {
		t.Errorf("unexpected campaign: %+v", campaign)
	}
}

func TestCampaignUnmarshal_LeavesADataArrayAlone(t *testing.T) {
	t.Parallel()

	// A `data` array belongs to a list response; unwrapping it would make the
	// campaign decode fail instead of leaving the fields empty.
	var campaign Campaign
	if err := json.Unmarshal([]byte(`{"id":"cp-1","data":[{"id":"nested"}]}`), &campaign); err != nil {
		t.Fatalf("decode campaign: %v", err)
	}

	if campaign.ID != "cp-1" {
		t.Errorf("unexpected campaign: %+v", campaign)
	}
}

func TestCampaignView_HandlesAMissingDuration(t *testing.T) {
	t.Parallel()

	campaign := Campaign{ID: "cp-1"}
	if view := campaign.View(); view.StartAt != "" || view.EndAt != "" {
		t.Errorf("expected no duration, got %+v", view)
	}
}

func TestCampaignMarshal_ReplaysTheOriginalBody(t *testing.T) {
	t.Parallel()

	var campaign Campaign
	if err := json.Unmarshal([]byte(`{"data":{"id":"cp-1","unknown_field":"kept"}}`), &campaign); err != nil {
		t.Fatalf("decode campaign: %v", err)
	}

	out, err := json.Marshal(campaign)
	if err != nil {
		t.Fatalf("marshal campaign: %v", err)
	}

	if !strings.Contains(string(out), "unknown_field") || !strings.Contains(string(out), `"data"`) {
		t.Errorf("expected the whole response to be replayed, got %s", out)
	}
}

func TestCampaignViews_FlattensTheList(t *testing.T) {
	t.Parallel()

	views := CampaignViews([]Campaign{{ID: "cp-1"}, {ID: "cp-2"}})
	if len(views) != 2 || views[1].ID != "cp-2" {
		t.Errorf("unexpected views: %+v", views)
	}
}

func TestCampaignRuleView_JoinsTheValues(t *testing.T) {
	t.Parallel()

	var rule CampaignRule
	if err := json.Unmarshal([]byte(`{"data":{"id":"rl-1","rule_type":"PAYMENT_METHOD",
		"conditional":"IN","values":["CARD","PAYPAL"],"status":"ACTIVE"}}`), &rule); err != nil {
		t.Fatalf("decode rule: %v", err)
	}

	view := rule.View()
	if view.ID != "rl-1" || view.Values != "CARD,PAYPAL" || view.Status != "ACTIVE" {
		t.Errorf("unexpected view: %+v", view)
	}
}

func TestCampaignRuleMarshal_ReplaysTheOriginalBody(t *testing.T) {
	t.Parallel()

	var rule CampaignRule
	if err := json.Unmarshal([]byte(`{"id":"rl-1","unknown_field":"kept"}`), &rule); err != nil {
		t.Fatalf("decode rule: %v", err)
	}

	out, err := json.Marshal(rule)
	if err != nil {
		t.Fatalf("marshal rule: %v", err)
	}

	if !strings.Contains(string(out), "unknown_field") {
		t.Errorf("expected the unknown field to survive, got %s", out)
	}
}

func TestCampaignRuleViews_FlattensTheList(t *testing.T) {
	t.Parallel()

	views := CampaignRuleViews([]CampaignRule{{ID: "rl-1"}, {ID: "rl-2"}})
	if len(views) != 2 || views[1].ID != "rl-2" {
		t.Errorf("unexpected views: %+v", views)
	}
}
