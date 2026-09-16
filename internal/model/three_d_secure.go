package model

import (
	"encoding/json"
	"strings"
)

// ThreeDSecureSetup is the 3-D Secure setup Yuno creates for a browser, used
// to collect the device fingerprints a later payment authenticates with.
type ThreeDSecureSetup struct {
	ID                 string              `json:"three_d_secure_setup_id,omitempty"`
	AccountID          string              `json:"account_id,omitempty"`
	Type               string              `json:"type,omitempty"`
	BrowserInfo        *BrowserInfo        `json:"browser_info,omitempty"`
	DeviceFingerprints []DeviceFingerprint `json:"device_fingerprints,omitempty"`

	// Raw is the verbatim response body, set when the setup was decoded from one.
	Raw json.RawMessage `json:"-"`
}

// BrowserInfo is the browser the 3-D Secure setup was created for.
type BrowserInfo struct {
	UserAgent             string `json:"user_agent,omitempty"`
	AcceptHeader          string `json:"accept_header,omitempty"`
	ColorDepth            string `json:"color_depth,omitempty"`
	ScreenHeight          string `json:"screen_height,omitempty"`
	ScreenWidth           string `json:"screen_width,omitempty"`
	Language              string `json:"language,omitempty"`
	JavascriptEnabled     *bool  `json:"javascript_enabled,omitempty"`
	JavaEnabled           *bool  `json:"java_enabled,omitempty"`
	BrowserTimeDifference string `json:"browser_time_difference,omitempty"`
	Platform              string `json:"platform,omitempty"`
}

// DeviceFingerprint is one provider fingerprint of a 3-D Secure setup.
type DeviceFingerprint struct {
	ProviderID string `json:"provider_id,omitempty"`
	ID         string `json:"id,omitempty"`
}

// UnmarshalJSON decodes the typed fields and keeps the original body.
func (s *ThreeDSecureSetup) UnmarshalJSON(data []byte) error {
	type alias ThreeDSecureSetup

	var decoded alias
	if err := json.Unmarshal(data, &decoded); err != nil {
		return err
	}

	*s = ThreeDSecureSetup(decoded)
	s.Raw = append(json.RawMessage(nil), data...)

	return nil
}

// MarshalJSON replays the original response when there is one.
//
//nolint:gocritic // hugeParam: a value receiver is required by json.Marshaler here
func (s ThreeDSecureSetup) MarshalJSON() ([]byte, error) {
	if len(s.Raw) > 0 {
		return s.Raw, nil
	}

	type alias ThreeDSecureSetup

	return json.Marshal(alias(s))
}

// ThreeDSecureSetupView is the flat table row of a 3-D Secure setup.
type ThreeDSecureSetupView struct {
	ID           string `json:"three_d_secure_setup_id"`
	AccountID    string `json:"account_id"`
	Type         string `json:"type"`
	Screen       string `json:"screen"`
	Language     string `json:"language"`
	Fingerprints string `json:"device_fingerprints"`
}

// View flattens a 3-D Secure setup into a table row.
func (s *ThreeDSecureSetup) View() ThreeDSecureSetupView {
	view := ThreeDSecureSetupView{
		ID:           s.ID,
		AccountID:    s.AccountID,
		Type:         s.Type,
		Fingerprints: fingerprintsLabel(s.DeviceFingerprints),
	}

	if info := s.BrowserInfo; info != nil {
		view.Screen = strings.TrimSpace(info.ScreenWidth + "x" + info.ScreenHeight)
		view.Language = info.Language
	}

	return view
}

// fingerprintsLabel renders the fingerprints as `PROVIDER=ID` pairs.
func fingerprintsLabel(fingerprints []DeviceFingerprint) string {
	labels := make([]string, 0, len(fingerprints))
	for _, f := range fingerprints {
		labels = append(labels, f.ProviderID+"="+f.ID)
	}

	return strings.Join(labels, ",")
}

// DryRunProviderEvent is a simulated provider event, used to replay a provider
// callback against the sandbox without the provider being involved.
type DryRunProviderEvent struct {
	ID                string  `json:"id,omitempty"`
	Status            string  `json:"status,omitempty"`
	MerchantReference string  `json:"merchant_reference,omitempty"`
	PaymentID         *string `json:"payment_id,omitempty"`
	AccountID         *string `json:"account_id,omitempty"`
	ProviderID        string  `json:"provider_id,omitempty"`
	PaymentMethodType string  `json:"payment_method_type,omitempty"`
	OperationType     string  `json:"operation_type,omitempty"`

	// Raw is the verbatim response body, set when the event was decoded from one.
	Raw json.RawMessage `json:"-"`
}

// UnmarshalJSON decodes the typed fields and keeps the original body.
func (e *DryRunProviderEvent) UnmarshalJSON(data []byte) error {
	type alias DryRunProviderEvent

	var decoded alias
	if err := json.Unmarshal(data, &decoded); err != nil {
		return err
	}

	*e = DryRunProviderEvent(decoded)
	e.Raw = append(json.RawMessage(nil), data...)

	return nil
}

// MarshalJSON replays the original response when there is one.
//
//nolint:gocritic // hugeParam: a value receiver is required by json.Marshaler here
func (e DryRunProviderEvent) MarshalJSON() ([]byte, error) {
	if len(e.Raw) > 0 {
		return e.Raw, nil
	}

	type alias DryRunProviderEvent

	return json.Marshal(alias(e))
}

// DryRunProviderEventView is the flat table row of a dry run provider event.
type DryRunProviderEventView struct {
	ID                string `json:"id"`
	Status            string `json:"status"`
	MerchantReference string `json:"merchant_reference"`
	PaymentID         string `json:"payment_id"`
	ProviderID        string `json:"provider_id"`
	PaymentMethodType string `json:"payment_method_type"`
	OperationType     string `json:"operation_type"`
}

// View flattens a dry run provider event into a table row.
func (e *DryRunProviderEvent) View() DryRunProviderEventView {
	return DryRunProviderEventView{
		ID:                e.ID,
		Status:            e.Status,
		MerchantReference: e.MerchantReference,
		PaymentID:         stringOrEmpty(e.PaymentID),
		ProviderID:        e.ProviderID,
		PaymentMethodType: e.PaymentMethodType,
		OperationType:     e.OperationType,
	}
}

// stringOrEmpty dereferences an optional string field.
func stringOrEmpty(v *string) string {
	if v == nil {
		return ""
	}

	return *v
}
