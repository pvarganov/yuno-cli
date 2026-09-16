package model

import (
	"encoding/json"
)

// Seller is a marketplace seller registered under an account. Only the fields
// the CLI renders are typed; the untouched response stays in Raw so `--json`
// never loses a field the spec adds later.
type Seller struct {
	SellerID             string    `json:"seller_id,omitempty"`
	MerchantSellerID     string    `json:"merchant_seller_id,omitempty"`
	Name                 string    `json:"name,omitempty"`
	Email                string    `json:"email,omitempty"`
	Country              string    `json:"country,omitempty"`
	Website              string    `json:"website,omitempty"`
	Industry             string    `json:"industry,omitempty"`
	MerchantCategoryCode string    `json:"merchant_category_code,omitempty"`
	Document             *Document `json:"document,omitempty"`
	Phone                *Phone    `json:"phone,omitempty"`
	CreatedAt            string    `json:"created_at,omitempty"`
	UpdatedAt            string    `json:"updated_at,omitempty"`

	// Raw is the verbatim response body, set when the seller was decoded from one.
	Raw json.RawMessage `json:"-"`
}

// UnmarshalJSON decodes the typed fields and keeps the original body.
func (s *Seller) UnmarshalJSON(data []byte) error {
	type alias Seller

	var decoded alias
	if err := json.Unmarshal(data, &decoded); err != nil {
		return err
	}

	*s = Seller(decoded)
	s.Raw = append(json.RawMessage(nil), data...)

	return nil
}

// MarshalJSON replays the original response when there is one.
//
//nolint:gocritic // hugeParam: a value receiver is required by json.Marshaler here
func (s Seller) MarshalJSON() ([]byte, error) {
	if len(s.Raw) > 0 {
		return s.Raw, nil
	}

	type alias Seller

	return json.Marshal(alias(s))
}

// SellerView is the flat table row of a seller.
type SellerView struct {
	SellerID         string `json:"seller_id"`
	MerchantSellerID string `json:"merchant_seller_id"`
	Name             string `json:"name"`
	Email            string `json:"email"`
	Country          string `json:"country"`
	Document         string `json:"document"`
	CreatedAt        string `json:"created_at"`
}

// View flattens a seller into a table row.
func (s *Seller) View() SellerView {
	return SellerView{
		SellerID:         s.SellerID,
		MerchantSellerID: s.MerchantSellerID,
		Name:             s.Name,
		Email:            s.Email,
		Country:          s.Country,
		Document:         s.Document.Label(),
		CreatedAt:        s.CreatedAt,
	}
}
