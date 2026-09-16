package cmd_test

import (
	"net/http"
	"strings"
	"testing"

	"github.com/pvarganov/yuno-cli/internal/config"
)

// planResponse is the plan payload the fake API returns.
const planResponse = `{"id":"plan-1","account_id":"acc-1","name":"Gold","status":"ACTIVE",
	"merchant_reference":"gold-1","base_amount":{"currency":"USD","value":9.99},
	"frequency":{"type":"MONTH","value":1},"subscribers_count":7,
	"phases":[{"name":"trial"}],"created_at":"2026-09-16T10:00:00Z"}`

func TestPlanCreate_PostsTheBody(t *testing.T) {
	isolateConfig(t)
	seedCredentials(t)

	got := startAPI(t, http.StatusCreated, planResponse)

	out, err := runCLI(t, "", "plan", "create", "--yes",
		"--account-id", "acc-1", "--name", "Gold", "--currency", "USD", "--amount", "9.99",
		"--frequency-type", "MONTH", "--frequency-value", "1",
		"--payment-method", "CARD", "--payment-method", "WALLET",
		"--phases", `[{"name":"trial"}]`)
	if err != nil {
		t.Fatalf("plan create failed: %v (%s)", err, out)
	}

	if got.Method != http.MethodPost || got.Path != "/v1/subscriptions/plans" {
		t.Errorf("expected POST /v1/subscriptions/plans, got %s %s", got.Method, got.Path)
	}

	body := decodeBody(t, got.Body)

	base, ok := body["base_amount"].(map[string]any)
	if !ok || base["currency"] != "USD" || base["value"] != 9.99 {
		t.Errorf("unexpected base amount in body: %s", got.Body)
	}

	methods, ok := body["allowed_payment_methods"].([]any)
	if !ok || len(methods) != 2 || methods[1] != "WALLET" {
		t.Errorf("unexpected payment methods in body: %s", got.Body)
	}

	if phases, ok := body["phases"].([]any); !ok || len(phases) != 1 {
		t.Errorf("unexpected phases in body: %s", got.Body)
	}

	for _, want := range []string{"BASE_AMOUNT", "USD 9.99", "1 MONTH", "Gold"} {
		if !strings.Contains(out, want) {
			t.Errorf("expected table output to contain %q, got:\n%s", want, out)
		}
	}
}

func TestPlanCreate_RequiresABody(t *testing.T) {
	isolateConfig(t)
	seedCredentials(t)
	startAPI(t, http.StatusCreated, planResponse)

	if _, err := runCLI(t, "", "plan", "create", "--yes"); err == nil ||
		!strings.Contains(err.Error(), "request body is required") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestPlanList_SendsTheAccountAndPaging(t *testing.T) {
	isolateConfig(t)
	seedCredentials(t)

	got := startAPI(t, http.StatusOK, `{"data":[`+planResponse+`],"pagination":{"has_more":false}}`)

	out, err := runCLI(t, "", "plan", "list", "--account-id", "acc-1", "--limit", "5", "--page-size", "5")
	if err != nil {
		t.Fatalf("plan list failed: %v (%s)", err, out)
	}

	if got.Method != http.MethodGet || got.Path != "/v1/subscriptions/plans" {
		t.Errorf("expected GET /v1/subscriptions/plans, got %s %s", got.Method, got.Path)
	}

	for _, want := range []string{"account_id=acc-1", "limit=5", "offset=0"} {
		if !strings.Contains(got.Query, want) {
			t.Errorf("expected query to contain %q, got %q", want, got.Query)
		}
	}

	if !strings.Contains(out, "plan-1") {
		t.Errorf("expected the plan in the table, got:\n%s", out)
	}
}

func TestPlanList_FallsBackToTheProfileAccountID(t *testing.T) {
	isolateConfig(t)
	writeConfig(t, &config.Config{
		DefaultProfile: "sandbox",
		Profiles: map[string]config.Profile{
			"sandbox": {
				Environment:      config.EnvironmentSandbox,
				PublicAPIKey:     "pub-key",
				PrivateSecretKey: "sec-key",
				AccountID:        "acc-profile",
			},
		},
	})

	got := startAPI(t, http.StatusOK, `{"data":[],"pagination":{"has_more":false}}`)

	if out, err := runCLI(t, "", "plan", "list"); err != nil {
		t.Fatalf("plan list failed: %v (%s)", err, out)
	}

	if !strings.Contains(got.Query, "account_id=acc-profile") {
		t.Errorf("expected the profile account id in the query, got %q", got.Query)
	}
}

func TestPlanGet_HitsThePlanPath(t *testing.T) {
	isolateConfig(t)
	seedCredentials(t)

	got := startAPI(t, http.StatusOK, planResponse)

	out, err := runCLI(t, "", "plan", "get", "plan-1", "--json")
	if err != nil {
		t.Fatalf("plan get failed: %v (%s)", err, out)
	}

	if got.Method != http.MethodGet || got.Path != "/v1/subscriptions/plans/plan-1" {
		t.Errorf("expected GET /v1/subscriptions/plans/plan-1, got %s %s", got.Method, got.Path)
	}

	if !strings.Contains(out, "phases") {
		t.Errorf("expected the raw response to survive --json, got:\n%s", out)
	}
}

func TestPlanStatus_PostsTheStatus(t *testing.T) {
	isolateConfig(t)
	seedCredentials(t)

	got := startAPI(t, http.StatusOK, `{"id":"plan-1","status":"CANCELED","affected_subscriptions":2}`)

	out, err := runCLI(t, "", "plan", "status", "plan-1", "--yes", "--status", "CANCELED")
	if err != nil {
		t.Fatalf("plan status failed: %v (%s)", err, out)
	}

	if got.Method != http.MethodPost || got.Path != "/v1/subscriptions/plans/plan-1/status" {
		t.Errorf("expected POST /v1/subscriptions/plans/plan-1/status, got %s %s", got.Method, got.Path)
	}

	if body := decodeBody(t, got.Body); body["status"] != "CANCELED" {
		t.Errorf("unexpected body: %s", got.Body)
	}

	for _, want := range []string{"AFFECTED_SUBSCRIPTIONS", "CANCELED", "2"} {
		if !strings.Contains(out, want) {
			t.Errorf("expected table output to contain %q, got:\n%s", want, out)
		}
	}
}

func TestPlanStatus_RequiresTheStatus(t *testing.T) {
	isolateConfig(t)
	seedCredentials(t)
	startAPI(t, http.StatusOK, `{"id":"plan-1"}`)

	if _, err := runCLI(t, "", "plan", "status", "plan-1", "--yes"); err == nil ||
		!strings.Contains(err.Error(), "--status is required") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestPlanGet_ForbiddenMentionsTheScope(t *testing.T) {
	isolateConfig(t)
	seedCredentials(t)
	startAPI(t, http.StatusForbidden, `{"code":"INSUFFICIENT_SCOPE","message":"nope"}`)

	_, err := runCLI(t, "", "plan", "get", "plan-1")
	if err == nil {
		t.Fatal("expected an error on 403")
	}

	if !strings.Contains(err.Error(), "subscriptions:read") {
		t.Errorf("expected the scope hint, got %v", err)
	}
}
