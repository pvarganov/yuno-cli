package api

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGetPayment_OmitsTheUnsetQueryFlags(t *testing.T) {
	var gotQuery, gotPath string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.EscapedPath()
		gotQuery = r.URL.RawQuery
		_, _ = w.Write([]byte(`{"id":"pay-1","status":"SUCCEEDED"}`))
	}))
	defer srv.Close()

	c, _ := newTestClient(t, srv, testProfile())

	payment, err := c.GetPayment(t.Context(), "pay 1", false, false)
	if err != nil {
		t.Fatalf("GetPayment: %v", err)
	}

	if gotPath != "/payments/pay%201" {
		t.Errorf("expected the payment id to be escaped, got %s", gotPath)
	}

	if gotQuery != "" {
		t.Errorf("expected no query parameters, got %s", gotQuery)
	}

	if payment.Status != "SUCCEEDED" {
		t.Errorf("unexpected payment: %+v", payment)
	}
}

func TestGetPayment_NotFoundIsWrapped(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"code":"NOT_FOUND","message":"payment not found"}`))
	}))
	defer srv.Close()

	c, _ := newTestClient(t, srv, testProfile())

	_, err := c.GetPayment(t.Context(), "pay-1", true, true)
	if err == nil {
		t.Fatal("expected an error on 404")
	}

	if !errors.Is(err, ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}

	if !strings.Contains(err.Error(), "get payment pay-1") {
		t.Errorf("expected the error to name the operation, got %v", err)
	}
}

func TestListIssuers_BuildsTheQuery(t *testing.T) {
	var gotQuery string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		_, _ = w.Write([]byte(`{"issuers":[{"id":"1001","name":"BANCO"}]}`))
	}))
	defer srv.Close()

	c, _ := newTestClient(t, srv, testProfile())

	issuers, err := c.ListIssuers(t.Context(), "CO", "PSE", "")
	if err != nil {
		t.Fatalf("ListIssuers: %v", err)
	}

	if gotQuery != "country_code=CO&payment_method=PSE" {
		t.Errorf("unexpected query: %s", gotQuery)
	}

	if len(issuers.Issuers) != 1 {
		t.Errorf("unexpected issuers: %+v", issuers)
	}
}

func TestPaymentActions_HitTheExpectedPaths(t *testing.T) {
	var gotMethod, gotPath string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		_, _ = w.Write([]byte(`{"id":"pay-1"}`))
	}))
	defer srv.Close()

	c, _ := newTestClient(t, srv, testProfile())
	body := map[string]any{"merchant_reference": "ref-1"}

	tests := []struct {
		name       string
		call       func() error
		wantMethod string
		wantPath   string
	}{
		{
			name:       "refund",
			call:       func() error { _, err := c.RefundPayment(t.Context(), "p", "t", body); return err },
			wantMethod: http.MethodPost,
			wantPath:   "/payments/p/transactions/t/refund",
		},
		{
			name:       "cancel",
			call:       func() error { _, err := c.CancelTransaction(t.Context(), "p", "t", body); return err },
			wantMethod: http.MethodPost,
			wantPath:   "/payments/p/transactions/t/cancel",
		},
		{
			name:       "capture",
			call:       func() error { _, err := c.CaptureTransaction(t.Context(), "p", "t", body); return err },
			wantMethod: http.MethodPost,
			wantPath:   "/payments/p/transactions/t/capture",
		},
		{
			name:       "cancel or refund payment",
			call:       func() error { _, err := c.CancelOrRefundPayment(t.Context(), "p", body); return err },
			wantMethod: http.MethodPost,
			wantPath:   "/payments/p/cancel-or-refund",
		},
		{
			name: "cancel or refund transaction",
			call: func() error {
				_, err := c.CancelOrRefundTransaction(t.Context(), "p", "t", body)

				return err
			},
			wantMethod: http.MethodPost,
			wantPath:   "/payments/p/transactions/t/cancel-or-refund",
		},
		{
			name:       "create dispute",
			call:       func() error { _, err := c.CreateDispute(t.Context(), "p", "t", body); return err },
			wantMethod: http.MethodPost,
			wantPath:   "/payments/p/transactions/t/dispute",
		},
		{
			name:       "update dispute",
			call:       func() error { _, err := c.UpdateDispute(t.Context(), "p", "t", body); return err },
			wantMethod: http.MethodPatch,
			wantPath:   "/payments/p/transactions/t/dispute",
		},
		{
			name:       "fulfillment",
			call:       func() error { _, err := c.CreateFulfillment(t.Context(), "p", body); return err },
			wantMethod: http.MethodPost,
			wantPath:   "/payments/p/fulfillments",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.call(); err != nil {
				t.Fatalf("%s: %v", tt.name, err)
			}

			if gotMethod != tt.wantMethod || gotPath != tt.wantPath {
				t.Errorf("expected %s %s, got %s %s", tt.wantMethod, tt.wantPath, gotMethod, gotPath)
			}
		})
	}
}

func TestCreatePayment_ServerErrorIsWrapped(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"code":"VALIDATION_ERROR","message":"amount is required"}`))
	}))
	defer srv.Close()

	c, _ := newTestClient(t, srv, testProfile())

	_, err := c.CreatePayment(t.Context(), map[string]any{})
	if err == nil {
		t.Fatal("expected an error on 400")
	}

	if !strings.Contains(err.Error(), "create payment") || !strings.Contains(err.Error(), "amount is required") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestGetPaymentByMerchantOrderID_DecodesTheList(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`[{"id":"pay-1"},{"id":"pay-2"}]`))
	}))
	defer srv.Close()

	c, _ := newTestClient(t, srv, testProfile())

	payments, err := c.GetPaymentByMerchantOrderID(t.Context(), "order-42")
	if err != nil {
		t.Fatalf("GetPaymentByMerchantOrderID: %v", err)
	}

	if len(payments) != 2 || payments[1].ID != "pay-2" {
		t.Errorf("unexpected payments: %+v", payments)
	}
}
