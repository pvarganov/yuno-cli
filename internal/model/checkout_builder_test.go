package model

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestCheckoutView_UsesTheStatus(t *testing.T) {
	t.Parallel()

	var checkout Checkout
	if err := json.Unmarshal([]byte(`{"id":"ck-1","name":"Promo","description":"Seasonal",
		"status":"PUBLISHED","is_default":true,"created_at":"2026-09-16T10:00:00Z"}`), &checkout); err != nil {
		t.Fatalf("decode checkout: %v", err)
	}

	view := checkout.View()
	if view.ID != "ck-1" || view.Status != "PUBLISHED" || !view.Default {
		t.Errorf("unexpected view: %+v", view)
	}
}

func TestCheckoutStatusLabel_FallsBackToIsActive(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name string
		body string
		want string
	}{
		{name: "active", body: `{"id":"ck-1","is_active":true}`, want: "ACTIVE"},
		{name: "inactive", body: `{"id":"ck-1","is_active":false}`, want: "INACTIVE"},
		{name: "unknown", body: `{"id":"ck-1"}`, want: ""},
		{name: "status wins", body: `{"id":"ck-1","status":"ARCHIVED","is_active":true}`, want: "ARCHIVED"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			var checkout Checkout
			if err := json.Unmarshal([]byte(tc.body), &checkout); err != nil {
				t.Fatalf("decode checkout: %v", err)
			}

			if got := checkout.StatusLabel(); got != tc.want {
				t.Errorf("expected %q, got %q", tc.want, got)
			}
		})
	}
}

func TestCheckoutMarshal_ReplaysTheOriginalBody(t *testing.T) {
	t.Parallel()

	var checkout Checkout
	if err := json.Unmarshal([]byte(`{"id":"ck-1","unknown_field":"kept"}`), &checkout); err != nil {
		t.Fatalf("decode checkout: %v", err)
	}

	out, err := json.Marshal(checkout)
	if err != nil {
		t.Fatalf("marshal checkout: %v", err)
	}

	if !strings.Contains(string(out), "unknown_field") {
		t.Errorf("expected the unknown field to survive, got %s", out)
	}
}

func TestCheckoutViews_FlattensTheList(t *testing.T) {
	t.Parallel()

	views := CheckoutViews([]Checkout{{ID: "ck-1"}, {ID: "ck-2"}})
	if len(views) != 2 || views[1].ID != "ck-2" {
		t.Errorf("unexpected views: %+v", views)
	}
}
