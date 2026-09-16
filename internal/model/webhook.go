package model

import (
	"encoding/json"
	"strings"
)

// Webhook is a Yuno webhook subscription: the endpoint Yuno posts the events of
// an account to. Only the fields the CLI renders are typed; the untouched
// response stays in Raw so `--json` never loses a field the spec adds later.
type Webhook struct {
	ID                   string   `json:"id,omitempty"`
	AccountID            string   `json:"account_id,omitempty"`
	Name                 string   `json:"name,omitempty"`
	State                string   `json:"state,omitempty"`
	URL                  string   `json:"url,omitempty"`
	APIKey               *string  `json:"api_key,omitempty"`
	Secret               *string  `json:"secret,omitempty"`
	HMACClientSecret     *string  `json:"hmac_client_secret,omitempty"`
	OAuth2ClientID       *string  `json:"oauth2_client_id,omitempty"`
	OAuth2ClientSecret   *string  `json:"oauth2_client_secret,omitempty"`
	OAuth2AuthURL        *string  `json:"oauth2_authentication_url,omitempty"`
	EnrollmentTriggers   []string `json:"enrollment_triggers,omitempty"`
	PaymentTriggers      []string `json:"payment_triggers,omitempty"`
	ReportTriggers       []string `json:"report_triggers,omitempty"`
	SubscriptionTriggers []string `json:"subscription_triggers,omitempty"`
	OnboardingTriggers   []string `json:"onboarding_triggers,omitempty"`
	RenewalDays          *int     `json:"renewal_days,omitempty"`
	CreatedAt            string   `json:"created_at,omitempty"`
	UpdatedAt            string   `json:"updated_at,omitempty"`

	// Raw is the verbatim response body, set when the webhook was decoded from one.
	Raw json.RawMessage `json:"-"`
}

// UnmarshalJSON decodes the typed fields and keeps the original body.
func (w *Webhook) UnmarshalJSON(data []byte) error {
	type alias Webhook

	var decoded alias
	if err := json.Unmarshal(data, &decoded); err != nil {
		return err
	}

	*w = Webhook(decoded)
	w.Raw = append(json.RawMessage(nil), data...)

	return nil
}

// MarshalJSON replays the original response when there is one.
//
//nolint:gocritic // hugeParam: a value receiver is required by json.Marshaler here
func (w Webhook) MarshalJSON() ([]byte, error) {
	if len(w.Raw) > 0 {
		return w.Raw, nil
	}

	type alias Webhook

	return json.Marshal(alias(w))
}

// WebhookView is the flat table row of a webhook.
type WebhookView struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	State     string `json:"state"`
	URL       string `json:"url"`
	Triggers  string `json:"triggers"`
	CreatedAt string `json:"created_at"`
}

// View flattens a webhook into a table row.
func (w *Webhook) View() WebhookView {
	return WebhookView{
		ID:        w.ID,
		Name:      w.Name,
		State:     w.State,
		URL:       w.URL,
		Triggers:  w.TriggerLabel(),
		CreatedAt: w.CreatedAt,
	}
}

// TriggerLabel counts the events the webhook subscribes to, per trigger family,
// so a table row stays readable when a webhook listens to a dozen events.
func (w *Webhook) TriggerLabel() string {
	families := []struct {
		name     string
		triggers []string
	}{
		{"payment", w.PaymentTriggers},
		{"enrollment", w.EnrollmentTriggers},
		{"report", w.ReportTriggers},
		{"subscription", w.SubscriptionTriggers},
		{"onboarding", w.OnboardingTriggers},
	}

	var parts []string

	for _, family := range families {
		if len(family.triggers) > 0 {
			parts = append(parts, family.name+":"+strings.Join(family.triggers, "|"))
		}
	}

	return strings.Join(parts, " ")
}

// WebhookViews flattens a list of webhooks.
func WebhookViews(webhooks []Webhook) []WebhookView {
	views := make([]WebhookView, 0, len(webhooks))
	for i := range webhooks {
		views = append(views, webhooks[i].View())
	}

	return views
}
