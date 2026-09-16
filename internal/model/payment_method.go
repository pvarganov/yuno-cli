package model

import (
	"encoding/json"
	"strings"
)

// CustomerPaymentMethod is a payment method enrolled for a customer, as
// returned by the enrollment, retrieval and unenrollment operations.
type CustomerPaymentMethod struct {
	ID           string        `json:"id,omitempty"`
	AccountID    string        `json:"account_id,omitempty"`
	CustomerID   string        `json:"customer_id,omitempty"`
	Name         string        `json:"name,omitempty"`
	Description  string        `json:"description,omitempty"`
	Type         string        `json:"type,omitempty"`
	Category     string        `json:"category,omitempty"`
	Country      string        `json:"country,omitempty"`
	Status       string        `json:"status,omitempty"`
	VaultedToken string        `json:"vaulted_token,omitempty"`
	Enrollment   *SessionRef   `json:"enrollment,omitempty"`
	CardData     *EnrolledCard `json:"card_data,omitempty"`
	CreatedAt    string        `json:"created_at,omitempty"`
	UpdatedAt    string        `json:"updated_at,omitempty"`

	// Raw is the verbatim response body, so `--json` keeps the card detail and
	// provider fields the typed struct does not know.
	Raw json.RawMessage `json:"-"`
}

// UnmarshalJSON decodes the typed fields and keeps the original body.
func (m *CustomerPaymentMethod) UnmarshalJSON(data []byte) error {
	type alias CustomerPaymentMethod

	var decoded alias
	if err := json.Unmarshal(data, &decoded); err != nil {
		return err
	}

	*m = CustomerPaymentMethod(decoded)
	m.Raw = append(json.RawMessage(nil), data...)

	return nil
}

// MarshalJSON replays the original response when there is one.
//
//nolint:gocritic // hugeParam: a value receiver is required by json.Marshaler here
func (m CustomerPaymentMethod) MarshalJSON() ([]byte, error) {
	if len(m.Raw) > 0 {
		return m.Raw, nil
	}

	type alias CustomerPaymentMethod

	return json.Marshal(alias(m))
}

// CustomerPaymentMethodView is the flat table row of an enrolled payment method.
type CustomerPaymentMethodView struct {
	ID           string `json:"id"`
	Type         string `json:"type"`
	Category     string `json:"category"`
	Name         string `json:"name"`
	Card         string `json:"card"`
	Status       string `json:"status"`
	VaultedToken string `json:"vaulted_token"`
	Session      string `json:"session"`
	CreatedAt    string `json:"created_at"`
}

// View flattens an enrolled payment method into a table row.
func (m *CustomerPaymentMethod) View() CustomerPaymentMethodView {
	session := ""
	if m.Enrollment != nil {
		session = m.Enrollment.Session
	}

	return CustomerPaymentMethodView{
		ID:           m.ID,
		Card:         m.CardLabel(),
		Type:         m.Type,
		Category:     m.Category,
		Name:         m.Name,
		Status:       m.Status,
		VaultedToken: m.VaultedToken,
		Session:      session,
		CreatedAt:    m.CreatedAt,
	}
}

// CustomerPaymentMethodViews flattens a list of enrolled payment methods.
func CustomerPaymentMethodViews(methods []CustomerPaymentMethod) []CustomerPaymentMethodView {
	views := make([]CustomerPaymentMethodView, 0, len(methods))
	for i := range methods {
		views = append(views, methods[i].View())
	}

	return views
}

// EnrolledCard is the `card_data` block Yuno attaches to an enrolled payment
// method. Only the string fields are typed: the numeric expiration fields are
// spelled differently across endpoints and survive in Raw anyway.
type EnrolledCard struct {
	Brand      string `json:"brand,omitempty"`
	IIN        string `json:"iin,omitempty"`
	LFD        string `json:"lfd,omitempty"`
	Type       string `json:"type,omitempty"`
	IssuerName string `json:"issuer_name,omitempty"`
}

// CardLabel renders the card of a payment method as `VISA ****1111`. The PAN is
// never reconstructed: only the brand and the last four digits are used.
func (m *CustomerPaymentMethod) CardLabel() string {
	if m.CardData == nil {
		return ""
	}

	parts := make([]string, 0, 2)
	if m.CardData.Brand != "" {
		parts = append(parts, m.CardData.Brand)
	}

	if m.CardData.LFD != "" {
		parts = append(parts, "****"+m.CardData.LFD)
	}

	return strings.Join(parts, " ")
}

// CustomerPaymentMethodList is the wrapper `GET /customers/{customer_id}/
// payment-methods` answers with.
type CustomerPaymentMethodList struct {
	PaymentMethods []CustomerPaymentMethod `json:"payment_methods"`
}

// AccountUpdaterResult is the answer of `POST /payment-methods/account-updater`.
type AccountUpdaterResult struct {
	Accepted int `json:"accepted"`
}

// AccountUpdaterView is the flat table row of an account updater registration.
type AccountUpdaterView struct {
	Accepted int `json:"accepted"`
}

// View flattens an account updater result into a table row.
func (r *AccountUpdaterResult) View() AccountUpdaterView {
	return AccountUpdaterView{Accepted: r.Accepted}
}
