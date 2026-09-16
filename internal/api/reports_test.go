package api

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

// reportBody is the report payload the fake API answers with.
const reportBody = `{"id":"rp-1","type":"PAYMENTS","status":"SUCCEEDED",
	"merchant_reference_id":"ref-1","start_date":"2026-09-01T00:00:00Z",
	"end_date":"2026-09-15T00:00:00Z","expires_at":"2026-09-30T00:00:00Z"}`

func TestCreateReport_PostsTheBody(t *testing.T) {
	var gotMethod, gotPath string

	var gotBody map[string]any

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod, gotPath = r.Method, r.URL.Path
		_ = json.NewDecoder(r.Body).Decode(&gotBody)

		w.WriteHeader(http.StatusCreated)
		_, _ = io.WriteString(w, reportBody)
	}))
	defer srv.Close()

	c, _ := newTestClient(t, srv, testProfile())

	report, err := c.CreateReport(t.Context(), map[string]any{"type": "PAYMENTS"})
	if err != nil {
		t.Fatalf("CreateReport: %v", err)
	}

	if gotMethod != http.MethodPost || gotPath != "/reports" {
		t.Errorf("expected POST /reports, got %s %s", gotMethod, gotPath)
	}

	if gotBody["type"] != "PAYMENTS" || report.ID != "rp-1" {
		t.Errorf("unexpected create: body %+v, report %+v", gotBody, report)
	}
}

func TestCreateReport_ErrorIsWrapped(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = io.WriteString(w, `{"code":"INSUFFICIENT_SCOPE","message":"nope"}`)
	}))
	defer srv.Close()

	c, _ := newTestClient(t, srv, testProfile())

	_, err := c.CreateReport(t.Context(), map[string]any{})
	if err == nil || !errors.Is(err, ErrForbidden) || !strings.Contains(err.Error(), "create report") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestListReports_PagesFromPageNumberOne(t *testing.T) {
	var queries []string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		queries = append(queries, r.URL.RawQuery)

		if r.URL.Query().Get("page_number") == "1" {
			_, _ = io.WriteString(w, `[`+reportBody+`,`+reportBody+`]`)

			return
		}

		_, _ = io.WriteString(w, `[]`)
	}))
	defer srv.Close()

	c, _ := newTestClient(t, srv, testProfile())

	reports, err := c.ListReports(t.Context(), url.Values{"account_id": []string{"acc-1"}}, 0, 2)
	if err != nil {
		t.Fatalf("ListReports: %v", err)
	}

	if len(reports) != 2 {
		t.Fatalf("expected 2 reports, got %d", len(reports))
	}

	if len(queries) != 2 {
		t.Fatalf("expected two requests, got %v", queries)
	}

	for i, want := range []string{"page_number=1", "page_number=2"} {
		if !strings.Contains(queries[i], want) || !strings.Contains(queries[i], "account_id=acc-1") {
			t.Errorf("request %d: expected %s and the filter, got %s", i, want, queries[i])
		}
	}
}

func TestListReports_StopsAtTheLimit(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, `[`+reportBody+`,`+reportBody+`,`+reportBody+`]`)
	}))
	defer srv.Close()

	c, _ := newTestClient(t, srv, testProfile())

	reports, err := c.ListReports(t.Context(), nil, 2, 3)
	if err != nil {
		t.Fatalf("ListReports: %v", err)
	}

	if len(reports) != 2 {
		t.Errorf("expected the list truncated to 2, got %d", len(reports))
	}
}

func TestListReports_ErrorIsWrapped(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = io.WriteString(w, `{"code":"BAD_REQUEST","message":"nope"}`)
	}))
	defer srv.Close()

	c, _ := newTestClient(t, srv, testProfile())

	if _, err := c.ListReports(t.Context(), nil, 0, 0); err == nil ||
		!strings.Contains(err.Error(), "list reports") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestGetReport_HitsTheIDPath(t *testing.T) {
	var gotPath string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path

		_, _ = io.WriteString(w, reportBody)
	}))
	defer srv.Close()

	c, _ := newTestClient(t, srv, testProfile())

	report, err := c.GetReport(t.Context(), "rp-1")
	if err != nil {
		t.Fatalf("GetReport: %v", err)
	}

	if gotPath != "/reports/rp-1" || report.Status != "SUCCEEDED" {
		t.Errorf("unexpected get: %s, %+v", gotPath, report)
	}
}

func TestGetReport_ErrorIsWrapped(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = io.WriteString(w, `{"code":"NOT_FOUND","message":"nope"}`)
	}))
	defer srv.Close()

	c, _ := newTestClient(t, srv, testProfile())

	if _, err := c.GetReport(t.Context(), "rp-1"); err == nil ||
		!strings.Contains(err.Error(), "get report rp-1") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestDownloadReport_ReturnsTheLink(t *testing.T) {
	var gotPath string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path

		_, _ = io.WriteString(w, `{"type":"PAYMENTS","status":"AVAILABLE","download_link":"https://s3/report.csv"}`)
	}))
	defer srv.Close()

	c, _ := newTestClient(t, srv, testProfile())

	download, err := c.DownloadReport(t.Context(), "rp-1")
	if err != nil {
		t.Fatalf("DownloadReport: %v", err)
	}

	if gotPath != "/reports/rp-1/download" || download.DownloadLink != "https://s3/report.csv" {
		t.Errorf("unexpected download: %s, %+v", gotPath, download)
	}
}

func TestDownloadReport_ErrorIsWrapped(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = io.WriteString(w, `{"code":"BAD_REQUEST","message":"not ready"}`)
	}))
	defer srv.Close()

	c, _ := newTestClient(t, srv, testProfile())

	if _, err := c.DownloadReport(t.Context(), "rp-1"); err == nil ||
		!strings.Contains(err.Error(), "download report rp-1") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestFetchFile_StreamsTheBodyWithoutTheAPIKeys(t *testing.T) {
	var gotAuth string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get(HeaderPrivateSecretKey)

		_, _ = io.WriteString(w, "id,amount\nrp-1,100\n")
	}))
	defer srv.Close()

	c, _ := newTestClient(t, srv, testProfile())

	var buf bytes.Buffer

	written, err := c.FetchFile(t.Context(), srv.URL+"/report.csv", &buf)
	if err != nil {
		t.Fatalf("FetchFile: %v", err)
	}

	if gotAuth != "" {
		t.Errorf("the api key must not be sent to the storage host, got %q", gotAuth)
	}

	if written != int64(buf.Len()) || !strings.Contains(buf.String(), "rp-1,100") {
		t.Errorf("unexpected download: %d bytes, %q", written, buf.String())
	}
}

func TestFetchFile_RejectsANonHTTPLink(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {}))
	defer srv.Close()

	c, _ := newTestClient(t, srv, testProfile())

	if _, err := c.FetchFile(t.Context(), "file:///etc/passwd", io.Discard); err == nil ||
		!strings.Contains(err.Error(), "want an http or https url") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestFetchFile_ReportsAFailedStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer srv.Close()

	c, _ := newTestClient(t, srv, testProfile())

	if _, err := c.FetchFile(t.Context(), srv.URL+"/report.csv", io.Discard); err == nil ||
		!strings.Contains(err.Error(), "unexpected status") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestFetchFile_ReportsAFailingWriter(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, "payload")
	}))
	defer srv.Close()

	c, _ := newTestClient(t, srv, testProfile())

	if _, err := c.FetchFile(t.Context(), srv.URL+"/report.csv", failingWriter{}); err == nil ||
		!strings.Contains(err.Error(), "write downloaded report") {
		t.Errorf("unexpected error: %v", err)
	}
}

// failingWriter refuses every write, standing in for a full disk.
type failingWriter struct{}

func (failingWriter) Write(_ []byte) (int, error) {
	return 0, fmt.Errorf("disk full")
}
