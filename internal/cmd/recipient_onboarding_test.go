package cmd_test

import (
	"net/http"
	"strings"
	"testing"
)

// onboardingResponse is the onboarding payload the fake API returns for the
// single-onboarding endpoint.
const onboardingResponse = `{"id":"onb-1","account_id":"acc-1","recipient_id":"rec-1","type":"INDIVIDUAL",
	"workflow":"AUTOMATIC","status":"IN_PROGRESS","provider":{"id":"NUVEI"},
	"created_at":"2026-09-16T10:00:00Z"}`

// recipientTransferResponse is the onboarding transfer the fake API returns.
const recipientTransferResponse = `{"id":"tr-1","origin_onboarding":{"id":"onb-1","status":"SUCCEEDED"},
	"destination_onboarding":{"id":"onb-2","status":"IN_PROGRESS"},"created_at":"2026-09-16T10:00:00Z"}`

func TestOnboardingCreate_PostsUnderTheRecipient(t *testing.T) {
	isolateConfig(t)
	seedCredentials(t)

	got := startAPI(t, http.StatusCreated, recipientResponse)

	out, err := runCLI(t, "", "recipient", "onboarding", "create", "rec-1", "--yes",
		"--account-id", "acc-1", "--type", "INDIVIDUAL", "--workflow", "AUTOMATIC",
		"--provider-id", "NUVEI", "--connection-id", "conn-1",
		"--terms-accepted", "--terms-ip", "1.2.3.4")
	if err != nil {
		t.Fatalf("onboarding create failed: %v (%s)", err, out)
	}

	if got.Method != http.MethodPost || got.Path != "/v1/recipients/rec-1/onboardings" {
		t.Errorf("expected POST /v1/recipients/rec-1/onboardings, got %s %s", got.Method, got.Path)
	}

	body := decodeBody(t, got.Body)

	provider, ok := body["provider"].(map[string]any)
	if !ok || provider["id"] != "NUVEI" || provider["connection_id"] != "conn-1" {
		t.Errorf("unexpected provider in body: %s", got.Body)
	}

	terms, ok := body["terms_of_service"].(map[string]any)
	if !ok || terms["acceptance"] != true || terms["ip"] != "1.2.3.4" {
		t.Errorf("unexpected terms of service in body: %s", got.Body)
	}
}

func TestOnboardingCreate_RequiresABody(t *testing.T) {
	isolateConfig(t)
	seedCredentials(t)
	startAPI(t, http.StatusCreated, recipientResponse)

	if _, err := runCLI(t, "", "recipient", "onboarding", "create", "rec-1", "--yes"); err == nil ||
		!strings.Contains(err.Error(), "request body is required") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestOnboardingGet_RendersTheOnboardingRow(t *testing.T) {
	isolateConfig(t)
	seedCredentials(t)

	got := startAPI(t, http.StatusOK, onboardingResponse)

	out, err := runCLI(t, "", "recipient", "onboarding", "get", "rec-1", "onb-1")
	if err != nil {
		t.Fatalf("onboarding get failed: %v (%s)", err, out)
	}

	if got.Method != http.MethodGet || got.Path != "/v1/recipients/rec-1/onboardings/onb-1" {
		t.Errorf("expected GET /v1/recipients/rec-1/onboardings/onb-1, got %s %s", got.Method, got.Path)
	}

	for _, want := range []string{"onb-1", "IN_PROGRESS", "NUVEI"} {
		if !strings.Contains(out, want) {
			t.Errorf("expected table output to contain %q, got:\n%s", want, out)
		}
	}
}

func TestOnboardingUpdate_PatchesTheOnboardingPath(t *testing.T) {
	isolateConfig(t)
	seedCredentials(t)

	got := startAPI(t, http.StatusOK, recipientResponse)

	out, err := runCLI(t, "", "recipient", "onboarding", "update", "rec-1", "onb-1", "--yes",
		"--callback-url", "https://example.com/hook")
	if err != nil {
		t.Fatalf("onboarding update failed: %v (%s)", err, out)
	}

	if got.Method != http.MethodPatch || got.Path != "/v1/recipients/rec-1/onboardings/onb-1" {
		t.Errorf("expected PATCH /v1/recipients/rec-1/onboardings/onb-1, got %s %s", got.Method, got.Path)
	}

	if body := decodeBody(t, got.Body); body["callback_url"] != "https://example.com/hook" {
		t.Errorf("unexpected body: %s", got.Body)
	}
}

func TestOnboardingContinue_SendsTheRequestedData(t *testing.T) {
	isolateConfig(t)
	seedCredentials(t)

	got := startAPI(t, http.StatusOK, recipientResponse)

	out, err := runCLI(t, "", "recipient", "onboarding", "continue", "rec-1", "onb-1", "--yes",
		"--withdrawal-methods", `{"bank":{"code":"001","account":"123"}}`)
	if err != nil {
		t.Fatalf("onboarding continue failed: %v (%s)", err, out)
	}

	if got.Method != http.MethodPost || got.Path != "/v1/recipients/rec-1/onboardings/onb-1/continue" {
		t.Errorf("expected POST .../continue, got %s %s", got.Method, got.Path)
	}

	body := decodeBody(t, got.Body)

	methods, ok := body["withdrawal_methods"].(map[string]any)
	if !ok {
		t.Fatalf("unexpected body: %s", got.Body)
	}

	bank, ok := methods["bank"].(map[string]any)
	if !ok || bank["code"] != "001" {
		t.Errorf("unexpected withdrawal methods in body: %s", got.Body)
	}
}

func TestOnboardingContinue_RequiresABody(t *testing.T) {
	isolateConfig(t)
	seedCredentials(t)
	startAPI(t, http.StatusOK, recipientResponse)

	_, err := runCLI(t, "", "recipient", "onboarding", "continue", "rec-1", "onb-1", "--yes")
	if err == nil || !strings.Contains(err.Error(), "request body is required") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestOnboardingActions_PostTheBodylessLifecyclePaths(t *testing.T) {
	for _, action := range []string{"cancel", "block", "unblock"} {
		isolateConfig(t)
		seedCredentials(t)

		got := startAPI(t, http.StatusOK, recipientResponse)

		out, err := runCLI(t, "", "recipient", "onboarding", action, "rec-1", "onb-1", "--yes")
		if err != nil {
			t.Fatalf("onboarding %s failed: %v (%s)", action, err, out)
		}

		want := "/v1/recipients/rec-1/onboardings/onb-1/" + action
		if got.Method != http.MethodPost || got.Path != want {
			t.Errorf("expected POST %s, got %s %s", want, got.Method, got.Path)
		}

		if got.Body != "" {
			t.Errorf("expected %s to send no body, got %s", action, got.Body)
		}
	}
}

func TestOnboardingTransfers_ListsTheTransfersOfAnOnboarding(t *testing.T) {
	isolateConfig(t)
	seedCredentials(t)

	got := startAPI(t, http.StatusOK, "["+recipientTransferResponse+"]")

	out, err := runCLI(t, "", "recipient", "onboarding", "transfers", "onb-1")
	if err != nil {
		t.Fatalf("onboarding transfers failed: %v (%s)", err, out)
	}

	if got.Method != http.MethodGet || got.Path != "/v1/onboardings/onb-1/transfers" {
		t.Errorf("expected GET /v1/onboardings/onb-1/transfers, got %s %s", got.Method, got.Path)
	}

	if !strings.Contains(out, "tr-1") {
		t.Errorf("expected the transfer in the table, got:\n%s", out)
	}
}

func TestRecipientTransferRequest_PostsTheTransferPath(t *testing.T) {
	isolateConfig(t)
	seedCredentials(t)

	got := startAPI(t, http.StatusCreated, recipientTransferResponse)

	out, err := runCLI(t, "", "recipient", "transfer", "request", "rec-1", "onb-2", "--yes")
	if err != nil {
		t.Fatalf("recipient transfer request failed: %v (%s)", err, out)
	}

	if got.Method != http.MethodPost || got.Path != "/v1/recipients/rec-1/onboardings/onb-2/transfer" {
		t.Errorf("expected POST .../onb-2/transfer, got %s %s", got.Method, got.Path)
	}

	for _, want := range []string{"tr-1", "onb-1", "onb-2", "IN_PROGRESS"} {
		if !strings.Contains(out, want) {
			t.Errorf("expected table output to contain %q, got:\n%s", want, out)
		}
	}
}

func TestRecipientTransferList_ListsTheTransfersOfARecipient(t *testing.T) {
	isolateConfig(t)
	seedCredentials(t)

	got := startAPI(t, http.StatusOK, "["+recipientTransferResponse+"]")

	out, err := runCLI(t, "", "recipient", "transfer", "list", "rec-1")
	if err != nil {
		t.Fatalf("recipient transfer list failed: %v (%s)", err, out)
	}

	if got.Method != http.MethodGet || got.Path != "/v1/recipients/rec-1/transfers" {
		t.Errorf("expected GET /v1/recipients/rec-1/transfers, got %s %s", got.Method, got.Path)
	}

	if !strings.Contains(out, "tr-1") {
		t.Errorf("expected the transfer in the table, got:\n%s", out)
	}
}

func TestRecipientTransferGet_HitsTheTopLevelTransferPath(t *testing.T) {
	isolateConfig(t)
	seedCredentials(t)

	got := startAPI(t, http.StatusOK, recipientTransferResponse)

	out, err := runCLI(t, "", "recipient", "transfer", "get", "tr-1", "--json")
	if err != nil {
		t.Fatalf("recipient transfer get failed: %v (%s)", err, out)
	}

	if got.Method != http.MethodGet || got.Path != "/v1/transfers/tr-1" {
		t.Errorf("expected GET /v1/transfers/tr-1, got %s %s", got.Method, got.Path)
	}

	if !strings.Contains(out, `"destination_onboarding"`) {
		t.Errorf("expected the whole payload as json, got:\n%s", out)
	}
}

func TestRecipientTransferReverse_PostsTheReversePath(t *testing.T) {
	isolateConfig(t)
	seedCredentials(t)

	got := startAPI(t, http.StatusCreated, recipientTransferResponse)

	out, err := runCLI(t, "", "recipient", "transfer", "reverse", "tr-1", "--yes")
	if err != nil {
		t.Fatalf("recipient transfer reverse failed: %v (%s)", err, out)
	}

	if got.Method != http.MethodPost || got.Path != "/v1/recipients/onboardings/reverse-transfer/tr-1" {
		t.Errorf("expected POST /v1/recipients/onboardings/reverse-transfer/tr-1, got %s %s", got.Method, got.Path)
	}
}

func TestRecipientTransferReversal_PostsUnderTheTransaction(t *testing.T) {
	isolateConfig(t)
	seedCredentials(t)

	got := startAPI(t, http.StatusCreated, `{"code":"OK","messages":["reversed"]}`)

	out, err := runCLI(t, "", "recipient", "transfer", "reversal", "pay-1", "txn-1", "--yes",
		"--currency", "USD", "--amount", "10", "--description", "oops")
	if err != nil {
		t.Fatalf("recipient transfer reversal failed: %v (%s)", err, out)
	}

	want := "/v1/payments/pay-1/transactions/txn-1/split-marketplace/transfer-reversal"
	if got.Method != http.MethodPost || got.Path != want {
		t.Errorf("expected POST %s, got %s %s", want, got.Method, got.Path)
	}

	body := decodeBody(t, got.Body)

	amount, ok := body["amount"].(map[string]any)
	if !ok || amount["currency"] != "USD" || amount["value"] != 10.0 {
		t.Errorf("unexpected amount in body: %s", got.Body)
	}

	if !strings.Contains(out, "reversed") {
		t.Errorf("expected the acknowledgement in the output, got:\n%s", out)
	}
}

func TestRecipientTransferReversal_ReversesEverythingWithoutAnAmount(t *testing.T) {
	isolateConfig(t)
	seedCredentials(t)

	got := startAPI(t, http.StatusCreated, `{"code":"OK"}`)

	if out, err := runCLI(t, "", "recipient", "transfer", "reversal", "pay-1", "txn-1", "--yes"); err != nil {
		t.Fatalf("recipient transfer reversal failed: %v (%s)", err, out)
	}

	if got.Body != "" {
		t.Errorf("expected no body when no amount is given, got %s", got.Body)
	}
}
