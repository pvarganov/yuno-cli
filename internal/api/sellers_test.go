package api

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// sellerBody is the seller payload the fake API answers with.
const sellerBody = `{"seller_id":"se-1","merchant_seller_id":"shop-1","name":"Acme Shop",
	"email":"shop@acme.com","country":"US","document":{"document_type":"EIN","document_number":"123"},
	"created_at":"2026-09-16T10:00:00Z"}`

func TestCreateSeller_PostsTheBody(t *testing.T) {
	var gotMethod, gotPath string

	var gotBody map[string]any

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod, gotPath = r.Method, r.URL.Path
		_ = json.NewDecoder(r.Body).Decode(&gotBody)

		w.WriteHeader(http.StatusCreated)
		_, _ = io.WriteString(w, sellerBody)
	}))
	defer srv.Close()

	c, _ := newTestClient(t, srv, testProfile())

	seller, err := c.CreateSeller(t.Context(), map[string]any{"merchant_seller_id": "shop-1"})
	if err != nil {
		t.Fatalf("CreateSeller: %v", err)
	}

	if gotMethod != http.MethodPost || gotPath != "/sellers" {
		t.Errorf("expected POST /sellers, got %s %s", gotMethod, gotPath)
	}

	if gotBody["merchant_seller_id"] != "shop-1" || seller.SellerID != "se-1" {
		t.Errorf("unexpected create: body %+v, seller %+v", gotBody, seller)
	}
}

func TestCreateSeller_ErrorIsWrapped(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusConflict)
		_, _ = io.WriteString(w, `{"code":"CONFLICT","message":"exists"}`)
	}))
	defer srv.Close()

	c, _ := newTestClient(t, srv, testProfile())

	if _, err := c.CreateSeller(t.Context(), map[string]any{}); err == nil ||
		!strings.Contains(err.Error(), "create seller") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestGetSeller_EscapesTheID(t *testing.T) {
	var gotMethod, gotPath string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod, gotPath = r.Method, r.URL.EscapedPath()

		_, _ = io.WriteString(w, sellerBody)
	}))
	defer srv.Close()

	c, _ := newTestClient(t, srv, testProfile())

	seller, err := c.GetSeller(t.Context(), "shop 1")
	if err != nil {
		t.Fatalf("GetSeller: %v", err)
	}

	if gotMethod != http.MethodGet || gotPath != "/sellers/shop%201" {
		t.Errorf("expected GET /sellers/shop%%201, got %s %s", gotMethod, gotPath)
	}

	if seller.MerchantSellerID != "shop-1" {
		t.Errorf("unexpected seller: %+v", seller)
	}
}

func TestGetSeller_ErrorIsWrapped(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = io.WriteString(w, `{"code":"NOT_FOUND","message":"nope"}`)
	}))
	defer srv.Close()

	c, _ := newTestClient(t, srv, testProfile())

	if _, err := c.GetSeller(t.Context(), "shop-1"); err == nil ||
		!strings.Contains(err.Error(), "get seller shop-1") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestUpdateSeller_UsesPut(t *testing.T) {
	var gotMethod, gotPath string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod, gotPath = r.Method, r.URL.Path

		_, _ = io.WriteString(w, sellerBody)
	}))
	defer srv.Close()

	c, _ := newTestClient(t, srv, testProfile())

	if _, err := c.UpdateSeller(t.Context(), "shop-1", map[string]any{"name": "Acme EU"}); err != nil {
		t.Fatalf("UpdateSeller: %v", err)
	}

	if gotMethod != http.MethodPut || gotPath != "/sellers/shop-1" {
		t.Errorf("expected PUT /sellers/shop-1, got %s %s", gotMethod, gotPath)
	}
}

func TestUpdateSeller_ErrorIsWrapped(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = io.WriteString(w, `{"code":"BAD_REQUEST","message":"nope"}`)
	}))
	defer srv.Close()

	c, _ := newTestClient(t, srv, testProfile())

	if _, err := c.UpdateSeller(t.Context(), "shop-1", map[string]any{}); err == nil ||
		!strings.Contains(err.Error(), "update seller shop-1") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestDeleteSeller_AcceptsAnEmptyBody(t *testing.T) {
	var gotMethod, gotPath string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod, gotPath = r.Method, r.URL.Path

		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	c, _ := newTestClient(t, srv, testProfile())

	data, err := c.DeleteSeller(t.Context(), "shop-1")
	if err != nil {
		t.Fatalf("DeleteSeller: %v", err)
	}

	if gotMethod != http.MethodDelete || gotPath != "/sellers/shop-1" {
		t.Errorf("expected DELETE /sellers/shop-1, got %s %s", gotMethod, gotPath)
	}

	if len(data) != 0 {
		t.Errorf("expected an empty body, got %s", data)
	}
}

func TestDeleteSeller_ErrorIsWrapped(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = io.WriteString(w, `{"code":"NOT_FOUND","message":"nope"}`)
	}))
	defer srv.Close()

	c, _ := newTestClient(t, srv, testProfile())

	if _, err := c.DeleteSeller(t.Context(), "shop-1"); err == nil ||
		!strings.Contains(err.Error(), "delete seller shop-1") {
		t.Errorf("unexpected error: %v", err)
	}
}
