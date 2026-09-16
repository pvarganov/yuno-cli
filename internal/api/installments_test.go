package api

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestListInstallmentPlans_SendsTheFiltersAndDecodesAnArray(t *testing.T) {
	var gotQuery, gotPath string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotQuery = r.URL.RawQuery

		_, _ = io.WriteString(w, `[{"id":"ip-1","name":"plan_007"},{"id":"ip-2"}]`)
	}))
	defer srv.Close()

	c, _ := newTestClient(t, srv, testProfile())

	filters := url.Values{}
	filters.Set("account_id", "acc-1")
	filters.Set("currency", "USD")

	plans, err := c.ListInstallmentPlans(t.Context(), filters)
	if err != nil {
		t.Fatalf("ListInstallmentPlans: %v", err)
	}

	if gotPath != "/installments-plans" {
		t.Errorf("expected /installments-plans, got %s", gotPath)
	}

	for _, want := range []string{"account_id=acc-1", "currency=USD"} {
		if !strings.Contains(gotQuery, want) {
			t.Errorf("expected the query to contain %q, got %q", want, gotQuery)
		}
	}

	if len(plans) != 2 || plans[0].Name != "plan_007" {
		t.Errorf("unexpected plans: %+v", plans)
	}
}

func TestListInstallmentPlans_ErrorIsWrapped(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = io.WriteString(w, `{"code":"INSUFFICIENT_SCOPE","message":"nope"}`)
	}))
	defer srv.Close()

	c, _ := newTestClient(t, srv, testProfile())

	_, err := c.ListInstallmentPlans(t.Context(), nil)
	if err == nil {
		t.Fatal("expected an error on 403")
	}

	if !errors.Is(err, ErrForbidden) || !strings.Contains(err.Error(), "list installment plans") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestGetInstallmentPlan_AcceptsAnObjectAndAnArray(t *testing.T) {
	for name, response := range map[string]string{
		"object": `{"id":"ip-1","country_code":"US"}`,
		"array":  `[{"id":"ip-1","country_code":"US"}]`,
	} {
		t.Run(name, func(t *testing.T) {
			var gotPath string

			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				gotPath = r.URL.Path
				_, _ = io.WriteString(w, response)
			}))
			defer srv.Close()

			c, _ := newTestClient(t, srv, testProfile())

			plans, err := c.GetInstallmentPlan(t.Context(), "ip-1")
			if err != nil {
				t.Fatalf("GetInstallmentPlan: %v", err)
			}

			if gotPath != "/installments-plans/ip-1" {
				t.Errorf("expected /installments-plans/ip-1, got %s", gotPath)
			}

			if len(plans) != 1 || plans[0].ID != "ip-1" || plans[0].CountryCode != "US" {
				t.Errorf("unexpected plans: %+v", plans)
			}
		})
	}
}

func TestGetInstallmentPlan_BrokenJSONIsWrapped(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, `{"id":`)
	}))
	defer srv.Close()

	c, _ := newTestClient(t, srv, testProfile())

	if _, err := c.GetInstallmentPlan(t.Context(), "ip-1"); err == nil ||
		!strings.Contains(err.Error(), "get installment plan ip-1") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestCreateInstallmentPlan_PostsTheBody(t *testing.T) {
	var gotMethod, gotPath, gotBody string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		gotMethod, gotPath, gotBody = r.Method, r.URL.Path, string(body)

		_, _ = io.WriteString(w, `{"id":"ip-1","name":"plan_007"}`)
	}))
	defer srv.Close()

	c, _ := newTestClient(t, srv, testProfile(), WithConfirmer(nil))

	plan, err := c.CreateInstallmentPlan(t.Context(), map[string]any{"name": "plan_007"})
	if err != nil {
		t.Fatalf("CreateInstallmentPlan: %v", err)
	}

	if gotMethod != http.MethodPost || gotPath != "/installments-plans" {
		t.Errorf("expected POST /installments-plans, got %s %s", gotMethod, gotPath)
	}

	if !strings.Contains(gotBody, `"name":"plan_007"`) {
		t.Errorf("unexpected body: %s", gotBody)
	}

	if plan.ID != "ip-1" {
		t.Errorf("unexpected plan: %+v", plan)
	}
}

func TestUpdateInstallmentPlan_PatchesTheBody(t *testing.T) {
	var gotMethod, gotPath string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod, gotPath = r.Method, r.URL.Path

		_, _ = io.WriteString(w, `{"id":"ip-1","name":"plan_008"}`)
	}))
	defer srv.Close()

	c, _ := newTestClient(t, srv, testProfile())

	plan, err := c.UpdateInstallmentPlan(t.Context(), "ip-1", map[string]any{"name": "plan_008"})
	if err != nil {
		t.Fatalf("UpdateInstallmentPlan: %v", err)
	}

	if gotMethod != http.MethodPatch || gotPath != "/installments-plans/ip-1" {
		t.Errorf("expected PATCH /installments-plans/ip-1, got %s %s", gotMethod, gotPath)
	}

	if plan.Name != "plan_008" {
		t.Errorf("unexpected plan: %+v", plan)
	}
}

func TestUpdateInstallmentPlan_ErrorIsWrapped(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = io.WriteString(w, `{"code":"INVALID","message":"bad plan"}`)
	}))
	defer srv.Close()

	c, _ := newTestClient(t, srv, testProfile())

	_, err := c.UpdateInstallmentPlan(t.Context(), "ip-1", map[string]any{})
	if err == nil || !strings.Contains(err.Error(), "update installment plan ip-1") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestDeleteInstallmentPlan_SendsDelete(t *testing.T) {
	var gotMethod, gotPath string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod, gotPath = r.Method, r.URL.Path

		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	c, _ := newTestClient(t, srv, testProfile())

	data, err := c.DeleteInstallmentPlan(t.Context(), "ip-1")
	if err != nil {
		t.Fatalf("DeleteInstallmentPlan: %v", err)
	}

	if gotMethod != http.MethodDelete || gotPath != "/installments-plans/ip-1" {
		t.Errorf("expected DELETE /installments-plans/ip-1, got %s %s", gotMethod, gotPath)
	}

	if len(data) != 0 {
		t.Errorf("expected an empty body, got %q", data)
	}
}

func TestDeleteInstallmentPlan_ErrorIsWrapped(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = io.WriteString(w, `{"code":"NOT_FOUND","message":"gone"}`)
	}))
	defer srv.Close()

	c, _ := newTestClient(t, srv, testProfile())

	_, err := c.DeleteInstallmentPlan(t.Context(), "ip-1")
	if err == nil || !strings.Contains(err.Error(), "delete installment plan ip-1") {
		t.Errorf("unexpected error: %v", err)
	}
}
