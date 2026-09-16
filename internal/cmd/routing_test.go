package cmd_test

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
)

// routingResponse is the list payload the fake API returns.
const routingResponse = `[{"id":"r-1","account_id":"acc-1","payment_method":"CARD",
	"name":"Card routing","default_route":{"steps":[{"index":1,"provider_id":"STRIPE",
	"connection_id":"c-1"},{"index":2,"provider_id":"ADYEN","connection_id":"c-2"}]},
	"condition_sets":[{"sort_number":1,"name":"High value","route":{"steps":[]}}]}]`

func TestRoutingList_QueriesAccountAndPaymentMethod(t *testing.T) {
	isolateConfig(t)
	seedCredentials(t)

	got := startAPI(t, http.StatusOK, routingResponse)

	out, err := runCLI(t, "", "routing", "list",
		"--account-id", "acc-1", "--payment-method", "CARD")
	if err != nil {
		t.Fatalf("routing list failed: %v (%s)", err, out)
	}

	if got.Method != http.MethodGet || got.Path != "/v1/routing" {
		t.Errorf("expected GET /v1/routing, got %s %s", got.Method, got.Path)
	}

	if got.Query != "account_id=acc-1&payment_method=CARD" {
		t.Errorf("unexpected query: %s", got.Query)
	}

	for _, want := range []string{"PAYMENT_METHOD", "PROVIDERS", "STRIPE > ADYEN", "Card routing"} {
		if !strings.Contains(out, want) {
			t.Errorf("expected table output to contain %q, got:\n%s", want, out)
		}
	}
}

func TestRoutingList_UsesProfileAccountID(t *testing.T) {
	isolateConfig(t)
	seedCredentials(t)
	t.Setenv("YUNO_ACCOUNT_ID", "profile-acc")

	got := startAPI(t, http.StatusOK, `[]`)

	if out, err := runCLI(t, "", "routing", "list"); err != nil {
		t.Fatalf("routing list failed: %v (%s)", err, out)
	}

	if got.Query != "account_id=profile-acc" {
		t.Errorf("expected the profile account id in the query, got %s", got.Query)
	}
}

func TestRoutingList_RequiresAccountID(t *testing.T) {
	isolateConfig(t)
	seedCredentials(t)
	startAPI(t, http.StatusOK, `[]`)

	_, err := runCLI(t, "", "routing", "list")
	if err == nil {
		t.Fatal("expected an error when no account id is available")
	}

	if !strings.Contains(err.Error(), "--account-id is required") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestRoutingList_JSONOutput(t *testing.T) {
	isolateConfig(t)
	seedCredentials(t)
	startAPI(t, http.StatusOK, routingResponse)

	out, err := runCLI(t, "", "routing", "list", "--account-id", "acc-1", "--json")
	if err != nil {
		t.Fatalf("routing list failed: %v (%s)", err, out)
	}

	var decoded []map[string]any
	if err := json.Unmarshal([]byte(out), &decoded); err != nil {
		t.Fatalf("output is not json: %v (%s)", err, out)
	}

	if len(decoded) != 1 || decoded[0]["id"] != "r-1" {
		t.Errorf("unexpected json output: %s", out)
	}
}

func TestRoutingGet_HitsTheRoutingPath(t *testing.T) {
	isolateConfig(t)
	seedCredentials(t)

	got := startAPI(t, http.StatusOK, strings.Trim(routingResponse, "[]"))

	out, err := runCLI(t, "", "routing", "get", "r-1")
	if err != nil {
		t.Fatalf("routing get failed: %v (%s)", err, out)
	}

	if got.Method != http.MethodGet || got.Path != "/v1/routing/r-1" {
		t.Errorf("expected GET /v1/routing/r-1, got %s %s", got.Method, got.Path)
	}

	if !strings.Contains(out, "STRIPE > ADYEN") {
		t.Errorf("expected the provider chain in the output, got:\n%s", out)
	}
}

func TestRoutingCreate_SendsTheBody(t *testing.T) {
	isolateConfig(t)
	seedCredentials(t)

	got := startAPI(t, http.StatusCreated, strings.Trim(routingResponse, "[]"))

	out, err := runCLI(t, "", "routing", "create", "--yes",
		"--account-id", "acc-1", "--payment-method", "CARD", "--name", "Card routing",
		"--default-route", `{"steps":[{"index":1,"provider_id":"STRIPE","connection_id":"c-1"}]}`)
	if err != nil {
		t.Fatalf("routing create failed: %v (%s)", err, out)
	}

	if got.Method != http.MethodPost || got.Path != "/v1/routing" {
		t.Errorf("expected POST /v1/routing, got %s %s", got.Method, got.Path)
	}

	var body map[string]any
	if err := json.Unmarshal([]byte(got.Body), &body); err != nil {
		t.Fatalf("request body is not json: %v (%s)", err, got.Body)
	}

	if body["account_id"] != "acc-1" || body["name"] != "Card routing" {
		t.Errorf("unexpected request body: %s", got.Body)
	}

	route, ok := body["default_route"].(map[string]any)
	if !ok || len(route["steps"].([]any)) != 1 {
		t.Errorf("expected the default route to be sent as an object: %s", got.Body)
	}
}

func TestRoutingCreate_RequiresABody(t *testing.T) {
	isolateConfig(t)
	seedCredentials(t)
	startAPI(t, http.StatusCreated, `{}`)

	_, err := runCLI(t, "", "routing", "create", "--yes")
	if err == nil {
		t.Fatal("expected an error when no body is given")
	}

	if !strings.Contains(err.Error(), "request body is required") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestRoutingUpdate_PatchesOnlyChangedFields(t *testing.T) {
	isolateConfig(t)
	seedCredentials(t)

	got := startAPI(t, http.StatusOK, strings.Trim(routingResponse, "[]"))

	out, err := runCLI(t, "", "routing", "update", "r-1", "--yes", "--name", "Renamed")
	if err != nil {
		t.Fatalf("routing update failed: %v (%s)", err, out)
	}

	if got.Method != http.MethodPatch || got.Path != "/v1/routing/r-1" {
		t.Errorf("expected PATCH /v1/routing/r-1, got %s %s", got.Method, got.Path)
	}

	var body map[string]any
	if err := json.Unmarshal([]byte(got.Body), &body); err != nil {
		t.Fatalf("request body is not json: %v (%s)", err, got.Body)
	}

	if len(body) != 1 || body["name"] != "Renamed" {
		t.Errorf("expected only the changed field in the patch body, got %s", got.Body)
	}
}

func TestRoutingRecommend_SendsTheFileBody(t *testing.T) {
	isolateConfig(t)
	seedCredentials(t)

	response := `{"recommendation_id":"rec-1","optimized_for":"APPROVAL_RATE",
		"decision_source":"MERCHANT_HISTORY","recommended":{"provider_id":"ADYEN"},
		"ranking":[{"provider_id":"ADYEN","approval_rate":0.9987,"avg_latency_ms":180,"sample_size":767},
		{"provider_id":"STRIPE","approval_rate":null,"avg_latency_ms":null,"sample_size":12}]}`

	got := startAPI(t, http.StatusOK, response)

	payload := `{"account_id":"acc-1","payment":{"amount":{"currency":"USD","value":99}},"candidates":[]}`

	out, err := runCLI(t, payload, "routing", "recommend", "--yes", "--data", "@-")
	if err != nil {
		t.Fatalf("routing recommend failed: %v (%s)", err, out)
	}

	if got.Method != http.MethodPost || got.Path != "/v1/routing/recommendations" {
		t.Errorf("expected POST /v1/routing/recommendations, got %s %s", got.Method, got.Path)
	}

	if !strings.Contains(got.Body, `"candidates"`) {
		t.Errorf("expected the stdin body to be forwarded, got %s", got.Body)
	}

	for _, want := range []string{"ADYEN", "0.9987", "true", "STRIPE"} {
		if !strings.Contains(out, want) {
			t.Errorf("expected output to contain %q, got:\n%s", want, out)
		}
	}
}

func TestRouting_ForbiddenMentionsTheScope(t *testing.T) {
	isolateConfig(t)
	seedCredentials(t)

	startAPI(t, http.StatusForbidden,
		`{"type":"forbidden","code":"INSUFFICIENT_SCOPE","message":"missing scope"}`)

	_, err := runCLI(t, "", "routing", "list", "--account-id", "acc-1")
	if err == nil {
		t.Fatal("expected a 403 error")
	}

	for _, want := range []string{"INSUFFICIENT_SCOPE", "routing:read"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("expected the error to mention %q, got: %v", want, err)
		}
	}
}
