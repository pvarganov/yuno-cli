package api

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestListRecipients_PagesWithLimitAndOffset(t *testing.T) {
	var queries []string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		queries = append(queries, r.URL.RawQuery)

		if len(queries) == 1 {
			_, _ = io.WriteString(w, `{"data":[{"id":"rec-1"},{"id":"rec-2"}],"pagination":{"has_next":true}}`)

			return
		}

		_, _ = io.WriteString(w, `{"data":[{"id":"rec-3"}],"pagination":{"has_next":false}}`)
	}))
	defer srv.Close()

	c, _ := newTestClient(t, srv, testProfile())

	recipients, err := c.ListRecipients(t.Context(), url.Values{"country": {"BR"}}, 0, 2)
	if err != nil {
		t.Fatalf("ListRecipients: %v", err)
	}

	if len(recipients) != 3 || recipients[2].ID != "rec-3" {
		t.Errorf("unexpected recipients: %+v", recipients)
	}

	if len(queries) != 2 {
		t.Fatalf("expected two requests, got %v", queries)
	}

	for _, want := range []string{"country=BR", "limit=2", "offset=0"} {
		if !strings.Contains(queries[0], want) {
			t.Errorf("expected the first query to contain %q, got %s", want, queries[0])
		}
	}

	if !strings.Contains(queries[1], "offset=2") {
		t.Errorf("expected the second page to start at offset 2, got %s", queries[1])
	}
}

func TestListRecipients_ErrorIsWrapped(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = io.WriteString(w, `{"code":"INSUFFICIENT_SCOPE","message":"nope"}`)
	}))
	defer srv.Close()

	c, _ := newTestClient(t, srv, testProfile())

	_, err := c.ListRecipients(t.Context(), nil, 0, 0)
	if err == nil || !errors.Is(err, ErrForbidden) || !strings.Contains(err.Error(), "list recipients") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestGetRecipient_HitsTheIDPath(t *testing.T) {
	var gotMethod, gotPath string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod, gotPath = r.Method, r.URL.Path

		_, _ = io.WriteString(w, `{"id":"rec-1","first_name":"Ada","last_name":"Lovelace",
			"onboardings":[{"id":"onb-1","status":"SUCCEEDED","provider":{"id":"NUVEI"}}]}`)
	}))
	defer srv.Close()

	c, _ := newTestClient(t, srv, testProfile())

	recipient, err := c.GetRecipient(t.Context(), "rec-1")
	if err != nil {
		t.Fatalf("GetRecipient: %v", err)
	}

	if gotMethod != http.MethodGet || gotPath != "/recipients/rec-1" {
		t.Errorf("expected GET /recipients/rec-1, got %s %s", gotMethod, gotPath)
	}

	if len(recipient.Onboardings) != 1 || recipient.Onboardings[0].ID != "onb-1" {
		t.Errorf("unexpected onboardings: %+v", recipient.Onboardings)
	}
}

func TestGetRecipient_ErrorIsWrapped(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = io.WriteString(w, `{"code":"NOT_FOUND","message":"gone"}`)
	}))
	defer srv.Close()

	c, _ := newTestClient(t, srv, testProfile())

	_, err := c.GetRecipient(t.Context(), "rec-1")
	if err == nil || !errors.Is(err, ErrNotFound) || !strings.Contains(err.Error(), "get recipient rec-1") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestCreateRecipient_PostsTheBody(t *testing.T) {
	var gotMethod, gotPath, gotBody string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		gotMethod, gotPath, gotBody = r.Method, r.URL.Path, string(body)

		w.WriteHeader(http.StatusCreated)
		_, _ = io.WriteString(w, `{"id":"rec-1"}`)
	}))
	defer srv.Close()

	c, _ := newTestClient(t, srv, testProfile())

	recipient, err := c.CreateRecipient(t.Context(), map[string]any{"country": "BR"})
	if err != nil {
		t.Fatalf("CreateRecipient: %v", err)
	}

	if gotMethod != http.MethodPost || gotPath != "/recipients" {
		t.Errorf("expected POST /recipients, got %s %s", gotMethod, gotPath)
	}

	if !strings.Contains(gotBody, `"country":"BR"`) || recipient.ID != "rec-1" {
		t.Errorf("unexpected request or recipient: %s / %+v", gotBody, recipient)
	}
}

func TestCreateRecipient_ErrorIsWrapped(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = io.WriteString(w, `{"code":"VALIDATION_ERROR","message":"bad"}`)
	}))
	defer srv.Close()

	c, _ := newTestClient(t, srv, testProfile())

	if _, err := c.CreateRecipient(t.Context(), map[string]any{}); err == nil ||
		!strings.Contains(err.Error(), "create recipient") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestUpdateRecipient_PatchesTheIDPath(t *testing.T) {
	var gotMethod, gotPath string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod, gotPath = r.Method, r.URL.Path

		_, _ = io.WriteString(w, `{"id":"rec-1","email":"ada@example.com"}`)
	}))
	defer srv.Close()

	c, _ := newTestClient(t, srv, testProfile())

	recipient, err := c.UpdateRecipient(t.Context(), "rec-1", map[string]any{"email": "ada@example.com"})
	if err != nil {
		t.Fatalf("UpdateRecipient: %v", err)
	}

	if gotMethod != http.MethodPatch || gotPath != "/recipients/rec-1" {
		t.Errorf("expected PATCH /recipients/rec-1, got %s %s", gotMethod, gotPath)
	}

	if recipient.Email != "ada@example.com" {
		t.Errorf("unexpected recipient: %+v", recipient)
	}
}

func TestUpdateRecipient_ErrorIsWrapped(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = io.WriteString(w, `{"code":"VALIDATION_ERROR","message":"bad"}`)
	}))
	defer srv.Close()

	c, _ := newTestClient(t, srv, testProfile())

	if _, err := c.UpdateRecipient(t.Context(), "rec-1", map[string]any{}); err == nil ||
		!strings.Contains(err.Error(), "update recipient rec-1") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestDeleteRecipient_SendsDelete(t *testing.T) {
	var gotMethod, gotPath string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod, gotPath = r.Method, r.URL.Path

		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	c, _ := newTestClient(t, srv, testProfile())

	data, err := c.DeleteRecipient(t.Context(), "rec-1")
	if err != nil {
		t.Fatalf("DeleteRecipient: %v", err)
	}

	if gotMethod != http.MethodDelete || gotPath != "/recipients/rec-1" {
		t.Errorf("expected DELETE /recipients/rec-1, got %s %s", gotMethod, gotPath)
	}

	if len(data) != 0 {
		t.Errorf("expected an empty body, got %q", data)
	}
}

func TestDeleteRecipient_ErrorIsWrapped(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusConflict)
		_, _ = io.WriteString(w, `{"code":"CONFLICT","message":"has onboardings"}`)
	}))
	defer srv.Close()

	c, _ := newTestClient(t, srv, testProfile())

	if _, err := c.DeleteRecipient(t.Context(), "rec-1"); err == nil ||
		!strings.Contains(err.Error(), "delete recipient rec-1") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestCreateOnboarding_PostsUnderTheRecipient(t *testing.T) {
	var gotMethod, gotPath, gotBody string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		gotMethod, gotPath, gotBody = r.Method, r.URL.Path, string(body)

		w.WriteHeader(http.StatusCreated)
		_, _ = io.WriteString(w, `{"id":"rec-1","onboardings":[{"id":"onb-1","status":"IN_PROGRESS"}]}`)
	}))
	defer srv.Close()

	c, _ := newTestClient(t, srv, testProfile())

	recipient, err := c.CreateOnboarding(t.Context(), "rec-1", map[string]any{"workflow": "AUTOMATIC"})
	if err != nil {
		t.Fatalf("CreateOnboarding: %v", err)
	}

	if gotMethod != http.MethodPost || gotPath != "/recipients/rec-1/onboardings" {
		t.Errorf("expected POST /recipients/rec-1/onboardings, got %s %s", gotMethod, gotPath)
	}

	if !strings.Contains(gotBody, `"workflow":"AUTOMATIC"`) || len(recipient.Onboardings) != 1 {
		t.Errorf("unexpected request or recipient: %s / %+v", gotBody, recipient)
	}
}

func TestCreateOnboarding_ErrorIsWrapped(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = io.WriteString(w, `{"code":"VALIDATION_ERROR","message":"bad"}`)
	}))
	defer srv.Close()

	c, _ := newTestClient(t, srv, testProfile())

	if _, err := c.CreateOnboarding(t.Context(), "rec-1", map[string]any{}); err == nil ||
		!strings.Contains(err.Error(), "create onboarding for recipient rec-1") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestGetOnboarding_HitsTheOnboardingPath(t *testing.T) {
	var gotMethod, gotPath string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod, gotPath = r.Method, r.URL.Path

		_, _ = io.WriteString(w, `{"id":"onb-1","recipient_id":"rec-1","status":"SUCCEEDED",
			"provider":{"id":"NUVEI"}}`)
	}))
	defer srv.Close()

	c, _ := newTestClient(t, srv, testProfile())

	onboarding, err := c.GetOnboarding(t.Context(), "rec-1", "onb-1")
	if err != nil {
		t.Fatalf("GetOnboarding: %v", err)
	}

	if gotMethod != http.MethodGet || gotPath != "/recipients/rec-1/onboardings/onb-1" {
		t.Errorf("expected GET /recipients/rec-1/onboardings/onb-1, got %s %s", gotMethod, gotPath)
	}

	if onboarding.Status != "SUCCEEDED" || onboarding.Provider == nil || onboarding.Provider.ID != "NUVEI" {
		t.Errorf("unexpected onboarding: %+v", onboarding)
	}
}

func TestGetOnboarding_ErrorIsWrapped(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = io.WriteString(w, `{"code":"NOT_FOUND","message":"gone"}`)
	}))
	defer srv.Close()

	c, _ := newTestClient(t, srv, testProfile())

	_, err := c.GetOnboarding(t.Context(), "rec-1", "onb-1")
	if err == nil || !errors.Is(err, ErrNotFound) || !strings.Contains(err.Error(), "get onboarding onb-1") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestUpdateOnboarding_PatchesTheOnboardingPath(t *testing.T) {
	var gotMethod, gotPath string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod, gotPath = r.Method, r.URL.Path

		_, _ = io.WriteString(w, `{"id":"rec-1"}`)
	}))
	defer srv.Close()

	c, _ := newTestClient(t, srv, testProfile())

	if _, err := c.UpdateOnboarding(t.Context(), "rec-1", "onb-1", map[string]any{"workflow": "MANUAL"}); err != nil {
		t.Fatalf("UpdateOnboarding: %v", err)
	}

	if gotMethod != http.MethodPatch || gotPath != "/recipients/rec-1/onboardings/onb-1" {
		t.Errorf("expected PATCH /recipients/rec-1/onboardings/onb-1, got %s %s", gotMethod, gotPath)
	}
}

func TestUpdateOnboarding_ErrorIsWrapped(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = io.WriteString(w, `{"code":"VALIDATION_ERROR","message":"bad"}`)
	}))
	defer srv.Close()

	c, _ := newTestClient(t, srv, testProfile())

	if _, err := c.UpdateOnboarding(t.Context(), "rec-1", "onb-1", map[string]any{}); err == nil ||
		!strings.Contains(err.Error(), "update onboarding onb-1") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestOnboardingAction_PostsEveryLifecycleAction(t *testing.T) {
	for _, action := range []string{"continue", "cancel", "block", "unblock"} {
		var gotMethod, gotPath string

		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			gotMethod, gotPath = r.Method, r.URL.Path

			_, _ = io.WriteString(w, `{"id":"rec-1"}`)
		}))

		c, _ := newTestClient(t, srv, testProfile())

		if _, err := c.OnboardingAction(t.Context(), "rec-1", "onb-1", action, nil); err != nil {
			t.Fatalf("OnboardingAction %s: %v", action, err)
		}

		want := "/recipients/rec-1/onboardings/onb-1/" + action
		if gotMethod != http.MethodPost || gotPath != want {
			t.Errorf("expected POST %s, got %s %s", want, gotMethod, gotPath)
		}

		srv.Close()
	}
}

func TestOnboardingAction_ErrorIsWrapped(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusConflict)
		_, _ = io.WriteString(w, `{"code":"CONFLICT","message":"already blocked"}`)
	}))
	defer srv.Close()

	c, _ := newTestClient(t, srv, testProfile())

	if _, err := c.OnboardingAction(t.Context(), "rec-1", "onb-1", "block", nil); err == nil ||
		!strings.Contains(err.Error(), "block onboarding onb-1") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestCreateOnboardingTransfer_PostsTheTransferPath(t *testing.T) {
	var gotMethod, gotPath string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod, gotPath = r.Method, r.URL.Path

		w.WriteHeader(http.StatusCreated)
		_, _ = io.WriteString(w, `{"id":"tr-1","origin_onboarding":{"id":"onb-1"},
			"destination_onboarding":{"id":"onb-2","status":"IN_PROGRESS"}}`)
	}))
	defer srv.Close()

	c, _ := newTestClient(t, srv, testProfile())

	transfer, err := c.CreateOnboardingTransfer(t.Context(), "rec-1", "onb-2")
	if err != nil {
		t.Fatalf("CreateOnboardingTransfer: %v", err)
	}

	if gotMethod != http.MethodPost || gotPath != "/recipients/rec-1/onboardings/onb-2/transfer" {
		t.Errorf("expected POST /recipients/rec-1/onboardings/onb-2/transfer, got %s %s", gotMethod, gotPath)
	}

	if transfer.DestinationOnboarding == nil || transfer.DestinationOnboarding.ID != "onb-2" {
		t.Errorf("unexpected transfer: %+v", transfer)
	}
}

func TestCreateOnboardingTransfer_ErrorIsWrapped(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusConflict)
		_, _ = io.WriteString(w, `{"code":"CONFLICT","message":"not onboarded"}`)
	}))
	defer srv.Close()

	c, _ := newTestClient(t, srv, testProfile())

	if _, err := c.CreateOnboardingTransfer(t.Context(), "rec-1", "onb-2"); err == nil ||
		!strings.Contains(err.Error(), "transfer onboarding onb-2") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestReverseOnboardingTransfer_PostsTheReversePath(t *testing.T) {
	var gotMethod, gotPath string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod, gotPath = r.Method, r.URL.Path

		w.WriteHeader(http.StatusCreated)
		_, _ = io.WriteString(w, `{"id":"tr-1"}`)
	}))
	defer srv.Close()

	c, _ := newTestClient(t, srv, testProfile())

	if _, err := c.ReverseOnboardingTransfer(t.Context(), "tr-1"); err != nil {
		t.Fatalf("ReverseOnboardingTransfer: %v", err)
	}

	if gotMethod != http.MethodPost || gotPath != "/recipients/onboardings/reverse-transfer/tr-1" {
		t.Errorf("expected POST /recipients/onboardings/reverse-transfer/tr-1, got %s %s", gotMethod, gotPath)
	}
}

func TestReverseOnboardingTransfer_ErrorIsWrapped(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = io.WriteString(w, `{"code":"INSUFFICIENT_SCOPE","message":"nope"}`)
	}))
	defer srv.Close()

	c, _ := newTestClient(t, srv, testProfile())

	if _, err := c.ReverseOnboardingTransfer(t.Context(), "tr-1"); err == nil ||
		!strings.Contains(err.Error(), "reverse onboarding transfer tr-1") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestGetRecipientTransfer_HitsTheTransferPath(t *testing.T) {
	var gotMethod, gotPath string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod, gotPath = r.Method, r.URL.Path

		_, _ = io.WriteString(w, `{"id":"tr-1","origin_onboarding":{"id":"onb-1"}}`)
	}))
	defer srv.Close()

	c, _ := newTestClient(t, srv, testProfile())

	transfer, err := c.GetRecipientTransfer(t.Context(), "tr-1")
	if err != nil {
		t.Fatalf("GetRecipientTransfer: %v", err)
	}

	if gotMethod != http.MethodGet || gotPath != "/transfers/tr-1" {
		t.Errorf("expected GET /transfers/tr-1, got %s %s", gotMethod, gotPath)
	}

	if transfer.OriginOnboarding == nil || transfer.OriginOnboarding.ID != "onb-1" {
		t.Errorf("unexpected transfer: %+v", transfer)
	}
}

func TestGetRecipientTransfer_ErrorIsWrapped(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = io.WriteString(w, `{"code":"NOT_FOUND","message":"gone"}`)
	}))
	defer srv.Close()

	c, _ := newTestClient(t, srv, testProfile())

	if _, err := c.GetRecipientTransfer(t.Context(), "tr-1"); err == nil ||
		!strings.Contains(err.Error(), "get recipient transfer tr-1") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestListRecipientTransfers_ReadsTheBareArray(t *testing.T) {
	var gotMethod, gotPath string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod, gotPath = r.Method, r.URL.Path

		_, _ = io.WriteString(w, `[{"id":"tr-1"},{"id":"tr-2"}]`)
	}))
	defer srv.Close()

	c, _ := newTestClient(t, srv, testProfile())

	transfers, err := c.ListRecipientTransfers(t.Context(), "rec-1")
	if err != nil {
		t.Fatalf("ListRecipientTransfers: %v", err)
	}

	if gotMethod != http.MethodGet || gotPath != "/recipients/rec-1/transfers" {
		t.Errorf("expected GET /recipients/rec-1/transfers, got %s %s", gotMethod, gotPath)
	}

	if len(transfers) != 2 || transfers[1].ID != "tr-2" {
		t.Errorf("unexpected transfers: %+v", transfers)
	}
}

func TestListRecipientTransfers_ErrorIsWrapped(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = io.WriteString(w, `{"code":"INSUFFICIENT_SCOPE","message":"nope"}`)
	}))
	defer srv.Close()

	c, _ := newTestClient(t, srv, testProfile())

	if _, err := c.ListRecipientTransfers(t.Context(), "rec-1"); err == nil ||
		!strings.Contains(err.Error(), "list transfers of recipient rec-1") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestListOnboardingTransfers_HitsTheOnboardingPath(t *testing.T) {
	var gotMethod, gotPath string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod, gotPath = r.Method, r.URL.Path

		_, _ = io.WriteString(w, `[{"id":"tr-1"}]`)
	}))
	defer srv.Close()

	c, _ := newTestClient(t, srv, testProfile())

	transfers, err := c.ListOnboardingTransfers(t.Context(), "onb-1")
	if err != nil {
		t.Fatalf("ListOnboardingTransfers: %v", err)
	}

	if gotMethod != http.MethodGet || gotPath != "/onboardings/onb-1/transfers" {
		t.Errorf("expected GET /onboardings/onb-1/transfers, got %s %s", gotMethod, gotPath)
	}

	if len(transfers) != 1 {
		t.Errorf("unexpected transfers: %+v", transfers)
	}
}

func TestListOnboardingTransfers_ErrorIsWrapped(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = io.WriteString(w, `{"code":"NOT_FOUND","message":"gone"}`)
	}))
	defer srv.Close()

	c, _ := newTestClient(t, srv, testProfile())

	if _, err := c.ListOnboardingTransfers(t.Context(), "onb-1"); err == nil ||
		!strings.Contains(err.Error(), "list transfers of onboarding onb-1") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestReversePaymentTransfer_PostsUnderTheTransaction(t *testing.T) {
	var gotMethod, gotPath, gotBody string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		gotMethod, gotPath, gotBody = r.Method, r.URL.Path, string(body)

		w.WriteHeader(http.StatusCreated)
		_, _ = io.WriteString(w, `{"code":"OK","messages":["reversed"]}`)
	}))
	defer srv.Close()

	c, _ := newTestClient(t, srv, testProfile())

	data, err := c.ReversePaymentTransfer(t.Context(), "pay-1", "txn-1", map[string]any{"description": "oops"})
	if err != nil {
		t.Fatalf("ReversePaymentTransfer: %v", err)
	}

	want := "/payments/pay-1/transactions/txn-1/split-marketplace/transfer-reversal"
	if gotMethod != http.MethodPost || gotPath != want {
		t.Errorf("expected POST %s, got %s %s", want, gotMethod, gotPath)
	}

	if !strings.Contains(gotBody, `"description":"oops"`) || !strings.Contains(string(data), "reversed") {
		t.Errorf("unexpected request or response: %s / %s", gotBody, data)
	}
}

func TestReversePaymentTransfer_ErrorIsWrapped(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = io.WriteString(w, `{"code":"VALIDATION_ERROR","message":"bad"}`)
	}))
	defer srv.Close()

	c, _ := newTestClient(t, srv, testProfile())

	if _, err := c.ReversePaymentTransfer(t.Context(), "pay-1", "txn-1", nil); err == nil ||
		!strings.Contains(err.Error(), "reverse transfer of payment pay-1") {
		t.Errorf("unexpected error: %v", err)
	}
}
