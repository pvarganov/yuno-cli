package model

import (
	"encoding/json"
	"strings"
)

// Campaign is a communications campaign: the AI caller or WhatsApp outreach
// Yuno runs on an account, with the schedule and the window it runs in. Only
// the fields the CLI tabulates are typed; the untouched response stays in Raw
// so `--json` never loses a field the spec adds later.
type Campaign struct {
	ID               string            `json:"id,omitempty"`
	Name             string            `json:"name,omitempty"`
	AccountID        string            `json:"account_id,omitempty"`
	OrganizationCode string            `json:"organization_code,omitempty"`
	Country          string            `json:"country,omitempty"`
	Channel          string            `json:"channel,omitempty"`
	Focus            string            `json:"focus,omitempty"`
	Status           string            `json:"status,omitempty"`
	Schedule         *CampaignSchedule `json:"schedule,omitempty"`
	Duration         *CampaignDuration `json:"duration,omitempty"`
	Rules            []CampaignRule    `json:"rules,omitempty"`
	CreatedAt        string            `json:"created_at,omitempty"`
	UpdatedAt        string            `json:"updated_at,omitempty"`

	// Raw is the verbatim response body, set when the campaign was decoded from one.
	Raw json.RawMessage `json:"-"`
}

// CampaignSchedule is the daily send window of a campaign.
type CampaignSchedule struct {
	DailyStartTime string `json:"daily_start_time,omitempty"`
	DailyEndTime   string `json:"daily_end_time,omitempty"`
	TimeZone       string `json:"time_zone,omitempty"`
}

// CampaignDuration is the active period of a campaign.
type CampaignDuration struct {
	StartAt string `json:"start_at,omitempty"`
	EndAt   string `json:"end_at,omitempty"`
}

// UnmarshalJSON decodes the typed fields and keeps the original body. The
// single-campaign endpoints wrap the resource in a `data` envelope while the
// list items arrive bare, so both shapes have to decode.
func (c *Campaign) UnmarshalJSON(data []byte) error {
	type alias Campaign

	var decoded alias
	if err := json.Unmarshal(unwrapData(data), &decoded); err != nil {
		return err
	}

	*c = Campaign(decoded)
	c.Raw = append(json.RawMessage(nil), data...)

	return nil
}

// MarshalJSON replays the original response when there is one.
//
//nolint:gocritic // hugeParam: a value receiver is required by json.Marshaler here
func (c Campaign) MarshalJSON() ([]byte, error) {
	if len(c.Raw) > 0 {
		return c.Raw, nil
	}

	type alias Campaign

	return json.Marshal(alias(c))
}

// CampaignView is the flat table row of a campaign.
type CampaignView struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Country   string `json:"country"`
	Channel   string `json:"channel"`
	Status    string `json:"status"`
	StartAt   string `json:"start_at"`
	EndAt     string `json:"end_at"`
	CreatedAt string `json:"created_at"`
}

// View flattens a campaign into a table row.
func (c *Campaign) View() CampaignView {
	view := CampaignView{
		ID:        c.ID,
		Name:      c.Name,
		Country:   c.Country,
		Channel:   c.Channel,
		Status:    c.Status,
		CreatedAt: c.CreatedAt,
	}

	if c.Duration != nil {
		view.StartAt, view.EndAt = c.Duration.StartAt, c.Duration.EndAt
	}

	return view
}

// CampaignViews flattens a list of campaigns into table rows.
func CampaignViews(campaigns []Campaign) []CampaignView {
	views := make([]CampaignView, 0, len(campaigns))
	for i := range campaigns {
		views = append(views, campaigns[i].View())
	}

	return views
}

// CampaignRule is one targeting rule of a campaign: which payments the campaign
// reaches out about.
type CampaignRule struct {
	ID          string   `json:"id,omitempty"`
	RuleType    string   `json:"rule_type,omitempty"`
	Values      []string `json:"values,omitempty"`
	Conditional string   `json:"conditional,omitempty"`
	MetadataKey string   `json:"metadata_key,omitempty"`
	Status      string   `json:"status,omitempty"`
	CreatedAt   string   `json:"created_at,omitempty"`
	UpdatedAt   string   `json:"updated_at,omitempty"`

	// Raw is the verbatim response body, set when the rule was decoded from one.
	Raw json.RawMessage `json:"-"`
}

// UnmarshalJSON decodes the typed fields and keeps the original body, unwrapping
// the `data` envelope the single-rule endpoints answer with.
func (r *CampaignRule) UnmarshalJSON(data []byte) error {
	type alias CampaignRule

	var decoded alias
	if err := json.Unmarshal(unwrapData(data), &decoded); err != nil {
		return err
	}

	*r = CampaignRule(decoded)
	r.Raw = append(json.RawMessage(nil), data...)

	return nil
}

// MarshalJSON replays the original response when there is one.
//
//nolint:gocritic // hugeParam: a value receiver is required by json.Marshaler here
func (r CampaignRule) MarshalJSON() ([]byte, error) {
	if len(r.Raw) > 0 {
		return r.Raw, nil
	}

	type alias CampaignRule

	return json.Marshal(alias(r))
}

// CampaignRuleList is the `data` envelope `POST /campaigns/{id}/rules` answers
// with: creating rules returns every rule that was created, not just one.
type CampaignRuleList struct {
	Data []CampaignRule `json:"data,omitempty"`
}

// CampaignRuleView is the flat table row of a campaign rule.
type CampaignRuleView struct {
	ID          string `json:"id"`
	RuleType    string `json:"rule_type"`
	Conditional string `json:"conditional"`
	Values      string `json:"values"`
	MetadataKey string `json:"metadata_key"`
	Status      string `json:"status"`
	CreatedAt   string `json:"created_at"`
}

// View flattens a campaign rule into a table row.
func (r *CampaignRule) View() CampaignRuleView {
	return CampaignRuleView{
		ID:          r.ID,
		RuleType:    r.RuleType,
		Conditional: r.Conditional,
		Values:      strings.Join(r.Values, ","),
		MetadataKey: r.MetadataKey,
		Status:      r.Status,
		CreatedAt:   r.CreatedAt,
	}
}

// CampaignRuleViews flattens a list of campaign rules into table rows.
func CampaignRuleViews(rules []CampaignRule) []CampaignRuleView {
	views := make([]CampaignRuleView, 0, len(rules))
	for i := range rules {
		views = append(views, rules[i].View())
	}

	return views
}

// unwrapData returns the body of a `data` envelope when the response is one, and
// the response itself otherwise. Only an object payload is unwrapped: a `data`
// array belongs to a list response, which the pager decodes on its own.
func unwrapData(data []byte) []byte {
	var envelope struct {
		Data json.RawMessage `json:"data"`
	}

	if err := json.Unmarshal(data, &envelope); err != nil {
		return data
	}

	body := json.RawMessage(strings.TrimSpace(string(envelope.Data)))
	if len(body) == 0 || body[0] != '{' {
		return data
	}

	return body
}
