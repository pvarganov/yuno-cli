package model

import (
	"encoding/json"
)

// Report is a Yuno report run: an asynchronous export of payments, payouts or
// settlements. Only the fields the CLI renders are typed; the untouched
// response stays in Raw so `--json` never loses a field the spec adds later.
type Report struct {
	ID                  string `json:"id,omitempty"`
	Type                string `json:"type,omitempty"`
	Status              string `json:"status,omitempty"`
	MerchantReferenceID string `json:"merchant_reference_id,omitempty"`
	StartDate           string `json:"start_date,omitempty"`
	EndDate             string `json:"end_date,omitempty"`
	ExpiresAt           string `json:"expires_at,omitempty"`
	CreatedAt           string `json:"created_at,omitempty"`
	UpdatedAt           string `json:"updated_at,omitempty"`

	// Raw is the verbatim response body, set when the report was decoded from one.
	Raw json.RawMessage `json:"-"`
}

// UnmarshalJSON decodes the typed fields and keeps the original body.
func (r *Report) UnmarshalJSON(data []byte) error {
	type alias Report

	var decoded alias
	if err := json.Unmarshal(data, &decoded); err != nil {
		return err
	}

	*r = Report(decoded)
	r.Raw = append(json.RawMessage(nil), data...)

	return nil
}

// MarshalJSON replays the original response when there is one.
//
//nolint:gocritic // hugeParam: a value receiver is required by json.Marshaler here
func (r Report) MarshalJSON() ([]byte, error) {
	if len(r.Raw) > 0 {
		return r.Raw, nil
	}

	type alias Report

	return json.Marshal(alias(r))
}

// ReportView is the flat table row of a report.
type ReportView struct {
	ID                  string `json:"id"`
	Type                string `json:"type"`
	Status              string `json:"status"`
	MerchantReferenceID string `json:"merchant_reference_id"`
	StartDate           string `json:"start_date"`
	EndDate             string `json:"end_date"`
	ExpiresAt           string `json:"expires_at"`
}

// View flattens a report into a table row.
func (r *Report) View() ReportView {
	return ReportView{
		ID:                  r.ID,
		Type:                r.Type,
		Status:              r.Status,
		MerchantReferenceID: r.MerchantReferenceID,
		StartDate:           r.StartDate,
		EndDate:             r.EndDate,
		ExpiresAt:           r.ExpiresAt,
	}
}

// ReportViews flattens a list of reports.
func ReportViews(reports []Report) []ReportView {
	views := make([]ReportView, 0, len(reports))
	for i := range reports {
		views = append(views, reports[i].View())
	}

	return views
}

// ReportDownload is the answer of `GET /reports/{id}/download`: a pre-signed
// link to the generated file, valid until the report expires.
type ReportDownload struct {
	Type         string `json:"type,omitempty"`
	Status       string `json:"status,omitempty"`
	DownloadLink string `json:"download_link,omitempty"`

	// Raw is the verbatim response body, set when the answer was decoded from one.
	Raw json.RawMessage `json:"-"`
}

// UnmarshalJSON decodes the typed fields and keeps the original body.
func (d *ReportDownload) UnmarshalJSON(data []byte) error {
	type alias ReportDownload

	var decoded alias
	if err := json.Unmarshal(data, &decoded); err != nil {
		return err
	}

	*d = ReportDownload(decoded)
	d.Raw = append(json.RawMessage(nil), data...)

	return nil
}

// MarshalJSON replays the original response when there is one.
//
//nolint:gocritic // hugeParam: a value receiver is required by json.Marshaler here
func (d ReportDownload) MarshalJSON() ([]byte, error) {
	if len(d.Raw) > 0 {
		return d.Raw, nil
	}

	type alias ReportDownload

	return json.Marshal(alias(d))
}

// ReportDownloadView is the flat table row of a download link.
type ReportDownloadView struct {
	Type         string `json:"type"`
	Status       string `json:"status"`
	DownloadLink string `json:"download_link"`
}

// View flattens a download answer into a table row.
func (d *ReportDownload) View() ReportDownloadView {
	return ReportDownloadView{
		Type:         d.Type,
		Status:       d.Status,
		DownloadLink: d.DownloadLink,
	}
}
