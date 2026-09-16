package api

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestListPayouts_FiltersByMerchantReference(t *testing.T) {
	var gotMethod, gotPath, gotQuery string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod, gotPath, gotQuery = r.Method, r.URL.Path, r.URL.RawQuery

		_, _ = io.WriteString(w, `[{"id":"po-1","status":"SUCCEEDED","merchant_reference":"ref-1"}]`)
	}))
	defer srv.Close()

	c, _ := newTestClient(t, srv, testProfile())

	payouts, err := c.ListPayouts(t.Context(), "ref-1")
	if err != nil {
		t.Fatalf("ListPayouts: %v", err)
	}

	if gotMethod != http.MethodGet || gotPath != "/payouts" {
		t.Errorf("expected GET /payouts, got %s %s", gotMethod, gotPath)
	}

	if gotQuery != "merchant_reference=ref-1" {
		t.Errorf("unexpected query: %s", gotQuery)
	}

	if len(payouts) != 1 || payouts[0].ID != "po-1" {
		t.Errorf("unexpected payouts: %+v", payouts)
	}
}

func TestListPayouts_ErrorIsWrapped(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = io.WriteString(w, `{"code":"INSUFFICIENT_SCOPE","message":"nope"}`)
	}))
	defer srv.Close()

	c, _ := newTestClient(t, srv, testProfile())

	_, err := c.ListPayouts(t.Context(), "ref-1")
	if err == nil || !errors.Is(err, ErrForbidden) || !strings.Contains(err.Error(), "list payouts") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestListPayouts_RejectsAMalformedBody(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, `[{"id":`)
	}))
	defer srv.Close()

	c, _ := newTestClient(t, srv, testProfile())

	if _, err := c.ListPayouts(t.Context(), "ref-1"); err == nil {
		t.Fatal("expected a decode error")
	}
}

func TestGetPayout_HitsTheIDPath(t *testing.T) {
	var gotMethod, gotPath string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod, gotPath = r.Method, r.URL.Path

		_, _ = io.WriteString(w, `{"id":"po-1","status":"SUCCEEDED","amount":{"currency":"USD","value":100}}`)
	}))
	defer srv.Close()

	c, _ := newTestClient(t, srv, testProfile())

	payout, err := c.GetPayout(t.Context(), "po-1")
	if err != nil {
		t.Fatalf("GetPayout: %v", err)
	}

	if gotMethod != http.MethodGet || gotPath != "/payouts/po-1" {
		t.Errorf("expected GET /payouts/po-1, got %s %s", gotMethod, gotPath)
	}

	if payout.Amount == nil || payout.Amount.Value != 100 {
		t.Errorf("unexpected payout: %+v", payout)
	}
}

func TestGetPayout_ErrorIsWrapped(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = io.WriteString(w, `{"code":"NOT_FOUND","message":"gone"}`)
	}))
	defer srv.Close()

	c, _ := newTestClient(t, srv, testProfile())

	_, err := c.GetPayout(t.Context(), "po-1")
	if err == nil || !errors.Is(err, ErrNotFound) || !strings.Contains(err.Error(), "get payout po-1") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestCreatePayout_PostsTheBody(t *testing.T) {
	var gotMethod, gotPath, gotBody string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		gotMethod, gotPath, gotBody = r.Method, r.URL.Path, string(body)

		w.WriteHeader(http.StatusCreated)
		_, _ = io.WriteString(w, `{"id":"po-1","status":"CREATED"}`)
	}))
	defer srv.Close()

	c, _ := newTestClient(t, srv, testProfile())

	payout, err := c.CreatePayout(t.Context(), map[string]any{"country": "US"})
	if err != nil {
		t.Fatalf("CreatePayout: %v", err)
	}

	if gotMethod != http.MethodPost || gotPath != "/payouts" {
		t.Errorf("expected POST /payouts, got %s %s", gotMethod, gotPath)
	}

	if !strings.Contains(gotBody, `"country":"US"`) {
		t.Errorf("unexpected body: %s", gotBody)
	}

	if payout.ID != "po-1" {
		t.Errorf("unexpected payout: %+v", payout)
	}
}

func TestCreatePayout_ErrorIsWrapped(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = io.WriteString(w, `{"code":"VALIDATION_ERROR","message":"bad"}`)
	}))
	defer srv.Close()

	c, _ := newTestClient(t, srv, testProfile())

	_, err := c.CreatePayout(t.Context(), map[string]any{})
	if err == nil || !strings.Contains(err.Error(), "create payout") {
		t.Errorf("unexpected error: %v", err)
	}
}
