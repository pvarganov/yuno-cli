package api

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// destinationBody is the destination payload the fake API answers with.
const destinationBody = `{"id":"d-1","account_id":"acc-1","hostname":"api.processor.com",
	"status":"ENABLED","purpose":"card charges","created_at":"2026-09-16T10:00:00Z"}`

func TestListPCIProxyDestinations_UnwrapsTheEnvelope(t *testing.T) {
	var gotMethod, gotPath string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod, gotPath = r.Method, r.URL.Path

		_, _ = io.WriteString(w, `{"destinations":[`+destinationBody+`]}`)
	}))
	defer srv.Close()

	c, _ := newTestClient(t, srv, testProfile())

	destinations, err := c.ListPCIProxyDestinations(t.Context())
	if err != nil {
		t.Fatalf("ListPCIProxyDestinations: %v", err)
	}

	if gotMethod != http.MethodGet || gotPath != "/pci-proxy/destinations" {
		t.Errorf("expected GET /pci-proxy/destinations, got %s %s", gotMethod, gotPath)
	}

	if len(destinations) != 1 || destinations[0].Hostname != "api.processor.com" {
		t.Errorf("unexpected destinations: %+v", destinations)
	}
}

func TestListPCIProxyDestinations_ErrorIsWrapped(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = io.WriteString(w, `{"code":"FORBIDDEN","message":"nope"}`)
	}))
	defer srv.Close()

	c, _ := newTestClient(t, srv, testProfile())

	if _, err := c.ListPCIProxyDestinations(t.Context()); err == nil ||
		!strings.Contains(err.Error(), "list pci proxy destinations") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestCreatePCIProxyDestination_PostsTheBody(t *testing.T) {
	var gotMethod, gotPath string

	var gotBody map[string]any

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod, gotPath = r.Method, r.URL.Path
		_ = json.NewDecoder(r.Body).Decode(&gotBody)

		w.WriteHeader(http.StatusCreated)
		_, _ = io.WriteString(w, destinationBody)
	}))
	defer srv.Close()

	c, _ := newTestClient(t, srv, testProfile())

	destination, err := c.CreatePCIProxyDestination(t.Context(), map[string]any{"hostname": "api.processor.com"})
	if err != nil {
		t.Fatalf("CreatePCIProxyDestination: %v", err)
	}

	if gotMethod != http.MethodPost || gotPath != "/pci-proxy/destinations" {
		t.Errorf("expected POST /pci-proxy/destinations, got %s %s", gotMethod, gotPath)
	}

	if gotBody["hostname"] != "api.processor.com" || destination.ID != "d-1" {
		t.Errorf("unexpected create: body %+v, destination %+v", gotBody, destination)
	}
}

func TestCreatePCIProxyDestination_ErrorIsWrapped(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = io.WriteString(w, `{"code":"INVALID_REQUEST","message":"nope"}`)
	}))
	defer srv.Close()

	c, _ := newTestClient(t, srv, testProfile())

	if _, err := c.CreatePCIProxyDestination(t.Context(), map[string]any{}); err == nil ||
		!strings.Contains(err.Error(), "create pci proxy destination") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestSwitchPCIProxyDestination_HitsTheActionPath(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		call func(c *Client) error
		path string
	}{
		{
			name: "enable",
			call: func(c *Client) error {
				_, err := c.EnablePCIProxyDestination(t.Context(), "d 1")

				return err
			},
			path: "/pci-proxy/destinations/d%201/enable",
		},
		{
			name: "disable",
			call: func(c *Client) error {
				_, err := c.DisablePCIProxyDestination(t.Context(), "d 1")

				return err
			},
			path: "/pci-proxy/destinations/d%201/disable",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			var gotMethod, gotPath string

			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				gotMethod, gotPath = r.Method, r.URL.EscapedPath()

				_, _ = io.WriteString(w, destinationBody)
			}))
			defer srv.Close()

			c, _ := newTestClient(t, srv, testProfile())

			if err := tc.call(c); err != nil {
				t.Fatalf("%s: %v", tc.name, err)
			}

			if gotMethod != http.MethodPost || gotPath != tc.path {
				t.Errorf("expected POST %s, got %s %s", tc.path, gotMethod, gotPath)
			}
		})
	}
}

func TestEnablePCIProxyDestination_ErrorIsWrapped(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = io.WriteString(w, `{"code":"NOT_FOUND","message":"nope"}`)
	}))
	defer srv.Close()

	c, _ := newTestClient(t, srv, testProfile())

	if _, err := c.EnablePCIProxyDestination(t.Context(), "d-1"); err == nil ||
		!strings.Contains(err.Error(), "enable pci proxy destination d-1") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestDeletePCIProxyDestination_AcceptsAnEmptyBody(t *testing.T) {
	var gotMethod, gotPath string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod, gotPath = r.Method, r.URL.Path

		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	c, _ := newTestClient(t, srv, testProfile())

	data, err := c.DeletePCIProxyDestination(t.Context(), "d-1")
	if err != nil {
		t.Fatalf("DeletePCIProxyDestination: %v", err)
	}

	if gotMethod != http.MethodDelete || gotPath != "/pci-proxy/destinations/d-1" {
		t.Errorf("expected DELETE /pci-proxy/destinations/d-1, got %s %s", gotMethod, gotPath)
	}

	if len(data) != 0 {
		t.Errorf("expected an empty body, got %s", data)
	}
}

func TestDeletePCIProxyDestination_ErrorIsWrapped(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = io.WriteString(w, `{"code":"NOT_FOUND","message":"nope"}`)
	}))
	defer srv.Close()

	c, _ := newTestClient(t, srv, testProfile())

	if _, err := c.DeletePCIProxyDestination(t.Context(), "d-1"); err == nil ||
		!strings.Contains(err.Error(), "delete pci proxy destination d-1") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestForwardPCIProxy_SendsTheProxyHeaders(t *testing.T) {
	var gotMethod, gotPath string

	var gotHeader http.Header

	var gotBody map[string]any

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod, gotPath = r.Method, r.URL.Path
		gotHeader = r.Header.Clone()
		_ = json.NewDecoder(r.Body).Decode(&gotBody)

		_, _ = io.WriteString(w, `{"charge_id":"ch_1"}`)
	}))
	defer srv.Close()

	c, _ := newTestClient(t, srv, testProfile())

	headers := map[string]string{
		HeaderProxyDestinationURL: "https://api.processor.com/charges",
		HeaderProxyTimeout:        "45",
		HeaderProxyAuth:           "DLOCAL_HMAC",
		HeaderProxyAccountID:      "acc-1",
	}

	data, err := c.ForwardPCIProxy(t.Context(), http.MethodPost, map[string]any{"amount": 2500}, headers)
	if err != nil {
		t.Fatalf("ForwardPCIProxy: %v", err)
	}

	if gotMethod != http.MethodPost || gotPath != "/pci-proxy/forward" {
		t.Errorf("expected POST /pci-proxy/forward, got %s %s", gotMethod, gotPath)
	}

	for name, want := range headers {
		if got := gotHeader.Get(name); got != want {
			t.Errorf("header %s: expected %q, got %q", name, want, got)
		}
	}

	if gotBody["amount"] != float64(2500) || !strings.Contains(string(data), "ch_1") {
		t.Errorf("unexpected forward: body %+v, response %s", gotBody, data)
	}
}

func TestForwardPCIProxy_RequiresTheDestinationURL(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		t.Error("the proxy must not be called without a destination url")
	}))
	defer srv.Close()

	c, _ := newTestClient(t, srv, testProfile())

	_, err := c.ForwardPCIProxy(t.Context(), http.MethodPost, map[string]any{}, nil)
	if err == nil || !strings.Contains(err.Error(), HeaderProxyDestinationURL) {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestForwardPCIProxy_ErrorIsWrapped(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = io.WriteString(w, `{"code":"INVALID_REQUEST","message":"nope"}`)
	}))
	defer srv.Close()

	c, _ := newTestClient(t, srv, testProfile())

	_, err := c.ForwardPCIProxy(t.Context(), http.MethodPost, map[string]any{},
		map[string]string{HeaderProxyDestinationURL: "https://api.processor.com/charges"})
	if err == nil || !strings.Contains(err.Error(), "forward pci proxy request") {
		t.Errorf("unexpected error: %v", err)
	}
}
