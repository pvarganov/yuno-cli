package model

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
)

// Payout is a Yuno payout: money sent from an account to a beneficiary. Only
// the fields the CLI renders are typed; the untouched response stays in Raw so
// `--json` never loses a field the spec adds later.
type Payout struct {
	ID                string              `json:"id,omitempty"`
	AccountID         string              `json:"account_id,omitempty"`
	Status            string              `json:"status,omitempty"`
	MerchantReference string              `json:"merchant_reference,omitempty"`
	Purpose           string              `json:"purpose,omitempty"`
	Country           string              `json:"country,omitempty"`
	Description       string              `json:"description,omitempty"`
	Amount            *Amount             `json:"amount,omitempty"`
	Beneficiary       *PayoutBeneficiary  `json:"beneficiary,omitempty"`
	Transactions      []PayoutTransaction `json:"transactions,omitempty"`
	CreatedAt         string              `json:"created_at,omitempty"`
	UpdatedAt         string              `json:"updated_at,omitempty"`

	// Raw is the verbatim response body, set when the payout was decoded from one.
	Raw json.RawMessage `json:"-"`
}

// PayoutBeneficiary is the party a payout pays out to.
type PayoutBeneficiary struct {
	MerchantBeneficiaryID string    `json:"merchant_beneficiary_id,omitempty"`
	NationalEntity        string    `json:"national_entity,omitempty"`
	FirstName             string    `json:"first_name,omitempty"`
	LastName              string    `json:"last_name,omitempty"`
	LegalName             string    `json:"legal_name,omitempty"`
	Email                 string    `json:"email,omitempty"`
	Country               string    `json:"country,omitempty"`
	DateOfBirth           string    `json:"date_of_birth,omitempty"`
	Document              *Document `json:"document,omitempty"`
	Phone                 *Phone    `json:"phone,omitempty"`
}

// PayoutTransaction is one attempt of a payout against a provider.
type PayoutTransaction struct {
	ID                string  `json:"id,omitempty"`
	Status            string  `json:"status,omitempty"`
	Type              string  `json:"type,omitempty"`
	ResponseCode      string  `json:"response_code,omitempty"`
	MerchantReference string  `json:"merchant_reference,omitempty"`
	Purpose           string  `json:"purpose,omitempty"`
	Description       string  `json:"description,omitempty"`
	Amount            *Amount `json:"amount,omitempty"`
	CreatedAt         string  `json:"created_at,omitempty"`
	UpdatedAt         string  `json:"updated_at,omitempty"`
}

// UnmarshalJSON decodes the typed fields and keeps the original body.
func (p *Payout) UnmarshalJSON(data []byte) error {
	type alias Payout

	var decoded alias
	if err := json.Unmarshal(data, &decoded); err != nil {
		return err
	}

	*p = Payout(decoded)
	p.Raw = append(json.RawMessage(nil), data...)

	return nil
}

// MarshalJSON replays the original response when there is one.
//
//nolint:gocritic // hugeParam: a value receiver is required by json.Marshaler here
func (p Payout) MarshalJSON() ([]byte, error) {
	if len(p.Raw) > 0 {
		return p.Raw, nil
	}

	type alias Payout

	return json.Marshal(alias(p))
}

// DecodePayouts decodes a payout response that Yuno returns either as a single
// object (`GET /payouts/{payout_id}`) or as an array (`GET /payouts`).
func DecodePayouts(data []byte) ([]Payout, error) {
	trimmed := bytes.TrimSpace(data)
	if len(trimmed) == 0 || bytes.Equal(trimmed, []byte("null")) {
		return nil, nil
	}

	if trimmed[0] == '[' {
		var payouts []Payout
		if err := json.Unmarshal(trimmed, &payouts); err != nil {
			return nil, fmt.Errorf("decode payouts: %w", err)
		}

		return payouts, nil
	}

	var payout Payout
	if err := json.Unmarshal(trimmed, &payout); err != nil {
		return nil, fmt.Errorf("decode payout: %w", err)
	}

	return []Payout{payout}, nil
}

// PayoutView is the flat table row of a payout.
type PayoutView struct {
	ID                string `json:"id"`
	Status            string `json:"status"`
	MerchantReference string `json:"merchant_reference"`
	Country           string `json:"country"`
	Purpose           string `json:"purpose"`
	Amount            string `json:"amount"`
	Beneficiary       string `json:"beneficiary"`
	CreatedAt         string `json:"created_at"`
}

// View flattens a payout into a table row.
func (p *Payout) View() PayoutView {
	return PayoutView{
		ID:                p.ID,
		Status:            p.Status,
		MerchantReference: p.MerchantReference,
		Country:           p.Country,
		Purpose:           p.Purpose,
		Amount:            p.Amount.Label(),
		Beneficiary:       p.Beneficiary.Label(),
		CreatedAt:         p.CreatedAt,
	}
}

// PayoutViews flattens a list of payouts.
func PayoutViews(payouts []Payout) []PayoutView {
	views := make([]PayoutView, 0, len(payouts))
	for i := range payouts {
		views = append(views, payouts[i].View())
	}

	return views
}

// Label renders the beneficiary as the best name it carries, falling back to
// the merchant reference of the beneficiary.
func (b *PayoutBeneficiary) Label() string {
	if b == nil {
		return ""
	}

	if name := strings.TrimSpace(b.FirstName + " " + b.LastName); name != "" {
		return name
	}

	if b.LegalName != "" {
		return b.LegalName
	}

	return b.MerchantBeneficiaryID
}
