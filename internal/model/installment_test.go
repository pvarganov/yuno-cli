package model

import (
	"encoding/json"
	"testing"
)

func TestInstallmentPlanView_FlattensThePlan(t *testing.T) {
	t.Parallel()

	var plan InstallmentPlan
	if err := json.Unmarshal([]byte(`{"id":"ip-1","name":"plan_007","account_id":["acc-1","acc-2"],
		"merchant_reference":"ref-1","country_code":"US",
		"installments_plan":[{"installment":3,"rate":1.2},{"installment":6}],
		"amount":{"Currency":"USD","min_value":0,"max_value":100000},
		"created_at":"2026-09-16T10:00:00Z"}`), &plan); err != nil {
		t.Fatalf("decode plan: %v", err)
	}

	view := plan.View()

	if view.Accounts != "acc-1,acc-2" {
		t.Errorf("unexpected accounts: %q", view.Accounts)
	}

	if view.Installments != "3,6" {
		t.Errorf("unexpected installments: %q", view.Installments)
	}

	if view.AmountRange != "USD 0-100000" {
		t.Errorf("unexpected amount range: %q", view.AmountRange)
	}
}

func TestInstallmentPlanMarshal_ReplaysTheRawResponse(t *testing.T) {
	t.Parallel()

	raw := `{"id":"ip-1","unknown_field":"kept"}`

	var plan InstallmentPlan
	if err := json.Unmarshal([]byte(raw), &plan); err != nil {
		t.Fatalf("decode plan: %v", err)
	}

	encoded, err := json.Marshal(plan)
	if err != nil {
		t.Fatalf("encode plan: %v", err)
	}

	if string(encoded) != raw {
		t.Errorf("expected the raw response back, got %s", encoded)
	}
}

func TestDecodeInstallmentPlans(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		payload string
		want    int
		wantErr bool
	}{
		"object":  {payload: `{"id":"ip-1"}`, want: 1},
		"array":   {payload: `[{"id":"ip-1"},{"id":"ip-2"}]`, want: 2},
		"empty":   {payload: ``, want: 0},
		"null":    {payload: `null`, want: 0},
		"broken":  {payload: `{"id":`, wantErr: true},
		"notplan": {payload: `["ip-1"]`, wantErr: true},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			plans, err := DecodeInstallmentPlans([]byte(tc.payload))
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected an error, got %+v", plans)
				}

				return
			}

			if err != nil {
				t.Fatalf("DecodeInstallmentPlans: %v", err)
			}

			if len(plans) != tc.want {
				t.Errorf("expected %d plans, got %d", tc.want, len(plans))
			}
		})
	}
}

func TestInstallmentAmountLabel(t *testing.T) {
	t.Parallel()

	low, high := 10.0, 20.0

	tests := map[string]struct {
		amount *InstallmentAmount
		want   string
	}{
		"nil":            {amount: nil, want: ""},
		"currency only":  {amount: &InstallmentAmount{Currency: "USD"}, want: "USD"},
		"range":          {amount: &InstallmentAmount{Currency: "USD", MinValue: &low, MaxValue: &high}, want: "USD 10-20"},
		"no currency":    {amount: &InstallmentAmount{MinValue: &low, MaxValue: &high}, want: "10-20"},
		"capital C key":  {amount: &InstallmentAmount{CurrencyAlt: "BRL"}, want: "BRL"},
		"only max value": {amount: &InstallmentAmount{Currency: "USD", MaxValue: &high}, want: "USD 0-20"},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			if got := tc.amount.Label(); got != tc.want {
				t.Errorf("expected %q, got %q", tc.want, got)
			}
		})
	}
}

func TestInstallmentPlanViews_FlattensTheList(t *testing.T) {
	t.Parallel()

	views := InstallmentPlanViews([]InstallmentPlan{{ID: "ip-1"}, {ID: "ip-2"}})
	if len(views) != 2 || views[1].ID != "ip-2" {
		t.Errorf("unexpected views: %+v", views)
	}
}
