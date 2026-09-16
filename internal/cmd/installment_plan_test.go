package cmd_test

import (
	"net/http"
	"strings"
	"testing"

	"github.com/pvarganov/yuno-cli/internal/config"
)

// installmentPlanResponse is the installments plan payload the fake API returns.
const installmentPlanResponse = `{"id":"ip-1","name":"plan_007","account_id":["acc-1"],
	"merchant_reference":"ref-1","country_code":"US",
	"installments_plan":[{"installment":3,"rate":1.2},{"installment":6,"rate":1.4}],
	"amount":{"currency":"USD","min_value":0,"max_value":100000},
	"created_at":"2026-09-16T10:00:00Z"}`

func TestInstallmentPlanCreate_PostsTheBody(t *testing.T) {
	isolateConfig(t)
	seedCredentials(t)

	got := startAPI(t, http.StatusOK, installmentPlanResponse)

	out, err := runCLI(t, "", "installment-plan", "create", "--yes",
		"--name", "plan_007", "--account-id", "acc-1", "--account-id", "acc-2",
		"--merchant-reference", "ref-1", "--country-code", "US",
		"--installments", `[{"installment":3,"rate":1.2}]`,
		"--currency", "USD", "--min-amount", "0", "--max-amount", "100000",
		"--brand", "VISA", "--payment-method-type", "CARD")
	if err != nil {
		t.Fatalf("installment-plan create failed: %v (%s)", err, out)
	}

	if got.Method != http.MethodPost || got.Path != "/v1/installments-plans" {
		t.Errorf("expected POST /v1/installments-plans, got %s %s", got.Method, got.Path)
	}

	body := decodeBody(t, got.Body)

	accounts, ok := body["account_id"].([]any)
	if !ok || len(accounts) != 2 || accounts[1] != "acc-2" {
		t.Errorf("unexpected accounts in body: %s", got.Body)
	}

	options, ok := body["installments_plan"].([]any)
	if !ok || len(options) != 1 {
		t.Errorf("unexpected installments in body: %s", got.Body)
	}

	amount, ok := body["amount"].(map[string]any)
	if !ok || amount["currency"] != "USD" || amount["max_value"] != 100000.0 {
		t.Errorf("unexpected amount in body: %s", got.Body)
	}

	for _, want := range []string{"ip-1", "plan_007", "3,6", "USD 0-100000"} {
		if !strings.Contains(out, want) {
			t.Errorf("expected table output to contain %q, got:\n%s", want, out)
		}
	}
}

func TestInstallmentPlanCreate_RequiresABody(t *testing.T) {
	isolateConfig(t)
	seedCredentials(t)
	startAPI(t, http.StatusOK, installmentPlanResponse)

	if _, err := runCLI(t, "", "installment-plan", "create", "--yes"); err == nil ||
		!strings.Contains(err.Error(), "request body is required") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestInstallmentPlanList_SendsTheFilters(t *testing.T) {
	isolateConfig(t)
	seedCredentials(t)

	got := startAPI(t, http.StatusOK, `[`+installmentPlanResponse+`]`)

	out, err := runCLI(t, "", "installment-plan", "list", "--account-id", "acc-1",
		"--currency", "USD", "--iin", "411111", "--amount", "5000",
		"--payment-method-type", "CARD")
	if err != nil {
		t.Fatalf("installment-plan list failed: %v (%s)", err, out)
	}

	if got.Method != http.MethodGet || got.Path != "/v1/installments-plans" {
		t.Errorf("expected GET /v1/installments-plans, got %s %s", got.Method, got.Path)
	}

	for _, want := range []string{"account_id=acc-1", "currency=USD", "iin=411111", "amount=5000",
		"payment_method_type=CARD"} {
		if !strings.Contains(got.Query, want) {
			t.Errorf("expected query to contain %q, got %q", want, got.Query)
		}
	}

	if !strings.Contains(out, "ip-1") {
		t.Errorf("expected the plan in the table, got:\n%s", out)
	}
}

func TestInstallmentPlanList_FallsBackToTheProfileAccountID(t *testing.T) {
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

	got := startAPI(t, http.StatusOK, `[]`)

	if out, err := runCLI(t, "", "installment-plan", "list"); err != nil {
		t.Fatalf("installment-plan list failed: %v (%s)", err, out)
	}

	if !strings.Contains(got.Query, "account_id=acc-profile") {
		t.Errorf("expected the profile account id in the query, got %q", got.Query)
	}
}

func TestInstallmentPlanGet_AcceptsAnArrayResponse(t *testing.T) {
	isolateConfig(t)
	seedCredentials(t)

	got := startAPI(t, http.StatusOK, `[`+installmentPlanResponse+`]`)

	out, err := runCLI(t, "", "installment-plan", "get", "ip-1")
	if err != nil {
		t.Fatalf("installment-plan get failed: %v (%s)", err, out)
	}

	if got.Method != http.MethodGet || got.Path != "/v1/installments-plans/ip-1" {
		t.Errorf("expected GET /v1/installments-plans/ip-1, got %s %s", got.Method, got.Path)
	}

	if !strings.Contains(out, "plan_007") {
		t.Errorf("expected the plan in the table, got:\n%s", out)
	}
}

func TestInstallmentPlanUpdate_PatchesOnlyTheChangedFields(t *testing.T) {
	isolateConfig(t)
	seedCredentials(t)

	got := startAPI(t, http.StatusOK, installmentPlanResponse)

	out, err := runCLI(t, "", "installment-plan", "update", "ip-1", "--yes", "--name", "plan_008")
	if err != nil {
		t.Fatalf("installment-plan update failed: %v (%s)", err, out)
	}

	if got.Method != http.MethodPatch || got.Path != "/v1/installments-plans/ip-1" {
		t.Errorf("expected PATCH /v1/installments-plans/ip-1, got %s %s", got.Method, got.Path)
	}

	body := decodeBody(t, got.Body)
	if len(body) != 1 || body["name"] != "plan_008" {
		t.Errorf("expected only the name in the body, got %s", got.Body)
	}
}

func TestInstallmentPlanDelete_SendsDelete(t *testing.T) {
	isolateConfig(t)
	seedCredentials(t)

	got := startAPI(t, http.StatusNoContent, "")

	if out, err := runCLI(t, "", "installment-plan", "delete", "ip-1", "--yes"); err != nil {
		t.Fatalf("installment-plan delete failed: %v (%s)", err, out)
	}

	if got.Method != http.MethodDelete || got.Path != "/v1/installments-plans/ip-1" {
		t.Errorf("expected DELETE /v1/installments-plans/ip-1, got %s %s", got.Method, got.Path)
	}
}

func TestInstallmentPlanDelete_ReportsTheAPIError(t *testing.T) {
	isolateConfig(t)
	seedCredentials(t)
	startAPI(t, http.StatusForbidden, `{"code":"INSUFFICIENT_SCOPE","message":"nope"}`)

	_, err := runCLI(t, "", "installment-plan", "delete", "ip-1", "--yes")
	if err == nil || !strings.Contains(err.Error(), "installments-plans") {
		t.Errorf("expected the scope hint, got %v", err)
	}
}
