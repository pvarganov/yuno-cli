package cmd_test

import (
	"net/http"
	"strings"
	"testing"
)

// checkoutBuilderResponse is the checkout payload the fake API returns.
const checkoutBuilderResponse = `{"id":"ck-1","name":"Promo Checkout","description":"Seasonal promo",
	"status":"PUBLISHED","is_default":true,"created_at":"2026-09-16T10:00:00Z"}`

func TestCheckoutBuilderCreate_PostsTheBody(t *testing.T) {
	isolateConfig(t)
	seedCredentials(t)

	got := startAPI(t, http.StatusCreated, checkoutBuilderResponse)

	out, err := runCLI(t, "", "checkout-builder", "create", "--yes",
		"--name", "Promo Checkout", "--description", "Seasonal promo")
	if err != nil {
		t.Fatalf("checkout-builder create failed: %v (%s)", err, out)
	}

	if got.Method != http.MethodPost || got.Path != "/v1/checkouts" {
		t.Errorf("expected POST /v1/checkouts, got %s %s", got.Method, got.Path)
	}

	body := decodeBody(t, got.Body)
	if body["name"] != "Promo Checkout" || body["description"] != "Seasonal promo" {
		t.Errorf("unexpected body: %s", got.Body)
	}

	for _, want := range []string{"ck-1", "Promo Checkout", "PUBLISHED"} {
		if !strings.Contains(out, want) {
			t.Errorf("expected table output to contain %q, got:\n%s", want, out)
		}
	}
}

func TestCheckoutBuilderCreate_RefusesAnEmptyBody(t *testing.T) {
	isolateConfig(t)
	seedCredentials(t)
	startAPI(t, http.StatusCreated, checkoutBuilderResponse)

	if _, err := runCLI(t, "", "checkout-builder", "create", "--yes"); err == nil ||
		!strings.Contains(err.Error(), "request body is required") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestCheckoutBuilderList_SendsTheFiltersAndPagesFromOne(t *testing.T) {
	isolateConfig(t)
	seedCredentials(t)

	got := startAPI(t, http.StatusOK,
		`{"data":[{"id":"ck-1","name":"Promo Checkout","status":"ARCHIVED"}],
		 "pagination":{"page":1,"size":1,"total":1}}`)

	out, err := runCLI(t, "", "checkout-builder", "list",
		"--status", "ARCHIVED", "--created-after", "2026-09-01", "--page-size", "1")
	if err != nil {
		t.Fatalf("checkout-builder list failed: %v (%s)", err, out)
	}

	if got.Method != http.MethodGet || got.Path != "/v1/checkouts" {
		t.Errorf("expected GET /v1/checkouts, got %s %s", got.Method, got.Path)
	}

	for _, want := range []string{"status=ARCHIVED", "created_after=2026-09-01", "page=1", "size=1"} {
		if !strings.Contains(got.Query, want) {
			t.Errorf("expected the query to contain %q, got %q", want, got.Query)
		}
	}

	if !strings.Contains(out, "ck-1") {
		t.Errorf("expected the checkout in the table, got:\n%s", out)
	}
}

func TestCheckoutBuilderList_PrintsNothingWhenEmpty(t *testing.T) {
	isolateConfig(t)
	seedCredentials(t)
	startAPI(t, http.StatusOK, `{"data":[],"pagination":{"page":1,"size":100,"total":0}}`)

	out, err := runCLI(t, "", "checkout-builder", "list")
	if err != nil {
		t.Fatalf("checkout-builder list failed: %v (%s)", err, out)
	}

	if strings.TrimSpace(out) != "" {
		t.Errorf("expected no output for an empty list, got:\n%s", out)
	}
}

func TestCheckoutBuilderGet_HitsTheCodePath(t *testing.T) {
	isolateConfig(t)
	seedCredentials(t)

	got := startAPI(t, http.StatusOK, checkoutBuilderResponse)

	out, err := runCLI(t, "", "checkout-builder", "get", "ck-1", "--json")
	if err != nil {
		t.Fatalf("checkout-builder get failed: %v (%s)", err, out)
	}

	if got.Method != http.MethodGet || got.Path != "/v1/checkouts/ck-1" {
		t.Errorf("expected GET /v1/checkouts/ck-1, got %s %s", got.Method, got.Path)
	}

	if !strings.Contains(out, `"id": "ck-1"`) {
		t.Errorf("expected the json body, got:\n%s", out)
	}
}

func TestCheckoutBuilderPublish_UsesPut(t *testing.T) {
	isolateConfig(t)
	seedCredentials(t)

	got := startAPI(t, http.StatusNoContent, "")

	out, err := runCLI(t, "", "checkout-builder", "publish", "ck-1", "--yes",
		"--config", `{"payment_methods":[{"payment_method_type":"CARD","order_to_show":0}]}`)
	if err != nil {
		t.Fatalf("checkout-builder publish failed: %v (%s)", err, out)
	}

	if got.Method != http.MethodPut || got.Path != "/v1/checkouts/ck-1" {
		t.Errorf("expected PUT /v1/checkouts/ck-1, got %s %s", got.Method, got.Path)
	}

	config, ok := decodeBody(t, got.Body)["config"].(map[string]any)
	if !ok || config["payment_methods"] == nil {
		t.Errorf("unexpected body: %s", got.Body)
	}
}

func TestCheckoutBuilderPublish_RejectsInvalidJSONFlags(t *testing.T) {
	isolateConfig(t)
	seedCredentials(t)
	startAPI(t, http.StatusNoContent, "")

	if _, err := runCLI(t, "", "checkout-builder", "publish", "ck-1", "--yes", "--styling", "{"); err == nil ||
		!strings.Contains(err.Error(), "not valid json") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestCheckoutBuilderUpdate_UsesPatch(t *testing.T) {
	isolateConfig(t)
	seedCredentials(t)

	got := startAPI(t, http.StatusOK, checkoutBuilderResponse)

	out, err := runCLI(t, "", "checkout-builder", "update", "ck-1", "--yes",
		"--status", "PUBLISHED", "--default")
	if err != nil {
		t.Fatalf("checkout-builder update failed: %v (%s)", err, out)
	}

	if got.Method != http.MethodPatch || got.Path != "/v1/checkouts/ck-1" {
		t.Errorf("expected PATCH /v1/checkouts/ck-1, got %s %s", got.Method, got.Path)
	}

	body := decodeBody(t, got.Body)
	if body["status"] != "PUBLISHED" || body["is_default"] != true {
		t.Errorf("unexpected body: %s", got.Body)
	}
}

func TestCheckoutBuilderUpdate_AsksBeforeWriting(t *testing.T) {
	isolateConfig(t)
	seedCredentials(t)
	startAPI(t, http.StatusOK, checkoutBuilderResponse)

	if _, err := runCLI(t, "n\n", "checkout-builder", "update", "ck-1", "--status", "ARCHIVED"); err == nil {
		t.Error("expected a declined confirmation to fail the command")
	}
}
