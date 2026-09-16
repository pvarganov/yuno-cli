package model

import (
	"encoding/json"
)

// Checkout is a checkout configuration built in the checkout builder: the
// payment method list, the general settings and the styling the SDK renders.
// Only the fields the CLI tabulates are typed; the untouched response stays in
// Raw so `--json` never loses a field the spec adds later.
type Checkout struct {
	ID          string `json:"id,omitempty"`
	Name        string `json:"name,omitempty"`
	Description string `json:"description,omitempty"`
	Status      string `json:"status,omitempty"`
	// IsDefault and IsActive are pointers because the create and the lifecycle
	// endpoints answer with them while the list and get endpoints answer with
	// `status` instead: a missing flag must not read as false.
	IsDefault *bool           `json:"is_default,omitempty"`
	IsActive  *bool           `json:"is_active,omitempty"`
	Config    json.RawMessage `json:"config,omitempty"`
	Styling   json.RawMessage `json:"styling,omitempty"`
	CreatedAt string          `json:"created_at,omitempty"`
	UpdatedAt string          `json:"updated_at,omitempty"`

	// Raw is the verbatim response body, set when the checkout was decoded from one.
	Raw json.RawMessage `json:"-"`
}

// UnmarshalJSON decodes the typed fields and keeps the original body.
func (c *Checkout) UnmarshalJSON(data []byte) error {
	type alias Checkout

	var decoded alias
	if err := json.Unmarshal(data, &decoded); err != nil {
		return err
	}

	*c = Checkout(decoded)
	c.Raw = append(json.RawMessage(nil), data...)

	return nil
}

// MarshalJSON replays the original response when there is one.
//
//nolint:gocritic // hugeParam: a value receiver is required by json.Marshaler here
func (c Checkout) MarshalJSON() ([]byte, error) {
	if len(c.Raw) > 0 {
		return c.Raw, nil
	}

	type alias Checkout

	return json.Marshal(alias(c))
}

// CheckoutView is the flat table row of a checkout configuration.
type CheckoutView struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Status      string `json:"status"`
	Default     bool   `json:"default"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
}

// View flattens a checkout into a table row.
func (c *Checkout) View() CheckoutView {
	return CheckoutView{
		ID:          c.ID,
		Name:        c.Name,
		Description: c.Description,
		Status:      c.StatusLabel(),
		Default:     boolValue(c.IsDefault),
		CreatedAt:   c.CreatedAt,
		UpdatedAt:   c.UpdatedAt,
	}
}

// StatusLabel is the lifecycle status of the checkout. The create and lifecycle
// endpoints answer with `is_active` instead of `status`, so the flag stands in
// when the status itself is absent.
func (c *Checkout) StatusLabel() string {
	if c.Status != "" {
		return c.Status
	}

	if c.IsActive == nil {
		return ""
	}

	if *c.IsActive {
		return "ACTIVE"
	}

	return "INACTIVE"
}

// CheckoutViews flattens a list of checkouts into table rows.
func CheckoutViews(checkouts []Checkout) []CheckoutView {
	views := make([]CheckoutView, 0, len(checkouts))
	for i := range checkouts {
		views = append(views, checkouts[i].View())
	}

	return views
}

// boolValue dereferences an optional flag, treating an absent one as false.
func boolValue(flag *bool) bool {
	return flag != nil && *flag
}
