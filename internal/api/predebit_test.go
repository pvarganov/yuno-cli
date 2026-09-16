package api

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCreatePreDebitNotification_PostsTheBody(t *testing.T) {
	var gotMethod, gotPath, gotBody string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		gotMethod, gotPath, gotBody = r.Method, r.URL.Path, string(body)

		w.WriteHeader(http.StatusCreated)
		_, _ = io.WriteString(w, `{"id":"pdn-1","status":"CREATED"}`)
	}))
	defer srv.Close()

	c, _ := newTestClient(t, srv, testProfile())

	notification, err := c.CreatePreDebitNotification(t.Context(), map[string]any{"merchant_reference": "ref-1"})
	if err != nil {
		t.Fatalf("CreatePreDebitNotification: %v", err)
	}

	if gotMethod != http.MethodPost || gotPath != "/predebit-notify" {
		t.Errorf("expected POST /predebit-notify, got %s %s", gotMethod, gotPath)
	}

	if !strings.Contains(gotBody, `"merchant_reference":"ref-1"`) {
		t.Errorf("unexpected body: %s", gotBody)
	}

	if notification.ID != "pdn-1" {
		t.Errorf("unexpected notification: %+v", notification)
	}
}

func TestCreatePreDebitNotification_ErrorIsWrapped(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = io.WriteString(w, `{"code":"INSUFFICIENT_SCOPE","message":"nope"}`)
	}))
	defer srv.Close()

	c, _ := newTestClient(t, srv, testProfile())

	_, err := c.CreatePreDebitNotification(t.Context(), map[string]any{})
	if err == nil || !errors.Is(err, ErrForbidden) ||
		!strings.Contains(err.Error(), "create pre-debit notification") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestGetPreDebitNotificationByReference_FiltersByMerchantReference(t *testing.T) {
	var gotPath, gotQuery string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath, gotQuery = r.URL.Path, r.URL.RawQuery

		_, _ = io.WriteString(w, `{"id":"pdn-1","amount":{"currency":"INR","value":"1000"}}`)
	}))
	defer srv.Close()

	c, _ := newTestClient(t, srv, testProfile())

	notification, err := c.GetPreDebitNotificationByReference(t.Context(), "ref-1")
	if err != nil {
		t.Fatalf("GetPreDebitNotificationByReference: %v", err)
	}

	if gotPath != "/predebit-notify" || gotQuery != "merchant_reference=ref-1" {
		t.Errorf("expected GET /predebit-notify?merchant_reference=ref-1, got %s?%s", gotPath, gotQuery)
	}

	if notification.Amount.Label() != "INR 1000" {
		t.Errorf("unexpected amount: %+v", notification.Amount)
	}
}

func TestGetPreDebitNotificationByReference_ErrorIsWrapped(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = io.WriteString(w, `{"code":"NOT_FOUND","message":"gone"}`)
	}))
	defer srv.Close()

	c, _ := newTestClient(t, srv, testProfile())

	_, err := c.GetPreDebitNotificationByReference(t.Context(), "ref-1")
	if err == nil || !errors.Is(err, ErrNotFound) ||
		!strings.Contains(err.Error(), "merchant reference ref-1") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestGetPreDebitNotification_HitsTheIDPath(t *testing.T) {
	var gotMethod, gotPath string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod, gotPath = r.Method, r.URL.Path

		_, _ = io.WriteString(w, `{"id":"pdn-1","status":"NOTIFIED"}`)
	}))
	defer srv.Close()

	c, _ := newTestClient(t, srv, testProfile())

	notification, err := c.GetPreDebitNotification(t.Context(), "pdn-1")
	if err != nil {
		t.Fatalf("GetPreDebitNotification: %v", err)
	}

	if gotMethod != http.MethodGet || gotPath != "/predebit-notify/pdn-1" {
		t.Errorf("expected GET /predebit-notify/pdn-1, got %s %s", gotMethod, gotPath)
	}

	if notification.Status != "NOTIFIED" {
		t.Errorf("unexpected notification: %+v", notification)
	}
}

func TestGetPreDebitNotification_ErrorIsWrapped(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = io.WriteString(w, `{"code":"UNAUTHORIZED","message":"nope"}`)
	}))
	defer srv.Close()

	c, _ := newTestClient(t, srv, testProfile())

	_, err := c.GetPreDebitNotification(t.Context(), "pdn-1")
	if err == nil || !errors.Is(err, ErrUnauthorized) ||
		!strings.Contains(err.Error(), "get pre-debit notification pdn-1") {
		t.Errorf("unexpected error: %v", err)
	}
}
