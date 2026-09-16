package model

import (
	"encoding/json"
)

// PCIProxyDestination is one hostname on the forward proxy allowlist. Only the
// fields the CLI renders are typed; the untouched response stays in Raw so
// `--json` never loses a field the spec adds later.
type PCIProxyDestination struct {
	ID        string `json:"id,omitempty"`
	AccountID string `json:"account_id,omitempty"`
	Hostname  string `json:"hostname,omitempty"`
	Status    string `json:"status,omitempty"`
	Purpose   string `json:"purpose,omitempty"`
	CreatedAt string `json:"created_at,omitempty"`
	UpdatedAt string `json:"updated_at,omitempty"`

	// Raw is the verbatim response body, set when the destination was decoded
	// from one.
	Raw json.RawMessage `json:"-"`
}

// UnmarshalJSON decodes the typed fields and keeps the original body.
func (d *PCIProxyDestination) UnmarshalJSON(data []byte) error {
	type alias PCIProxyDestination

	var decoded alias
	if err := json.Unmarshal(data, &decoded); err != nil {
		return err
	}

	*d = PCIProxyDestination(decoded)
	d.Raw = append(json.RawMessage(nil), data...)

	return nil
}

// MarshalJSON replays the original response when there is one.
//
//nolint:gocritic // hugeParam: a value receiver is required by json.Marshaler here
func (d PCIProxyDestination) MarshalJSON() ([]byte, error) {
	if len(d.Raw) > 0 {
		return d.Raw, nil
	}

	type alias PCIProxyDestination

	return json.Marshal(alias(d))
}

// PCIProxyDestinationView is the flat table row of a destination.
type PCIProxyDestinationView struct {
	ID        string `json:"id"`
	Hostname  string `json:"hostname"`
	Status    string `json:"status"`
	Purpose   string `json:"purpose"`
	AccountID string `json:"account_id"`
	CreatedAt string `json:"created_at"`
}

// View flattens a destination into a table row.
func (d *PCIProxyDestination) View() PCIProxyDestinationView {
	return PCIProxyDestinationView{
		ID:        d.ID,
		Hostname:  d.Hostname,
		Status:    d.Status,
		Purpose:   d.Purpose,
		AccountID: d.AccountID,
		CreatedAt: d.CreatedAt,
	}
}

// PCIProxyDestinationViews flattens a list of destinations into table rows.
func PCIProxyDestinationViews(destinations []PCIProxyDestination) []PCIProxyDestinationView {
	views := make([]PCIProxyDestinationView, 0, len(destinations))
	for i := range destinations {
		views = append(views, destinations[i].View())
	}

	return views
}

// PCIProxyDestinationList is the envelope `GET /pci-proxy/destinations` answers
// with: the allowlist is not paginated, it arrives whole under `destinations`.
type PCIProxyDestinationList struct {
	Destinations []PCIProxyDestination `json:"destinations"`
}
