package api

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestListRoutings_DecodesTheList(t *testing.T) {
	var gotQuery string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		_, _ = w.Write([]byte(`[{"id":"r-1","default_route":{"steps":[{"index":1,"provider_id":"STRIPE"}]}}]`))
	}))
	defer srv.Close()

	c, _ := newTestClient(t, srv, testProfile())

	routings, err := c.ListRoutings(t.Context(), "acc-1", "CARD")
	if err != nil {
		t.Fatalf("ListRoutings: %v", err)
	}

	if gotQuery != "account_id=acc-1&payment_method=CARD" {
		t.Errorf("unexpected query: %s", gotQuery)
	}

	if len(routings) != 1 || routings[0].DefaultRoute.Chain() != "STRIPE" {
		t.Errorf("unexpected routings: %+v", routings)
	}
}

func TestListRoutings_OmitsEmptyPaymentMethod(t *testing.T) {
	var gotQuery string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		_, _ = w.Write([]byte(`[]`))
	}))
	defer srv.Close()

	c, _ := newTestClient(t, srv, testProfile())

	if _, err := c.ListRoutings(t.Context(), "acc-1", ""); err != nil {
		t.Fatalf("ListRoutings: %v", err)
	}

	if gotQuery != "account_id=acc-1" {
		t.Errorf("unexpected query: %s", gotQuery)
	}
}

func TestListRoutings_ForbiddenIsNotRetried(t *testing.T) {
	calls := 0

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls++

		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"code":"INSUFFICIENT_SCOPE","message":"missing scope"}`))
	}))
	defer srv.Close()

	c, _ := newTestClient(t, srv, testProfile())

	_, err := c.ListRoutings(t.Context(), "acc-1", "")
	if err == nil {
		t.Fatal("expected a 403 error")
	}

	if !errors.Is(err, ErrForbidden) {
		t.Errorf("expected ErrForbidden, got %v", err)
	}

	if calls != 1 {
		t.Errorf("expected no retry on 403, got %d calls", calls)
	}

	if !strings.Contains(err.Error(), "list routings") {
		t.Errorf("expected the error to carry context, got %v", err)
	}
}

func TestRoutingWrites_UseTheRightMethodAndPath(t *testing.T) {
	var gotMethod, gotPath, gotBody string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path

		buf := make([]byte, r.ContentLength)
		_, _ = r.Body.Read(buf)
		gotBody = string(buf)

		_, _ = w.Write([]byte(`{"id":"r-1"}`))
	}))
	defer srv.Close()

	c, _ := newTestClient(t, srv, testProfile())

	tests := []struct {
		name       string
		call       func() error
		wantMethod string
		wantPath   string
	}{
		{
			name:       "create",
			call:       func() error { _, err := c.CreateRouting(t.Context(), map[string]any{"name": "n"}); return err },
			wantMethod: http.MethodPost,
			wantPath:   "/routing",
		},
		{
			name:       "update",
			call:       func() error { _, err := c.UpdateRouting(t.Context(), "r-1", map[string]any{"name": "n"}); return err },
			wantMethod: http.MethodPatch,
			wantPath:   "/routing/r-1",
		},
		{
			name:       "get",
			call:       func() error { _, err := c.GetRouting(t.Context(), "r-1"); return err },
			wantMethod: http.MethodGet,
			wantPath:   "/routing/r-1",
		},
		{
			name:       "recommend",
			call:       func() error { _, err := c.RecommendRouting(t.Context(), map[string]any{"account_id": "a"}); return err },
			wantMethod: http.MethodPost,
			wantPath:   "/routing/recommendations",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			gotBody = ""

			if err := tc.call(); err != nil {
				t.Fatalf("%s: %v", tc.name, err)
			}

			if gotMethod != tc.wantMethod || gotPath != tc.wantPath {
				t.Errorf("got %s %s, want %s %s", gotMethod, gotPath, tc.wantMethod, tc.wantPath)
			}

			if tc.wantMethod != http.MethodGet && gotBody == "" {
				t.Errorf("expected a request body for %s", tc.name)
			}
		})
	}
}

func TestConnectionCalls_UseTheRightMethodAndPath(t *testing.T) {
	var gotMethod, gotPath string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path

		_, _ = w.Write([]byte(`{"connection_id":"c-1","payment_methods":["CARD"]}`))
	}))
	defer srv.Close()

	c, _ := newTestClient(t, srv, testProfile())

	tests := []struct {
		name       string
		call       func() error
		wantMethod string
		wantPath   string
	}{
		{
			name:       "catalog",
			call:       func() error { _, err := c.GetProviderCatalog(t.Context(), "STRIPE"); return err },
			wantMethod: http.MethodGet,
			wantPath:   "/connections/catalog/STRIPE",
		},
		{
			name:       "get",
			call:       func() error { _, err := c.GetConnection(t.Context(), "c-1"); return err },
			wantMethod: http.MethodGet,
			wantPath:   "/connections/c-1",
		},
		{
			name: "create",
			call: func() error {
				_, err := c.CreateConnection(t.Context(), map[string]any{"provider_id": "ADYEN"})
				return err
			},
			wantMethod: http.MethodPost,
			wantPath:   "/connections",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if err := tc.call(); err != nil {
				t.Fatalf("%s: %v", tc.name, err)
			}

			if gotMethod != tc.wantMethod || gotPath != tc.wantPath {
				t.Errorf("got %s %s, want %s %s", gotMethod, gotPath, tc.wantMethod, tc.wantPath)
			}
		})
	}
}

func TestGetConnection_NotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"code":"NOT_FOUND","message":"connection not found"}`))
	}))
	defer srv.Close()

	c, _ := newTestClient(t, srv, testProfile())

	_, err := c.GetConnection(t.Context(), "missing")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}

	if !strings.Contains(err.Error(), "get connection missing") {
		t.Errorf("expected the error to carry context, got %v", err)
	}
}
