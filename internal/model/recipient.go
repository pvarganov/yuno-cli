package model

import (
	"encoding/json"
	"strings"
)

// Recipient is a marketplace recipient: the party a split marketplace transfer
// pays out to. Only the fields the CLI renders are typed; the untouched
// response stays in Raw so `--json` never loses a field the spec adds later.
type Recipient struct {
	ID                   string                `json:"id,omitempty"`
	AccountID            string                `json:"account_id,omitempty"`
	MerchantRecipientID  string                `json:"merchant_recipient_id,omitempty"`
	NationalEntity       string                `json:"national_entity,omitempty"`
	EntityType           string                `json:"entity_type,omitempty"`
	FirstName            string                `json:"first_name,omitempty"`
	LastName             string                `json:"last_name,omitempty"`
	LegalName            *string               `json:"legal_name,omitempty"`
	Email                string                `json:"email,omitempty"`
	DateOfBirth          string                `json:"date_of_birth,omitempty"`
	Country              string                `json:"country,omitempty"`
	Website              string                `json:"website,omitempty"`
	Industry             string                `json:"industry,omitempty"`
	MerchantCategoryCode string                `json:"merchant_category_code,omitempty"`
	Document             *Document             `json:"document,omitempty"`
	Phone                *Phone                `json:"phone,omitempty"`
	Address              *RecipientAddress     `json:"address,omitempty"`
	SplitConfiguration   *SplitConfiguration   `json:"split_configuration,omitempty"`
	Onboardings          []RecipientOnboarding `json:"onboardings,omitempty"`
	CreatedAt            string                `json:"created_at,omitempty"`
	UpdatedAt            string                `json:"updated_at,omitempty"`

	// Raw is the verbatim response body, set when the recipient was decoded
	// from one.
	Raw json.RawMessage `json:"-"`
}

// RecipientAddress is the postal address of a recipient.
type RecipientAddress struct {
	AddressLine1 string `json:"address_line_1,omitempty"`
	AddressLine2 string `json:"address_line_2,omitempty"`
	City         string `json:"city,omitempty"`
	State        string `json:"state,omitempty"`
	Country      string `json:"country,omitempty"`
	ZipCode      string `json:"zip_code,omitempty"`
	Neighborhood string `json:"neighborhood,omitempty"`
}

// SplitConfiguration is the share of a payment a recipient receives.
type SplitConfiguration struct {
	CalculationType string  `json:"calculation_type,omitempty"`
	Percentage      float64 `json:"percentage,omitempty"`
	FixedAmount     float64 `json:"fixed_amount,omitempty"`
	Currency        string  `json:"currency,omitempty"`
	RoundingMode    string  `json:"rounding_mode,omitempty"`
}

// RecipientOnboarding is one onboarding of a recipient with one provider: the
// KYC flow that has to succeed before the recipient can be transferred to.
type RecipientOnboarding struct {
	ID              string               `json:"id,omitempty"`
	AccountID       string               `json:"account_id,omitempty"`
	RecipientID     string               `json:"recipient_id,omitempty"`
	Type            string               `json:"type,omitempty"`
	Workflow        string               `json:"workflow,omitempty"`
	Status          string               `json:"status,omitempty"`
	Description     string               `json:"description,omitempty"`
	ResponseMessage *string              `json:"response_message,omitempty"`
	CallbackURL     string               `json:"callback_url,omitempty"`
	Provider        *OnboardingProvider  `json:"provider,omitempty"`
	TermsOfService  *OnboardingTermsOfSv `json:"terms_of_service,omitempty"`
	CreatedAt       string               `json:"created_at,omitempty"`
	UpdatedAt       string               `json:"updated_at,omitempty"`

	// Raw is the verbatim response body, set when the onboarding was decoded
	// from one.
	Raw json.RawMessage `json:"-"`
}

// OnboardingProvider is the provider side of an onboarding.
type OnboardingProvider struct {
	ID            string `json:"id,omitempty"`
	ConnectionID  string `json:"connection_id,omitempty"`
	RecipientID   string `json:"recipient_id,omitempty"`
	RecipientType string `json:"recipient_type,omitempty"`
}

// OnboardingTermsOfSv records the terms of service acceptance of an onboarding.
type OnboardingTermsOfSv struct {
	Acceptance bool   `json:"acceptance,omitempty"`
	Date       string `json:"date,omitempty"`
	IP         string `json:"ip,omitempty"`
}

// RecipientTransfer is a transfer of a recipient between two onboardings: the
// money moved when a recipient is migrated from one provider to another.
type RecipientTransfer struct {
	ID                    string               `json:"id,omitempty"`
	OriginOnboarding      *RecipientOnboarding `json:"origin_onboarding,omitempty"`
	DestinationOnboarding *RecipientOnboarding `json:"destination_onboarding,omitempty"`
	CreatedAt             string               `json:"created_at,omitempty"`
	UpdatedAt             string               `json:"updated_at,omitempty"`

	// Raw is the verbatim response body, set when the transfer was decoded
	// from one.
	Raw json.RawMessage `json:"-"`
}

// UnmarshalJSON decodes the typed fields and keeps the original body.
func (r *Recipient) UnmarshalJSON(data []byte) error {
	type alias Recipient

	var decoded alias
	if err := json.Unmarshal(data, &decoded); err != nil {
		return err
	}

	*r = Recipient(decoded)
	r.Raw = append(json.RawMessage(nil), data...)

	return nil
}

// MarshalJSON replays the original response when there is one.
//
//nolint:gocritic // hugeParam: a value receiver is required by json.Marshaler here
func (r Recipient) MarshalJSON() ([]byte, error) {
	if len(r.Raw) > 0 {
		return r.Raw, nil
	}

	type alias Recipient

	return json.Marshal(alias(r))
}

// UnmarshalJSON decodes the typed fields and keeps the original body.
func (o *RecipientOnboarding) UnmarshalJSON(data []byte) error {
	type alias RecipientOnboarding

	var decoded alias
	if err := json.Unmarshal(data, &decoded); err != nil {
		return err
	}

	*o = RecipientOnboarding(decoded)
	o.Raw = append(json.RawMessage(nil), data...)

	return nil
}

// MarshalJSON replays the original response when there is one.
//
//nolint:gocritic // hugeParam: a value receiver is required by json.Marshaler here
func (o RecipientOnboarding) MarshalJSON() ([]byte, error) {
	if len(o.Raw) > 0 {
		return o.Raw, nil
	}

	type alias RecipientOnboarding

	return json.Marshal(alias(o))
}

// UnmarshalJSON decodes the typed fields and keeps the original body.
func (t *RecipientTransfer) UnmarshalJSON(data []byte) error {
	type alias RecipientTransfer

	var decoded alias
	if err := json.Unmarshal(data, &decoded); err != nil {
		return err
	}

	*t = RecipientTransfer(decoded)
	t.Raw = append(json.RawMessage(nil), data...)

	return nil
}

// MarshalJSON replays the original response when there is one.
//
//nolint:gocritic // hugeParam: a value receiver is required by json.Marshaler here
func (t RecipientTransfer) MarshalJSON() ([]byte, error) {
	if len(t.Raw) > 0 {
		return t.Raw, nil
	}

	type alias RecipientTransfer

	return json.Marshal(alias(t))
}

// RecipientView is the flat table row of a recipient.
type RecipientView struct {
	ID                  string `json:"id"`
	MerchantRecipientID string `json:"merchant_recipient_id"`
	Name                string `json:"name"`
	EntityType          string `json:"entity_type"`
	Country             string `json:"country"`
	Email               string `json:"email"`
	Onboarding          string `json:"onboarding"`
	CreatedAt           string `json:"created_at"`
}

// View flattens a recipient into a table row.
func (r *Recipient) View() RecipientView {
	return RecipientView{
		ID:                  r.ID,
		MerchantRecipientID: r.MerchantRecipientID,
		Name:                r.Name(),
		EntityType:          r.EntityType,
		Country:             r.Country,
		Email:               r.Email,
		Onboarding:          r.OnboardingLabel(),
		CreatedAt:           r.CreatedAt,
	}
}

// Name renders the recipient as the best name it carries, falling back to the
// merchant reference.
func (r *Recipient) Name() string {
	if name := strings.TrimSpace(r.FirstName + " " + r.LastName); name != "" {
		return name
	}

	if r.LegalName != nil && *r.LegalName != "" {
		return *r.LegalName
	}

	return r.MerchantRecipientID
}

// OnboardingLabel summarises the onboardings of a recipient as
// `PROVIDER STATUS`, listing every onboarding the response carries.
func (r *Recipient) OnboardingLabel() string {
	labels := make([]string, 0, len(r.Onboardings))

	for i := range r.Onboardings {
		if label := r.Onboardings[i].Label(); label != "" {
			labels = append(labels, label)
		}
	}

	return strings.Join(labels, ", ")
}

// Label renders one onboarding as `PROVIDER STATUS`.
func (o *RecipientOnboarding) Label() string {
	if o == nil {
		return ""
	}

	provider := ""
	if o.Provider != nil {
		provider = o.Provider.ID
	}

	return strings.TrimSpace(provider + " " + o.Status)
}

// RecipientViews flattens a list of recipients.
func RecipientViews(recipients []Recipient) []RecipientView {
	views := make([]RecipientView, 0, len(recipients))
	for i := range recipients {
		views = append(views, recipients[i].View())
	}

	return views
}

// RecipientOnboardingView is the flat table row of an onboarding.
type RecipientOnboardingView struct {
	ID          string `json:"id"`
	RecipientID string `json:"recipient_id"`
	Status      string `json:"status"`
	Type        string `json:"type"`
	Workflow    string `json:"workflow"`
	Provider    string `json:"provider"`
	Response    string `json:"response"`
	CreatedAt   string `json:"created_at"`
}

// View flattens an onboarding into a table row.
func (o *RecipientOnboarding) View() RecipientOnboardingView {
	view := RecipientOnboardingView{
		ID:          o.ID,
		RecipientID: o.RecipientID,
		Status:      o.Status,
		Type:        o.Type,
		Workflow:    o.Workflow,
		CreatedAt:   o.CreatedAt,
	}

	if o.Provider != nil {
		view.Provider = o.Provider.ID
	}

	if o.ResponseMessage != nil {
		view.Response = *o.ResponseMessage
	}

	return view
}

// RecipientOnboardingViews flattens a list of onboardings.
func RecipientOnboardingViews(onboardings []RecipientOnboarding) []RecipientOnboardingView {
	views := make([]RecipientOnboardingView, 0, len(onboardings))
	for i := range onboardings {
		views = append(views, onboardings[i].View())
	}

	return views
}

// RecipientTransferView is the flat table row of a recipient transfer.
type RecipientTransferView struct {
	ID          string `json:"id"`
	Origin      string `json:"origin_onboarding"`
	Destination string `json:"destination_onboarding"`
	Status      string `json:"status"`
	CreatedAt   string `json:"created_at"`
}

// View flattens a recipient transfer into a table row. The status shown is the
// one of the destination onboarding, since that is the side the transfer is
// waiting on.
func (t *RecipientTransfer) View() RecipientTransferView {
	view := RecipientTransferView{
		ID:        t.ID,
		CreatedAt: t.CreatedAt,
	}

	if o := t.OriginOnboarding; o != nil {
		view.Origin = o.ID
	}

	if d := t.DestinationOnboarding; d != nil {
		view.Destination = d.ID
		view.Status = d.Status
	}

	return view
}

// RecipientTransferViews flattens a list of recipient transfers.
func RecipientTransferViews(transfers []RecipientTransfer) []RecipientTransferView {
	views := make([]RecipientTransferView, 0, len(transfers))
	for i := range transfers {
		views = append(views, transfers[i].View())
	}

	return views
}
