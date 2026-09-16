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

// webhookBody is the webhook payload the fake API answers with.
const webhookBody = `{"id":"wh-1","account_id":"acc-1","name":"listener","state":"ACTIVE",
	"url":"https://api.acme.com/yuno","api_key":"***","payment_triggers":["AUTHORIZE"],
	"created_at":"2026-09-16T10:00:00Z"}`

func TestListWebhooks_SendsAccountAndState(t *testing.T) {
	var gotMethod, gotPath, gotQuery string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod, gotPath, gotQuery = r.Method, r.URL.Path, r.URL.RawQuery

		_, _ = io.WriteString(w, "["+webhookBody+"]")
	}))
	defer srv.Close()

	c, _ := newTestClient(t, srv, testProfile())

	webhooks, err := c.ListWebhooks(t.Context(), "acc-1", "ACTIVE")
	if err != nil {
		t.Fatalf("ListWebhooks: %v", err)
	}

	if gotMethod != http.MethodGet || gotPath != "/webhooks" {
		t.Errorf("expected GET /webhooks, got %s %s", gotMethod, gotPath)
	}

	if gotQuery != "account_id=acc-1&state=ACTIVE" {
		t.Errorf("unexpected query: %s", gotQuery)
	}

	if len(webhooks) != 1 || webhooks[0].ID != "wh-1" {
		t.Errorf("unexpected webhooks: %+v", webhooks)
	}
}

func TestListWebhooks_OmitsAnEmptyState(t *testing.T) {
	var gotQuery string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery

		_, _ = io.WriteString(w, `[]`)
	}))
	defer srv.Close()

	c, _ := newTestClient(t, srv, testProfile())

	if _, err := c.ListWebhooks(t.Context(), "acc-1", ""); err != nil {
		t.Fatalf("ListWebhooks: %v", err)
	}

	if gotQuery != "account_id=acc-1" {
		t.Errorf("unexpected query: %s", gotQuery)
	}
}

func TestListWebhooks_ErrorIsWrapped(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = io.WriteString(w, `{"code":"INSUFFICIENT_SCOPE","message":"nope"}`)
	}))
	defer srv.Close()

	c, _ := newTestClient(t, srv, testProfile())

	_, err := c.ListWebhooks(t.Context(), "acc-1", "")
	if err == nil || !errors.Is(err, ErrForbidden) || !strings.Contains(err.Error(), "list webhooks of account acc-1") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestGetWebhook_SendsTheAccountID(t *testing.T) {
	var gotPath, gotQuery string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath, gotQuery = r.URL.Path, r.URL.RawQuery

		_, _ = io.WriteString(w, webhookBody)
	}))
	defer srv.Close()

	c, _ := newTestClient(t, srv, testProfile())

	webhook, err := c.GetWebhook(t.Context(), "wh-1", "acc-1")
	if err != nil {
		t.Fatalf("GetWebhook: %v", err)
	}

	if gotPath != "/webhooks/wh-1" || gotQuery != "account_id=acc-1" {
		t.Errorf("unexpected request: %s?%s", gotPath, gotQuery)
	}

	if webhook.State != "ACTIVE" {
		t.Errorf("unexpected webhook: %+v", webhook)
	}
}

func TestGetWebhook_ErrorIsWrapped(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = io.WriteString(w, `{"code":"NOT_FOUND","message":"nope"}`)
	}))
	defer srv.Close()

	c, _ := newTestClient(t, srv, testProfile())

	if _, err := c.GetWebhook(t.Context(), "wh-1", "acc-1"); err == nil ||
		!strings.Contains(err.Error(), "get webhook wh-1") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestCreateWebhook_PostsTheBody(t *testing.T) {
	var gotMethod, gotPath string

	var gotBody map[string]any

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod, gotPath = r.Method, r.URL.Path
		_ = json.NewDecoder(r.Body).Decode(&gotBody)

		w.WriteHeader(http.StatusCreated)
		_, _ = io.WriteString(w, webhookBody)
	}))
	defer srv.Close()

	c, _ := newTestClient(t, srv, testProfile())

	webhook, err := c.CreateWebhook(t.Context(), map[string]any{"account_id": "acc-1", "name": "listener"})
	if err != nil {
		t.Fatalf("CreateWebhook: %v", err)
	}

	if gotMethod != http.MethodPost || gotPath != "/webhooks" {
		t.Errorf("expected POST /webhooks, got %s %s", gotMethod, gotPath)
	}

	if gotBody["name"] != "listener" {
		t.Errorf("unexpected body: %+v", gotBody)
	}

	if webhook.ID != "wh-1" {
		t.Errorf("unexpected webhook: %+v", webhook)
	}
}

func TestCreateWebhook_ErrorIsWrapped(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusConflict)
		_, _ = io.WriteString(w, `{"code":"CONFLICT","message":"duplicate name"}`)
	}))
	defer srv.Close()

	c, _ := newTestClient(t, srv, testProfile())

	if _, err := c.CreateWebhook(t.Context(), map[string]any{}); err == nil ||
		!strings.Contains(err.Error(), "create webhook") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestUpdateWebhook_PatchesTheIDPath(t *testing.T) {
	var gotMethod, gotPath string

	var gotBody map[string]any

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod, gotPath = r.Method, r.URL.Path
		_ = json.NewDecoder(r.Body).Decode(&gotBody)

		_, _ = io.WriteString(w, webhookBody)
	}))
	defer srv.Close()

	c, _ := newTestClient(t, srv, testProfile())

	if _, err := c.UpdateWebhook(t.Context(), "wh-1",
		map[string]any{"account_id": "acc-1", "state": "INACTIVE"}); err != nil {
		t.Fatalf("UpdateWebhook: %v", err)
	}

	if gotMethod != http.MethodPatch || gotPath != "/webhooks/wh-1" {
		t.Errorf("expected PATCH /webhooks/wh-1, got %s %s", gotMethod, gotPath)
	}

	if gotBody["state"] != "INACTIVE" {
		t.Errorf("unexpected body: %+v", gotBody)
	}
}

func TestUpdateWebhook_ErrorIsWrapped(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = io.WriteString(w, `{"code":"BAD_REQUEST","message":"nope"}`)
	}))
	defer srv.Close()

	c, _ := newTestClient(t, srv, testProfile())

	if _, err := c.UpdateWebhook(t.Context(), "wh-1", map[string]any{}); err == nil ||
		!strings.Contains(err.Error(), "update webhook wh-1") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestDeleteWebhook_SendsTheAccountID(t *testing.T) {
	var gotMethod, gotPath, gotQuery string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod, gotPath, gotQuery = r.Method, r.URL.Path, r.URL.RawQuery

		_, _ = io.WriteString(w, `{"deleted":true}`)
	}))
	defer srv.Close()

	c, _ := newTestClient(t, srv, testProfile())

	data, err := c.DeleteWebhook(t.Context(), "wh-1", "acc-1")
	if err != nil {
		t.Fatalf("DeleteWebhook: %v", err)
	}

	if gotMethod != http.MethodDelete || gotPath != "/webhooks/wh-1" || gotQuery != "account_id=acc-1" {
		t.Errorf("unexpected request: %s %s?%s", gotMethod, gotPath, gotQuery)
	}

	if !strings.Contains(string(data), "deleted") {
		t.Errorf("unexpected response: %s", data)
	}
}

func TestDeleteWebhook_ErrorIsWrapped(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = io.WriteString(w, `{"code":"NOT_FOUND","message":"nope"}`)
	}))
	defer srv.Close()

	c, _ := newTestClient(t, srv, testProfile())

	if _, err := c.DeleteWebhook(t.Context(), "wh-1", "acc-1"); err == nil ||
		!strings.Contains(err.Error(), "delete webhook wh-1") {
		t.Errorf("unexpected error: %v", err)
	}
}
