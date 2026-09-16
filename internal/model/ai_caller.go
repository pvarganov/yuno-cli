package model

import (
	"encoding/json"
)

// AICallerOutreach is what the AI caller answers with when an outreach is
// accepted: an identifier for the run and a human readable message. The
// abandoned-flow endpoint answers 200 with no body at all, so every field is
// optional.
type AICallerOutreach struct {
	ID      string `json:"id,omitempty"`
	Message string `json:"message,omitempty"`

	// Raw is the verbatim response body, set when the outreach was decoded from one.
	Raw json.RawMessage `json:"-"`
}

// UnmarshalJSON decodes the typed fields and keeps the original body.
func (o *AICallerOutreach) UnmarshalJSON(data []byte) error {
	type alias AICallerOutreach

	var decoded alias
	if err := json.Unmarshal(data, &decoded); err != nil {
		return err
	}

	*o = AICallerOutreach(decoded)
	o.Raw = append(json.RawMessage(nil), data...)

	return nil
}

// MarshalJSON replays the original response when there is one.
//
//nolint:gocritic // hugeParam: a value receiver is required by json.Marshaler here
func (o AICallerOutreach) MarshalJSON() ([]byte, error) {
	if len(o.Raw) > 0 {
		return o.Raw, nil
	}

	type alias AICallerOutreach

	return json.Marshal(alias(o))
}

// AICallerOutreachView is the flat table row of an AI caller outreach.
type AICallerOutreachView struct {
	ID      string `json:"id"`
	Message string `json:"message"`
}

// View flattens an outreach into a table row.
func (o *AICallerOutreach) View() AICallerOutreachView {
	return AICallerOutreachView{ID: o.ID, Message: o.Message}
}
