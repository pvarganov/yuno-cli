package cmd_test

import (
	"net/http"
	"strings"
	"testing"
)

// Banking payloads the fake API returns.
const (
	bankingEntityResponse = `{"id":"be-1","account_id":"acc-1","merchant_entity_id":"me-1",
		"national_entity":"US","entity_detail":{"entity_type":"BUSINESS","legal_name":"Acme Inc"},
		"created_at":"2026-09-16T10:00:00Z"}`
	bankingOnboardingResponse = `{"id":"bo-1","entity_id":"be-1","provider":"COLUMN",
		"onboarding_type":"BANK_ACCOUNT","status":"PENDING","expires_at":"2026-10-16T10:00:00Z"}`
	bankingAccountResponse = `{"id":"ba-1","entity_id":"be-1","onboarding_id":"bo-1","account_id":"acc-1",
		"provider":"COLUMN","account_type":"CHECKING","status":"ACTIVE","currency":"USD",
		"balance":{"currency":"USD","value":1000}}`
	bankingTransferResponse = `{"id":"bt-1","source_account_id":"ba-1","account_id":"acc-1",
		"status":"PENDING","direction":"OUTBOUND","payment_rail":"ACH",
		"amount":{"currency":"USD","value":250}}`
)

// bankingCase is one banking command mapped onto the request it must produce.
type bankingCase struct {
	name       string
	args       []string
	response   string
	wantMethod string
	wantPath   string
	wantBody   map[string]any
	wantOutput []string
}

// runBankingCommands drives a table of banking command cases.
func runBankingCommands(t *testing.T, cases []bankingCase) {
	t.Helper()

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			isolateConfig(t)
			seedCredentials(t)

			got := startAPI(t, http.StatusOK, tc.response)

			out, err := runCLI(t, "", append(tc.args, "--yes")...)
			if err != nil {
				t.Fatalf("%s failed: %v (%s)", tc.name, err, out)
			}

			if got.Method != tc.wantMethod || got.Path != tc.wantPath {
				t.Errorf("expected %s %s, got %s %s", tc.wantMethod, tc.wantPath, got.Method, got.Path)
			}

			if len(tc.wantBody) > 0 {
				body := decodeBody(t, got.Body)
				for key, want := range tc.wantBody {
					if body[key] != want {
						t.Errorf("body %s: expected %v, got %v", key, want, body[key])
					}
				}
			}

			for _, want := range tc.wantOutput {
				if !strings.Contains(out, want) {
					t.Errorf("expected output to contain %q, got:\n%s", want, out)
				}
			}
		})
	}
}

func TestBankingEntityCommands(t *testing.T) {
	runBankingCommands(t, []bankingCase{
		{
			name: "create",
			args: []string{
				"banking", "entity", "create", "--account-id", "acc-1",
				"--merchant-entity-id", "me-1", "--national-entity", "US",
				"--entity-detail", `{"entity_type":"BUSINESS","legal_name":"Acme Inc"}`,
			},
			response:   bankingEntityResponse,
			wantMethod: http.MethodPost,
			wantPath:   "/v1/banking/entities",
			wantBody:   map[string]any{"merchant_entity_id": "me-1", "national_entity": "US"},
			wantOutput: []string{"be-1", "Acme Inc"},
		},
		{
			name:       "get",
			args:       []string{"banking", "entity", "get", "be-1"},
			response:   bankingEntityResponse,
			wantMethod: http.MethodGet,
			wantPath:   "/v1/banking/entities/be-1",
			wantOutput: []string{"be-1", "me-1"},
		},
		{
			name:       "update",
			args:       []string{"banking", "entity", "update", "be-1", "--account-id", "acc-1"},
			response:   bankingEntityResponse,
			wantMethod: http.MethodPatch,
			wantPath:   "/v1/banking/entities/be-1",
			wantBody:   map[string]any{"account_id": "acc-1"},
		},
		{
			name: "onboarding create",
			args: []string{
				"banking", "entity", "onboarding", "create", "be-1",
				"--account-id", "acc-1", "--yuno-connection-id", "conn-1",
				"--onboarding-type", "BANK_ACCOUNT",
			},
			response:   bankingOnboardingResponse,
			wantMethod: http.MethodPost,
			wantPath:   "/v1/banking/entities/be-1/onboardings",
			wantBody:   map[string]any{"yuno_connection_id": "conn-1", "onboarding_type": "BANK_ACCOUNT"},
			wantOutput: []string{"bo-1", "PENDING"},
		},
		{
			name:       "onboarding get",
			args:       []string{"banking", "entity", "onboarding", "get", "be-1", "bo-1"},
			response:   bankingOnboardingResponse,
			wantMethod: http.MethodGet,
			wantPath:   "/v1/banking/entities/be-1/onboardings/bo-1",
		},
		{
			name: "onboarding update",
			args: []string{
				"banking", "entity", "onboarding", "update", "be-1", "bo-1",
				"--account-id", "acc-1", "--documentation", `[{"type":"ID"}]`,
			},
			response:   bankingOnboardingResponse,
			wantMethod: http.MethodPatch,
			wantPath:   "/v1/banking/entities/be-1/onboardings/bo-1",
			wantBody:   map[string]any{"account_id": "acc-1"},
		},
		{
			name:       "onboarding cancel",
			args:       []string{"banking", "entity", "onboarding", "cancel", "be-1", "bo-1"},
			response:   bankingOnboardingResponse,
			wantMethod: http.MethodPost,
			wantPath:   "/v1/banking/entities/be-1/onboardings/bo-1/cancel",
		},
	})
}

func TestBankingAccountAndTransferCommands(t *testing.T) {
	runBankingCommands(t, []bankingCase{
		{
			name: "account create",
			args: []string{
				"banking", "account", "create", "--account-id", "acc-1",
				"--onboarding-id", "bo-1", "--account-type", "CHECKING", "--currency", "USD",
			},
			response:   bankingAccountResponse,
			wantMethod: http.MethodPost,
			wantPath:   "/v1/banking/accounts",
			wantBody:   map[string]any{"onboarding_id": "bo-1", "currency": "USD"},
			wantOutput: []string{"ba-1", "USD 1000"},
		},
		{
			name:       "account get",
			args:       []string{"banking", "account", "get", "ba-1"},
			response:   bankingAccountResponse,
			wantMethod: http.MethodGet,
			wantPath:   "/v1/banking/accounts/ba-1",
		},
		{
			name:       "account update",
			args:       []string{"banking", "account", "update", "ba-1", "--account-type", "SAVINGS"},
			response:   bankingAccountResponse,
			wantMethod: http.MethodPatch,
			wantPath:   "/v1/banking/accounts/ba-1",
			wantBody:   map[string]any{"account_type": "SAVINGS"},
		},
		{
			name:       "account close",
			args:       []string{"banking", "account", "close", "ba-1"},
			response:   bankingAccountResponse,
			wantMethod: http.MethodDelete,
			wantPath:   "/v1/banking/accounts/ba-1",
		},
		{
			name: "transfer create",
			args: []string{
				"banking", "transfer", "create", "--account-id", "acc-1",
				"--source-account-id", "ba-1", "--direction", "OUTBOUND", "--payment-rail", "ACH",
				"--amount", `{"value":250,"currency":"USD"}`,
				"--destination-account", `{"account_number":"9876543210"}`,
			},
			response:   bankingTransferResponse,
			wantMethod: http.MethodPost,
			wantPath:   "/v1/banking/transfers",
			wantBody:   map[string]any{"source_account_id": "ba-1", "payment_rail": "ACH"},
			wantOutput: []string{"bt-1", "USD 250"},
		},
		{
			name:       "transfer get",
			args:       []string{"banking", "transfer", "get", "ba-1", "bt-1"},
			response:   bankingTransferResponse,
			wantMethod: http.MethodGet,
			wantPath:   "/v1/banking/accounts/ba-1/transfers/bt-1",
		},
		{
			name:       "transfer cancel",
			args:       []string{"banking", "transfer", "cancel", "ba-1", "bt-1"},
			response:   bankingTransferResponse,
			wantMethod: http.MethodPost,
			wantPath:   "/v1/banking/accounts/ba-1/transfers/bt-1/cancel",
		},
	})
}

func TestBankingEntityCreate_RequiresABody(t *testing.T) {
	isolateConfig(t)
	seedCredentials(t)
	startAPI(t, http.StatusOK, bankingEntityResponse)

	if _, err := runCLI(t, "", "banking", "entity", "create", "--yes"); err == nil ||
		!strings.Contains(err.Error(), "request body is required") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestBankingTransferCreate_RejectsInvalidJSONFlags(t *testing.T) {
	isolateConfig(t)
	seedCredentials(t)
	startAPI(t, http.StatusOK, bankingTransferResponse)

	_, err := runCLI(t, "", "banking", "transfer", "create", "--yes", "--amount", "{")
	if err == nil || !strings.Contains(err.Error(), "not valid json") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestBankingAccountGet_ReportsAPIErrors(t *testing.T) {
	isolateConfig(t)
	seedCredentials(t)
	startAPI(t, http.StatusNotFound, `{"code":"NOT_FOUND","message":"no such account"}`)

	_, err := runCLI(t, "", "banking", "account", "get", "ba-1")
	if err == nil || !strings.Contains(err.Error(), "get banking account ba-1") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestBankingEntityGet_ScopeHintOnForbidden(t *testing.T) {
	isolateConfig(t)
	seedCredentials(t)
	startAPI(t, http.StatusForbidden, `{"code":"FORBIDDEN","message":"nope"}`)

	_, err := runCLI(t, "", "banking", "entity", "get", "be-1")
	if err == nil || !strings.Contains(err.Error(), "banking:read") {
		t.Errorf("unexpected error: %v", err)
	}
}
