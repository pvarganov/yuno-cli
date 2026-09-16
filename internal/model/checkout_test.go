package model_test

import (
	"encoding/json"
	"testing"

	"github.com/pvarganov/yuno-cli/internal/model"
)

func TestCheckoutSessionView_FallsBackToTheIDAndEmptyAmount(t *testing.T) {
	t.Parallel()

	session := model.CheckoutSession{ID: "chk-1", Country: "AR"}

	view := session.View()
	if view.Session != "chk-1" {
		t.Errorf("expected the id to be used as the session, got %q", view.Session)
	}

	if view.Amount != "" {
		t.Errorf("expected an empty amount cell, got %q", view.Amount)
	}
}

func TestCheckoutSessionMarshalJSON_ReplaysTheResponse(t *testing.T) {
	t.Parallel()

	raw := `{"checkout_session":"chk-1","metadata":[{"key":"tier"}]}`

	var session model.CheckoutSession
	if err := json.Unmarshal([]byte(raw), &session); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	out, err := json.Marshal(session)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	if string(out) != raw {
		t.Errorf("expected the verbatim response, got %s", out)
	}
}

func TestCheckoutPaymentMethodView_PrefersTheCheckoutSession(t *testing.T) {
	t.Parallel()

	method := model.CheckoutPaymentMethod{
		Type:       "CARD",
		Checkout:   &model.SessionRef{Session: "chk-1"},
		Enrollment: &model.SessionRef{Session: "cs-1"},
	}

	if got := method.View().Session; got != "chk-1" {
		t.Errorf("expected the checkout session, got %q", got)
	}

	enrollable := model.CheckoutPaymentMethod{Enrollment: &model.SessionRef{Session: "cs-1"}}
	if got := enrollable.View().Session; got != "cs-1" {
		t.Errorf("expected the enrollment session, got %q", got)
	}

	none := model.CheckoutPaymentMethod{}
	if got := none.View().Session; got != "" {
		t.Errorf("expected an empty session, got %q", got)
	}
}

func TestCustomerPaymentMethodViews_FlattensTheList(t *testing.T) {
	t.Parallel()

	methods := []model.CustomerPaymentMethod{
		{ID: "pm-1", Status: "ENROLLED", Enrollment: &model.SessionRef{Session: "cs-1"}},
		{ID: "pm-2", Status: "UNENROLLED"},
	}

	views := model.CustomerPaymentMethodViews(methods)
	if len(views) != 2 || views[0].Session != "cs-1" || views[1].Session != "" {
		t.Errorf("unexpected views: %+v", views)
	}
}
