package api

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCreateTransfer_PostsTheBody(t *testing.T) {
	var gotMethod, gotPath, gotBody string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		gotMethod, gotPath, gotBody = r.Method, r.URL.Path, string(body)

		_, _ = io.WriteString(w, `{"id":"tr-1","status":"CREATED","recipient_id":"rec-1"}`)
	}))
	defer srv.Close()

	c, _ := newTestClient(t, srv, testProfile())

	transfer, err := c.CreateTransfer(t.Context(), map[string]any{"recipient_id": "rec-1"})
	if err != nil {
		t.Fatalf("CreateTransfer: %v", err)
	}

	if gotMethod != http.MethodPost || gotPath != "/split-marketplace/transfers" {
		t.Errorf("expected POST /split-marketplace/transfers, got %s %s", gotMethod, gotPath)
	}

	if !strings.Contains(gotBody, `"recipient_id":"rec-1"`) {
		t.Errorf("unexpected body: %s", gotBody)
	}

	if transfer.ID != "tr-1" {
		t.Errorf("unexpected transfer: %+v", transfer)
	}
}

func TestCreateTransfer_ErrorIsWrapped(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = io.WriteString(w, `{"code":"INSUFFICIENT_SCOPE","message":"nope"}`)
	}))
	defer srv.Close()

	c, _ := newTestClient(t, srv, testProfile())

	_, err := c.CreateTransfer(t.Context(), map[string]any{})
	if err == nil || !errors.Is(err, ErrForbidden) || !strings.Contains(err.Error(), "create transfer") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestGetTransfer_HitsTheIDPath(t *testing.T) {
	var gotMethod, gotPath string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod, gotPath = r.Method, r.URL.Path

		_, _ = io.WriteString(w, `{"id":"tr-1","status":"SUCCEEDED","provider":"NUVEI"}`)
	}))
	defer srv.Close()

	c, _ := newTestClient(t, srv, testProfile())

	transfer, err := c.GetTransfer(t.Context(), "tr-1")
	if err != nil {
		t.Fatalf("GetTransfer: %v", err)
	}

	if gotMethod != http.MethodGet || gotPath != "/split-marketplace/transfers/tr-1" {
		t.Errorf("expected GET /split-marketplace/transfers/tr-1, got %s %s", gotMethod, gotPath)
	}

	if transfer.Provider == nil || *transfer.Provider != "NUVEI" {
		t.Errorf("unexpected transfer: %+v", transfer)
	}
}

func TestGetTransfer_ErrorIsWrapped(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = io.WriteString(w, `{"code":"NOT_FOUND","message":"gone"}`)
	}))
	defer srv.Close()

	c, _ := newTestClient(t, srv, testProfile())

	_, err := c.GetTransfer(t.Context(), "tr-1")
	if err == nil || !errors.Is(err, ErrNotFound) || !strings.Contains(err.Error(), "get transfer tr-1") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestReverseTransfer_PostsToTheReversePath(t *testing.T) {
	var gotMethod, gotPath, gotBody string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		gotMethod, gotPath, gotBody = r.Method, r.URL.Path, string(body)

		_, _ = io.WriteString(w, `{"id":"tr-1","status":"REVERSED"}`)
	}))
	defer srv.Close()

	c, _ := newTestClient(t, srv, testProfile())

	transfer, err := c.ReverseTransfer(t.Context(), "tr-1", map[string]any{"reason": "DUPLICATE"})
	if err != nil {
		t.Fatalf("ReverseTransfer: %v", err)
	}

	if gotMethod != http.MethodPost || gotPath != "/split-marketplace/transfers/tr-1/reverse" {
		t.Errorf("expected POST /split-marketplace/transfers/tr-1/reverse, got %s %s", gotMethod, gotPath)
	}

	if !strings.Contains(gotBody, `"reason":"DUPLICATE"`) {
		t.Errorf("unexpected body: %s", gotBody)
	}

	if transfer.Status != "REVERSED" {
		t.Errorf("unexpected transfer: %+v", transfer)
	}
}

func TestReverseTransfer_ErrorIsWrapped(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = io.WriteString(w, `{"code":"VALIDATION_ERROR","message":"bad"}`)
	}))
	defer srv.Close()

	c, _ := newTestClient(t, srv, testProfile())

	_, err := c.ReverseTransfer(t.Context(), "tr-1", nil)
	if err == nil || !strings.Contains(err.Error(), "reverse transfer tr-1") {
		t.Errorf("unexpected error: %v", err)
	}
}
