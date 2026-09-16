package cmd_test

import (
	"net/http"
	"strings"
	"testing"
)

// campaignResponse is the campaign payload the fake API returns, `data` envelope
// included.
const campaignResponse = `{"data":{"id":"cp-1","name":"Recovery CO","country":"CO",
	"channel":"PHONE_CALL","status":"ACTIVE",
	"duration":{"start_at":"2026-10-01","end_at":"2026-12-31"},
	"created_at":"2026-09-16T10:00:00Z"}}`

// campaignRuleResponse is the rule payload the fake API returns.
const campaignRuleResponse = `{"data":{"id":"rl-1","rule_type":"PAYMENT_METHOD","conditional":"IN",
	"values":["CARD","PAYPAL"],"status":"ACTIVE","created_at":"2026-09-16T10:00:00Z"}}`

func TestCampaignCreate_NestsTheScheduleAndDuration(t *testing.T) {
	isolateConfig(t)
	seedCredentials(t)

	got := startAPI(t, http.StatusCreated, campaignResponse)

	out, err := runCLI(t, "", "campaign", "create", "--yes",
		"--name", "Recovery CO", "--account-id", "acc-1", "--country", "CO",
		"--channel", "PHONE_CALL", "--daily-start-time", "09:00", "--daily-end-time", "18:00",
		"--time-zone", "America/Bogota", "--start-at", "2026-10-01", "--end-at", "2026-12-31")
	if err != nil {
		t.Fatalf("campaign create failed: %v (%s)", err, out)
	}

	if got.Method != http.MethodPost || got.Path != "/v1/campaigns" {
		t.Errorf("expected POST /v1/campaigns, got %s %s", got.Method, got.Path)
	}

	body := decodeBody(t, got.Body)

	schedule, ok := body["schedule"].(map[string]any)
	if !ok || schedule["daily_start_time"] != "09:00" || schedule["time_zone"] != "America/Bogota" {
		t.Errorf("unexpected schedule: %s", got.Body)
	}

	duration, ok := body["duration"].(map[string]any)
	if !ok || duration["start_at"] != "2026-10-01" || duration["end_at"] != "2026-12-31" {
		t.Errorf("unexpected duration: %s", got.Body)
	}

	for _, want := range []string{"cp-1", "Recovery CO", "2026-12-31"} {
		if !strings.Contains(out, want) {
			t.Errorf("expected table output to contain %q, got:\n%s", want, out)
		}
	}
}

func TestCampaignList_SendsTheFiltersAndPagesByOffset(t *testing.T) {
	isolateConfig(t)
	seedCredentials(t)

	got := startAPI(t, http.StatusOK,
		`{"data":[{"id":"cp-1","name":"Recovery CO","status":"ACTIVE"}],
		 "meta":{"total":1,"limit":1,"offset":0}}`)

	out, err := runCLI(t, "", "campaign", "list",
		"--start-date", "2026-09-01", "--end-date", "2026-12-31", "--page-size", "1")
	if err != nil {
		t.Fatalf("campaign list failed: %v (%s)", err, out)
	}

	if got.Method != http.MethodGet || got.Path != "/v1/campaigns" {
		t.Errorf("expected GET /v1/campaigns, got %s %s", got.Method, got.Path)
	}

	for _, want := range []string{"start_date=2026-09-01", "end_date=2026-12-31", "limit=1", "offset=0"} {
		if !strings.Contains(got.Query, want) {
			t.Errorf("expected the query to contain %q, got %q", want, got.Query)
		}
	}

	if !strings.Contains(out, "cp-1") {
		t.Errorf("expected the campaign in the table, got:\n%s", out)
	}
}

func TestCampaignGet_UnwrapsTheEnvelopeForTheTable(t *testing.T) {
	isolateConfig(t)
	seedCredentials(t)

	got := startAPI(t, http.StatusOK, campaignResponse)

	out, err := runCLI(t, "", "campaign", "get", "cp-1")
	if err != nil {
		t.Fatalf("campaign get failed: %v (%s)", err, out)
	}

	if got.Method != http.MethodGet || got.Path != "/v1/campaigns/cp-1" {
		t.Errorf("expected GET /v1/campaigns/cp-1, got %s %s", got.Method, got.Path)
	}

	for _, want := range []string{"cp-1", "Recovery CO", "PHONE_CALL"} {
		if !strings.Contains(out, want) {
			t.Errorf("expected table output to contain %q, got:\n%s", want, out)
		}
	}
}

func TestCampaignUpdate_UsesPatch(t *testing.T) {
	isolateConfig(t)
	seedCredentials(t)

	got := startAPI(t, http.StatusOK, campaignResponse)

	out, err := runCLI(t, "", "campaign", "update", "cp-1", "--yes", "--status", "PAUSED")
	if err != nil {
		t.Fatalf("campaign update failed: %v (%s)", err, out)
	}

	if got.Method != http.MethodPatch || got.Path != "/v1/campaigns/cp-1" {
		t.Errorf("expected PATCH /v1/campaigns/cp-1, got %s %s", got.Method, got.Path)
	}

	if decodeBody(t, got.Body)["status"] != "PAUSED" {
		t.Errorf("unexpected body: %s", got.Body)
	}
}

func TestCampaignRuleCreate_WrapsTheFieldFlagsInARulesArray(t *testing.T) {
	isolateConfig(t)
	seedCredentials(t)

	got := startAPI(t, http.StatusCreated,
		`{"data":[{"id":"rl-1","rule_type":"PAYMENT_METHOD","conditional":"IN",
			"values":["CARD","PAYPAL"],"status":"ACTIVE"}]}`)

	out, err := runCLI(t, "", "campaign", "rule", "create", "cp-1", "--yes",
		"--rule-type", "PAYMENT_METHOD", "--conditional", "IN", "--value", "CARD", "--value", "PAYPAL")
	if err != nil {
		t.Fatalf("campaign rule create failed: %v (%s)", err, out)
	}

	if got.Method != http.MethodPost || got.Path != "/v1/campaigns/cp-1/rules" {
		t.Errorf("expected POST /v1/campaigns/cp-1/rules, got %s %s", got.Method, got.Path)
	}

	rules, ok := decodeBody(t, got.Body)["rules"].([]any)
	if !ok || len(rules) != 1 {
		t.Fatalf("expected one wrapped rule, got: %s", got.Body)
	}

	rule, ok := rules[0].(map[string]any)
	if !ok || rule["rule_type"] != "PAYMENT_METHOD" || rule["conditional"] != "IN" {
		t.Errorf("unexpected rule: %s", got.Body)
	}

	values, ok := rule["values"].([]any)
	if !ok || len(values) != 2 || values[1] != "PAYPAL" {
		t.Errorf("unexpected values: %s", got.Body)
	}

	if !strings.Contains(out, "CARD,PAYPAL") {
		t.Errorf("expected the values joined in the table, got:\n%s", out)
	}
}

func TestCampaignRuleCreate_KeepsAnExplicitRulesArray(t *testing.T) {
	isolateConfig(t)
	seedCredentials(t)

	got := startAPI(t, http.StatusCreated, `{"data":[{"id":"rl-1"},{"id":"rl-2"}]}`)

	out, err := runCLI(t, `{"rules":[{"rule_type":"METADATA","metadata_key":"tier"},{"rule_type":"COUNTRY"}]}`,
		"campaign", "rule", "create", "cp-1", "--yes", "--data", "@-")
	if err != nil {
		t.Fatalf("campaign rule create failed: %v (%s)", err, out)
	}

	rules, ok := decodeBody(t, got.Body)["rules"].([]any)
	if !ok || len(rules) != 2 {
		t.Errorf("expected the array to be forwarded as it was written, got: %s", got.Body)
	}
}

func TestCampaignRuleGet_HitsTheNestedPath(t *testing.T) {
	isolateConfig(t)
	seedCredentials(t)

	got := startAPI(t, http.StatusOK, campaignRuleResponse)

	out, err := runCLI(t, "", "campaign", "rule", "get", "cp-1", "rl-1", "--json")
	if err != nil {
		t.Fatalf("campaign rule get failed: %v (%s)", err, out)
	}

	if got.Method != http.MethodGet || got.Path != "/v1/campaigns/cp-1/rules/rl-1" {
		t.Errorf("expected GET /v1/campaigns/cp-1/rules/rl-1, got %s %s", got.Method, got.Path)
	}

	if !strings.Contains(out, `"id": "rl-1"`) {
		t.Errorf("expected the json body, got:\n%s", out)
	}
}

func TestCampaignRuleUpdate_SendsTheBareRule(t *testing.T) {
	isolateConfig(t)
	seedCredentials(t)

	got := startAPI(t, http.StatusOK, campaignRuleResponse)

	out, err := runCLI(t, "", "campaign", "rule", "update", "cp-1", "rl-1", "--yes",
		"--rule-type", "METADATA", "--metadata-key", "tier")
	if err != nil {
		t.Fatalf("campaign rule update failed: %v (%s)", err, out)
	}

	if got.Method != http.MethodPatch || got.Path != "/v1/campaigns/cp-1/rules/rl-1" {
		t.Errorf("expected PATCH /v1/campaigns/cp-1/rules/rl-1, got %s %s", got.Method, got.Path)
	}

	body := decodeBody(t, got.Body)
	if body["rule_type"] != "METADATA" || body["metadata_key"] != "tier" {
		t.Errorf("unexpected body: %s", got.Body)
	}

	if _, wrapped := body["rules"]; wrapped {
		t.Errorf("update takes a bare rule, not a rules array: %s", got.Body)
	}
}

func TestCampaignRuleStatus_HitsTheStatusPath(t *testing.T) {
	isolateConfig(t)
	seedCredentials(t)

	got := startAPI(t, http.StatusOK, campaignRuleResponse)

	out, err := runCLI(t, "", "campaign", "rule", "status", "cp-1", "rl-1", "--yes", "--status", "INACTIVE")
	if err != nil {
		t.Fatalf("campaign rule status failed: %v (%s)", err, out)
	}

	if got.Method != http.MethodPatch || got.Path != "/v1/campaigns/cp-1/rules/rl-1/status" {
		t.Errorf("expected PATCH /v1/campaigns/cp-1/rules/rl-1/status, got %s %s", got.Method, got.Path)
	}

	if decodeBody(t, got.Body)["status"] != "INACTIVE" {
		t.Errorf("unexpected body: %s", got.Body)
	}
}

func TestCampaignRuleStatus_RefusesAnEmptyBody(t *testing.T) {
	isolateConfig(t)
	seedCredentials(t)
	startAPI(t, http.StatusOK, campaignRuleResponse)

	if _, err := runCLI(t, "", "campaign", "rule", "status", "cp-1", "rl-1", "--yes"); err == nil ||
		!strings.Contains(err.Error(), "request body is required") {
		t.Errorf("unexpected error: %v", err)
	}
}
