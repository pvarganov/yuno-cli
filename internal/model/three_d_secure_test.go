package model

import (
	"encoding/json"
	"testing"
)

func TestThreeDSecureSetupView_FlattensTheBrowserAndFingerprints(t *testing.T) {
	var setup ThreeDSecureSetup
	if err := json.Unmarshal([]byte(`{"three_d_secure_setup_id":"tds-1","account_id":"acc-1","type":"BROWSER",
		"browser_info":{"screen_width":"1920","screen_height":"1080","language":"en-US"},
		"device_fingerprints":[{"provider_id":"CYBERSOURCE","id":"fp-1"},
		{"provider_id":"NUVEI","id":"fp-2"}]}`), &setup); err != nil {
		t.Fatalf("unmarshal setup: %v", err)
	}

	view := setup.View()
	if view.Screen != "1920x1080" || view.Language != "en-US" {
		t.Errorf("unexpected browser columns: %+v", view)
	}

	if view.Fingerprints != "CYBERSOURCE=fp-1,NUVEI=fp-2" {
		t.Errorf("unexpected fingerprints: %q", view.Fingerprints)
	}
}

func TestThreeDSecureSetupView_HandlesAnEmptySetup(t *testing.T) {
	setup := ThreeDSecureSetup{}
	if view := setup.View(); view.Screen != "" || view.Fingerprints != "" {
		t.Errorf("unexpected view: %+v", view)
	}
}

func TestThreeDSecureSetupMarshalJSON_ReplaysTheRawBody(t *testing.T) {
	body := `{"three_d_secure_setup_id":"tds-1","unknown_field":"kept"}`

	var setup ThreeDSecureSetup
	if err := json.Unmarshal([]byte(body), &setup); err != nil {
		t.Fatalf("unmarshal setup: %v", err)
	}

	encoded, err := json.Marshal(setup)
	if err != nil {
		t.Fatalf("marshal setup: %v", err)
	}

	if string(encoded) != body {
		t.Errorf("expected the raw body back, got %s", encoded)
	}
}

func TestDryRunProviderEventView_DereferencesTheNullablePaymentID(t *testing.T) {
	var event DryRunProviderEvent
	if err := json.Unmarshal([]byte(`{"id":"dr-1","status":"PROCESSED","payment_id":null,
		"provider_id":"NUVEI","operation_type":"CREATE_PAYMENT"}`), &event); err != nil {
		t.Fatalf("unmarshal event: %v", err)
	}

	if view := event.View(); view.PaymentID != "" || view.OperationType != "CREATE_PAYMENT" {
		t.Errorf("unexpected view: %+v", view)
	}

	if err := json.Unmarshal([]byte(`{"id":"dr-1","payment_id":"pay-1"}`), &event); err != nil {
		t.Fatalf("unmarshal event: %v", err)
	}

	if view := event.View(); view.PaymentID != "pay-1" {
		t.Errorf("unexpected view: %+v", view)
	}
}

func TestDryRunProviderEventMarshalJSON_ReplaysTheRawBody(t *testing.T) {
	body := `{"id":"dr-1","unknown_field":"kept"}`

	var event DryRunProviderEvent
	if err := json.Unmarshal([]byte(body), &event); err != nil {
		t.Fatalf("unmarshal event: %v", err)
	}

	encoded, err := json.Marshal(event)
	if err != nil {
		t.Fatalf("marshal event: %v", err)
	}

	if string(encoded) != body {
		t.Errorf("expected the raw body back, got %s", encoded)
	}

	encoded, err = json.Marshal(DryRunProviderEvent{ID: "dr-2"})
	if err != nil {
		t.Fatalf("marshal event: %v", err)
	}

	if string(encoded) != `{"id":"dr-2"}` {
		t.Errorf("unexpected encoding: %s", encoded)
	}
}
