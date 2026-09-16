package model

import (
	"encoding/json"
	"strings"
)

// Transfer is a standalone split-marketplace transfer: money moved from the
// marketplace account to one recipient. The untouched response stays in Raw so
// `--json` never loses a field the typed struct does not know.
type Transfer struct {
	ID                string                `json:"id,omitempty"`
	RecipientID       string                `json:"recipient_id,omitempty"`
	Amount            *Amount               `json:"amount,omitempty"`
	Status            string                `json:"status,omitempty"`
	Reason            string                `json:"reason,omitempty"`
	Description       string                `json:"description,omitempty"`
	MerchantReference string                `json:"merchant_reference,omitempty"`
	Provider          *string               `json:"provider,omitempty"`
	ProviderData      *TransferProviderData `json:"provider_data,omitempty"`
	Transactions      []TransferTransaction `json:"transactions,omitempty"`
	CreatedAt         string                `json:"created_at,omitempty"`
	UpdatedAt         string                `json:"updated_at,omitempty"`
	CompletedAt       string                `json:"completed_at,omitempty"`

	// Raw is the verbatim response body, set when the transfer was decoded from one.
	Raw json.RawMessage `json:"-"`
}

// TransferProviderData is the provider side of a transfer.
type TransferProviderData struct {
	ID              string `json:"id,omitempty"`
	RecipientID     string `json:"recipient_id,omitempty"`
	TransferID      string `json:"transfer_id,omitempty"`
	ResponseCode    string `json:"response_code,omitempty"`
	ResponseMessage string `json:"response_message,omitempty"`
}

// TransferTransaction is one attempt of a transfer against a provider.
type TransferTransaction struct {
	Code                    string `json:"code,omitempty"`
	Type                    string `json:"type,omitempty"`
	Status                  string `json:"status,omitempty"`
	ProviderTransferID      string `json:"provider_transfer_id,omitempty"`
	ProviderResponseMessage string `json:"provider_response_message,omitempty"`
	CreatedAt               string `json:"created_at,omitempty"`
	UpdatedAt               string `json:"updated_at,omitempty"`
}

// UnmarshalJSON decodes the typed fields and keeps the original body.
func (t *Transfer) UnmarshalJSON(data []byte) error {
	type alias Transfer

	var decoded alias
	if err := json.Unmarshal(data, &decoded); err != nil {
		return err
	}

	*t = Transfer(decoded)
	t.Raw = append(json.RawMessage(nil), data...)

	return nil
}

// MarshalJSON replays the original response when there is one.
//
//nolint:gocritic // hugeParam: a value receiver is required by json.Marshaler here
func (t Transfer) MarshalJSON() ([]byte, error) {
	if len(t.Raw) > 0 {
		return t.Raw, nil
	}

	type alias Transfer

	return json.Marshal(alias(t))
}

// TransferView is the flat table row of a transfer.
type TransferView struct {
	ID                string `json:"id"`
	Status            string `json:"status"`
	RecipientID       string `json:"recipient_id"`
	Amount            string `json:"amount"`
	MerchantReference string `json:"merchant_reference"`
	Provider          string `json:"provider"`
	ProviderResponse  string `json:"provider_response"`
	CreatedAt         string `json:"created_at"`
}

// View flattens a transfer into a table row.
func (t *Transfer) View() TransferView {
	view := TransferView{
		ID:                t.ID,
		Status:            t.Status,
		RecipientID:       t.RecipientID,
		Amount:            t.Amount.Label(),
		MerchantReference: t.MerchantReference,
		CreatedAt:         t.CreatedAt,
	}

	if t.Provider != nil {
		view.Provider = *t.Provider
	}

	if data := t.ProviderData; data != nil {
		if view.Provider == "" {
			view.Provider = data.ID
		}

		view.ProviderResponse = strings.TrimSpace(data.ResponseCode + " " + data.ResponseMessage)
	}

	return view
}

// TransferViews flattens a list of transfers.
func TransferViews(transfers []Transfer) []TransferView {
	views := make([]TransferView, 0, len(transfers))
	for i := range transfers {
		views = append(views, transfers[i].View())
	}

	return views
}
