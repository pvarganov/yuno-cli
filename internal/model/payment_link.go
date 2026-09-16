package model

import (
	"encoding/json"
	"strconv"
	"strings"
)

// PaymentLink is a Yuno payment link. `create` answers with `id`, the other
// operations with `code`; both are kept so the table always shows one. The
// untouched response stays in Raw so `--json` never loses a field.
type PaymentLink struct {
	ID                 string        `json:"id,omitempty"`
	Code               string        `json:"code,omitempty"`
	Country            string        `json:"country,omitempty"`
	Status             string        `json:"status,omitempty"`
	Description        string        `json:"description,omitempty"`
	MerchantOrderID    string        `json:"merchant_order_id,omitempty"`
	Amount             *Amount       `json:"amount,omitempty"`
	PaymentMethodTypes []string      `json:"payment_method_types,omitempty"`
	OneTimeUse         *bool         `json:"one_time_use,omitempty"`
	CallbackURL        string        `json:"callback_url,omitempty"`
	CheckoutURL        string        `json:"checkout_url,omitempty"`
	Availability       *Availability `json:"availability,omitempty"`
	PaymentsNumber     *int          `json:"payments_number,omitempty"`
	CreatedAt          string        `json:"created_at,omitempty"`
	UpdatedAt          string        `json:"updated_at,omitempty"`

	// Raw is the verbatim response body, set when the link was decoded from one.
	Raw json.RawMessage `json:"-"`
}

// UnmarshalJSON decodes the typed fields and keeps the original body.
func (l *PaymentLink) UnmarshalJSON(data []byte) error {
	type alias PaymentLink

	var decoded alias
	if err := json.Unmarshal(data, &decoded); err != nil {
		return err
	}

	*l = PaymentLink(decoded)
	l.Raw = append(json.RawMessage(nil), data...)

	return nil
}

// MarshalJSON replays the original response when there is one.
//
//nolint:gocritic // hugeParam: a value receiver is required by json.Marshaler here
func (l PaymentLink) MarshalJSON() ([]byte, error) {
	if len(l.Raw) > 0 {
		return l.Raw, nil
	}

	type alias PaymentLink

	return json.Marshal(alias(l))
}

// PaymentLinkView is the flat table row of a payment link.
type PaymentLinkView struct {
	Code            string `json:"code"`
	Status          string `json:"status"`
	Country         string `json:"country"`
	Amount          string `json:"amount"`
	MerchantOrderID string `json:"merchant_order_id"`
	PaymentMethods  string `json:"payment_method_types"`
	CheckoutURL     string `json:"checkout_url"`
	ExpiresAt       string `json:"finish_at"`
}

// View flattens a payment link into a table row.
func (l *PaymentLink) View() PaymentLinkView {
	return PaymentLinkView{
		Code:            l.Reference(),
		Status:          l.Status,
		Country:         l.Country,
		Amount:          l.Amount.Label(),
		MerchantOrderID: l.MerchantOrderID,
		PaymentMethods:  strings.Join(l.PaymentMethodTypes, ","),
		CheckoutURL:     l.CheckoutURL,
		ExpiresAt:       l.Availability.FinishLabel(),
	}
}

// Reference returns the identifier the other payment link commands take, which
// is `code` when Yuno sent one and `id` otherwise.
func (l *PaymentLink) Reference() string {
	if l.Code != "" {
		return l.Code
	}

	return l.ID
}

// PaymentLinkViews flattens a list of payment links.
func PaymentLinkViews(links []PaymentLink) []PaymentLinkView {
	views := make([]PaymentLinkView, 0, len(links))
	for i := range links {
		views = append(views, links[i].View())
	}

	return views
}

// FinishLabel renders the end of an availability window.
func (a *Availability) FinishLabel() string {
	if a == nil {
		return ""
	}

	return a.FinishAt
}

// ConversionRate is the result of a currency conversion quote.
type ConversionRate struct {
	ID     string            `json:"id"`
	Amount *ConversionAmount `json:"amount,omitempty"`

	// Raw is the verbatim response body, set when the quote was decoded from one.
	Raw json.RawMessage `json:"-"`
}

// ConversionAmount is the converted amount of a currency conversion quote.
type ConversionAmount struct {
	Value              float64             `json:"value,omitempty"`
	Currency           string              `json:"currency,omitempty"`
	CurrencyConversion *CurrencyConversion `json:"currency_conversion,omitempty"`
}

// CurrencyConversion is the cardholder side of a currency conversion quote.
type CurrencyConversion struct {
	CardholderCurrency string                  `json:"cardholder_currency,omitempty"`
	CardholderAmount   float64                 `json:"cardholder_amount,omitempty"`
	Rate               float64                 `json:"rate,omitempty"`
	ProviderData       *ConversionProviderData `json:"provider_data,omitempty"`
}

// ConversionProviderData identifies the provider that quoted the conversion.
type ConversionProviderData struct {
	ID              string `json:"id,omitempty"`
	TransactionID   string `json:"transaction_id,omitempty"`
	ResponseCode    string `json:"response_code,omitempty"`
	ResponseMessage string `json:"response_message,omitempty"`
}

// UnmarshalJSON decodes the typed fields and keeps the original body.
func (r *ConversionRate) UnmarshalJSON(data []byte) error {
	type alias ConversionRate

	var decoded alias
	if err := json.Unmarshal(data, &decoded); err != nil {
		return err
	}

	*r = ConversionRate(decoded)
	r.Raw = append(json.RawMessage(nil), data...)

	return nil
}

// MarshalJSON replays the original response when there is one.
//
//nolint:gocritic // hugeParam: a value receiver is required by json.Marshaler here
func (r ConversionRate) MarshalJSON() ([]byte, error) {
	if len(r.Raw) > 0 {
		return r.Raw, nil
	}

	type alias ConversionRate

	return json.Marshal(alias(r))
}

// ConversionRateView is the flat table row of a currency conversion quote.
type ConversionRateView struct {
	ID                  string `json:"id"`
	Amount              string `json:"amount"`
	CardholderAmount    string `json:"cardholder_amount"`
	Rate                string `json:"rate"`
	Provider            string `json:"provider"`
	ProviderResponse    string `json:"provider_response"`
	ProviderTransaction string `json:"provider_transaction_id"`
}

// View flattens a currency conversion quote into a table row.
func (r *ConversionRate) View() ConversionRateView {
	view := ConversionRateView{ID: r.ID}
	if r.Amount == nil {
		return view
	}

	view.Amount = strings.TrimSpace(r.Amount.Currency + " " + formatAmount(r.Amount.Value))

	conversion := r.Amount.CurrencyConversion
	if conversion == nil {
		return view
	}

	view.CardholderAmount = strings.TrimSpace(
		conversion.CardholderCurrency + " " + formatAmount(conversion.CardholderAmount))
	view.Rate = strconv.FormatFloat(conversion.Rate, 'f', -1, 64)

	if provider := conversion.ProviderData; provider != nil {
		view.Provider = provider.ID
		view.ProviderResponse = strings.TrimSpace(provider.ResponseCode + " " + provider.ResponseMessage)
		view.ProviderTransaction = provider.TransactionID
	}

	return view
}
