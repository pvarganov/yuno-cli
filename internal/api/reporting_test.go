package api

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestReportTransactions_PostsTheBatch(t *testing.T) {
	var gotMethod, gotPath string

	var gotBody map[string]any

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod, gotPath = r.Method, r.URL.Path
		_ = json.NewDecoder(r.Body).Decode(&gotBody)

		w.WriteHeader(http.StatusAccepted)
		_, _ = io.WriteString(w, `{"ingested":1,"duplicated":0,"failed":0}`)
	}))
	defer srv.Close()

	c, _ := newTestClient(t, srv, testProfile())

	data, err := c.ReportTransactions(t.Context(), map[string]any{
		"account_id": "acc-1",
		"events":     []any{map[string]any{"report_id": "ev-1"}},
	})
	if err != nil {
		t.Fatalf("ReportTransactions: %v", err)
	}

	if gotMethod != http.MethodPost || gotPath != "/reporting/transactions" {
		t.Errorf("expected POST /reporting/transactions, got %s %s", gotMethod, gotPath)
	}

	if gotBody["account_id"] != "acc-1" {
		t.Errorf("unexpected body: %+v", gotBody)
	}

	if !strings.Contains(string(data), `"ingested":1`) {
		t.Errorf("unexpected response: %s", data)
	}
}

func TestReportTransactions_ErrorIsWrapped(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = io.WriteString(w, `{"code":"INSUFFICIENT_SCOPE","message":"nope"}`)
	}))
	defer srv.Close()

	c, _ := newTestClient(t, srv, testProfile())

	_, err := c.ReportTransactions(t.Context(), map[string]any{})
	if err == nil || !errors.Is(err, ErrForbidden) || !strings.Contains(err.Error(), "report transactions") {
		t.Errorf("unexpected error: %v", err)
	}
}
