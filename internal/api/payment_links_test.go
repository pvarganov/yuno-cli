package api

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCreatePaymentLink_PostsTheBody(t *testing.T) {
	var gotMethod, gotPath, gotBody string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		gotMethod, gotPath, gotBody = r.Method, r.URL.Path, string(body)

		_, _ = io.WriteString(w, `{"id":"pl-1","status":"CREATED"}`)
	}))
	defer srv.Close()

	c, _ := newTestClient(t, srv, testProfile())

	link, err := c.CreatePaymentLink(t.Context(), map[string]any{"country": "US"})
	if err != nil {
		t.Fatalf("CreatePaymentLink: %v", err)
	}

	if gotMethod != http.MethodPost || gotPath != "/payment-links" {
		t.Errorf("expected POST /payment-links, got %s %s", gotMethod, gotPath)
	}

	if !strings.Contains(gotBody, `"country":"US"`) {
		t.Errorf("unexpected body: %s", gotBody)
	}

	if link.Reference() != "pl-1" {
		t.Errorf("unexpected link: %+v", link)
	}
}

func TestCreatePaymentLink_ErrorIsWrapped(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = io.WriteString(w, `{"code":"INSUFFICIENT_SCOPE","message":"nope"}`)
	}))
	defer srv.Close()

	c, _ := newTestClient(t, srv, testProfile())

	_, err := c.CreatePaymentLink(t.Context(), map[string]any{})
	if err == nil {
		t.Fatal("expected an error on 403")
	}

	if !errors.Is(err, ErrForbidden) || !strings.Contains(err.Error(), "create payment link") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestGetPaymentLink_HitsTheCodePath(t *testing.T) {
	var gotMethod, gotPath string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod, gotPath = r.Method, r.URL.Path

		_, _ = io.WriteString(w, `{"code":"pl-1","status":"CREATED","checkout_url":"https://pay.y.uno/pl-1"}`)
	}))
	defer srv.Close()

	c, _ := newTestClient(t, srv, testProfile())

	link, err := c.GetPaymentLink(t.Context(), "pl-1")
	if err != nil {
		t.Fatalf("GetPaymentLink: %v", err)
	}

	if gotMethod != http.MethodGet || gotPath != "/payment-links/pl-1" {
		t.Errorf("expected GET /payment-links/pl-1, got %s %s", gotMethod, gotPath)
	}

	if link.CheckoutURL != "https://pay.y.uno/pl-1" {
		t.Errorf("unexpected link: %+v", link)
	}
}

func TestGetPaymentLink_ErrorIsWrapped(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = io.WriteString(w, `{"code":"NOT_FOUND","message":"gone"}`)
	}))
	defer srv.Close()

	c, _ := newTestClient(t, srv, testProfile())

	_, err := c.GetPaymentLink(t.Context(), "pl-1")
	if err == nil || !strings.Contains(err.Error(), "get payment link pl-1") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestCancelPaymentLink_PostsToTheCancelPath(t *testing.T) {
	var gotMethod, gotPath string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod, gotPath = r.Method, r.URL.Path

		_, _ = io.WriteString(w, `{"code":"pl-1","status":"CANCELED"}`)
	}))
	defer srv.Close()

	c, _ := newTestClient(t, srv, testProfile())

	link, err := c.CancelPaymentLink(t.Context(), "pl-1")
	if err != nil {
		t.Fatalf("CancelPaymentLink: %v", err)
	}

	if gotMethod != http.MethodPost || gotPath != "/payment-links/pl-1/cancel" {
		t.Errorf("expected POST /payment-links/pl-1/cancel, got %s %s", gotMethod, gotPath)
	}

	if link.Status != "CANCELED" {
		t.Errorf("unexpected link: %+v", link)
	}
}

func TestCancelPaymentLink_ErrorIsWrapped(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusConflict)
		_, _ = io.WriteString(w, `{"code":"CONFLICT","message":"already canceled"}`)
	}))
	defer srv.Close()

	c, _ := newTestClient(t, srv, testProfile())

	_, err := c.CancelPaymentLink(t.Context(), "pl-1")
	if err == nil || !strings.Contains(err.Error(), "cancel payment link pl-1") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestGetConversionRate_PostsTheBody(t *testing.T) {
	var gotMethod, gotPath, gotBody string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		gotMethod, gotPath, gotBody = r.Method, r.URL.Path, string(body)

		_, _ = io.WriteString(w, `{"id":"cr-1","amount":{"value":10000,"currency":"COP",
			"currency_conversion":{"cardholder_currency":"USD","cardholder_amount":2.58,"rate":3879.81}}}`)
	}))
	defer srv.Close()

	c, _ := newTestClient(t, srv, testProfile())

	rate, err := c.GetConversionRate(t.Context(), map[string]any{"account_id": "acc-1"})
	if err != nil {
		t.Fatalf("GetConversionRate: %v", err)
	}

	if gotMethod != http.MethodPost || gotPath != "/currency-conversion" {
		t.Errorf("expected POST /currency-conversion, got %s %s", gotMethod, gotPath)
	}

	if !strings.Contains(gotBody, `"account_id":"acc-1"`) {
		t.Errorf("unexpected body: %s", gotBody)
	}

	if rate.Amount == nil || rate.Amount.CurrencyConversion.CardholderAmount != 2.58 {
		t.Errorf("unexpected rate: %+v", rate)
	}
}

func TestGetConversionRate_ErrorIsWrapped(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = io.WriteString(w, `{"code":"INVALID","message":"bad provider"}`)
	}))
	defer srv.Close()

	c, _ := newTestClient(t, srv, testProfile())

	_, err := c.GetConversionRate(t.Context(), map[string]any{})
	if err == nil || !strings.Contains(err.Error(), "get conversion rate") {
		t.Errorf("unexpected error: %v", err)
	}
}
