package model

import (
	"encoding/json"
	"strings"
)

// Customer is a Yuno customer. Only the fields the CLI renders are typed; the
// untouched response is kept in Raw so `--json` never loses a field the spec
// adds later (document, addresses, metadata).
type Customer struct {
	ID                 string    `json:"id"`
	MerchantCustomerID string    `json:"merchant_customer_id,omitempty"`
	FirstName          string    `json:"first_name,omitempty"`
	LastName           string    `json:"last_name,omitempty"`
	Email              string    `json:"email,omitempty"`
	Country            string    `json:"country,omitempty"`
	Gender             string    `json:"gender,omitempty"`
	DateOfBirth        string    `json:"date_of_birth,omitempty"`
	Nationality        string    `json:"nationality,omitempty"`
	Document           *Document `json:"document,omitempty"`
	Phone              *Phone    `json:"phone,omitempty"`
	CreatedAt          string    `json:"created_at,omitempty"`
	UpdatedAt          string    `json:"updated_at,omitempty"`

	// Raw is the verbatim response body, set when the customer was decoded
	// from one.
	Raw json.RawMessage `json:"-"`
}

// Document is the identity document of a customer.
type Document struct {
	DocumentNumber string `json:"document_number,omitempty"`
	DocumentType   string `json:"document_type,omitempty"`
}

// Phone is the phone number of a customer.
type Phone struct {
	CountryCode string `json:"country_code,omitempty"`
	Number      string `json:"number,omitempty"`
}

// UnmarshalJSON decodes the typed fields and keeps the original body.
func (c *Customer) UnmarshalJSON(data []byte) error {
	type alias Customer

	var decoded alias
	if err := json.Unmarshal(data, &decoded); err != nil {
		return err
	}

	*c = Customer(decoded)
	c.Raw = append(json.RawMessage(nil), data...)

	return nil
}

// MarshalJSON replays the original response when there is one, so `--json`
// shows every field Yuno sent and not only the typed ones.
// The receiver stays a value on purpose: json.Marshal only calls a pointer
// method on an addressable value, and a customer copied out of a response is
// often not one.
//
//nolint:gocritic // hugeParam: a value receiver is required by json.Marshaler here
func (c Customer) MarshalJSON() ([]byte, error) {
	if len(c.Raw) > 0 {
		return c.Raw, nil
	}

	type alias Customer

	return json.Marshal(alias(c))
}

// CustomerView is the flat table row of a customer.
type CustomerView struct {
	ID                 string `json:"id"`
	MerchantCustomerID string `json:"merchant_customer_id"`
	Name               string `json:"name"`
	Email              string `json:"email"`
	Country            string `json:"country"`
	Document           string `json:"document"`
	CreatedAt          string `json:"created_at"`
}

// View flattens a customer into a table row.
func (c *Customer) View() CustomerView {
	return CustomerView{
		ID:                 c.ID,
		MerchantCustomerID: c.MerchantCustomerID,
		Name:               strings.TrimSpace(c.FirstName + " " + c.LastName),
		Email:              c.Email,
		Country:            c.Country,
		Document:           c.Document.Label(),
		CreatedAt:          c.CreatedAt,
	}
}

// CustomerViews flattens a list of customers.
func CustomerViews(customers []Customer) []CustomerView {
	views := make([]CustomerView, 0, len(customers))
	for i := range customers {
		views = append(views, customers[i].View())
	}

	return views
}

// Label renders a document as `CC 12345`, and an empty string when the
// customer has none.
func (d *Document) Label() string {
	if d == nil {
		return ""
	}

	return strings.TrimSpace(d.DocumentType + " " + d.DocumentNumber)
}

// CustomerSession is the enrollment session of a customer.
type CustomerSession struct {
	CustomerSession string `json:"customer_session"`
	CustomerID      string `json:"customer_id,omitempty"`
	Country         string `json:"country,omitempty"`
	CallbackURL     string `json:"callback_url,omitempty"`
	CheckoutID      string `json:"checkout_id,omitempty"`
	CreatedAt       string `json:"created_at,omitempty"`
}

// CustomerSessionView is the flat table row of a customer session.
type CustomerSessionView struct {
	CustomerSession string `json:"customer_session"`
	CustomerID      string `json:"customer_id"`
	Country         string `json:"country"`
	CheckoutID      string `json:"checkout_id"`
	CreatedAt       string `json:"created_at"`
}

// View flattens a customer session into a table row.
func (s *CustomerSession) View() CustomerSessionView {
	return CustomerSessionView{
		CustomerSession: s.CustomerSession,
		CustomerID:      s.CustomerID,
		Country:         s.Country,
		CheckoutID:      s.CheckoutID,
		CreatedAt:       s.CreatedAt,
	}
}

// NetworkTokenCryptogram is the cryptogram generated for a network token.
type NetworkTokenCryptogram struct {
	VaultedToken string `json:"vaulted_token,omitempty"`
	NetworkToken string `json:"network_token,omitempty"`
	Cryptogram   string `json:"cryptogram,omitempty"`
	ECI          string `json:"eci,omitempty"`
	CreatedAt    string `json:"created_at,omitempty"`
}

// NetworkTokenCryptogramView is the flat table row of a cryptogram. The token
// and the cryptogram themselves are masked by the formatter.
type NetworkTokenCryptogramView struct {
	VaultedToken string `json:"vaulted_token"`
	NetworkToken string `json:"network_token"`
	Cryptogram   string `json:"cryptogram"`
	ECI          string `json:"eci"`
	CreatedAt    string `json:"created_at"`
}

// View flattens a cryptogram into a table row.
func (c *NetworkTokenCryptogram) View() NetworkTokenCryptogramView {
	return NetworkTokenCryptogramView(*c)
}
