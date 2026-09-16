package model

import (
	"encoding/json"
	"strconv"
	"strings"
)

// Payment is a Yuno payment. Only the fields the CLI renders are typed; the
// untouched response is kept in Raw so `--json` never loses a field the spec
// adds later.
type Payment struct {
	ID              string        `json:"id"`
	AccountID       string        `json:"account_id,omitempty"`
	MerchantOrderID string        `json:"merchant_order_id,omitempty"`
	Description     string        `json:"description,omitempty"`
	Country         string        `json:"country,omitempty"`
	Status          string        `json:"status,omitempty"`
	SubStatus       string        `json:"sub_status,omitempty"`
	Workflow        string        `json:"workflow,omitempty"`
	Amount          Amount        `json:"amount"`
	PaymentMethod   PaymentMethod `json:"payment_method"`
	Transactions    []Transaction `json:"transactions,omitempty"`
	CreatedAt       string        `json:"created_at,omitempty"`
	UpdatedAt       string        `json:"updated_at,omitempty"`

	// Raw is the verbatim response body, set when the payment was decoded
	// from one.
	Raw json.RawMessage `json:"-"`
}

// Amount is the value and currency of a payment or a transaction.
type Amount struct {
	Currency string   `json:"currency,omitempty"`
	Value    float64  `json:"value,omitempty"`
	Captured *float64 `json:"captured,omitempty"`
	Refunded *float64 `json:"refunded,omitempty"`
}

// PaymentMethod is the instrument a payment was made with.
type PaymentMethod struct {
	Type         string               `json:"type,omitempty"`
	Token        string               `json:"token,omitempty"`
	VaultedToken string               `json:"vaulted_token,omitempty"`
	Detail       *PaymentMethodDetail `json:"detail,omitempty"`
}

// PaymentMethodDetail holds the instrument specific data of a payment method.
type PaymentMethodDetail struct {
	Card *CardDetail `json:"card,omitempty"`
}

// CardDetail is the card data of a payment method. The number is whatever the
// provider returned, which is normally already truncated; the formatter masks
// it again on the way out.
type CardDetail struct {
	Brand        string `json:"brand,omitempty"`
	Number       string `json:"number,omitempty"`
	Type         string `json:"type,omitempty"`
	IIN          string `json:"iin,omitempty"`
	LFD          string `json:"lfd,omitempty"`
	ExpirationMo string `json:"expiration_month,omitempty"`
	ExpirationYr string `json:"expiration_year,omitempty"`
}

// Transaction is one provider attempt of a payment.
type Transaction struct {
	ID          string `json:"id"`
	Type        string `json:"type,omitempty"`
	Status      string `json:"status,omitempty"`
	Category    string `json:"category,omitempty"`
	ProviderID  string `json:"provider_id,omitempty"`
	Amount      Amount `json:"amount"`
	ReasonCode  string `json:"reason,omitempty"`
	CreatedAt   string `json:"created_at,omitempty"`
	UpdatedAt   string `json:"updated_at,omitempty"`
	Description string `json:"description,omitempty"`
}

// UnmarshalJSON decodes the typed fields and keeps the original body.
func (p *Payment) UnmarshalJSON(data []byte) error {
	type alias Payment

	var decoded alias
	if err := json.Unmarshal(data, &decoded); err != nil {
		return err
	}

	*p = Payment(decoded)
	p.Raw = append(json.RawMessage(nil), data...)

	return nil
}

// MarshalJSON replays the original response when there is one, so `--json`
// shows every field Yuno sent and not only the typed ones.
// The receiver stays a value on purpose: json.Marshal only calls a pointer
// method on an addressable value, and a payment copied out of a response is
// often not one.
//
//nolint:gocritic // hugeParam: a value receiver is required by json.Marshaler here
func (p Payment) MarshalJSON() ([]byte, error) {
	if len(p.Raw) > 0 {
		return p.Raw, nil
	}

	type alias Payment

	return json.Marshal(alias(p))
}

// PaymentView is the flat table row of a payment.
type PaymentView struct {
	ID              string `json:"id"`
	MerchantOrderID string `json:"merchant_order_id"`
	Status          string `json:"status"`
	SubStatus       string `json:"sub_status"`
	Currency        string `json:"currency"`
	Amount          string `json:"amount"`
	PaymentMethod   string `json:"payment_method"`
	CreatedAt       string `json:"created_at"`
}

// View flattens a payment into a table row.
func (p *Payment) View() PaymentView {
	return PaymentView{
		ID:              p.ID,
		MerchantOrderID: p.MerchantOrderID,
		Status:          p.Status,
		SubStatus:       p.SubStatus,
		Currency:        p.Amount.Currency,
		Amount:          formatAmount(p.Amount.Value),
		PaymentMethod:   p.PaymentMethod.Label(),
		CreatedAt:       p.CreatedAt,
	}
}

// PaymentViews flattens a list of payments.
func PaymentViews(payments []Payment) []PaymentView {
	views := make([]PaymentView, 0, len(payments))
	for i := range payments {
		views = append(views, payments[i].View())
	}

	return views
}

// TransactionView is the flat table row of a payment transaction.
type TransactionView struct {
	ID         string `json:"id"`
	Type       string `json:"type"`
	Status     string `json:"status"`
	Category   string `json:"category"`
	ProviderID string `json:"provider_id"`
	Currency   string `json:"currency"`
	Amount     string `json:"amount"`
}

// TransactionViews flattens the transactions of a payment, falling back to the
// payment row when the payment has none.
func (p *Payment) TransactionViews() []TransactionView {
	views := make([]TransactionView, 0, len(p.Transactions))

	for i := range p.Transactions {
		t := &p.Transactions[i]

		views = append(views, TransactionView{
			ID:         t.ID,
			Type:       t.Type,
			Status:     t.Status,
			Category:   t.Category,
			ProviderID: t.ProviderID,
			Currency:   t.Amount.Currency,
			Amount:     formatAmount(t.Amount.Value),
		})
	}

	return views
}

// Label renders a payment method as `CARD VISA 1234`, skipping the parts the
// provider did not return.
func (m *PaymentMethod) Label() string {
	parts := []string{m.Type}

	if m.Detail != nil && m.Detail.Card != nil {
		card := m.Detail.Card

		if card.Brand != "" {
			parts = append(parts, card.Brand)
		}

		if card.LFD != "" {
			parts = append(parts, card.LFD)
		}
	}

	return strings.TrimSpace(strings.Join(parts, " "))
}

// Issuer is one bank of `GET /issuers`.
type Issuer struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// IssuerList is the envelope `GET /issuers` responds with.
type IssuerList struct {
	Issuers []Issuer `json:"issuers"`
}

// IssuerView is the flat table row of an issuer.
type IssuerView struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// Views flattens the issuers of a list response.
func (l *IssuerList) Views() []IssuerView {
	views := make([]IssuerView, 0, len(l.Issuers))
	for _, issuer := range l.Issuers {
		views = append(views, IssuerView(issuer))
	}

	return views
}

// formatAmount renders a monetary value without trailing zeros, since Yuno
// sends both integer minor units and decimal values depending on the currency.
func formatAmount(value float64) string {
	return strconv.FormatFloat(value, 'f', -1, 64)
}
