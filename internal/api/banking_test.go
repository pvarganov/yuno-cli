package api

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// Banking payloads the fake API answers with.
const (
	bankingEntityBody = `{"id":"be-1","account_id":"acc-1","merchant_entity_id":"me-1",
		"national_entity":"US","entity_detail":{"entity_type":"BUSINESS","legal_name":"Acme Inc"},
		"created_at":"2026-09-16T10:00:00Z"}`
	bankingOnboardingBody = `{"id":"bo-1","entity_id":"be-1","provider":"COLUMN",
		"onboarding_type":"BANK_ACCOUNT","status":"PENDING","expires_at":"2026-10-16T10:00:00Z"}`
	bankingAccountBody = `{"id":"ba-1","entity_id":"be-1","onboarding_id":"bo-1","account_id":"acc-1",
		"provider":"COLUMN","account_type":"CHECKING","status":"ACTIVE","currency":"USD",
		"balance":{"currency":"USD","value":1000}}`
	bankingTransferBody = `{"id":"bt-1","source_account_id":"ba-1","account_id":"acc-1",
		"status":"PENDING","direction":"OUTBOUND","payment_rail":"ACH",
		"amount":{"currency":"USD","value":250}}`
)

// captureBanking starts a fake API that records the request, answers with the
// given body and returns a client wired to it.
func captureBanking(t *testing.T, status int, response string) (c *Client, method, path *string) {
	t.Helper()

	var gotMethod, gotPath string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod, gotPath = r.Method, r.URL.EscapedPath()

		w.WriteHeader(status)
		_, _ = io.WriteString(w, response)
	}))
	t.Cleanup(srv.Close)

	client, _ := newTestClient(t, srv, testProfile())

	return client, &gotMethod, &gotPath
}

func TestBankingEntity_Endpoints(t *testing.T) {
	tests := []struct {
		name       string
		response   string
		call       func(c *Client) error
		wantMethod string
		wantPath   string
	}{
		{
			name:     "create",
			response: bankingEntityBody,
			call: func(c *Client) error {
				entity, err := c.CreateBankingEntity(t.Context(), map[string]any{"merchant_entity_id": "me-1"})
				if err == nil && entity.ID != "be-1" {
					t.Errorf("unexpected entity: %+v", entity)
				}

				return err
			},
			wantMethod: http.MethodPost,
			wantPath:   "/banking/entities",
		},
		{
			name:     "get",
			response: bankingEntityBody,
			call: func(c *Client) error {
				_, err := c.GetBankingEntity(t.Context(), "be 1")

				return err
			},
			wantMethod: http.MethodGet,
			wantPath:   "/banking/entities/be%201",
		},
		{
			name:     "update",
			response: bankingEntityBody,
			call: func(c *Client) error {
				_, err := c.UpdateBankingEntity(t.Context(), "be-1", map[string]any{"account_id": "acc-1"})

				return err
			},
			wantMethod: http.MethodPatch,
			wantPath:   "/banking/entities/be-1",
		},
		{
			name:     "create onboarding",
			response: bankingOnboardingBody,
			call: func(c *Client) error {
				_, err := c.CreateBankingOnboarding(t.Context(), "be-1", map[string]any{"account_id": "acc-1"})

				return err
			},
			wantMethod: http.MethodPost,
			wantPath:   "/banking/entities/be-1/onboardings",
		},
		{
			name:     "get onboarding",
			response: bankingOnboardingBody,
			call: func(c *Client) error {
				_, err := c.GetBankingOnboarding(t.Context(), "be-1", "bo-1")

				return err
			},
			wantMethod: http.MethodGet,
			wantPath:   "/banking/entities/be-1/onboardings/bo-1",
		},
		{
			name:     "update onboarding",
			response: bankingOnboardingBody,
			call: func(c *Client) error {
				_, err := c.UpdateBankingOnboarding(t.Context(), "be-1", "bo-1", map[string]any{"account_id": "acc-1"})

				return err
			},
			wantMethod: http.MethodPatch,
			wantPath:   "/banking/entities/be-1/onboardings/bo-1",
		},
		{
			name:     "cancel onboarding",
			response: bankingOnboardingBody,
			call: func(c *Client) error {
				_, err := c.CancelBankingOnboarding(t.Context(), "be-1", "bo-1")

				return err
			},
			wantMethod: http.MethodPost,
			wantPath:   "/banking/entities/be-1/onboardings/bo-1/cancel",
		},
	}

	runBankingCases(t, tests)
}

func TestBankingAccountAndTransfer_Endpoints(t *testing.T) {
	tests := []struct {
		name       string
		response   string
		call       func(c *Client) error
		wantMethod string
		wantPath   string
	}{
		{
			name:     "create account",
			response: bankingAccountBody,
			call: func(c *Client) error {
				account, err := c.CreateBankingAccount(t.Context(), map[string]any{"account_id": "acc-1"})
				if err == nil && account.Currency != "USD" {
					t.Errorf("unexpected account: %+v", account)
				}

				return err
			},
			wantMethod: http.MethodPost,
			wantPath:   "/banking/accounts",
		},
		{
			name:     "get account",
			response: bankingAccountBody,
			call: func(c *Client) error {
				_, err := c.GetBankingAccount(t.Context(), "ba-1")

				return err
			},
			wantMethod: http.MethodGet,
			wantPath:   "/banking/accounts/ba-1",
		},
		{
			name:     "update account",
			response: bankingAccountBody,
			call: func(c *Client) error {
				_, err := c.UpdateBankingAccount(t.Context(), "ba-1", map[string]any{"account_type": "CHECKING"})

				return err
			},
			wantMethod: http.MethodPatch,
			wantPath:   "/banking/accounts/ba-1",
		},
		{
			name:     "close account",
			response: bankingAccountBody,
			call: func(c *Client) error {
				_, err := c.CloseBankingAccount(t.Context(), "ba-1")

				return err
			},
			wantMethod: http.MethodDelete,
			wantPath:   "/banking/accounts/ba-1",
		},
		{
			name:     "create transfer",
			response: bankingTransferBody,
			call: func(c *Client) error {
				transfer, err := c.CreateBankingTransfer(t.Context(), map[string]any{"source_account_id": "ba-1"})
				if err == nil && transfer.Direction != "OUTBOUND" {
					t.Errorf("unexpected transfer: %+v", transfer)
				}

				return err
			},
			wantMethod: http.MethodPost,
			wantPath:   "/banking/transfers",
		},
		{
			name:     "get transfer",
			response: bankingTransferBody,
			call: func(c *Client) error {
				_, err := c.GetBankingTransfer(t.Context(), "ba-1", "bt-1")

				return err
			},
			wantMethod: http.MethodGet,
			wantPath:   "/banking/accounts/ba-1/transfers/bt-1",
		},
		{
			name:     "cancel transfer",
			response: bankingTransferBody,
			call: func(c *Client) error {
				_, err := c.CancelBankingTransfer(t.Context(), "ba-1", "bt-1")

				return err
			},
			wantMethod: http.MethodPost,
			wantPath:   "/banking/accounts/ba-1/transfers/bt-1/cancel",
		},
	}

	runBankingCases(t, tests)
}

// runBankingCases drives one table of banking endpoint cases.
func runBankingCases(t *testing.T, tests []struct {
	name       string
	response   string
	call       func(c *Client) error
	wantMethod string
	wantPath   string
},
) {
	t.Helper()

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			c, method, path := captureBanking(t, http.StatusOK, tc.response)

			if err := tc.call(c); err != nil {
				t.Fatalf("%s: %v", tc.name, err)
			}

			if *method != tc.wantMethod || *path != tc.wantPath {
				t.Errorf("expected %s %s, got %s %s", tc.wantMethod, tc.wantPath, *method, *path)
			}
		})
	}
}

func TestBanking_ErrorsAreWrapped(t *testing.T) {
	tests := []struct {
		name string
		call func(c *Client) error
		want string
	}{
		{
			name: "create entity",
			call: func(c *Client) error {
				_, err := c.CreateBankingEntity(t.Context(), map[string]any{})

				return err
			},
			want: "create banking entity",
		},
		{
			name: "get entity",
			call: func(c *Client) error {
				_, err := c.GetBankingEntity(t.Context(), "be-1")

				return err
			},
			want: "get banking entity be-1",
		},
		{
			name: "update entity",
			call: func(c *Client) error {
				_, err := c.UpdateBankingEntity(t.Context(), "be-1", map[string]any{})

				return err
			},
			want: "update banking entity be-1",
		},
		{
			name: "create onboarding",
			call: func(c *Client) error {
				_, err := c.CreateBankingOnboarding(t.Context(), "be-1", map[string]any{})

				return err
			},
			want: "create onboarding of banking entity be-1",
		},
		{
			name: "get onboarding",
			call: func(c *Client) error {
				_, err := c.GetBankingOnboarding(t.Context(), "be-1", "bo-1")

				return err
			},
			want: "get banking onboarding bo-1",
		},
		{
			name: "update onboarding",
			call: func(c *Client) error {
				_, err := c.UpdateBankingOnboarding(t.Context(), "be-1", "bo-1", map[string]any{})

				return err
			},
			want: "update banking onboarding bo-1",
		},
		{
			name: "cancel onboarding",
			call: func(c *Client) error {
				_, err := c.CancelBankingOnboarding(t.Context(), "be-1", "bo-1")

				return err
			},
			want: "cancel banking onboarding bo-1",
		},
		{
			name: "create account",
			call: func(c *Client) error {
				_, err := c.CreateBankingAccount(t.Context(), map[string]any{})

				return err
			},
			want: "create banking account",
		},
		{
			name: "get account",
			call: func(c *Client) error {
				_, err := c.GetBankingAccount(t.Context(), "ba-1")

				return err
			},
			want: "get banking account ba-1",
		},
		{
			name: "update account",
			call: func(c *Client) error {
				_, err := c.UpdateBankingAccount(t.Context(), "ba-1", map[string]any{})

				return err
			},
			want: "update banking account ba-1",
		},
		{
			name: "close account",
			call: func(c *Client) error {
				_, err := c.CloseBankingAccount(t.Context(), "ba-1")

				return err
			},
			want: "close banking account ba-1",
		},
		{
			name: "create transfer",
			call: func(c *Client) error {
				_, err := c.CreateBankingTransfer(t.Context(), map[string]any{})

				return err
			},
			want: "create banking transfer",
		},
		{
			name: "get transfer",
			call: func(c *Client) error {
				_, err := c.GetBankingTransfer(t.Context(), "ba-1", "bt-1")

				return err
			},
			want: "get banking transfer bt-1",
		},
		{
			name: "cancel transfer",
			call: func(c *Client) error {
				_, err := c.CancelBankingTransfer(t.Context(), "ba-1", "bt-1")

				return err
			},
			want: "cancel banking transfer bt-1",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			c, _, _ := captureBanking(t, http.StatusUnprocessableEntity, `{"code":"INVALID","message":"nope"}`)

			err := tc.call(c)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}
