package cmd_test

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReportingTransactions_PostsTheBatch(t *testing.T) {
	isolateConfig(t)
	seedCredentials(t)

	got := startAPI(t, http.StatusAccepted, `{"ingested":1,"duplicated":0,"failed":0}`)

	out, err := runCLI(t, "", "reporting", "transactions", "--yes",
		"--account-id", "acc-1",
		"--events", `[{"report_id":"ev-1","merchant_order_id":"mo-1"}]`)
	if err != nil {
		t.Fatalf("reporting transactions failed: %v (%s)", err, out)
	}

	if got.Method != http.MethodPost || got.Path != "/v1/reporting/transactions" {
		t.Errorf("expected POST /v1/reporting/transactions, got %s %s", got.Method, got.Path)
	}

	body := decodeBody(t, got.Body)
	if body["account_id"] != "acc-1" {
		t.Errorf("unexpected body: %s", got.Body)
	}

	events, ok := body["events"].([]any)
	if !ok || len(events) != 1 {
		t.Errorf("unexpected events: %s", got.Body)
	}

	if !strings.Contains(out, "ingested") {
		t.Errorf("expected the per-event result printed, got:\n%s", out)
	}
}

func TestReportingTransactions_ReadsTheBatchFromAFile(t *testing.T) {
	isolateConfig(t)
	seedCredentials(t)

	got := startAPI(t, http.StatusAccepted, `{"ingested":2}`)

	path := filepath.Join(t.TempDir(), "events.json")
	if err := os.WriteFile(path, []byte(
		`{"account_id":"acc-1","events":[{"report_id":"ev-1"},{"report_id":"ev-2"}]}`), 0o600); err != nil {
		t.Fatalf("write the events file: %v", err)
	}

	if out, err := runCLI(t, "", "reporting", "transactions", "--yes", "--file", path); err != nil {
		t.Fatalf("reporting transactions failed: %v (%s)", err, out)
	}

	body := decodeBody(t, got.Body)

	events, ok := body["events"].([]any)
	if !ok || len(events) != 2 {
		t.Errorf("unexpected events: %s", got.Body)
	}
}

func TestReportingTransactions_RequiresABody(t *testing.T) {
	isolateConfig(t)
	seedCredentials(t)
	startAPI(t, http.StatusAccepted, `{}`)

	if _, err := runCLI(t, "", "reporting", "transactions", "--yes"); err == nil ||
		!strings.Contains(err.Error(), "request body is required") {
		t.Errorf("unexpected error: %v", err)
	}
}
