package api

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCreateThreeDSecureSetup_PostsTheBody(t *testing.T) {
	var gotMethod, gotPath, gotBody string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		gotMethod, gotPath, gotBody = r.Method, r.URL.Path, string(body)

		w.WriteHeader(http.StatusCreated)
		_, _ = io.WriteString(w, `{"three_d_secure_setup_id":"tds-1","type":"BROWSER",
			"device_fingerprints":[{"provider_id":"CYBERSOURCE","id":"fp-1"}]}`)
	}))
	defer srv.Close()

	c, _ := newTestClient(t, srv, testProfile())

	setup, err := c.CreateThreeDSecureSetup(t.Context(), map[string]any{"type": "BROWSER"})
	if err != nil {
		t.Fatalf("CreateThreeDSecureSetup: %v", err)
	}

	if gotMethod != http.MethodPost || gotPath != "/three-d-secure/setups" {
		t.Errorf("expected POST /three-d-secure/setups, got %s %s", gotMethod, gotPath)
	}

	if !strings.Contains(gotBody, `"type":"BROWSER"`) {
		t.Errorf("unexpected body: %s", gotBody)
	}

	if setup.ID != "tds-1" || len(setup.DeviceFingerprints) != 1 {
		t.Errorf("unexpected setup: %+v", setup)
	}
}

func TestCreateThreeDSecureSetup_ErrorIsWrapped(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = io.WriteString(w, `{"code":"VALIDATION_ERROR","message":"bad"}`)
	}))
	defer srv.Close()

	c, _ := newTestClient(t, srv, testProfile())

	_, err := c.CreateThreeDSecureSetup(t.Context(), map[string]any{})
	if err == nil || !strings.Contains(err.Error(), "create three-d-secure setup") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestCreateDryRunProviderEvent_PostsTheBody(t *testing.T) {
	var gotMethod, gotPath, gotBody string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		gotMethod, gotPath, gotBody = r.Method, r.URL.Path, string(body)

		w.WriteHeader(http.StatusCreated)
		_, _ = io.WriteString(w, `{"id":"dr-1","status":"PROCESSED","payment_id":null}`)
	}))
	defer srv.Close()

	c, _ := newTestClient(t, srv, testProfile())

	event, err := c.CreateDryRunProviderEvent(t.Context(), map[string]any{"merchant_reference": "ref-1"})
	if err != nil {
		t.Fatalf("CreateDryRunProviderEvent: %v", err)
	}

	if gotMethod != http.MethodPost || gotPath != "/dry-run/provider-events" {
		t.Errorf("expected POST /dry-run/provider-events, got %s %s", gotMethod, gotPath)
	}

	if !strings.Contains(gotBody, `"merchant_reference":"ref-1"`) {
		t.Errorf("unexpected body: %s", gotBody)
	}

	if event.ID != "dr-1" || event.PaymentID != nil {
		t.Errorf("unexpected event: %+v", event)
	}
}

func TestCreateDryRunProviderEvent_ErrorIsWrapped(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = io.WriteString(w, `{"code":"INSUFFICIENT_SCOPE","message":"nope"}`)
	}))
	defer srv.Close()

	c, _ := newTestClient(t, srv, testProfile())

	_, err := c.CreateDryRunProviderEvent(t.Context(), map[string]any{})
	if err == nil || !errors.Is(err, ErrForbidden) ||
		!strings.Contains(err.Error(), "create dry run provider event") {
		t.Errorf("unexpected error: %v", err)
	}
}
