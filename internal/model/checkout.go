package model

import (
	"encoding/json"
)

// CheckoutSession is a checkout session of the SDK workflow. Yuno names the
// session `checkout_session` when it creates one and `id` when it returns an
// existing one, so both are typed and View picks whichever is set.
type CheckoutSession struct {
	ID                 string  `json:"id,omitempty"`
	CheckoutSession    string  `json:"checkout_session,omitempty"`
	MerchantOrderID    string  `json:"merchant_order_id,omitempty"`
	PaymentDescription string  `json:"payment_description,omitempty"`
	Country            string  `json:"country,omitempty"`
	AccountID          string  `json:"account_id,omitempty"`
	CustomerID         string  `json:"customer_id,omitempty"`
	CallbackURL        string  `json:"callback_url,omitempty"`
	CheckoutID         string  `json:"checkout_id,omitempty"`
	Workflow           string  `json:"workflow,omitempty"`
	Amount             *Amount `json:"amount,omitempty"`
	Used               *bool   `json:"used,omitempty"`
	CreatedAt          string  `json:"created_at,omitempty"`

	// Raw is the verbatim response body, so `--json` keeps the fields the
	// typed struct does not know (metadata, installments, recurring_payment).
	Raw json.RawMessage `json:"-"`
}

// UnmarshalJSON decodes the typed fields and keeps the original body.
func (s *CheckoutSession) UnmarshalJSON(data []byte) error {
	type alias CheckoutSession

	var decoded alias
	if err := json.Unmarshal(data, &decoded); err != nil {
		return err
	}

	*s = CheckoutSession(decoded)
	s.Raw = append(json.RawMessage(nil), data...)

	return nil
}

// MarshalJSON replays the original response when there is one.
// The receiver stays a value on purpose: json.Marshal only calls a pointer
// method on an addressable value, and a session copied out of a response is
// often not one.
//
//nolint:gocritic // hugeParam: a value receiver is required by json.Marshaler here
func (s CheckoutSession) MarshalJSON() ([]byte, error) {
	if len(s.Raw) > 0 {
		return s.Raw, nil
	}

	type alias CheckoutSession

	return json.Marshal(alias(s))
}

// CheckoutSessionView is the flat table row of a checkout session.
type CheckoutSessionView struct {
	Session         string `json:"session"`
	MerchantOrderID string `json:"merchant_order_id"`
	Country         string `json:"country"`
	CustomerID      string `json:"customer_id"`
	Amount          string `json:"amount"`
	Workflow        string `json:"workflow"`
	CreatedAt       string `json:"created_at"`
}

// View flattens a checkout session into a table row.
func (s *CheckoutSession) View() CheckoutSessionView {
	return CheckoutSessionView{
		Session:         s.Session(),
		MerchantOrderID: s.MerchantOrderID,
		Country:         s.Country,
		CustomerID:      s.CustomerID,
		Amount:          s.Amount.Label(),
		Workflow:        s.Workflow,
		CreatedAt:       s.CreatedAt,
	}
}

// Session returns the session identifier, whichever field carries it.
func (s *CheckoutSession) Session() string {
	if s.CheckoutSession != "" {
		return s.CheckoutSession
	}

	return s.ID
}

// Label renders an amount as `USD 5.20`, and an empty string when there is none.
func (a *Amount) Label() string {
	if a == nil || a.Currency == "" && a.Value == 0 {
		return ""
	}

	return a.Currency + " " + formatAmount(a.Value)
}

// CheckoutPaymentMethod is one payment method offered for a checkout session or
// for enrollment in a customer session.
type CheckoutPaymentMethod struct {
	Name                   string          `json:"name,omitempty"`
	Description            string          `json:"description,omitempty"`
	Type                   string          `json:"type,omitempty"`
	Category               string          `json:"category,omitempty"`
	VaultedToken           string          `json:"vaulted_token,omitempty"`
	Icon                   string          `json:"icon,omitempty"`
	LastSuccessfullyUsed   bool            `json:"last_successfully_used,omitempty"`
	LastSuccessfullyUsedAt string          `json:"last_successfully_used_at,omitempty"`
	Checkout               *SessionRef     `json:"checkout,omitempty"`
	Enrollment             *SessionRef     `json:"enrollment,omitempty"`
	Raw                    json.RawMessage `json:"-"`
}

// UnmarshalJSON decodes the typed fields and keeps the original body, so the
// checkout conditions Yuno attaches survive `--json`.
func (m *CheckoutPaymentMethod) UnmarshalJSON(data []byte) error {
	type alias CheckoutPaymentMethod

	var decoded alias
	if err := json.Unmarshal(data, &decoded); err != nil {
		return err
	}

	*m = CheckoutPaymentMethod(decoded)
	m.Raw = append(json.RawMessage(nil), data...)

	return nil
}

// MarshalJSON replays the original response when there is one.
//
//nolint:gocritic // hugeParam: a value receiver is required by json.Marshaler here
func (m CheckoutPaymentMethod) MarshalJSON() ([]byte, error) {
	if len(m.Raw) > 0 {
		return m.Raw, nil
	}

	type alias CheckoutPaymentMethod

	return json.Marshal(alias(m))
}

// SessionRef is the session a payment method is bound to, plus whether the SDK
// has to take an action for it.
type SessionRef struct {
	Session           string `json:"session,omitempty"`
	SDKRequiredAction bool   `json:"sdk_required_action,omitempty"`
}

// CheckoutPaymentMethodList is the wrapper `GET /checkout/customers/sessions/
// {customer_session}/payment-methods` answers with.
type CheckoutPaymentMethodList struct {
	PaymentMethods []CheckoutPaymentMethod `json:"payment_methods"`
}

// CheckoutPaymentMethodView is the flat table row of a checkout payment method.
type CheckoutPaymentMethodView struct {
	Type         string `json:"type"`
	Category     string `json:"category"`
	Name         string `json:"name"`
	VaultedToken string `json:"vaulted_token"`
	Session      string `json:"session"`
}

// View flattens a checkout payment method into a table row.
func (m *CheckoutPaymentMethod) View() CheckoutPaymentMethodView {
	return CheckoutPaymentMethodView{
		Type:         m.Type,
		Category:     m.Category,
		Name:         m.Name,
		VaultedToken: m.VaultedToken,
		Session:      m.SessionCode(),
	}
}

// SessionCode returns the checkout or enrollment session of a payment method.
func (m *CheckoutPaymentMethod) SessionCode() string {
	if m.Checkout != nil && m.Checkout.Session != "" {
		return m.Checkout.Session
	}

	if m.Enrollment != nil {
		return m.Enrollment.Session
	}

	return ""
}

// CheckoutPaymentMethodViews flattens a list of checkout payment methods.
func CheckoutPaymentMethodViews(methods []CheckoutPaymentMethod) []CheckoutPaymentMethodView {
	views := make([]CheckoutPaymentMethodView, 0, len(methods))
	for i := range methods {
		views = append(views, methods[i].View())
	}

	return views
}
