package model

import (
	"encoding/json"
	"testing"
)

func TestTransferView_PrefersTheProviderField(t *testing.T) {
	var transfer Transfer
	if err := json.Unmarshal([]byte(`{"id":"tr-1","status":"SUCCEEDED","recipient_id":"rec-1",
		"amount":{"currency":"USD","value":25},"provider":"NUVEI",
		"provider_data":{"id":"OTHER","response_code":"00","response_message":"OK"}}`), &transfer); err != nil {
		t.Fatalf("unmarshal transfer: %v", err)
	}

	view := transfer.View()
	if view.Provider != "NUVEI" || view.ProviderResponse != "00 OK" || view.Amount != "USD 25" {
		t.Errorf("unexpected view: %+v", view)
	}
}

func TestTransferView_FallsBackToTheProviderData(t *testing.T) {
	var transfer Transfer
	if err := json.Unmarshal([]byte(`{"id":"tr-1","provider":null,
		"provider_data":{"id":"NUVEI"}}`), &transfer); err != nil {
		t.Fatalf("unmarshal transfer: %v", err)
	}

	if view := transfer.View(); view.Provider != "NUVEI" || view.ProviderResponse != "" {
		t.Errorf("unexpected view: %+v", view)
	}
}

func TestTransferView_HandlesAnEmptyTransfer(t *testing.T) {
	transfer := Transfer{}
	if view := transfer.View(); view.Provider != "" || view.Amount != "" {
		t.Errorf("unexpected view: %+v", view)
	}
}

func TestTransferMarshalJSON_ReplaysTheRawBody(t *testing.T) {
	body := `{"id":"tr-1","unknown_field":"kept"}`

	var transfer Transfer
	if err := json.Unmarshal([]byte(body), &transfer); err != nil {
		t.Fatalf("unmarshal transfer: %v", err)
	}

	encoded, err := json.Marshal(transfer)
	if err != nil {
		t.Fatalf("marshal transfer: %v", err)
	}

	if string(encoded) != body {
		t.Errorf("expected the raw body back, got %s", encoded)
	}
}

func TestTransferViews_FlattensAList(t *testing.T) {
	views := TransferViews([]Transfer{{ID: "tr-1"}, {ID: "tr-2"}})
	if len(views) != 2 || views[0].ID != "tr-1" {
		t.Errorf("unexpected views: %+v", views)
	}
}
