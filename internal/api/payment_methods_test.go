package api

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

const enrolledPaymentMethodBody = `{"id":"pm-1","customer_id":"cus-1","type":"CARD","category":"CARD",
	"status":"ENROLLED","vaulted_token":"46adcbc0-aa26-4867-a4e7-28a5ad9100ae",
	"card_data":{"iin":"41111111","lfd":"1111","brand":"VISA","type":"CREDIT"}}`

func TestListCustomerPaymentMethods_UnwrapsTheEnvelope(t *testing.T) {
	var gotPath, gotMethod string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath, gotMethod = r.URL.EscapedPath(), r.Method
		_, _ = w.Write([]byte(`{"payment_methods":[` + enrolledPaymentMethodBody + `]}`))
	}))
	defer srv.Close()

	c, _ := newTestClient(t, srv, testProfile())

	methods, err := c.ListCustomerPaymentMethods(t.Context(), "cus 1")
	if err != nil {
		t.Fatalf("ListCustomerPaymentMethods: %v", err)
	}

	if gotMethod != http.MethodGet || gotPath != "/customers/cus%201/payment-methods" {
		t.Errorf("expected GET /customers/cus%%201/payment-methods, got %s %s", gotMethod, gotPath)
	}

	if len(methods) != 1 || methods[0].ID != "pm-1" {
		t.Fatalf("unexpected payment methods: %+v", methods)
	}

	if methods[0].CardData == nil || methods[0].CardData.LFD != "1111" {
		t.Errorf("expected the card data to be decoded, got %+v", methods[0].CardData)
	}
}

func TestListCustomerPaymentMethods_ErrorIsWrapped(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"code":"AUTHORIZATION_REQUIRED","message":"no"}`))
	}))
	defer srv.Close()

	c, _ := newTestClient(t, srv, testProfile())

	_, err := c.ListCustomerPaymentMethods(t.Context(), "cus-1")
	if err == nil {
		t.Fatal("expected an error on 403")
	}

	if !errors.Is(err, ErrForbidden) {
		t.Errorf("expected ErrForbidden, got %v", err)
	}

	if !strings.Contains(err.Error(), "list payment methods of customer cus-1") {
		t.Errorf("expected the error to name the operation, got %v", err)
	}
}

func TestGetCustomerPaymentMethod_EscapesBothIDs(t *testing.T) {
	var gotPath string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.EscapedPath()
		_, _ = w.Write([]byte(enrolledPaymentMethodBody))
	}))
	defer srv.Close()

	c, _ := newTestClient(t, srv, testProfile())

	method, err := c.GetCustomerPaymentMethod(t.Context(), "cus 1", "pm 1")
	if err != nil {
		t.Fatalf("GetCustomerPaymentMethod: %v", err)
	}

	if gotPath != "/customers/cus%201/payment-methods/pm%201" {
		t.Errorf("unexpected path %s", gotPath)
	}

	if method.Status != "ENROLLED" {
		t.Errorf("unexpected payment method: %+v", method)
	}
}

func TestGetCustomerPaymentMethod_NotFoundIsWrapped(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"code":"NOT_FOUND","message":"nope"}`))
	}))
	defer srv.Close()

	c, _ := newTestClient(t, srv, testProfile())

	_, err := c.GetCustomerPaymentMethod(t.Context(), "cus-1", "pm-1")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}

	if !strings.Contains(err.Error(), "get payment method pm-1 of customer cus-1") {
		t.Errorf("expected the error to name the operation, got %v", err)
	}
}

func TestEnrollCustomerPaymentMethod_PostsTheBody(t *testing.T) {
	var gotPath, gotMethod string

	var gotBody map[string]any

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath, gotMethod = r.URL.EscapedPath(), r.Method
		data, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(data, &gotBody)
		_, _ = w.Write([]byte(enrolledPaymentMethodBody))
	}))
	defer srv.Close()

	c, _ := newTestClient(t, srv, testProfile())

	_, err := c.EnrollCustomerPaymentMethod(t.Context(), "cus-1", map[string]any{"type": "CARD"})
	if err != nil {
		t.Fatalf("EnrollCustomerPaymentMethod: %v", err)
	}

	if gotMethod != http.MethodPost || gotPath != "/customers/cus-1/payment-methods" {
		t.Errorf("expected POST /customers/cus-1/payment-methods, got %s %s", gotMethod, gotPath)
	}

	if gotBody["type"] != "CARD" {
		t.Errorf("unexpected body: %v", gotBody)
	}
}

func TestEnrollCustomerPaymentMethod_BadRequestIsWrapped(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"code":"INVALID_REQUEST","messages":["bad"]}`))
	}))
	defer srv.Close()

	c, _ := newTestClient(t, srv, testProfile())

	_, err := c.EnrollCustomerPaymentMethod(t.Context(), "cus-1", map[string]any{})
	if err == nil {
		t.Fatal("expected an error on 400")
	}

	if !strings.Contains(err.Error(), "enroll payment method for customer cus-1") {
		t.Errorf("expected the error to name the operation, got %v", err)
	}
}

func TestUnenrollCustomerPaymentMethod_HitsTheUnenrollPath(t *testing.T) {
	var gotPath, gotMethod string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath, gotMethod = r.URL.EscapedPath(), r.Method
		_, _ = w.Write([]byte(`{"id":"pm-1","status":"UNENROLLED"}`))
	}))
	defer srv.Close()

	c, _ := newTestClient(t, srv, testProfile())

	method, err := c.UnenrollCustomerPaymentMethod(t.Context(), "cus-1", "pm-1")
	if err != nil {
		t.Fatalf("UnenrollCustomerPaymentMethod: %v", err)
	}

	if gotMethod != http.MethodPost || gotPath != "/customers/cus-1/payment-methods/pm-1/unenroll" {
		t.Errorf("unexpected request %s %s", gotMethod, gotPath)
	}

	if method.Status != "UNENROLLED" {
		t.Errorf("unexpected payment method: %+v", method)
	}
}

func TestUnenrollCustomerPaymentMethod_ErrorIsWrapped(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"code":"NOT_FOUND"}`))
	}))
	defer srv.Close()

	c, _ := newTestClient(t, srv, testProfile())

	_, err := c.UnenrollCustomerPaymentMethod(t.Context(), "cus-1", "pm-1")
	if !strings.Contains(errString(err), "unenroll payment method pm-1 of customer cus-1") {
		t.Errorf("expected the error to name the operation, got %v", err)
	}
}

func TestReassignPaymentMethod_OverridesTheAccountCode(t *testing.T) {
	var gotPath, gotMethod, gotAccount string

	var gotBody map[string]any

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath, gotMethod = r.URL.EscapedPath(), r.Method
		gotAccount = r.Header.Get(HeaderAccountCode)
		data, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(data, &gotBody)
		_, _ = w.Write([]byte(enrolledPaymentMethodBody))
	}))
	defer srv.Close()

	profile := testProfile()
	profile.AccountCode = "profile-account"

	c, _ := newTestClient(t, srv, profile)

	_, err := c.ReassignPaymentMethod(t.Context(), "pm-1", "flag-account", map[string]any{"customer_id": "cus-2"})
	if err != nil {
		t.Fatalf("ReassignPaymentMethod: %v", err)
	}

	if gotMethod != http.MethodPatch || gotPath != "/payment-methods/pm-1" {
		t.Errorf("expected PATCH /payment-methods/pm-1, got %s %s", gotMethod, gotPath)
	}

	if gotAccount != "flag-account" {
		t.Errorf("expected the account code override, got %q", gotAccount)
	}

	if gotBody["customer_id"] != "cus-2" {
		t.Errorf("unexpected body: %v", gotBody)
	}
}

func TestReassignPaymentMethod_KeepsTheProfileAccountCode(t *testing.T) {
	var gotAccount string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAccount = r.Header.Get(HeaderAccountCode)
		_, _ = w.Write([]byte(enrolledPaymentMethodBody))
	}))
	defer srv.Close()

	profile := testProfile()
	profile.AccountCode = "profile-account"

	c, _ := newTestClient(t, srv, profile)

	if _, err := c.ReassignPaymentMethod(t.Context(), "pm-1", "", map[string]any{}); err != nil {
		t.Fatalf("ReassignPaymentMethod: %v", err)
	}

	if gotAccount != "profile-account" {
		t.Errorf("expected the profile account code, got %q", gotAccount)
	}
}

func TestReassignPaymentMethod_ConflictIsWrapped(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusConflict)
		_, _ = w.Write([]byte(`{"code":"CONFLICT","message":"customer has data"}`))
	}))
	defer srv.Close()

	c, _ := newTestClient(t, srv, testProfile())

	_, err := c.ReassignPaymentMethod(t.Context(), "pm-1", "", map[string]any{})
	if !strings.Contains(errString(err), "reassign payment method pm-1") {
		t.Errorf("expected the error to name the operation, got %v", err)
	}
}

func TestRegisterCardsForAccountUpdater_PostsTheIDs(t *testing.T) {
	var gotPath, gotMethod string

	var gotBody map[string]any

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath, gotMethod = r.URL.EscapedPath(), r.Method
		data, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(data, &gotBody)
		_, _ = w.Write([]byte(`{"accepted":2}`))
	}))
	defer srv.Close()

	c, _ := newTestClient(t, srv, testProfile())

	result, err := c.RegisterCardsForAccountUpdater(t.Context(), map[string]any{
		"payment_method_ids": []any{"pm-1", "pm-2"},
	})
	if err != nil {
		t.Fatalf("RegisterCardsForAccountUpdater: %v", err)
	}

	if gotMethod != http.MethodPost || gotPath != "/payment-methods/account-updater" {
		t.Errorf("expected POST /payment-methods/account-updater, got %s %s", gotMethod, gotPath)
	}

	if ids, ok := gotBody["payment_method_ids"].([]any); !ok || len(ids) != 2 {
		t.Errorf("unexpected body: %v", gotBody)
	}

	if result.Accepted != 2 {
		t.Errorf("Accepted = %d, want 2", result.Accepted)
	}
}

func TestRegisterCardsForAccountUpdater_ErrorIsWrapped(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"code":"INVALID_REQUEST","messages":["too many"]}`))
	}))
	defer srv.Close()

	c, _ := newTestClient(t, srv, testProfile())

	_, err := c.RegisterCardsForAccountUpdater(t.Context(), map[string]any{})
	if !strings.Contains(errString(err), "register cards for account updater") {
		t.Errorf("expected the error to name the operation, got %v", err)
	}
}

// errString renders an error for a substring assertion, tolerating nil.
func errString(err error) string {
	if err == nil {
		return ""
	}

	return err.Error()
}
