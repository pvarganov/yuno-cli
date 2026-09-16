package api

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestAICallerDeclinedPayments_HitsTheRecoverPath(t *testing.T) {
	var gotMethod, gotPath string

	var gotBody map[string]any

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod, gotPath = r.Method, r.URL.Path
		_ = json.NewDecoder(r.Body).Decode(&gotBody)

		_, _ = io.WriteString(w, `{"id":"call-1","message":"outreach scheduled"}`)
	}))
	defer srv.Close()

	c, _ := newTestClient(t, srv, testProfile())

	outreach, err := c.AICallerDeclinedPayments(t.Context(), map[string]any{"settings": map[string]any{}})
	if err != nil {
		t.Fatalf("AICallerDeclinedPayments: %v", err)
	}

	if gotMethod != http.MethodPost || gotPath != "/smart-support/external/payments/recover" {
		t.Errorf("expected POST /smart-support/external/payments/recover, got %s %s", gotMethod, gotPath)
	}

	if outreach.ID != "call-1" || outreach.Message != "outreach scheduled" {
		t.Errorf("unexpected outreach: %+v", outreach)
	}
}

func TestAICallerDeclinedPayments_ErrorIsWrapped(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = io.WriteString(w, `{"message":"missing payment"}`)
	}))
	defer srv.Close()

	c, _ := newTestClient(t, srv, testProfile())

	if _, err := c.AICallerDeclinedPayments(t.Context(), map[string]any{}); err == nil ||
		!strings.Contains(err.Error(), "start declined payment outreach") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestAICallerRecover_HitsThePaymentsPath(t *testing.T) {
	var gotMethod, gotPath string

	var gotBody map[string]any

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod, gotPath = r.Method, r.URL.Path
		_ = json.NewDecoder(r.Body).Decode(&gotBody)

		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c, _ := newTestClient(t, srv, testProfile())

	data, err := c.AICallerRecover(t.Context(), map[string]any{"customer": map[string]any{"email": "a@b.c"}})
	if err != nil {
		t.Fatalf("AICallerRecover: %v", err)
	}

	if gotMethod != http.MethodPost || gotPath != "/smart-support/external/payments" {
		t.Errorf("expected POST /smart-support/external/payments, got %s %s", gotMethod, gotPath)
	}

	if _, ok := gotBody["customer"]; !ok || len(data) != 0 {
		t.Errorf("unexpected recover: body %+v, response %s", gotBody, data)
	}
}

func TestAICallerRecover_ErrorIsWrapped(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = io.WriteString(w, `{"message":"boom"}`)
	}))
	defer srv.Close()

	c, _ := newTestClient(t, srv, testProfile())

	if _, err := c.AICallerRecover(t.Context(), map[string]any{}); err == nil ||
		!strings.Contains(err.Error(), "start abandoned flow outreach") {
		t.Errorf("unexpected error: %v", err)
	}
}
