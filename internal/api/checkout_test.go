package api

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCreateCheckoutSession_PostsTheBody(t *testing.T) {
	var gotMethod, gotPath, gotBody string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		gotMethod, gotPath, gotBody = r.Method, r.URL.EscapedPath(), string(body)

		_, _ = w.Write([]byte(`{"checkout_session":"chk-1","merchant_order_id":"order-42","workflow":"SDK_CHECKOUT"}`))
	}))
	defer srv.Close()

	c, _ := newTestClient(t, srv, testProfile())

	session, err := c.CreateCheckoutSession(t.Context(), map[string]any{"merchant_order_id": "order-42"})
	if err != nil {
		t.Fatalf("CreateCheckoutSession: %v", err)
	}

	if gotMethod != http.MethodPost || gotPath != "/checkout/sessions" {
		t.Errorf("expected POST /checkout/sessions, got %s %s", gotMethod, gotPath)
	}

	if !strings.Contains(gotBody, `"merchant_order_id":"order-42"`) {
		t.Errorf("unexpected body: %s", gotBody)
	}

	if session.Session() != "chk-1" {
		t.Errorf("unexpected session: %+v", session)
	}
}

func TestGetCheckoutSession_EscapesTheSession(t *testing.T) {
	var gotMethod, gotPath string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod, gotPath = r.Method, r.URL.EscapedPath()

		_, _ = w.Write([]byte(`{"id":"chk 1","country":"AR"}`))
	}))
	defer srv.Close()

	c, _ := newTestClient(t, srv, testProfile())

	session, err := c.GetCheckoutSession(t.Context(), "chk 1")
	if err != nil {
		t.Fatalf("GetCheckoutSession: %v", err)
	}

	if gotMethod != http.MethodGet || gotPath != "/checkout/sessions/chk%201" {
		t.Errorf("expected GET /checkout/sessions/chk%%201, got %s %s", gotMethod, gotPath)
	}

	// The retrieval endpoint names the session `id`, not `checkout_session`.
	if session.Session() != "chk 1" {
		t.Errorf("unexpected session: %+v", session)
	}
}

func TestGetCheckoutSession_NotFoundIsWrapped(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"code":"NOT_FOUND","message":"checkout session not found"}`))
	}))
	defer srv.Close()

	c, _ := newTestClient(t, srv, testProfile())

	_, err := c.GetCheckoutSession(t.Context(), "chk-1")
	if err == nil {
		t.Fatal("expected an error on 404")
	}

	if !errors.Is(err, ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}

	if !strings.Contains(err.Error(), "get checkout session chk-1") {
		t.Errorf("expected the error to name the operation, got %v", err)
	}
}

func TestUpdateCheckoutSession_PatchesTheSession(t *testing.T) {
	var gotMethod, gotPath, gotBody string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		gotMethod, gotPath, gotBody = r.Method, r.URL.EscapedPath(), string(body)

		_, _ = w.Write([]byte(`{"checkout_session":"chk-1","used":false}`))
	}))
	defer srv.Close()

	c, _ := newTestClient(t, srv, testProfile())

	if _, err := c.UpdateCheckoutSession(t.Context(), "chk-1", map[string]any{"country": "BR"}); err != nil {
		t.Fatalf("UpdateCheckoutSession: %v", err)
	}

	if gotMethod != http.MethodPatch || gotPath != "/checkout/sessions/chk-1" {
		t.Errorf("expected PATCH /checkout/sessions/chk-1, got %s %s", gotMethod, gotPath)
	}

	if !strings.Contains(gotBody, `"country":"BR"`) {
		t.Errorf("unexpected body: %s", gotBody)
	}
}

func TestListCheckoutSessionPaymentMethods_SetsTheCategory(t *testing.T) {
	var gotPath, gotQuery string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath, gotQuery = r.URL.EscapedPath(), r.URL.RawQuery

		_, _ = w.Write([]byte(`[{"name":"VISA ****1111","type":"CARD","category":"CARD",
			"vaulted_token":"vt-1","checkout":{"session":"chk-1"}}]`))
	}))
	defer srv.Close()

	c, _ := newTestClient(t, srv, testProfile())

	methods, err := c.ListCheckoutSessionPaymentMethods(t.Context(), "chk-1", "CARD")
	if err != nil {
		t.Fatalf("ListCheckoutSessionPaymentMethods: %v", err)
	}

	if gotPath != "/checkout/sessions/chk-1/payment-methods" || gotQuery != "category=CARD" {
		t.Errorf("unexpected request: %s?%s", gotPath, gotQuery)
	}

	if len(methods) != 1 || methods[0].SessionCode() != "chk-1" {
		t.Errorf("unexpected payment methods: %+v", methods)
	}
}

func TestListCheckoutSessionPaymentMethods_OmitsAnEmptyCategory(t *testing.T) {
	var gotQuery string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery

		_, _ = w.Write([]byte(`[]`))
	}))
	defer srv.Close()

	c, _ := newTestClient(t, srv, testProfile())

	if _, err := c.ListCheckoutSessionPaymentMethods(t.Context(), "chk-1", ""); err != nil {
		t.Fatalf("ListCheckoutSessionPaymentMethods: %v", err)
	}

	if gotQuery != "" {
		t.Errorf("expected no query, got %q", gotQuery)
	}
}

func TestListEnrollablePaymentMethods_UnwrapsTheList(t *testing.T) {
	var gotPath string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.EscapedPath()

		_, _ = w.Write([]byte(`{"payment_methods":[{"name":"Visa Credit Card","type":"VISA",
			"category":"CARD","enrollment":{"session":"cs-1"}}]}`))
	}))
	defer srv.Close()

	c, _ := newTestClient(t, srv, testProfile())

	methods, err := c.ListEnrollablePaymentMethods(t.Context(), "cs-1")
	if err != nil {
		t.Fatalf("ListEnrollablePaymentMethods: %v", err)
	}

	if gotPath != "/checkout/customers/sessions/cs-1/payment-methods" {
		t.Errorf("unexpected path: %s", gotPath)
	}

	if len(methods) != 1 || methods[0].SessionCode() != "cs-1" {
		t.Errorf("unexpected payment methods: %+v", methods)
	}
}

func TestListEnrollablePaymentMethods_ErrorIsWrapped(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"code":"INSUFFICIENT_SCOPE","message":"forbidden"}`))
	}))
	defer srv.Close()

	c, _ := newTestClient(t, srv, testProfile())

	_, err := c.ListEnrollablePaymentMethods(t.Context(), "cs-1")
	if err == nil {
		t.Fatal("expected an error on 403")
	}

	if !errors.Is(err, ErrForbidden) {
		t.Errorf("expected ErrForbidden, got %v", err)
	}

	if !strings.Contains(err.Error(), "list enrollable payment methods of customer session cs-1") {
		t.Errorf("expected the error to name the operation, got %v", err)
	}
}

func TestEnrollPaymentMethod_PostsToTheCustomerSession(t *testing.T) {
	var gotMethod, gotPath, gotBody string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		gotMethod, gotPath, gotBody = r.Method, r.URL.EscapedPath(), string(body)

		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"id":"pm-1","status":"READY_TO_ENROLL","enrollment":{"session":"cs-1"}}`))
	}))
	defer srv.Close()

	c, _ := newTestClient(t, srv, testProfile())

	method, err := c.EnrollPaymentMethod(t.Context(), "cs-1", map[string]any{"payment_method_type": "CARD"})
	if err != nil {
		t.Fatalf("EnrollPaymentMethod: %v", err)
	}

	if gotMethod != http.MethodPost || gotPath != "/customers/sessions/cs-1/payment-methods" {
		t.Errorf("expected POST /customers/sessions/cs-1/payment-methods, got %s %s", gotMethod, gotPath)
	}

	if !strings.Contains(gotBody, `"payment_method_type":"CARD"`) {
		t.Errorf("unexpected body: %s", gotBody)
	}

	if method.ID != "pm-1" || method.Status != "READY_TO_ENROLL" {
		t.Errorf("unexpected payment method: %+v", method)
	}
}

func TestGetPaymentMethod_HitsThePaymentMethodPath(t *testing.T) {
	var gotMethod, gotPath string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod, gotPath = r.Method, r.URL.EscapedPath()

		_, _ = w.Write([]byte(`{"id":"pm-1","type":"CARD","status":"ENROLLED"}`))
	}))
	defer srv.Close()

	c, _ := newTestClient(t, srv, testProfile())

	if _, err := c.GetPaymentMethod(t.Context(), "pm-1"); err != nil {
		t.Fatalf("GetPaymentMethod: %v", err)
	}

	if gotMethod != http.MethodGet || gotPath != "/payment-methods/pm-1" {
		t.Errorf("expected GET /payment-methods/pm-1, got %s %s", gotMethod, gotPath)
	}
}

func TestUnenrollPaymentMethod_PostsWithoutABody(t *testing.T) {
	var gotMethod, gotPath, gotBody string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		gotMethod, gotPath, gotBody = r.Method, r.URL.EscapedPath(), string(body)

		_, _ = w.Write([]byte(`{"id":"pm-1","status":"UNENROLLED"}`))
	}))
	defer srv.Close()

	c, _ := newTestClient(t, srv, testProfile())

	method, err := c.UnenrollPaymentMethod(t.Context(), "pm-1")
	if err != nil {
		t.Fatalf("UnenrollPaymentMethod: %v", err)
	}

	if gotMethod != http.MethodPost || gotPath != "/customers/payment-methods/pm-1/unenroll" {
		t.Errorf("expected POST /customers/payment-methods/pm-1/unenroll, got %s %s", gotMethod, gotPath)
	}

	if gotBody != "" {
		t.Errorf("expected no request body, got %q", gotBody)
	}

	if method.Status != "UNENROLLED" {
		t.Errorf("unexpected payment method: %+v", method)
	}
}
