package model

import (
	"encoding/json"
	"strconv"
	"strings"
)

// Subscription is a Yuno subscription. Only the fields the CLI renders are
// typed; the untouched response is kept in Raw so `--json` never loses a field
// the spec adds later (phases, retries, pending_plan_change).
type Subscription struct {
	ID                 string         `json:"id"`
	Name               string         `json:"name,omitempty"`
	Description        string         `json:"description,omitempty"`
	AccountID          string         `json:"account_id,omitempty"`
	Country            string         `json:"country,omitempty"`
	MerchantReference  string         `json:"merchant_reference,omitempty"`
	Status             string         `json:"status,omitempty"`
	PlanID             string         `json:"plan_id,omitempty"`
	Amount             *Amount        `json:"amount,omitempty"`
	Frequency          *Frequency     `json:"frequency,omitempty"`
	BillingCycles      *BillingCycles `json:"billing_cycles,omitempty"`
	CustomerPayer      *CustomerPayer `json:"customer_payer,omitempty"`
	PaymentMethod      *PaymentMethod `json:"payment_method,omitempty"`
	CurrentPeriodStart string         `json:"current_period_start,omitempty"`
	CurrentPeriodEnd   string         `json:"current_period_end,omitempty"`
	CreatedAt          string         `json:"created_at,omitempty"`
	UpdatedAt          string         `json:"updated_at,omitempty"`

	// Raw is the verbatim response body, set when the subscription was decoded
	// from one.
	Raw json.RawMessage `json:"-"`
}

// Frequency is how often a subscription or a plan bills, e.g. 1 MONTH.
type Frequency struct {
	Type  string `json:"type,omitempty"`
	Value int    `json:"value,omitempty"`
}

// BillingCycles is how many cycles a subscription runs for.
type BillingCycles struct {
	Total *int `json:"total,omitempty"`
}

// CustomerPayer is the customer a subscription bills.
type CustomerPayer struct {
	ID string `json:"id,omitempty"`
}

// UnmarshalJSON decodes the typed fields and keeps the original body.
func (s *Subscription) UnmarshalJSON(data []byte) error {
	type alias Subscription

	var decoded alias
	if err := json.Unmarshal(data, &decoded); err != nil {
		return err
	}

	*s = Subscription(decoded)
	s.Raw = append(json.RawMessage(nil), data...)

	return nil
}

// MarshalJSON replays the original response when there is one, so `--json`
// shows every field Yuno sent and not only the typed ones.
// The receiver stays a value on purpose: json.Marshal only calls a pointer
// method on an addressable value, and a subscription copied out of a response
// is often not one.
//
//nolint:gocritic // hugeParam: a value receiver is required by json.Marshaler here
func (s Subscription) MarshalJSON() ([]byte, error) {
	if len(s.Raw) > 0 {
		return s.Raw, nil
	}

	type alias Subscription

	return json.Marshal(alias(s))
}

// SubscriptionView is the flat table row of a subscription.
type SubscriptionView struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Status     string `json:"status"`
	PlanID     string `json:"plan_id"`
	Amount     string `json:"amount"`
	Frequency  string `json:"frequency"`
	CustomerID string `json:"customer_id"`
	PeriodEnd  string `json:"current_period_end"`
	CreatedAt  string `json:"created_at"`
}

// View flattens a subscription into a table row.
func (s *Subscription) View() SubscriptionView {
	return SubscriptionView{
		ID:         s.ID,
		Name:       s.Name,
		Status:     s.Status,
		PlanID:     s.PlanID,
		Amount:     s.Amount.Label(),
		Frequency:  s.Frequency.Label(),
		CustomerID: s.CustomerPayer.Label(),
		PeriodEnd:  s.CurrentPeriodEnd,
		CreatedAt:  s.CreatedAt,
	}
}

// SubscriptionViews flattens a list of subscriptions.
func SubscriptionViews(subscriptions []Subscription) []SubscriptionView {
	views := make([]SubscriptionView, 0, len(subscriptions))
	for i := range subscriptions {
		views = append(views, subscriptions[i].View())
	}

	return views
}

// Label renders a frequency as `1 MONTH`, and an empty string when there is none.
func (f *Frequency) Label() string {
	if f == nil || f.Type == "" && f.Value == 0 {
		return ""
	}

	return strings.TrimSpace(strconv.Itoa(f.Value) + " " + f.Type)
}

// Label renders the id of the customer a subscription bills.
func (c *CustomerPayer) Label() string {
	if c == nil {
		return ""
	}

	return c.ID
}

// SubscriptionPayment is one payment of a subscription.
type SubscriptionPayment struct {
	ID           string  `json:"id"`
	PaymentID    string  `json:"payment_id,omitempty"`
	Status       string  `json:"status,omitempty"`
	SubStatus    string  `json:"sub_status,omitempty"`
	Provider     string  `json:"provider,omitempty"`
	Currency     string  `json:"currency,omitempty"`
	Amount       float64 `json:"amount,omitempty"`
	Total        float64 `json:"total,omitempty"`
	BillingCycle *int    `json:"billing_cycle,omitempty"`
	PlanID       string  `json:"plan_id,omitempty"`
	CreatedAt    string  `json:"created_at,omitempty"`
}

// SubscriptionPaymentView is the flat table row of a subscription payment.
type SubscriptionPaymentView struct {
	ID           string `json:"id"`
	PaymentID    string `json:"payment_id"`
	Status       string `json:"status"`
	Currency     string `json:"currency"`
	Amount       string `json:"amount"`
	BillingCycle string `json:"billing_cycle"`
	CreatedAt    string `json:"created_at"`
}

// View flattens a subscription payment into a table row.
func (p *SubscriptionPayment) View() SubscriptionPaymentView {
	cycle := ""
	if p.BillingCycle != nil {
		cycle = strconv.Itoa(*p.BillingCycle)
	}

	return SubscriptionPaymentView{
		ID:           p.ID,
		PaymentID:    p.PaymentID,
		Status:       p.Status,
		Currency:     p.Currency,
		Amount:       formatAmount(p.Amount),
		BillingCycle: cycle,
		CreatedAt:    p.CreatedAt,
	}
}

// SubscriptionPaymentViews flattens a list of subscription payments.
func SubscriptionPaymentViews(payments []SubscriptionPayment) []SubscriptionPaymentView {
	views := make([]SubscriptionPaymentView, 0, len(payments))
	for i := range payments {
		views = append(views, payments[i].View())
	}

	return views
}

// Plan is a subscription plan.
type Plan struct {
	ID                string     `json:"id"`
	AccountID         string     `json:"account_id,omitempty"`
	Name              string     `json:"name,omitempty"`
	Description       string     `json:"description,omitempty"`
	MerchantReference string     `json:"merchant_reference,omitempty"`
	Status            string     `json:"status,omitempty"`
	BaseAmount        *Amount    `json:"base_amount,omitempty"`
	Frequency         *Frequency `json:"frequency,omitempty"`
	SubscribersCount  *int       `json:"subscribers_count,omitempty"`
	Countries         []string   `json:"countries,omitempty"`
	CreatedAt         string     `json:"created_at,omitempty"`
	UpdatedAt         string     `json:"updated_at,omitempty"`

	// Raw is the verbatim response body, set when the plan was decoded from one.
	Raw json.RawMessage `json:"-"`
}

// UnmarshalJSON decodes the typed fields and keeps the original body.
func (p *Plan) UnmarshalJSON(data []byte) error {
	type alias Plan

	var decoded alias
	if err := json.Unmarshal(data, &decoded); err != nil {
		return err
	}

	*p = Plan(decoded)
	p.Raw = append(json.RawMessage(nil), data...)

	return nil
}

// MarshalJSON replays the original response when there is one.
//
//nolint:gocritic // hugeParam: a value receiver is required by json.Marshaler here
func (p Plan) MarshalJSON() ([]byte, error) {
	if len(p.Raw) > 0 {
		return p.Raw, nil
	}

	type alias Plan

	return json.Marshal(alias(p))
}

// PlanView is the flat table row of a plan.
type PlanView struct {
	ID                string `json:"id"`
	Name              string `json:"name"`
	Status            string `json:"status"`
	BaseAmount        string `json:"base_amount"`
	Frequency         string `json:"frequency"`
	SubscribersCount  string `json:"subscribers_count"`
	MerchantReference string `json:"merchant_reference"`
	CreatedAt         string `json:"created_at"`
}

// View flattens a plan into a table row.
func (p *Plan) View() PlanView {
	subscribers := ""
	if p.SubscribersCount != nil {
		subscribers = strconv.Itoa(*p.SubscribersCount)
	}

	return PlanView{
		ID:                p.ID,
		Name:              p.Name,
		Status:            p.Status,
		BaseAmount:        p.BaseAmount.Label(),
		Frequency:         p.Frequency.Label(),
		SubscribersCount:  subscribers,
		MerchantReference: p.MerchantReference,
		CreatedAt:         p.CreatedAt,
	}
}

// PlanViews flattens a list of plans.
func PlanViews(plans []Plan) []PlanView {
	views := make([]PlanView, 0, len(plans))
	for i := range plans {
		views = append(views, plans[i].View())
	}

	return views
}

// PlanStatus is the result of changing the status of a plan.
type PlanStatus struct {
	ID                    string `json:"id"`
	Status                string `json:"status,omitempty"`
	AffectedSubscriptions *int   `json:"affected_subscriptions,omitempty"`
}

// PlanStatusView is the flat table row of a plan status change.
type PlanStatusView struct {
	ID                    string `json:"id"`
	Status                string `json:"status"`
	AffectedSubscriptions string `json:"affected_subscriptions"`
}

// View flattens a plan status change into a table row.
func (s *PlanStatus) View() PlanStatusView {
	affected := ""
	if s.AffectedSubscriptions != nil {
		affected = strconv.Itoa(*s.AffectedSubscriptions)
	}

	return PlanStatusView{
		ID:                    s.ID,
		Status:                s.Status,
		AffectedSubscriptions: affected,
	}
}
