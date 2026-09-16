package model

import (
	"encoding/json"
	"strings"
)

// BankingEntity is a legal entity registered for banking connectivity. Only the
// fields the CLI renders are typed; the untouched response stays in Raw so
// `--json` never loses a field the spec adds later.
type BankingEntity struct {
	ID               string          `json:"id,omitempty"`
	AccountID        string          `json:"account_id,omitempty"`
	MerchantEntityID string          `json:"merchant_entity_id,omitempty"`
	NationalEntity   string          `json:"national_entity,omitempty"`
	EntityDetail     *BankingDetail  `json:"entity_detail,omitempty"`
	CreatedAt        string          `json:"created_at,omitempty"`
	UpdatedAt        string          `json:"updated_at,omitempty"`
	Raw              json.RawMessage `json:"-"`
}

// BankingDetail carries the individual or business description of an entity.
// Only the naming fields are typed, since they are the ones a table shows.
type BankingDetail struct {
	EntityType  string `json:"entity_type,omitempty"`
	LegalName   string `json:"legal_name,omitempty"`
	TradingName string `json:"trading_name,omitempty"`
	FirstName   string `json:"first_name,omitempty"`
	LastName    string `json:"last_name,omitempty"`
}

// Label renders the name of an entity, whichever field carries it.
func (d *BankingDetail) Label() string {
	switch {
	case d == nil:
		return ""
	case d.LegalName != "":
		return d.LegalName
	case d.TradingName != "":
		return d.TradingName
	case d.FirstName != "" || d.LastName != "":
		return strings.TrimSpace(d.FirstName + " " + d.LastName)
	default:
		return ""
	}
}

// UnmarshalJSON decodes the typed fields and keeps the original body.
func (e *BankingEntity) UnmarshalJSON(data []byte) error {
	type alias BankingEntity

	var decoded alias
	if err := json.Unmarshal(data, &decoded); err != nil {
		return err
	}

	*e = BankingEntity(decoded)
	e.Raw = append(json.RawMessage(nil), data...)

	return nil
}

// MarshalJSON replays the original response when there is one.
//
//nolint:gocritic // hugeParam: a value receiver is required by json.Marshaler here
func (e BankingEntity) MarshalJSON() ([]byte, error) {
	if len(e.Raw) > 0 {
		return e.Raw, nil
	}

	type alias BankingEntity

	return json.Marshal(alias(e))
}

// BankingEntityView is the flat table row of an entity.
type BankingEntityView struct {
	ID               string `json:"id"`
	MerchantEntityID string `json:"merchant_entity_id"`
	Name             string `json:"name"`
	NationalEntity   string `json:"national_entity"`
	AccountID        string `json:"account_id"`
	CreatedAt        string `json:"created_at"`
}

// View flattens an entity into a table row.
func (e *BankingEntity) View() BankingEntityView {
	return BankingEntityView{
		ID:               e.ID,
		MerchantEntityID: e.MerchantEntityID,
		Name:             e.EntityDetail.Label(),
		NationalEntity:   e.NationalEntity,
		AccountID:        e.AccountID,
		CreatedAt:        e.CreatedAt,
	}
}

// BankingOnboarding is one onboarding of an entity with a banking provider.
type BankingOnboarding struct {
	ID             string          `json:"id,omitempty"`
	EntityID       string          `json:"entity_id,omitempty"`
	YunoAccountID  string          `json:"yuno_account_id,omitempty"`
	Provider       string          `json:"provider,omitempty"`
	OnboardingType string          `json:"onboarding_type,omitempty"`
	Status         string          `json:"status,omitempty"`
	CreatedAt      string          `json:"created_at,omitempty"`
	UpdatedAt      string          `json:"updated_at,omitempty"`
	ExpiresAt      string          `json:"expires_at,omitempty"`
	Raw            json.RawMessage `json:"-"`
}

// UnmarshalJSON decodes the typed fields and keeps the original body.
func (o *BankingOnboarding) UnmarshalJSON(data []byte) error {
	type alias BankingOnboarding

	var decoded alias
	if err := json.Unmarshal(data, &decoded); err != nil {
		return err
	}

	*o = BankingOnboarding(decoded)
	o.Raw = append(json.RawMessage(nil), data...)

	return nil
}

// MarshalJSON replays the original response when there is one.
//
//nolint:gocritic // hugeParam: a value receiver is required by json.Marshaler here
func (o BankingOnboarding) MarshalJSON() ([]byte, error) {
	if len(o.Raw) > 0 {
		return o.Raw, nil
	}

	type alias BankingOnboarding

	return json.Marshal(alias(o))
}

// BankingOnboardingView is the flat table row of an onboarding.
type BankingOnboardingView struct {
	ID             string `json:"id"`
	EntityID       string `json:"entity_id"`
	OnboardingType string `json:"onboarding_type"`
	Status         string `json:"status"`
	Provider       string `json:"provider"`
	ExpiresAt      string `json:"expires_at"`
}

// View flattens an onboarding into a table row.
func (o *BankingOnboarding) View() BankingOnboardingView {
	return BankingOnboardingView{
		ID:             o.ID,
		EntityID:       o.EntityID,
		OnboardingType: o.OnboardingType,
		Status:         o.Status,
		Provider:       o.Provider,
		ExpiresAt:      o.ExpiresAt,
	}
}

// BankingAccount is a bank account opened for an onboarded entity.
type BankingAccount struct {
	ID            string          `json:"id,omitempty"`
	EntityID      string          `json:"entity_id,omitempty"`
	OnboardingID  string          `json:"onboarding_id,omitempty"`
	AccountID     string          `json:"account_id,omitempty"`
	Provider      string          `json:"provider,omitempty"`
	AccountType   string          `json:"account_type,omitempty"`
	Status        string          `json:"status,omitempty"`
	AccountNumber string          `json:"account_number,omitempty"`
	RoutingNumber string          `json:"routing_number,omitempty"`
	IBAN          string          `json:"iban,omitempty"`
	Currency      string          `json:"currency,omitempty"`
	Balance       *Amount         `json:"balance,omitempty"`
	Raw           json.RawMessage `json:"-"`
}

// UnmarshalJSON decodes the typed fields and keeps the original body.
func (a *BankingAccount) UnmarshalJSON(data []byte) error {
	type alias BankingAccount

	var decoded alias
	if err := json.Unmarshal(data, &decoded); err != nil {
		return err
	}

	*a = BankingAccount(decoded)
	a.Raw = append(json.RawMessage(nil), data...)

	return nil
}

// MarshalJSON replays the original response when there is one.
//
//nolint:gocritic // hugeParam: a value receiver is required by json.Marshaler here
func (a BankingAccount) MarshalJSON() ([]byte, error) {
	if len(a.Raw) > 0 {
		return a.Raw, nil
	}

	type alias BankingAccount

	return json.Marshal(alias(a))
}

// BankingAccountView is the flat table row of a bank account.
type BankingAccountView struct {
	ID          string `json:"id"`
	EntityID    string `json:"entity_id"`
	AccountType string `json:"account_type"`
	Status      string `json:"status"`
	Currency    string `json:"currency"`
	Balance     string `json:"balance"`
	Provider    string `json:"provider"`
}

// View flattens a bank account into a table row.
func (a *BankingAccount) View() BankingAccountView {
	return BankingAccountView{
		ID:          a.ID,
		EntityID:    a.EntityID,
		AccountType: a.AccountType,
		Status:      a.Status,
		Currency:    a.Currency,
		Balance:     a.Balance.Label(),
		Provider:    a.Provider,
	}
}

// BankingTransfer is one money movement between banking accounts.
type BankingTransfer struct {
	ID                   string          `json:"id,omitempty"`
	SourceAccountID      string          `json:"source_account_id,omitempty"`
	DestinationAccountID string          `json:"destination_account_id,omitempty"`
	AccountID            string          `json:"account_id,omitempty"`
	MerchantTransferID   string          `json:"merchant_transfer_id,omitempty"`
	Provider             string          `json:"provider,omitempty"`
	Status               string          `json:"status,omitempty"`
	Direction            string          `json:"direction,omitempty"`
	PaymentRail          string          `json:"payment_rail,omitempty"`
	Description          string          `json:"description,omitempty"`
	Amount               *Amount         `json:"amount,omitempty"`
	CreatedAt            string          `json:"created_at,omitempty"`
	UpdatedAt            string          `json:"updated_at,omitempty"`
	Raw                  json.RawMessage `json:"-"`
}

// UnmarshalJSON decodes the typed fields and keeps the original body.
func (t *BankingTransfer) UnmarshalJSON(data []byte) error {
	type alias BankingTransfer

	var decoded alias
	if err := json.Unmarshal(data, &decoded); err != nil {
		return err
	}

	*t = BankingTransfer(decoded)
	t.Raw = append(json.RawMessage(nil), data...)

	return nil
}

// MarshalJSON replays the original response when there is one.
//
//nolint:gocritic // hugeParam: a value receiver is required by json.Marshaler here
func (t BankingTransfer) MarshalJSON() ([]byte, error) {
	if len(t.Raw) > 0 {
		return t.Raw, nil
	}

	type alias BankingTransfer

	return json.Marshal(alias(t))
}

// BankingTransferView is the flat table row of a transfer.
type BankingTransferView struct {
	ID              string `json:"id"`
	SourceAccountID string `json:"source_account_id"`
	Direction       string `json:"direction"`
	Status          string `json:"status"`
	Amount          string `json:"amount"`
	PaymentRail     string `json:"payment_rail"`
	CreatedAt       string `json:"created_at"`
}

// View flattens a transfer into a table row.
func (t *BankingTransfer) View() BankingTransferView {
	return BankingTransferView{
		ID:              t.ID,
		SourceAccountID: t.SourceAccountID,
		Direction:       t.Direction,
		Status:          t.Status,
		Amount:          t.Amount.Label(),
		PaymentRail:     t.PaymentRail,
		CreatedAt:       t.CreatedAt,
	}
}
