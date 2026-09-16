package model

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestReportView_FlattensTheRun(t *testing.T) {
	t.Parallel()

	var report Report
	if err := json.Unmarshal([]byte(`{"id":"rp-1","type":"PAYMENTS","status":"SUCCEEDED",
		"merchant_reference_id":"ref-1","start_date":"2026-09-01T00:00:00Z",
		"end_date":"2026-09-15T00:00:00Z","expires_at":"2026-09-30T00:00:00Z"}`), &report); err != nil {
		t.Fatalf("decode report: %v", err)
	}

	view := report.View()
	if view.ID != "rp-1" || view.Type != "PAYMENTS" || view.Status != "SUCCEEDED" ||
		view.MerchantReferenceID != "ref-1" || view.ExpiresAt != "2026-09-30T00:00:00Z" {
		t.Errorf("unexpected view: %+v", view)
	}
}

func TestReportMarshal_ReplaysTheOriginalBody(t *testing.T) {
	t.Parallel()

	var report Report
	if err := json.Unmarshal([]byte(`{"id":"rp-1","unknown_field":"kept"}`), &report); err != nil {
		t.Fatalf("decode report: %v", err)
	}

	out, err := json.Marshal(report)
	if err != nil {
		t.Fatalf("marshal report: %v", err)
	}

	if !strings.Contains(string(out), "unknown_field") {
		t.Errorf("expected the original body replayed, got %s", out)
	}
}

func TestReportMarshal_FallsBackToTheTypedFields(t *testing.T) {
	t.Parallel()

	out, err := json.Marshal(Report{ID: "rp-1"})
	if err != nil {
		t.Fatalf("marshal report: %v", err)
	}

	if !strings.Contains(string(out), `"id":"rp-1"`) {
		t.Errorf("unexpected json: %s", out)
	}
}

func TestReportViews_FlattensTheList(t *testing.T) {
	t.Parallel()

	views := ReportViews([]Report{{ID: "rp-1"}, {ID: "rp-2"}})
	if len(views) != 2 || views[0].ID != "rp-1" {
		t.Errorf("unexpected views: %+v", views)
	}
}

func TestReportDownload_ViewAndMarshal(t *testing.T) {
	t.Parallel()

	var download ReportDownload
	if err := json.Unmarshal([]byte(
		`{"type":"PAYMENTS","status":"AVAILABLE","download_link":"https://s3/report.csv","extra":1}`,
	), &download); err != nil {
		t.Fatalf("decode download: %v", err)
	}

	if view := download.View(); view.DownloadLink != "https://s3/report.csv" || view.Status != "AVAILABLE" {
		t.Errorf("unexpected view: %+v", view)
	}

	out, err := json.Marshal(download)
	if err != nil {
		t.Fatalf("marshal download: %v", err)
	}

	if !strings.Contains(string(out), "extra") {
		t.Errorf("expected the original body replayed, got %s", out)
	}
}

func TestReportDownloadMarshal_FallsBackToTheTypedFields(t *testing.T) {
	t.Parallel()

	out, err := json.Marshal(ReportDownload{Status: "PENDING"})
	if err != nil {
		t.Fatalf("marshal download: %v", err)
	}

	if !strings.Contains(string(out), `"status":"PENDING"`) {
		t.Errorf("unexpected json: %s", out)
	}
}
