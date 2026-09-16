package model

import (
	"encoding/json"
	"strconv"
	"strings"
)

// PreDebitNotification is the notice sent to a payer before a recurring debit
// is collected. The untouched response stays in Raw so `--json` never loses a
// field the typed struct does not know.
type PreDebitNotification struct {
	ID                    string                `json:"id,omitempty"`
	AccountID             string                `json:"account_id,omitempty"`
	Status                string                `json:"status,omitempty"`
	MerchantReference     string                `json:"merchant_reference,omitempty"`
	Description           string                `json:"description,omitempty"`
	Amount                *LooseAmount          `json:"amount,omitempty"`
	BillingDate           string                `json:"billing_date,omitempty"`
	BillingSequenceNumber string                `json:"billing_sequence_number,omitempty"`
	OriginPaymentID       string                `json:"origin_payment_id,omitempty"`
	ProviderData          *PreDebitProviderData `json:"provider_data,omitempty"`
	CreatedAt             string                `json:"created_at,omitempty"`
	UpdatedAt             string                `json:"updated_at,omitempty"`

	// Raw is the verbatim response body, set when the notification was decoded
	// from one.
	Raw json.RawMessage `json:"-"`
}

// PreDebitProviderData identifies the provider that carried the notification.
type PreDebitProviderData struct {
	ID                     string `json:"id,omitempty"`
	ConnectionID           string `json:"connection_id,omitempty"`
	ProviderTransactionID  string `json:"provider_transaction_id,omitempty"`
	ProviderNotificationID string `json:"provider_notification_id,omitempty"`
}

// LooseAmount is an amount whose value Yuno spells as a string on some
// resources and as a number on others.
type LooseAmount struct {
	Currency string `json:"currency,omitempty"`
	Value    string `json:"value,omitempty"`
}

// UnmarshalJSON accepts both the string and the number spelling of the value.
func (a *LooseAmount) UnmarshalJSON(data []byte) error {
	var decoded struct {
		Currency string          `json:"currency"`
		Value    json.RawMessage `json:"value"`
	}

	if err := json.Unmarshal(data, &decoded); err != nil {
		return err
	}

	a.Currency = decoded.Currency
	a.Value = looseValue(decoded.Value)

	return nil
}

// MarshalJSON writes the value back as the string Yuno expects.
func (a LooseAmount) MarshalJSON() ([]byte, error) {
	type alias LooseAmount

	return json.Marshal(alias(a))
}

// looseValue renders a raw JSON amount value as a plain string.
func looseValue(raw json.RawMessage) string {
	trimmed := strings.TrimSpace(string(raw))
	if trimmed == "" || trimmed == "null" {
		return ""
	}

	if unquoted, err := strconv.Unquote(trimmed); err == nil {
		return unquoted
	}

	return trimmed
}

// Label renders a loose amount as `USD 1000`.
func (a *LooseAmount) Label() string {
	if a == nil {
		return ""
	}

	return strings.TrimSpace(a.Currency + " " + a.Value)
}

// UnmarshalJSON decodes the typed fields and keeps the original body.
func (n *PreDebitNotification) UnmarshalJSON(data []byte) error {
	type alias PreDebitNotification

	var decoded alias
	if err := json.Unmarshal(data, &decoded); err != nil {
		return err
	}

	*n = PreDebitNotification(decoded)
	n.Raw = append(json.RawMessage(nil), data...)

	return nil
}

// MarshalJSON replays the original response when there is one.
//
//nolint:gocritic // hugeParam: a value receiver is required by json.Marshaler here
func (n PreDebitNotification) MarshalJSON() ([]byte, error) {
	if len(n.Raw) > 0 {
		return n.Raw, nil
	}

	type alias PreDebitNotification

	return json.Marshal(alias(n))
}

// PreDebitNotificationView is the flat table row of a pre-debit notification.
type PreDebitNotificationView struct {
	ID                string `json:"id"`
	Status            string `json:"status"`
	MerchantReference string `json:"merchant_reference"`
	Amount            string `json:"amount"`
	BillingDate       string `json:"billing_date"`
	OriginPaymentID   string `json:"origin_payment_id"`
	Provider          string `json:"provider"`
	CreatedAt         string `json:"created_at"`
}

// View flattens a pre-debit notification into a table row.
func (n *PreDebitNotification) View() PreDebitNotificationView {
	view := PreDebitNotificationView{
		ID:                n.ID,
		Status:            n.Status,
		MerchantReference: n.MerchantReference,
		Amount:            n.Amount.Label(),
		BillingDate:       n.BillingDate,
		OriginPaymentID:   n.OriginPaymentID,
		CreatedAt:         n.CreatedAt,
	}

	if n.ProviderData != nil {
		view.Provider = n.ProviderData.ID
	}

	return view
}

// PreDebitNotificationViews flattens a list of pre-debit notifications.
func PreDebitNotificationViews(notifications []PreDebitNotification) []PreDebitNotificationView {
	views := make([]PreDebitNotificationView, 0, len(notifications))
	for i := range notifications {
		views = append(views, notifications[i].View())
	}

	return views
}
