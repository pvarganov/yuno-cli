package api

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// checkoutBuilderBody is the checkout payload the fake API answers with.
const checkoutBuilderBody = `{"id":"ck-1","name":"Promo Checkout","description":"Seasonal promo",
	"status":"PUBLISHED","is_default":true,"created_at":"2026-09-16T10:00:00Z"}`

func TestCreateCheckout_PostsTheBody(t *testing.T) {
	var gotMethod, gotPath string

	var gotBody map[string]any

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod, gotPath = r.Method, r.URL.Path
		_ = json.NewDecoder(r.Body).Decode(&gotBody)

		w.WriteHeader(http.StatusCreated)
		_, _ = io.WriteString(w, checkoutBuilderBody)
	}))
	defer srv.Close()

	c, _ := newTestClient(t, srv, testProfile())

	checkout, err := c.CreateCheckout(t.Context(), map[string]any{"name": "Promo Checkout"})
	if err != nil {
		t.Fatalf("CreateCheckout: %v", err)
	}

	if gotMethod != http.MethodPost || gotPath != "/checkouts" {
		t.Errorf("expected POST /checkouts, got %s %s", gotMethod, gotPath)
	}

	if gotBody["name"] != "Promo Checkout" || checkout.ID != "ck-1" {
		t.Errorf("unexpected create: body %+v, checkout %+v", gotBody, checkout)
	}
}

func TestCreateCheckout_ErrorIsWrapped(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = io.WriteString(w, `{"code":"BAD_REQUEST","message":"duplicate name"}`)
	}))
	defer srv.Close()

	c, _ := newTestClient(t, srv, testProfile())

	if _, err := c.CreateCheckout(t.Context(), map[string]any{}); err == nil ||
		!strings.Contains(err.Error(), "create checkout") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestListCheckouts_PagesFromPageOne(t *testing.T) {
	var gotQueries []string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQueries = append(gotQueries, r.URL.RawQuery)

		if r.URL.Query().Get("page") == "1" {
			_, _ = io.WriteString(w, `{"data":[{"id":"ck-1"},{"id":"ck-2"}],
				"pagination":{"page":1,"size":2,"total":3}}`)

			return
		}

		_, _ = io.WriteString(w, `{"data":[{"id":"ck-3"}],"pagination":{"page":2,"size":2,"total":3}}`)
	}))
	defer srv.Close()

	c, _ := newTestClient(t, srv, testProfile())

	checkouts, err := c.ListCheckouts(t.Context(), nil, 0, 2)
	if err != nil {
		t.Fatalf("ListCheckouts: %v", err)
	}

	if len(checkouts) != 3 || checkouts[2].ID != "ck-3" {
		t.Errorf("unexpected checkouts: %+v", checkouts)
	}

	if len(gotQueries) != 2 || !strings.Contains(gotQueries[0], "page=1") ||
		!strings.Contains(gotQueries[0], "size=2") {
		t.Errorf("expected the first page to be page one, got %v", gotQueries)
	}
}

func TestListCheckouts_KeepsTheFilters(t *testing.T) {
	var gotQuery string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery

		_, _ = io.WriteString(w, `{"data":[],"pagination":{"page":1,"size":100,"total":0}}`)
	}))
	defer srv.Close()

	c, _ := newTestClient(t, srv, testProfile())

	if _, err := c.ListCheckouts(t.Context(), map[string][]string{"status": {"ARCHIVED"}}, 0, 0); err != nil {
		t.Fatalf("ListCheckouts: %v", err)
	}

	if !strings.Contains(gotQuery, "status=ARCHIVED") {
		t.Errorf("expected the status filter to survive, got %q", gotQuery)
	}
}

func TestListCheckouts_ErrorIsWrapped(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = io.WriteString(w, `{"code":"UNAUTHORIZED","message":"nope"}`)
	}))
	defer srv.Close()

	c, _ := newTestClient(t, srv, testProfile())

	if _, err := c.ListCheckouts(t.Context(), nil, 0, 0); err == nil ||
		!strings.Contains(err.Error(), "list checkouts") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestGetCheckout_EscapesTheCode(t *testing.T) {
	var gotMethod, gotPath string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod, gotPath = r.Method, r.URL.EscapedPath()

		_, _ = io.WriteString(w, checkoutBuilderBody)
	}))
	defer srv.Close()

	c, _ := newTestClient(t, srv, testProfile())

	checkout, err := c.GetCheckout(t.Context(), "ck 1")
	if err != nil {
		t.Fatalf("GetCheckout: %v", err)
	}

	if gotMethod != http.MethodGet || gotPath != "/checkouts/ck%201" {
		t.Errorf("expected GET /checkouts/ck%%201, got %s %s", gotMethod, gotPath)
	}

	if checkout.Name != "Promo Checkout" {
		t.Errorf("unexpected checkout: %+v", checkout)
	}
}

func TestGetCheckout_ErrorIsWrapped(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = io.WriteString(w, `{"code":"NOT_FOUND","message":"nope"}`)
	}))
	defer srv.Close()

	c, _ := newTestClient(t, srv, testProfile())

	if _, err := c.GetCheckout(t.Context(), "ck-1"); err == nil ||
		!strings.Contains(err.Error(), "get checkout ck-1") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestPublishCheckout_UsesPut(t *testing.T) {
	var gotMethod, gotPath string

	var gotBody map[string]any

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod, gotPath = r.Method, r.URL.Path
		_ = json.NewDecoder(r.Body).Decode(&gotBody)

		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	c, _ := newTestClient(t, srv, testProfile())

	data, err := c.PublishCheckout(t.Context(), "ck-1", map[string]any{"config": map[string]any{}})
	if err != nil {
		t.Fatalf("PublishCheckout: %v", err)
	}

	if gotMethod != http.MethodPut || gotPath != "/checkouts/ck-1" {
		t.Errorf("expected PUT /checkouts/ck-1, got %s %s", gotMethod, gotPath)
	}

	if _, ok := gotBody["config"]; !ok || len(data) != 0 {
		t.Errorf("unexpected publish: body %+v, response %s", gotBody, data)
	}
}

func TestPublishCheckout_ErrorIsWrapped(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = io.WriteString(w, `{"code":"BAD_REQUEST","message":"nope"}`)
	}))
	defer srv.Close()

	c, _ := newTestClient(t, srv, testProfile())

	if _, err := c.PublishCheckout(t.Context(), "ck-1", map[string]any{}); err == nil ||
		!strings.Contains(err.Error(), "publish checkout ck-1") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestUpdateCheckout_UsesPatch(t *testing.T) {
	var gotMethod, gotPath string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod, gotPath = r.Method, r.URL.Path

		_, _ = io.WriteString(w, checkoutBuilderBody)
	}))
	defer srv.Close()

	c, _ := newTestClient(t, srv, testProfile())

	if _, err := c.UpdateCheckout(t.Context(), "ck-1", map[string]any{"status": "PUBLISHED"}); err != nil {
		t.Fatalf("UpdateCheckout: %v", err)
	}

	if gotMethod != http.MethodPatch || gotPath != "/checkouts/ck-1" {
		t.Errorf("expected PATCH /checkouts/ck-1, got %s %s", gotMethod, gotPath)
	}
}

func TestUpdateCheckout_ErrorIsWrapped(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = io.WriteString(w, `{"code":"NOT_FOUND","message":"nope"}`)
	}))
	defer srv.Close()

	c, _ := newTestClient(t, srv, testProfile())

	if _, err := c.UpdateCheckout(t.Context(), "ck-1", map[string]any{}); err == nil ||
		!strings.Contains(err.Error(), "update checkout ck-1") {
		t.Errorf("unexpected error: %v", err)
	}
}
