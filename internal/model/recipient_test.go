package model

import (
	"encoding/json"
	"testing"
)

func TestRecipientView_FlattensTheNestedFields(t *testing.T) {
	t.Parallel()

	const payload = `{"id":"rec-1","merchant_recipient_id":"mr-1","first_name":"Ada","last_name":"Lovelace",
		"entity_type":"INDIVIDUAL","country":"BR","email":"ada@example.com","created_at":"2026-09-16T10:00:00Z",
		"onboardings":[{"id":"onb-1","status":"SUCCEEDED","provider":{"id":"NUVEI"}},
			{"id":"onb-2","status":"IN_PROGRESS","provider":{"id":"DLOCAL"}}]}`

	var recipient Recipient
	if err := json.Unmarshal([]byte(payload), &recipient); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	view := recipient.View()
	if view.Name != "Ada Lovelace" {
		t.Errorf("unexpected name: %q", view.Name)
	}

	if view.Onboarding != "NUVEI SUCCEEDED, DLOCAL IN_PROGRESS" {
		t.Errorf("unexpected onboarding label: %q", view.Onboarding)
	}

	if view.ID != "rec-1" || view.MerchantRecipientID != "mr-1" || view.Country != "BR" {
		t.Errorf("unexpected view: %+v", view)
	}
}

func TestRecipientName_FallsBackToLegalNameAndReference(t *testing.T) {
	t.Parallel()

	legal := "Analytical Engines Ltd"
	business := Recipient{LegalName: &legal, MerchantRecipientID: "mr-1"}

	if got := business.Name(); got != legal {
		t.Errorf("expected the legal name, got %q", got)
	}

	bare := Recipient{MerchantRecipientID: "mr-1"}
	if got := bare.Name(); got != "mr-1" {
		t.Errorf("expected the merchant reference, got %q", got)
	}
}

func TestRecipientMarshalJSON_ReplaysTheRawBody(t *testing.T) {
	t.Parallel()

	const payload = `{"id":"rec-1","unknown_field":"kept"}`

	var recipient Recipient
	if err := json.Unmarshal([]byte(payload), &recipient); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	data, err := json.Marshal(&recipient)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	if string(data) != payload {
		t.Errorf("expected the original body, got %s", data)
	}
}

func TestRecipientMarshalJSON_WithoutRawUsesTheTypedFields(t *testing.T) {
	t.Parallel()

	data, err := json.Marshal(&Recipient{ID: "rec-1"})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	if string(data) != `{"id":"rec-1"}` {
		t.Errorf("unexpected json: %s", data)
	}
}

func TestOnboardingView_RendersTheProviderAndTheResponse(t *testing.T) {
	t.Parallel()

	const payload = `{"id":"onb-1","recipient_id":"rec-1","status":"REJECTED","provider":{"id":"NUVEI"},` +
		`"response_message":"document unreadable"}`

	var onboarding RecipientOnboarding
	if err := json.Unmarshal([]byte(payload), &onboarding); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	view := onboarding.View()
	if view.Provider != "NUVEI" || view.Response != "document unreadable" || view.Status != "REJECTED" {
		t.Errorf("unexpected view: %+v", view)
	}

	data, err := json.Marshal(&onboarding)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	if string(data) != payload {
		t.Errorf("expected the original body, got %s", data)
	}
}

func TestOnboardingLabel_HandlesNilAndMissingProvider(t *testing.T) {
	t.Parallel()

	var missing *RecipientOnboarding
	if got := missing.Label(); got != "" {
		t.Errorf("expected an empty label for a nil onboarding, got %q", got)
	}

	onboarding := RecipientOnboarding{Status: "SUCCEEDED"}
	if got := onboarding.Label(); got != "SUCCEEDED" {
		t.Errorf("expected the bare status, got %q", got)
	}
}

func TestRecipientOnboardingViews_FlattensTheList(t *testing.T) {
	t.Parallel()

	views := RecipientOnboardingViews([]RecipientOnboarding{{ID: "onb-1"}, {ID: "onb-2"}})
	if len(views) != 2 || views[1].ID != "onb-2" {
		t.Errorf("unexpected views: %+v", views)
	}
}

func TestRecipientTransferView_TakesTheDestinationStatus(t *testing.T) {
	t.Parallel()

	const payload = `{"id":"tr-1","origin_onboarding":{"id":"onb-1","status":"SUCCEEDED"},` +
		`"destination_onboarding":{"id":"onb-2","status":"IN_PROGRESS"},"created_at":"2026-09-16T10:00:00Z"}`

	var transfer RecipientTransfer
	if err := json.Unmarshal([]byte(payload), &transfer); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	view := transfer.View()
	if view.Origin != "onb-1" || view.Destination != "onb-2" || view.Status != "IN_PROGRESS" {
		t.Errorf("unexpected view: %+v", view)
	}

	data, err := json.Marshal(&transfer)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	if string(data) != payload {
		t.Errorf("expected the original body, got %s", data)
	}
}

func TestRecipientTransferView_HandlesMissingOnboardings(t *testing.T) {
	t.Parallel()

	transfer := RecipientTransfer{ID: "tr-1"}

	view := transfer.View()
	if view.Origin != "" || view.Destination != "" || view.Status != "" {
		t.Errorf("unexpected view: %+v", view)
	}
}

func TestRecipientViewsAndTransferViews_FlattenTheLists(t *testing.T) {
	t.Parallel()

	recipients := RecipientViews([]Recipient{{ID: "rec-1"}, {ID: "rec-2"}})
	if len(recipients) != 2 || recipients[0].ID != "rec-1" {
		t.Errorf("unexpected recipient views: %+v", recipients)
	}

	transfers := RecipientTransferViews([]RecipientTransfer{{ID: "tr-1"}})
	if len(transfers) != 1 || transfers[0].ID != "tr-1" {
		t.Errorf("unexpected transfer views: %+v", transfers)
	}
}
