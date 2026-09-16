package model

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

// InstallmentPlan is an installments plan: the set of installment options a
// shopper may pick for a given account, brand and country. Only the fields the
// CLI renders are typed; the untouched response is kept in Raw so `--json`
// never loses a field the spec adds later (issuer, iin, financial costs).
type InstallmentPlan struct {
	ID                       string              `json:"id"`
	Name                     string              `json:"name,omitempty"`
	AccountID                []string            `json:"account_id,omitempty"`
	MerchantReference        string              `json:"merchant_reference,omitempty"`
	CountryCode              string              `json:"country_code,omitempty"`
	Brand                    []string            `json:"brand,omitempty"`
	Issuer                   []string            `json:"issuer,omitempty"`
	PaymentMethodType        string              `json:"payment_method_type,omitempty"`
	FirstInstallmentDeferral *int                `json:"first_installment_deferral,omitempty"`
	InstallmentsPlan         []InstallmentOption `json:"installments_plan,omitempty"`
	Amount                   *InstallmentAmount  `json:"amount,omitempty"`
	Availability             *Availability       `json:"availability,omitempty"`
	CreatedAt                string              `json:"created_at,omitempty"`
	UpdatedAt                string              `json:"updated_at,omitempty"`

	// Raw is the verbatim response body, set when the plan was decoded from one.
	Raw json.RawMessage `json:"-"`
}

// InstallmentOption is one installment choice of a plan, e.g. 3 installments
// at a 1.2 rate.
type InstallmentOption struct {
	Installment int     `json:"installment,omitempty"`
	Rate        float64 `json:"rate,omitempty"`
	Type        string  `json:"type,omitempty"`
	ProviderID  string  `json:"provider_id,omitempty"`
}

// InstallmentAmount is the amount range a plan applies to. Yuno spells the
// currency key with a capital C on some responses, hence the second field.
type InstallmentAmount struct {
	Currency    string   `json:"currency,omitempty"`
	CurrencyAlt string   `json:"Currency,omitempty"`
	MinValue    *float64 `json:"min_value,omitempty"`
	MaxValue    *float64 `json:"max_value,omitempty"`
}

// Availability is the time window a plan or a payment link is usable in.
type Availability struct {
	StartAt  string `json:"start_at,omitempty"`
	FinishAt string `json:"finish_at,omitempty"`
}

// UnmarshalJSON decodes the typed fields and keeps the original body.
func (p *InstallmentPlan) UnmarshalJSON(data []byte) error {
	type alias InstallmentPlan

	var decoded alias
	if err := json.Unmarshal(data, &decoded); err != nil {
		return err
	}

	*p = InstallmentPlan(decoded)
	p.Raw = append(json.RawMessage(nil), data...)

	return nil
}

// MarshalJSON replays the original response when there is one.
//
//nolint:gocritic // hugeParam: a value receiver is required by json.Marshaler here
func (p InstallmentPlan) MarshalJSON() ([]byte, error) {
	if len(p.Raw) > 0 {
		return p.Raw, nil
	}

	type alias InstallmentPlan

	return json.Marshal(alias(p))
}

// DecodeInstallmentPlans decodes a plan response that Yuno returns either as a
// single object or as an array of them, depending on the endpoint.
func DecodeInstallmentPlans(data []byte) ([]InstallmentPlan, error) {
	trimmed := bytes.TrimSpace(data)
	if len(trimmed) == 0 || bytes.Equal(trimmed, []byte("null")) {
		return nil, nil
	}

	if trimmed[0] == '[' {
		var plans []InstallmentPlan
		if err := json.Unmarshal(trimmed, &plans); err != nil {
			return nil, fmt.Errorf("decode installment plans: %w", err)
		}

		return plans, nil
	}

	var plan InstallmentPlan
	if err := json.Unmarshal(trimmed, &plan); err != nil {
		return nil, fmt.Errorf("decode installment plan: %w", err)
	}

	return []InstallmentPlan{plan}, nil
}

// InstallmentPlanView is the flat table row of an installments plan.
type InstallmentPlanView struct {
	ID                string `json:"id"`
	Name              string `json:"name"`
	MerchantReference string `json:"merchant_reference"`
	CountryCode       string `json:"country_code"`
	Accounts          string `json:"account_id"`
	Installments      string `json:"installments"`
	AmountRange       string `json:"amount_range"`
	CreatedAt         string `json:"created_at"`
}

// View flattens an installments plan into a table row.
func (p *InstallmentPlan) View() InstallmentPlanView {
	return InstallmentPlanView{
		ID:                p.ID,
		Name:              p.Name,
		MerchantReference: p.MerchantReference,
		CountryCode:       p.CountryCode,
		Accounts:          strings.Join(p.AccountID, ","),
		Installments:      installmentsLabel(p.InstallmentsPlan),
		AmountRange:       p.Amount.Label(),
		CreatedAt:         p.CreatedAt,
	}
}

// InstallmentPlanViews flattens a list of installments plans.
func InstallmentPlanViews(plans []InstallmentPlan) []InstallmentPlanView {
	views := make([]InstallmentPlanView, 0, len(plans))
	for i := range plans {
		views = append(views, plans[i].View())
	}

	return views
}

// installmentsLabel renders the installment counts of a plan as `1,3,6`.
func installmentsLabel(options []InstallmentOption) string {
	counts := make([]string, 0, len(options))
	for _, o := range options {
		counts = append(counts, strconv.Itoa(o.Installment))
	}

	return strings.Join(counts, ",")
}

// Label renders an amount range as `USD 0-100000`, and an empty string when
// the plan does not restrict the amount.
func (a *InstallmentAmount) Label() string {
	if a == nil {
		return ""
	}

	currency := a.Currency
	if currency == "" {
		currency = a.CurrencyAlt
	}

	if a.MinValue == nil && a.MaxValue == nil {
		return currency
	}

	label := formatAmount(valueOrZero(a.MinValue)) + "-" + formatAmount(valueOrZero(a.MaxValue))
	if currency == "" {
		return label
	}

	return currency + " " + label
}

// valueOrZero dereferences an optional amount bound.
func valueOrZero(v *float64) float64 {
	if v == nil {
		return 0
	}

	return *v
}
