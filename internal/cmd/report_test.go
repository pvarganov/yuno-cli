package cmd_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pvarganov/yuno-cli/internal/config"
)

// reportResponse is the report payload the fake API returns.
const reportResponse = `{"id":"rp-1","type":"PAYMENTS","status":"SUCCEEDED",
	"merchant_reference_id":"ref-1","start_date":"2026-09-01T00:00:00Z",
	"end_date":"2026-09-15T00:00:00Z","expires_at":"2026-09-30T00:00:00Z"}`

// reportFile is the body the fake storage host serves behind the download link.
const reportFile = "id,amount\nrp-1,100\n"

// startReportDownloadAPI serves the download endpoint and the storage host the
// link points at, so the streaming path can be exercised end to end.
func startReportDownloadAPI(t *testing.T, status string) *httptest.Server {
	t.Helper()

	mux := http.NewServeMux()
	server := httptest.NewServer(mux)

	t.Cleanup(server.Close)

	link := server.URL + "/files/report.csv"
	if status != "AVAILABLE" {
		link = ""
	}

	mux.HandleFunc("/v1/reports/rp-1/download", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"type":"PAYMENTS","status":"`+status+`","download_link":"`+link+`"}`)
	})

	mux.HandleFunc("/files/report.csv", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, reportFile)
	})

	t.Setenv(config.EnvAPIEndpoint, server.URL+"/v1")

	return server
}

func TestReportCreate_PostsTheBody(t *testing.T) {
	isolateConfig(t)
	seedCredentials(t)

	got := startAPI(t, http.StatusCreated, reportResponse)

	out, err := runCLI(t, "", "report", "create", "--yes",
		"--type", "PAYMENTS", "--start-date", "2026-09-01T00:00:00Z",
		"--end-date", "2026-09-15T00:00:00Z", "--merchant-reference-id", "ref-1",
		"--payment-status", "SUCCEEDED,DECLINED")
	if err != nil {
		t.Fatalf("report create failed: %v (%s)", err, out)
	}

	if got.Method != http.MethodPost || got.Path != "/v1/reports" {
		t.Errorf("expected POST /v1/reports, got %s %s", got.Method, got.Path)
	}

	body := decodeBody(t, got.Body)
	if body["type"] != "PAYMENTS" || body["payment_status"] != "SUCCEEDED,DECLINED" {
		t.Errorf("unexpected body: %s", got.Body)
	}

	for _, want := range []string{"rp-1", "PAYMENTS", "SUCCEEDED"} {
		if !strings.Contains(out, want) {
			t.Errorf("expected table output to contain %q, got:\n%s", want, out)
		}
	}
}

func TestReportCreate_RequiresABody(t *testing.T) {
	isolateConfig(t)
	seedCredentials(t)
	startAPI(t, http.StatusCreated, reportResponse)

	if _, err := runCLI(t, "", "report", "create", "--yes"); err == nil ||
		!strings.Contains(err.Error(), "request body is required") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestReportList_PagesFromPageNumberOne(t *testing.T) {
	isolateConfig(t)
	seedCredentials(t)

	got := startAPI(t, http.StatusOK, "["+reportResponse+"]")

	out, err := runCLI(t, "", "report", "list", "--account-id", "acc-1", "--limit", "1", "--page-size", "1")
	if err != nil {
		t.Fatalf("report list failed: %v (%s)", err, out)
	}

	if got.Method != http.MethodGet || got.Path != "/v1/reports/list" {
		t.Errorf("expected GET /v1/reports/list, got %s %s", got.Method, got.Path)
	}

	for _, want := range []string{"account_id=acc-1", "page_number=1", "page_size=1"} {
		if !strings.Contains(got.Query, want) {
			t.Errorf("expected the query to contain %q, got %s", want, got.Query)
		}
	}

	if !strings.Contains(out, "rp-1") {
		t.Errorf("expected the report in the table, got:\n%s", out)
	}
}

func TestReportList_EmptyResponsePrintsNothing(t *testing.T) {
	isolateConfig(t)
	seedCredentials(t)
	startAPI(t, http.StatusOK, "[]")

	out, err := runCLI(t, "", "report", "list")
	if err != nil {
		t.Fatalf("report list failed: %v (%s)", err, out)
	}

	if strings.TrimSpace(out) != "" {
		t.Errorf("expected no output for an empty list, got:\n%s", out)
	}
}

func TestReportGet_HitsTheIDPath(t *testing.T) {
	isolateConfig(t)
	seedCredentials(t)

	got := startAPI(t, http.StatusOK, reportResponse)

	out, err := runCLI(t, "", "report", "get", "rp-1", "--json")
	if err != nil {
		t.Fatalf("report get failed: %v (%s)", err, out)
	}

	if got.Method != http.MethodGet || got.Path != "/v1/reports/rp-1" {
		t.Errorf("expected GET /v1/reports/rp-1, got %s %s", got.Method, got.Path)
	}

	if !strings.Contains(out, `"id": "rp-1"`) {
		t.Errorf("expected the json body, got:\n%s", out)
	}
}

func TestReportDownload_PrintsTheLinkWithoutOutput(t *testing.T) {
	isolateConfig(t)
	seedCredentials(t)
	startReportDownloadAPI(t, "AVAILABLE")

	out, err := runCLI(t, "", "report", "download", "rp-1")
	if err != nil {
		t.Fatalf("report download failed: %v (%s)", err, out)
	}

	if !strings.Contains(out, "/files/report.csv") || !strings.Contains(out, "AVAILABLE") {
		t.Errorf("expected the download link in the table, got:\n%s", out)
	}
}

func TestReportDownload_StreamsToAFile(t *testing.T) {
	isolateConfig(t)
	seedCredentials(t)
	startReportDownloadAPI(t, "AVAILABLE")

	target := filepath.Join(t.TempDir(), "report.csv")

	out, err := runCLI(t, "", "report", "download", "rp-1", "--output", target)
	if err != nil {
		t.Fatalf("report download failed: %v (%s)", err, out)
	}

	data, err := os.ReadFile(target)
	if err != nil {
		t.Fatalf("read the downloaded report: %v", err)
	}

	if string(data) != reportFile {
		t.Errorf("unexpected file content: %q", data)
	}

	info, err := os.Stat(target)
	if err != nil {
		t.Fatalf("stat the downloaded report: %v", err)
	}

	if mode := info.Mode().Perm(); mode != 0o600 {
		t.Errorf("expected the report file to be 0600, got %o", mode)
	}

	if !strings.Contains(out, "wrote") {
		t.Errorf("expected the byte count on stderr, got:\n%s", out)
	}
}

func TestReportDownload_StreamsToStdout(t *testing.T) {
	isolateConfig(t)
	seedCredentials(t)
	startReportDownloadAPI(t, "AVAILABLE")

	out, err := runCLI(t, "", "report", "download", "rp-1", "--output", "-")
	if err != nil {
		t.Fatalf("report download failed: %v (%s)", err, out)
	}

	if !strings.Contains(out, "rp-1,100") {
		t.Errorf("expected the file on stdout, got:\n%s", out)
	}
}

func TestReportDownload_RefusesAMissingLink(t *testing.T) {
	isolateConfig(t)
	seedCredentials(t)
	startReportDownloadAPI(t, "IN_PROCESS")

	_, err := runCLI(t, "", "report", "download", "rp-1", "--output", filepath.Join(t.TempDir(), "x.csv"))
	if err == nil || !strings.Contains(err.Error(), "no download link yet") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestReportDownload_ReportsAnUnwritableTarget(t *testing.T) {
	isolateConfig(t)
	seedCredentials(t)
	startReportDownloadAPI(t, "AVAILABLE")

	target := filepath.Join(t.TempDir(), "missing", "report.csv")

	if _, err := runCLI(t, "", "report", "download", "rp-1", "--output", target); err == nil ||
		!strings.Contains(err.Error(), "create report file") {
		t.Errorf("unexpected error: %v", err)
	}
}
