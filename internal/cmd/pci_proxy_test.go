package cmd_test

import (
	"net/http"
	"strings"
	"testing"
)

// destinationResponse is the destination payload the fake API returns.
const destinationResponse = `{"id":"d-1","account_id":"acc-1","hostname":"api.processor.com",
	"status":"ENABLED","purpose":"card charges","created_at":"2026-09-16T10:00:00Z"}`

func TestPCIProxyDestinationList_PrintsTheAllowlist(t *testing.T) {
	isolateConfig(t)
	seedCredentials(t)

	got := startAPI(t, http.StatusOK, `{"destinations":[`+destinationResponse+`]}`)

	out, err := runCLI(t, "", "pci-proxy", "destination", "list")
	if err != nil {
		t.Fatalf("pci-proxy destination list failed: %v (%s)", err, out)
	}

	if got.Method != http.MethodGet || got.Path != "/v1/pci-proxy/destinations" {
		t.Errorf("expected GET /v1/pci-proxy/destinations, got %s %s", got.Method, got.Path)
	}

	for _, want := range []string{"d-1", "api.processor.com", "ENABLED"} {
		if !strings.Contains(out, want) {
			t.Errorf("expected table output to contain %q, got:\n%s", want, out)
		}
	}
}

func TestPCIProxyDestinationCreate_PostsTheBody(t *testing.T) {
	isolateConfig(t)
	seedCredentials(t)

	got := startAPI(t, http.StatusCreated, destinationResponse)

	out, err := runCLI(t, "", "pci-proxy", "destination", "create", "--yes",
		"--hostname", "api.processor.com", "--purpose", "card charges", "--account-id", "acc-1")
	if err != nil {
		t.Fatalf("pci-proxy destination create failed: %v (%s)", err, out)
	}

	if got.Method != http.MethodPost || got.Path != "/v1/pci-proxy/destinations" {
		t.Errorf("expected POST /v1/pci-proxy/destinations, got %s %s", got.Method, got.Path)
	}

	body := decodeBody(t, got.Body)
	if body["hostname"] != "api.processor.com" || body["purpose"] != "card charges" {
		t.Errorf("unexpected body: %s", got.Body)
	}
}

func TestPCIProxyDestinationEnableDisable_HitTheActionPaths(t *testing.T) {
	tests := []struct {
		action string
		path   string
	}{
		{action: "enable", path: "/v1/pci-proxy/destinations/d-1/enable"},
		{action: "disable", path: "/v1/pci-proxy/destinations/d-1/disable"},
	}

	for _, tc := range tests {
		t.Run(tc.action, func(t *testing.T) {
			isolateConfig(t)
			seedCredentials(t)

			got := startAPI(t, http.StatusOK, destinationResponse)

			out, err := runCLI(t, "", "pci-proxy", "destination", tc.action, "d-1", "--yes")
			if err != nil {
				t.Fatalf("pci-proxy destination %s failed: %v (%s)", tc.action, err, out)
			}

			if got.Method != http.MethodPost || got.Path != tc.path {
				t.Errorf("expected POST %s, got %s %s", tc.path, got.Method, got.Path)
			}
		})
	}
}

func TestPCIProxyDestinationDelete_UsesDelete(t *testing.T) {
	isolateConfig(t)
	seedCredentials(t)

	got := startAPI(t, http.StatusNoContent, "")

	if _, err := runCLI(t, "", "pci-proxy", "destination", "delete", "d-1", "--yes"); err != nil {
		t.Fatalf("pci-proxy destination delete failed: %v", err)
	}

	if got.Method != http.MethodDelete || got.Path != "/v1/pci-proxy/destinations/d-1" {
		t.Errorf("expected DELETE /v1/pci-proxy/destinations/d-1, got %s %s", got.Method, got.Path)
	}
}

func TestPCIProxyForward_SendsTheProxyHeaders(t *testing.T) {
	isolateConfig(t)
	seedCredentials(t)

	got := startAPI(t, http.StatusOK, `{"charge_id":"ch_1"}`)

	out, err := runCLI(t, "", "pci-proxy", "forward", "--yes",
		"--destination-url", "https://api.processor.com/charges",
		"--timeout", "45", "--proxy-auth", "DLOCAL_HMAC", "--proxy-auth-secret-key", "shh",
		"--account-id", "acc-1", "--header", "X-Login: merchant",
		"--data", `{"amount":2500}`)
	if err != nil {
		t.Fatalf("pci-proxy forward failed: %v (%s)", err, out)
	}

	if got.Method != http.MethodPost || got.Path != "/v1/pci-proxy/forward" {
		t.Errorf("expected POST /v1/pci-proxy/forward, got %s %s", got.Method, got.Path)
	}

	wantHeaders := map[string]string{
		"yuno-proxy-destination-url": "https://api.processor.com/charges",
		"yuno-proxy-timeout":         "45",
		"yuno-proxy-auth":            "DLOCAL_HMAC",
		"yuno-proxy-auth-secret-key": "shh",
		"yuno-account-id":            "acc-1",
		"X-Login":                    "merchant",
	}

	for name, want := range wantHeaders {
		if value := got.Header.Get(name); value != want {
			t.Errorf("header %s: expected %q, got %q", name, want, value)
		}
	}

	if body := decodeBody(t, got.Body); body["amount"] != float64(2500) {
		t.Errorf("unexpected body: %s", got.Body)
	}

	if !strings.Contains(out, "ch_1") {
		t.Errorf("expected the destination response, got:\n%s", out)
	}
}

func TestPCIProxyForward_ForwardsOtherMethodsWithoutABody(t *testing.T) {
	isolateConfig(t)
	seedCredentials(t)

	got := startAPI(t, http.StatusOK, `{"charge_id":"ch_1"}`)

	out, err := runCLI(t, "", "pci-proxy", "forward", "--yes", "--method", "get",
		"--destination-url", "https://api.processor.com/charges/ch_1")
	if err != nil {
		t.Fatalf("pci-proxy forward failed: %v (%s)", err, out)
	}

	if got.Method != http.MethodGet || got.Body != "" {
		t.Errorf("expected a bodyless GET, got %s with body %q", got.Method, got.Body)
	}
}

func TestPCIProxyForward_RejectsAnUnsupportedMethod(t *testing.T) {
	isolateConfig(t)
	seedCredentials(t)
	startAPI(t, http.StatusOK, `{}`)

	_, err := runCLI(t, "", "pci-proxy", "forward", "--yes", "--method", "TRACE",
		"--destination-url", "https://api.processor.com/charges")
	if err == nil || !strings.Contains(err.Error(), "unsupported method") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestPCIProxyForward_RequiresABodyForPost(t *testing.T) {
	isolateConfig(t)
	seedCredentials(t)
	startAPI(t, http.StatusOK, `{}`)

	_, err := runCLI(t, "", "pci-proxy", "forward", "--yes",
		"--destination-url", "https://api.processor.com/charges")
	if err == nil || !strings.Contains(err.Error(), "request body is required") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestPCIProxyForward_RequiresTheDestinationURL(t *testing.T) {
	isolateConfig(t)
	seedCredentials(t)
	startAPI(t, http.StatusOK, `{}`)

	_, err := runCLI(t, "", "pci-proxy", "forward", "--yes", "--data", `{"amount":1}`)
	if err == nil || !strings.Contains(err.Error(), "destination-url") {
		t.Errorf("unexpected error: %v", err)
	}
}
