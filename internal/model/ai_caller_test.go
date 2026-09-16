package model

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestAICallerOutreachView_FlattensTheOutreach(t *testing.T) {
	t.Parallel()

	var outreach AICallerOutreach
	if err := json.Unmarshal([]byte(`{"id":"call-1","message":"outreach scheduled"}`), &outreach); err != nil {
		t.Fatalf("decode outreach: %v", err)
	}

	if view := outreach.View(); view.ID != "call-1" || view.Message != "outreach scheduled" {
		t.Errorf("unexpected view: %+v", view)
	}
}

func TestAICallerOutreachMarshal_ReplaysTheOriginalBody(t *testing.T) {
	t.Parallel()

	var outreach AICallerOutreach
	if err := json.Unmarshal([]byte(`{"id":"call-1","unknown_field":"kept"}`), &outreach); err != nil {
		t.Fatalf("decode outreach: %v", err)
	}

	out, err := json.Marshal(outreach)
	if err != nil {
		t.Fatalf("marshal outreach: %v", err)
	}

	if !strings.Contains(string(out), "unknown_field") {
		t.Errorf("expected the unknown field to survive, got %s", out)
	}
}
