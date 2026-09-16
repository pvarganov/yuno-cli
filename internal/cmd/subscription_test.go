package cmd_test

import (
	"net/http"
	"strings"
	"testing"
)

// subscriptionResponse is the subscription payload the fake API returns.
const subscriptionResponse = `{"id":"sub-1","name":"Gold","status":"ACTIVE","plan_id":"plan-1",
	"account_id":"acc-1","country":"CO","amount":{"currency":"USD","value":9.99},
	"frequency":{"type":"MONTH","value":1},"customer_payer":{"id":"cus-1"},
	"payment_method":{"type":"CARD","vaulted_token":"46adcbc0-aa26-4867-a4e7-28a5ad9100ae"},
	"phases":[{"name":"trial"}],"created_at":"2026-09-16T10:00:00Z"}`

func TestSubscriptionCreate_PostsTheBody(t *testing.T) {
	isolateConfig(t)
	seedCredentials(t)

	got := startAPI(t, http.StatusCreated, subscriptionResponse)

	out, err := runCLI(t, "", "subscription", "create", "--yes",
		"--account-id", "acc-1", "--name", "Gold", "--country", "CO",
		"--currency", "USD", "--amount", "9.99",
		"--frequency-type", "MONTH", "--frequency-value", "1",
		"--billing-cycles", "12", "--customer-id", "cus-1",
		"--payment-method-type", "CARD", "--vaulted-token", "vt-1")
	if err != nil {
		t.Fatalf("subscription create failed: %v (%s)", err, out)
	}

	if got.Method != http.MethodPost || got.Path != "/v1/subscriptions" {
		t.Errorf("expected POST /v1/subscriptions, got %s %s", got.Method, got.Path)
	}

	body := decodeBody(t, got.Body)
	if body["name"] != "Gold" || body["account_id"] != "acc-1" {
		t.Errorf("unexpected body: %s", got.Body)
	}

	amount, ok := body["amount"].(map[string]any)
	if !ok || amount["currency"] != "USD" || amount["value"] != 9.99 {
		t.Errorf("unexpected amount in body: %s", got.Body)
	}

	frequency, ok := body["frequency"].(map[string]any)
	if !ok || frequency["type"] != "MONTH" || frequency["value"] != float64(1) {
		t.Errorf("unexpected frequency in body: %s", got.Body)
	}

	cycles, ok := body["billing_cycles"].(map[string]any)
	if !ok || cycles["total"] != float64(12) {
		t.Errorf("unexpected billing cycles in body: %s", got.Body)
	}

	payer, ok := body["customer_payer"].(map[string]any)
	if !ok || payer["id"] != "cus-1" {
		t.Errorf("unexpected customer payer in body: %s", got.Body)
	}

	method, ok := body["payment_method"].(map[string]any)
	if !ok || method["type"] != "CARD" || method["vaulted_token"] != "vt-1" {
		t.Errorf("unexpected payment method in body: %s", got.Body)
	}

	for _, want := range []string{"STATUS", "ACTIVE", "USD 9.99", "1 MONTH"} {
		if !strings.Contains(out, want) {
			t.Errorf("expected table output to contain %q, got:\n%s", want, out)
		}
	}
}

func TestSubscriptionCreate_RequiresABody(t *testing.T) {
	isolateConfig(t)
	seedCredentials(t)
	startAPI(t, http.StatusCreated, subscriptionResponse)

	if _, err := runCLI(t, "", "subscription", "create", "--yes"); err == nil ||
		!strings.Contains(err.Error(), "request body is required") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestSubscriptionList_SendsTheFiltersAndPaging(t *testing.T) {
	isolateConfig(t)
	seedCredentials(t)

	got := startAPI(t, http.StatusOK, `{"items":[`+subscriptionResponse+`],"pagination":{"total":1}}`)

	out, err := runCLI(t, "", "subscription", "list",
		"--status", "ACTIVE", "--customer-id", "cus-1", "--plan-id", "plan-1",
		"--created-after", "2026-01-01", "--created-before", "2026-12-31",
		"--payment-method-type", "CARD", "--limit", "1", "--page-size", "25")
	if err != nil {
		t.Fatalf("subscription list failed: %v (%s)", err, out)
	}

	if got.Method != http.MethodGet || got.Path != "/v1/subscriptions" {
		t.Errorf("expected GET /v1/subscriptions, got %s %s", got.Method, got.Path)
	}

	for _, want := range []string{
		"status=ACTIVE", "customer_id=cus-1", "plan_id=plan-1",
		"created_at_from=2026-01-01", "created_at_to=2026-12-31",
		"payment_method_type=CARD", "page=0", "size=25",
	} {
		if !strings.Contains(got.Query, want) {
			t.Errorf("expected query to contain %q, got %q", want, got.Query)
		}
	}

	if !strings.Contains(out, "sub-1") {
		t.Errorf("expected the subscription in the table, got:\n%s", out)
	}
}

func TestSubscriptionGet_HitsTheSubscriptionPath(t *testing.T) {
	isolateConfig(t)
	seedCredentials(t)

	got := startAPI(t, http.StatusOK, subscriptionResponse)

	out, err := runCLI(t, "", "subscription", "get", "sub-1", "--json")
	if err != nil {
		t.Fatalf("subscription get failed: %v (%s)", err, out)
	}

	if got.Method != http.MethodGet || got.Path != "/v1/subscriptions/sub-1" {
		t.Errorf("expected GET /v1/subscriptions/sub-1, got %s %s", got.Method, got.Path)
	}

	if !strings.Contains(out, "phases") || !strings.Contains(out, "trial") {
		t.Errorf("expected the raw response to survive --json, got:\n%s", out)
	}
}

func TestSubscriptionGet_MasksTheVaultedTokenInTheTable(t *testing.T) {
	isolateConfig(t)
	seedCredentials(t)
	startAPI(t, http.StatusOK, subscriptionResponse)

	out, err := runCLI(t, "", "subscription", "get", "sub-1")
	if err != nil {
		t.Fatalf("subscription get failed: %v (%s)", err, out)
	}

	if strings.Contains(out, "46adcbc0-aa26-4867-a4e7-28a5ad9100ae") {
		t.Errorf("table output leaks the vaulted token:\n%s", out)
	}
}

func TestSubscriptionUpdate_PatchesTheSubscription(t *testing.T) {
	isolateConfig(t)
	seedCredentials(t)

	got := startAPI(t, http.StatusOK, subscriptionResponse)

	out, err := runCLI(t, "", "subscription", "update", "sub-1", "--yes", "--amount", "12.5")
	if err != nil {
		t.Fatalf("subscription update failed: %v (%s)", err, out)
	}

	if got.Method != http.MethodPatch || got.Path != "/v1/subscriptions/sub-1" {
		t.Errorf("expected PATCH /v1/subscriptions/sub-1, got %s %s", got.Method, got.Path)
	}

	body := decodeBody(t, got.Body)
	if _, ok := body["name"]; ok {
		t.Errorf("an unchanged flag must not land in the patch body: %s", got.Body)
	}

	amount, ok := body["amount"].(map[string]any)
	if !ok || amount["value"] != 12.5 {
		t.Errorf("unexpected amount in body: %s", got.Body)
	}
}

func TestSubscriptionPayments_ListsThePayments(t *testing.T) {
	isolateConfig(t)
	seedCredentials(t)

	response := `{"data":[{"id":"sp-1","payment_id":"pay-1","status":"SUCCEEDED",` +
		`"currency":"USD","amount":9.99,"billing_cycle":1,"created_at":"2026-09-16T10:00:00Z"}],` +
		`"pagination":{"has_more":false}}`

	got := startAPI(t, http.StatusOK, response)

	out, err := runCLI(t, "", "subscription", "payments", "sub-1", "--limit", "10", "--page-size", "10")
	if err != nil {
		t.Fatalf("subscription payments failed: %v (%s)", err, out)
	}

	if got.Method != http.MethodGet || got.Path != "/v1/subscriptions/sub-1/payments" {
		t.Errorf("expected GET /v1/subscriptions/sub-1/payments, got %s %s", got.Method, got.Path)
	}

	if !strings.Contains(got.Query, "limit=10") || !strings.Contains(got.Query, "offset=0") {
		t.Errorf("unexpected query %q", got.Query)
	}

	for _, want := range []string{"BILLING_CYCLE", "pay-1", "SUCCEEDED"} {
		if !strings.Contains(out, want) {
			t.Errorf("expected table output to contain %q, got:\n%s", want, out)
		}
	}
}

func TestSubscriptionSimpleActions_PostToTheActionPath(t *testing.T) {
	actions := map[string]string{
		"pause":  "/v1/subscriptions/sub-1/pause",
		"resume": "/v1/subscriptions/sub-1/resume",
		"retry":  "/v1/subscriptions/sub-1/retry",
	}

	for action, path := range actions {
		t.Run(action, func(t *testing.T) {
			isolateConfig(t)
			seedCredentials(t)

			got := startAPI(t, http.StatusOK, subscriptionResponse)

			out, err := runCLI(t, "", "subscription", action, "sub-1", "--yes")
			if err != nil {
				t.Fatalf("subscription %s failed: %v (%s)", action, err, out)
			}

			if got.Method != http.MethodPost || got.Path != path {
				t.Errorf("expected POST %s, got %s %s", path, got.Method, got.Path)
			}

			if strings.TrimSpace(got.Body) != "" {
				t.Errorf("expected no body, got %s", got.Body)
			}
		})
	}
}

func TestSubscriptionCancel_SendsTheRefundFlags(t *testing.T) {
	isolateConfig(t)
	seedCredentials(t)

	got := startAPI(t, http.StatusOK, subscriptionResponse)

	out, err := runCLI(t, "", "subscription", "cancel", "sub-1", "--yes",
		"--refund", "--refund-amount", "5", "--schedule", "END_OF_CYCLE")
	if err != nil {
		t.Fatalf("subscription cancel failed: %v (%s)", err, out)
	}

	if got.Method != http.MethodPost || got.Path != "/v1/subscriptions/sub-1/cancel" {
		t.Errorf("expected POST /v1/subscriptions/sub-1/cancel, got %s %s", got.Method, got.Path)
	}

	body := decodeBody(t, got.Body)
	if body["refund"] != true || body["refund_amount"] != float64(5) || body["schedule"] != "END_OF_CYCLE" {
		t.Errorf("unexpected body: %s", got.Body)
	}
}

func TestSubscriptionCancel_WorksWithoutABody(t *testing.T) {
	isolateConfig(t)
	seedCredentials(t)

	got := startAPI(t, http.StatusOK, subscriptionResponse)

	if out, err := runCLI(t, "", "subscription", "cancel", "sub-1", "--yes"); err != nil {
		t.Fatalf("subscription cancel failed: %v (%s)", err, out)
	}

	if strings.TrimSpace(got.Body) != "" {
		t.Errorf("expected no body, got %s", got.Body)
	}
}

func TestSubscriptionChangePlan_PostsThePlanBody(t *testing.T) {
	isolateConfig(t)
	seedCredentials(t)

	got := startAPI(t, http.StatusOK, subscriptionResponse)

	out, err := runCLI(t, "", "subscription", "change-plan", "sub-1", "--yes",
		"--plan-id", "plan-2", "--skip-trial")
	if err != nil {
		t.Fatalf("subscription change-plan failed: %v (%s)", err, out)
	}

	if got.Method != http.MethodPost || got.Path != "/v1/subscriptions/sub-1/plan" {
		t.Errorf("expected POST /v1/subscriptions/sub-1/plan, got %s %s", got.Method, got.Path)
	}

	body := decodeBody(t, got.Body)
	if body["plan_id"] != "plan-2" || body["skip_trial"] != true {
		t.Errorf("unexpected body: %s", got.Body)
	}
}

func TestSubscriptionChangePlan_RequiresABody(t *testing.T) {
	isolateConfig(t)
	seedCredentials(t)
	startAPI(t, http.StatusOK, subscriptionResponse)

	if _, err := runCLI(t, "", "subscription", "change-plan", "sub-1", "--yes"); err == nil ||
		!strings.Contains(err.Error(), "request body is required") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestSubscriptionList_ForbiddenMentionsTheScope(t *testing.T) {
	isolateConfig(t)
	seedCredentials(t)
	startAPI(t, http.StatusForbidden, `{"code":"INSUFFICIENT_SCOPE","message":"nope"}`)

	_, err := runCLI(t, "", "subscription", "list")
	if err == nil {
		t.Fatal("expected an error on 403")
	}

	if !strings.Contains(err.Error(), "subscriptions:read") {
		t.Errorf("expected the scope hint, got %v", err)
	}
}
