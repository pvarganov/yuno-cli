package api

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGetCustomer_EscapesTheID(t *testing.T) {
	var gotPath, gotMethod string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.EscapedPath()
		gotMethod = r.Method
		_, _ = w.Write([]byte(`{"id":"cus-1","email":"user@example.com"}`))
	}))
	defer srv.Close()

	c, _ := newTestClient(t, srv, testProfile())

	customer, err := c.GetCustomer(t.Context(), "cus 1")
	if err != nil {
		t.Fatalf("GetCustomer: %v", err)
	}

	if gotMethod != http.MethodGet || gotPath != "/customers/cus%201" {
		t.Errorf("expected GET /customers/cus%%201, got %s %s", gotMethod, gotPath)
	}

	if customer.Email != "user@example.com" {
		t.Errorf("unexpected customer: %+v", customer)
	}
}

func TestGetCustomer_NotFoundIsWrapped(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"code":"NOT_FOUND","message":"customer not found"}`))
	}))
	defer srv.Close()

	c, _ := newTestClient(t, srv, testProfile())

	if _, err := c.GetCustomer(t.Context(), "cus-1"); err == nil {
		t.Fatal("expected an error on 404")
	} else {
		if !errors.Is(err, ErrNotFound) {
			t.Errorf("expected ErrNotFound, got %v", err)
		}

		if !strings.Contains(err.Error(), "get customer cus-1") {
			t.Errorf("expected the error to name the operation, got %v", err)
		}
	}
}

func TestGetCustomerByMerchantCustomerID_SetsTheQuery(t *testing.T) {
	var gotPath, gotQuery string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.EscapedPath()
		gotQuery = r.URL.RawQuery
		_, _ = w.Write([]byte(`{"id":"cus-1","merchant_customer_id":"user-42"}`))
	}))
	defer srv.Close()

	c, _ := newTestClient(t, srv, testProfile())

	customer, err := c.GetCustomerByMerchantCustomerID(t.Context(), "user-42")
	if err != nil {
		t.Fatalf("GetCustomerByMerchantCustomerID: %v", err)
	}

	if gotPath != "/customers" || gotQuery != "merchant_customer_id=user-42" {
		t.Errorf("expected GET /customers?merchant_customer_id=user-42, got %s?%s", gotPath, gotQuery)
	}

	if customer.MerchantCustomerID != "user-42" {
		t.Errorf("unexpected customer: %+v", customer)
	}
}

func TestGetCustomerByMerchantCustomerID_ErrorNamesTheOperation(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"code":"NOT_FOUND","message":"customer not found"}`))
	}))
	defer srv.Close()

	c, _ := newTestClient(t, srv, testProfile())

	_, err := c.GetCustomerByMerchantCustomerID(t.Context(), "user-42")
	if err == nil || !strings.Contains(err.Error(), "get customer by merchant customer id user-42") {
		t.Errorf("expected the error to name the operation, got %v", err)
	}
}

func TestCreateAndUpdateCustomer_UseTheRightMethods(t *testing.T) {
	var gotMethod, gotPath, gotBody string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.EscapedPath()

		buf, _ := io.ReadAll(r.Body)
		gotBody = string(buf)

		_, _ = w.Write([]byte(`{"id":"cus-1"}`))
	}))
	defer srv.Close()

	c, _ := newTestClient(t, srv, testProfile())

	if _, err := c.CreateCustomer(t.Context(), map[string]any{"merchant_customer_id": "user-42"}); err != nil {
		t.Fatalf("CreateCustomer: %v", err)
	}

	if gotMethod != http.MethodPost || gotPath != "/customers" {
		t.Errorf("expected POST /customers, got %s %s", gotMethod, gotPath)
	}

	if !strings.Contains(gotBody, "user-42") {
		t.Errorf("expected the body to carry the merchant customer id, got %s", gotBody)
	}

	if _, err := c.UpdateCustomer(t.Context(), "cus-1", map[string]any{"email": ""}); err != nil {
		t.Fatalf("UpdateCustomer: %v", err)
	}

	if gotMethod != http.MethodPatch || gotPath != "/customers/cus-1" {
		t.Errorf("expected PATCH /customers/cus-1, got %s %s", gotMethod, gotPath)
	}
}

func TestUpdateCustomer_ErrorNamesTheOperation(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"code":"VALIDATION_ERROR","message":"invalid email"}`))
	}))
	defer srv.Close()

	c, _ := newTestClient(t, srv, testProfile())

	_, err := c.UpdateCustomer(t.Context(), "cus-1", map[string]any{"email": "x"})
	if err == nil || !strings.Contains(err.Error(), "update customer cus-1") {
		t.Errorf("expected the error to name the operation, got %v", err)
	}
}

func TestDeleteCustomer_AcceptsAnEmptyBody(t *testing.T) {
	var gotMethod, gotPath string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.EscapedPath()
		w.WriteHeader(http.StatusAccepted)
	}))
	defer srv.Close()

	c, _ := newTestClient(t, srv, testProfile())

	data, err := c.DeleteCustomer(t.Context(), "cus-1")
	if err != nil {
		t.Fatalf("DeleteCustomer: %v", err)
	}

	if gotMethod != http.MethodDelete || gotPath != "/customers/cus-1" {
		t.Errorf("expected DELETE /customers/cus-1, got %s %s", gotMethod, gotPath)
	}

	if len(data) != 0 {
		t.Errorf("expected an empty response body, got %s", data)
	}
}

func TestDeleteCustomer_ErrorNamesTheOperation(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"code":"INSUFFICIENT_SCOPE","message":"no scope"}`))
	}))
	defer srv.Close()

	c, _ := newTestClient(t, srv, testProfile())

	_, err := c.DeleteCustomer(t.Context(), "cus-1")
	if err == nil || !errors.Is(err, ErrForbidden) {
		t.Fatalf("expected ErrForbidden, got %v", err)
	}

	if !strings.Contains(err.Error(), "delete customer cus-1") {
		t.Errorf("expected the error to name the operation, got %v", err)
	}
}

func TestCreateCustomerSession_PostsToTheSessionsPath(t *testing.T) {
	var gotMethod, gotPath string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.EscapedPath()
		_, _ = w.Write([]byte(`{"customer_session":"sess-1","customer_id":"cus-1","country":"CO"}`))
	}))
	defer srv.Close()

	c, _ := newTestClient(t, srv, testProfile())

	session, err := c.CreateCustomerSession(t.Context(), map[string]any{"customer_id": "cus-1"})
	if err != nil {
		t.Fatalf("CreateCustomerSession: %v", err)
	}

	if gotMethod != http.MethodPost || gotPath != "/customers/sessions" {
		t.Errorf("expected POST /customers/sessions, got %s %s", gotMethod, gotPath)
	}

	if session.CustomerSession != "sess-1" {
		t.Errorf("unexpected session: %+v", session)
	}
}

func TestCreateCustomerSession_ErrorNamesTheOperation(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"code":"VALIDATION_ERROR","message":"account_id is required"}`))
	}))
	defer srv.Close()

	c, _ := newTestClient(t, srv, testProfile())

	_, err := c.CreateCustomerSession(t.Context(), map[string]any{})
	if err == nil || !strings.Contains(err.Error(), "create customer session") {
		t.Errorf("expected the error to name the operation, got %v", err)
	}
}

func TestGenerateNetworkTokenCryptogram_PostsToTheCryptogramsPath(t *testing.T) {
	var gotMethod, gotPath string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.EscapedPath()
		_, _ = w.Write([]byte(`{"vaulted_token":"vt-1","network_token":"nt-1","cryptogram":"cr-1","eci":"05"}`))
	}))
	defer srv.Close()

	c, _ := newTestClient(t, srv, testProfile())

	cryptogram, err := c.GenerateNetworkTokenCryptogram(t.Context(), map[string]any{"vaulted_token": "vt-1"})
	if err != nil {
		t.Fatalf("GenerateNetworkTokenCryptogram: %v", err)
	}

	if gotMethod != http.MethodPost || gotPath != "/network-tokens/cryptograms" {
		t.Errorf("expected POST /network-tokens/cryptograms, got %s %s", gotMethod, gotPath)
	}

	if cryptogram.Cryptogram != "cr-1" || cryptogram.ECI != "05" {
		t.Errorf("unexpected cryptogram: %+v", cryptogram)
	}
}

func TestGenerateNetworkTokenCryptogram_ErrorNamesTheOperation(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
		_, _ = w.Write([]byte(`{"code":"PROVIDER_ERROR","message":"scheme unavailable"}`))
	}))
	defer srv.Close()

	c, _ := newTestClient(t, srv, testProfile())

	_, err := c.GenerateNetworkTokenCryptogram(t.Context(), map[string]any{"vaulted_token": "vt-1"})
	if err == nil || !strings.Contains(err.Error(), "generate network token cryptogram") {
		t.Errorf("expected the error to name the operation, got %v", err)
	}
}
