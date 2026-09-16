package api

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// campaignBody is the campaign payload the fake API answers with: the single
// campaign endpoints wrap the resource in a `data` envelope.
const campaignBody = `{"data":{"id":"cp-1","name":"Recovery CO","country":"CO","channel":"PHONE_CALL",
	"status":"ACTIVE","duration":{"start_at":"2026-10-01","end_at":"2026-12-31"},
	"created_at":"2026-09-16T10:00:00Z"}}`

// campaignRuleBody is the rule payload the fake API answers with.
const campaignRuleBody = `{"data":{"id":"rl-1","rule_type":"PAYMENT_METHOD","conditional":"IN",
	"values":["CARD","PAYPAL"],"status":"ACTIVE","created_at":"2026-09-16T10:00:00Z"}}`

func TestCreateCampaign_UnwrapsTheDataEnvelope(t *testing.T) {
	var gotMethod, gotPath string

	var gotBody map[string]any

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod, gotPath = r.Method, r.URL.Path
		_ = json.NewDecoder(r.Body).Decode(&gotBody)

		w.WriteHeader(http.StatusCreated)
		_, _ = io.WriteString(w, campaignBody)
	}))
	defer srv.Close()

	c, _ := newTestClient(t, srv, testProfile())

	campaign, err := c.CreateCampaign(t.Context(), map[string]any{"name": "Recovery CO"})
	if err != nil {
		t.Fatalf("CreateCampaign: %v", err)
	}

	if gotMethod != http.MethodPost || gotPath != "/campaigns" {
		t.Errorf("expected POST /campaigns, got %s %s", gotMethod, gotPath)
	}

	if campaign.ID != "cp-1" || campaign.Duration == nil || campaign.Duration.EndAt != "2026-12-31" {
		t.Errorf("expected the data envelope to be unwrapped, got %+v", campaign)
	}
}

func TestCreateCampaign_KeepsTheRawEnvelope(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, campaignBody)
	}))
	defer srv.Close()

	c, _ := newTestClient(t, srv, testProfile())

	campaign, err := c.CreateCampaign(t.Context(), map[string]any{})
	if err != nil {
		t.Fatalf("CreateCampaign: %v", err)
	}

	replayed, err := json.Marshal(campaign)
	if err != nil {
		t.Fatalf("marshal campaign: %v", err)
	}

	if !strings.Contains(string(replayed), `"data"`) {
		t.Errorf("expected --json to replay the whole response, got %s", replayed)
	}
}

func TestCreateCampaign_ErrorIsWrapped(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = io.WriteString(w, `{"code":"BAD_REQUEST","message":"nope"}`)
	}))
	defer srv.Close()

	c, _ := newTestClient(t, srv, testProfile())

	if _, err := c.CreateCampaign(t.Context(), map[string]any{}); err == nil ||
		!strings.Contains(err.Error(), "create campaign") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestListCampaigns_WalksTheOffset(t *testing.T) {
	var gotQueries []string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQueries = append(gotQueries, r.URL.RawQuery)

		if r.URL.Query().Get("offset") == "0" {
			_, _ = io.WriteString(w, `{"data":[{"id":"cp-1"},{"id":"cp-2"}],
				"meta":{"total":3,"limit":2,"offset":0}}`)

			return
		}

		_, _ = io.WriteString(w, `{"data":[{"id":"cp-3"}],"meta":{"total":3,"limit":2,"offset":2}}`)
	}))
	defer srv.Close()

	c, _ := newTestClient(t, srv, testProfile())

	campaigns, err := c.ListCampaigns(t.Context(), nil, 0, 2)
	if err != nil {
		t.Fatalf("ListCampaigns: %v", err)
	}

	if len(campaigns) != 3 || campaigns[2].ID != "cp-3" {
		t.Errorf("unexpected campaigns: %+v", campaigns)
	}

	if len(gotQueries) != 2 || !strings.Contains(gotQueries[1], "offset=2") {
		t.Errorf("expected the second request to skip two items, got %v", gotQueries)
	}
}

func TestListCampaigns_ErrorIsWrapped(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = io.WriteString(w, `{"code":"FORBIDDEN","message":"nope"}`)
	}))
	defer srv.Close()

	c, _ := newTestClient(t, srv, testProfile())

	if _, err := c.ListCampaigns(t.Context(), nil, 0, 0); err == nil ||
		!strings.Contains(err.Error(), "list campaigns") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestGetCampaign_EscapesTheID(t *testing.T) {
	var gotMethod, gotPath string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod, gotPath = r.Method, r.URL.EscapedPath()

		_, _ = io.WriteString(w, campaignBody)
	}))
	defer srv.Close()

	c, _ := newTestClient(t, srv, testProfile())

	campaign, err := c.GetCampaign(t.Context(), "cp 1")
	if err != nil {
		t.Fatalf("GetCampaign: %v", err)
	}

	if gotMethod != http.MethodGet || gotPath != "/campaigns/cp%201" {
		t.Errorf("expected GET /campaigns/cp%%201, got %s %s", gotMethod, gotPath)
	}

	if campaign.Name != "Recovery CO" {
		t.Errorf("unexpected campaign: %+v", campaign)
	}
}

func TestGetCampaign_ErrorIsWrapped(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = io.WriteString(w, `{"code":"NOT_FOUND","message":"nope"}`)
	}))
	defer srv.Close()

	c, _ := newTestClient(t, srv, testProfile())

	if _, err := c.GetCampaign(t.Context(), "cp-1"); err == nil ||
		!strings.Contains(err.Error(), "get campaign cp-1") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestUpdateCampaign_UsesPatch(t *testing.T) {
	var gotMethod, gotPath string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod, gotPath = r.Method, r.URL.Path

		_, _ = io.WriteString(w, campaignBody)
	}))
	defer srv.Close()

	c, _ := newTestClient(t, srv, testProfile())

	if _, err := c.UpdateCampaign(t.Context(), "cp-1", map[string]any{"status": "PAUSED"}); err != nil {
		t.Fatalf("UpdateCampaign: %v", err)
	}

	if gotMethod != http.MethodPatch || gotPath != "/campaigns/cp-1" {
		t.Errorf("expected PATCH /campaigns/cp-1, got %s %s", gotMethod, gotPath)
	}
}

func TestUpdateCampaign_ErrorIsWrapped(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusConflict)
		_, _ = io.WriteString(w, `{"code":"CONFLICT","message":"nope"}`)
	}))
	defer srv.Close()

	c, _ := newTestClient(t, srv, testProfile())

	if _, err := c.UpdateCampaign(t.Context(), "cp-1", map[string]any{}); err == nil ||
		!strings.Contains(err.Error(), "update campaign cp-1") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestCreateCampaignRules_ReturnsEveryCreatedRule(t *testing.T) {
	var gotMethod, gotPath string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod, gotPath = r.Method, r.URL.Path

		w.WriteHeader(http.StatusCreated)
		_, _ = io.WriteString(w, `{"data":[{"id":"rl-1","rule_type":"PAYMENT_METHOD"},
			{"id":"rl-2","rule_type":"METADATA","metadata_key":"tier"}]}`)
	}))
	defer srv.Close()

	c, _ := newTestClient(t, srv, testProfile())

	rules, err := c.CreateCampaignRules(t.Context(), "cp-1", map[string]any{"rules": []any{}})
	if err != nil {
		t.Fatalf("CreateCampaignRules: %v", err)
	}

	if gotMethod != http.MethodPost || gotPath != "/campaigns/cp-1/rules" {
		t.Errorf("expected POST /campaigns/cp-1/rules, got %s %s", gotMethod, gotPath)
	}

	if len(rules) != 2 || rules[1].MetadataKey != "tier" {
		t.Errorf("unexpected rules: %+v", rules)
	}
}

func TestCreateCampaignRules_ErrorIsWrapped(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = io.WriteString(w, `{"code":"BAD_REQUEST","message":"nope"}`)
	}))
	defer srv.Close()

	c, _ := newTestClient(t, srv, testProfile())

	if _, err := c.CreateCampaignRules(t.Context(), "cp-1", map[string]any{}); err == nil ||
		!strings.Contains(err.Error(), "create rules of campaign cp-1") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestGetCampaignRule_EscapesBothIDs(t *testing.T) {
	var gotMethod, gotPath string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod, gotPath = r.Method, r.URL.EscapedPath()

		_, _ = io.WriteString(w, campaignRuleBody)
	}))
	defer srv.Close()

	c, _ := newTestClient(t, srv, testProfile())

	rule, err := c.GetCampaignRule(t.Context(), "cp 1", "rl 1")
	if err != nil {
		t.Fatalf("GetCampaignRule: %v", err)
	}

	if gotMethod != http.MethodGet || gotPath != "/campaigns/cp%201/rules/rl%201" {
		t.Errorf("expected both ids escaped, got %s %s", gotMethod, gotPath)
	}

	if rule.ID != "rl-1" || len(rule.Values) != 2 {
		t.Errorf("unexpected rule: %+v", rule)
	}
}

func TestGetCampaignRule_ErrorIsWrapped(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = io.WriteString(w, `{"code":"NOT_FOUND","message":"nope"}`)
	}))
	defer srv.Close()

	c, _ := newTestClient(t, srv, testProfile())

	if _, err := c.GetCampaignRule(t.Context(), "cp-1", "rl-1"); err == nil ||
		!strings.Contains(err.Error(), "get rule rl-1 of campaign cp-1") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestUpdateCampaignRule_UsesPatch(t *testing.T) {
	var gotMethod, gotPath string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod, gotPath = r.Method, r.URL.Path

		_, _ = io.WriteString(w, campaignRuleBody)
	}))
	defer srv.Close()

	c, _ := newTestClient(t, srv, testProfile())

	if _, err := c.UpdateCampaignRule(t.Context(), "cp-1", "rl-1", map[string]any{"conditional": "IN"}); err != nil {
		t.Fatalf("UpdateCampaignRule: %v", err)
	}

	if gotMethod != http.MethodPatch || gotPath != "/campaigns/cp-1/rules/rl-1" {
		t.Errorf("expected PATCH /campaigns/cp-1/rules/rl-1, got %s %s", gotMethod, gotPath)
	}
}

func TestUpdateCampaignRule_ErrorIsWrapped(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = io.WriteString(w, `{"code":"BAD_REQUEST","message":"nope"}`)
	}))
	defer srv.Close()

	c, _ := newTestClient(t, srv, testProfile())

	if _, err := c.UpdateCampaignRule(t.Context(), "cp-1", "rl-1", map[string]any{}); err == nil ||
		!strings.Contains(err.Error(), "update rule rl-1 of campaign cp-1") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestUpdateCampaignRuleStatus_HitsTheStatusPath(t *testing.T) {
	var gotMethod, gotPath string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod, gotPath = r.Method, r.URL.Path

		_, _ = io.WriteString(w, campaignRuleBody)
	}))
	defer srv.Close()

	c, _ := newTestClient(t, srv, testProfile())

	rule, err := c.UpdateCampaignRuleStatus(t.Context(), "cp-1", "rl-1", map[string]any{"status": "INACTIVE"})
	if err != nil {
		t.Fatalf("UpdateCampaignRuleStatus: %v", err)
	}

	if gotMethod != http.MethodPatch || gotPath != "/campaigns/cp-1/rules/rl-1/status" {
		t.Errorf("expected PATCH /campaigns/cp-1/rules/rl-1/status, got %s %s", gotMethod, gotPath)
	}

	if rule.Status != "ACTIVE" {
		t.Errorf("unexpected rule: %+v", rule)
	}
}

func TestUpdateCampaignRuleStatus_ErrorIsWrapped(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = io.WriteString(w, `{"code":"NOT_FOUND","message":"nope"}`)
	}))
	defer srv.Close()

	c, _ := newTestClient(t, srv, testProfile())

	if _, err := c.UpdateCampaignRuleStatus(t.Context(), "cp-1", "rl-1", map[string]any{}); err == nil ||
		!strings.Contains(err.Error(), "update status of rule rl-1 of campaign cp-1") {
		t.Errorf("unexpected error: %v", err)
	}
}
