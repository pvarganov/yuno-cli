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

func TestListSubscriptions_PagesWithPageAndSize(t *testing.T) {
	var gotQueries []string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQueries = append(gotQueries, r.URL.RawQuery)

		if r.URL.Query().Get("page") == "0" {
			_, _ = io.WriteString(w, `{"items":[{"id":"sub-1"},{"id":"sub-2"}],"pagination":{"page":0,"size":2,"total":3}}`)

			return
		}

		_, _ = io.WriteString(w, `{"items":[{"id":"sub-3"}],"pagination":{"page":1,"size":2,"total":3}}`)
	}))
	defer srv.Close()

	c, _ := newTestClient(t, srv, testProfile())

	filters := url.Values{}
	filters.Set("status", "ACTIVE")

	subscriptions, err := c.ListSubscriptions(t.Context(), filters, 0, 2)
	if err != nil {
		t.Fatalf("ListSubscriptions: %v", err)
	}

	if len(subscriptions) != 3 || subscriptions[2].ID != "sub-3" {
		t.Errorf("unexpected subscriptions: %+v", subscriptions)
	}

	if len(gotQueries) != 2 {
		t.Fatalf("expected two requests, got %d: %v", len(gotQueries), gotQueries)
	}

	for _, want := range []string{"status=ACTIVE", "page=0", "size=2"} {
		if !strings.Contains(gotQueries[0], want) {
			t.Errorf("expected the query to contain %q, got %q", want, gotQueries[0])
		}
	}
}

func TestListSubscriptions_StopsAtTheLimit(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, `{"items":[{"id":"sub-1"},{"id":"sub-2"}]}`)
	}))
	defer srv.Close()

	c, _ := newTestClient(t, srv, testProfile())

	subscriptions, err := c.ListSubscriptions(t.Context(), nil, 1, 2)
	if err != nil {
		t.Fatalf("ListSubscriptions: %v", err)
	}

	if len(subscriptions) != 1 {
		t.Errorf("expected one subscription, got %d", len(subscriptions))
	}
}

func TestListSubscriptions_ErrorIsWrapped(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = io.WriteString(w, `{"code":"INSUFFICIENT_SCOPE","message":"nope"}`)
	}))
	defer srv.Close()

	c, _ := newTestClient(t, srv, testProfile())

	_, err := c.ListSubscriptions(t.Context(), nil, 0, 0)
	if err == nil {
		t.Fatal("expected an error on 403")
	}

	if !errors.Is(err, ErrForbidden) || !strings.Contains(err.Error(), "list subscriptions") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestGetSubscription_EscapesTheID(t *testing.T) {
	var gotMethod, gotPath string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod, gotPath = r.Method, r.URL.EscapedPath()
		_, _ = io.WriteString(w, `{"id":"sub 1","status":"ACTIVE"}`)
	}))
	defer srv.Close()

	c, _ := newTestClient(t, srv, testProfile())

	subscription, err := c.GetSubscription(t.Context(), "sub 1")
	if err != nil {
		t.Fatalf("GetSubscription: %v", err)
	}

	if gotMethod != http.MethodGet || gotPath != "/subscriptions/sub%201" {
		t.Errorf("expected GET /subscriptions/sub%%201, got %s %s", gotMethod, gotPath)
	}

	if subscription.Status != "ACTIVE" {
		t.Errorf("unexpected subscription: %+v", subscription)
	}
}

func TestGetSubscription_NotFoundIsWrapped(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = io.WriteString(w, `{"code":"NOT_FOUND","message":"subscription not found"}`)
	}))
	defer srv.Close()

	c, _ := newTestClient(t, srv, testProfile())

	_, err := c.GetSubscription(t.Context(), "sub-1")
	if err == nil {
		t.Fatal("expected an error on 404")
	}

	if !errors.Is(err, ErrNotFound) || !strings.Contains(err.Error(), "get subscription sub-1") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestCreateAndUpdateSubscription_SendTheBody(t *testing.T) {
	var gotMethod, gotPath, gotBody string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		gotMethod, gotPath, gotBody = r.Method, r.URL.EscapedPath(), string(body)
		_, _ = io.WriteString(w, `{"id":"sub-1"}`)
	}))
	defer srv.Close()

	c, _ := newTestClient(t, srv, testProfile())

	if _, err := c.CreateSubscription(t.Context(), map[string]any{"name": "Gold"}); err != nil {
		t.Fatalf("CreateSubscription: %v", err)
	}

	if gotMethod != http.MethodPost || gotPath != "/subscriptions" || !strings.Contains(gotBody, "Gold") {
		t.Errorf("unexpected create request: %s %s %s", gotMethod, gotPath, gotBody)
	}

	if _, err := c.UpdateSubscription(t.Context(), "sub-1", map[string]any{"name": "Silver"}); err != nil {
		t.Fatalf("UpdateSubscription: %v", err)
	}

	if gotMethod != http.MethodPatch || gotPath != "/subscriptions/sub-1" || !strings.Contains(gotBody, "Silver") {
		t.Errorf("unexpected update request: %s %s %s", gotMethod, gotPath, gotBody)
	}
}

func TestSubscriptionAction_PostsToTheActionPath(t *testing.T) {
	for _, action := range []string{"cancel", "pause", "resume", "retry", "plan"} {
		t.Run(action, func(t *testing.T) {
			var gotMethod, gotPath string

			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				gotMethod, gotPath = r.Method, r.URL.EscapedPath()
				_, _ = io.WriteString(w, `{"id":"sub-1","status":"CANCELLED"}`)
			}))
			defer srv.Close()

			c, _ := newTestClient(t, srv, testProfile())

			subscription, err := c.SubscriptionAction(t.Context(), "sub-1", action, nil)
			if err != nil {
				t.Fatalf("SubscriptionAction(%s): %v", action, err)
			}

			if gotMethod != http.MethodPost || gotPath != "/subscriptions/sub-1/"+action {
				t.Errorf("expected POST /subscriptions/sub-1/%s, got %s %s", action, gotMethod, gotPath)
			}

			if subscription.ID != "sub-1" {
				t.Errorf("unexpected subscription: %+v", subscription)
			}
		})
	}
}

func TestSubscriptionAction_ErrorNamesTheAction(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnprocessableEntity)
		_, _ = io.WriteString(w, `{"code":"INVALID_STATE","message":"already cancelled"}`)
	}))
	defer srv.Close()

	c, _ := newTestClient(t, srv, testProfile())

	_, err := c.SubscriptionAction(t.Context(), "sub-1", "cancel", nil)
	if err == nil {
		t.Fatal("expected an error on 422")
	}

	if !strings.Contains(err.Error(), "cancel subscription sub-1") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestListSubscriptionPayments_PagesWithLimitAndOffset(t *testing.T) {
	var gotQueries []string
	var gotPath string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQueries = append(gotQueries, r.URL.RawQuery)
		gotPath = r.URL.EscapedPath()

		if r.URL.Query().Get("offset") == "0" {
			_, _ = io.WriteString(w, `{"data":[{"id":"sp-1"},{"id":"sp-2"}],"pagination":{"has_more":true}}`)

			return
		}

		_, _ = io.WriteString(w, `{"data":[{"id":"sp-3"}],"pagination":{"has_more":false}}`)
	}))
	defer srv.Close()

	c, _ := newTestClient(t, srv, testProfile())

	payments, err := c.ListSubscriptionPayments(t.Context(), "sub-1", 0, 2)
	if err != nil {
		t.Fatalf("ListSubscriptionPayments: %v", err)
	}

	if gotPath != "/subscriptions/sub-1/payments" {
		t.Errorf("unexpected path %s", gotPath)
	}

	if len(payments) != 3 || payments[2].ID != "sp-3" {
		t.Errorf("unexpected payments: %+v", payments)
	}

	if !strings.Contains(gotQueries[0], "limit=2") || !strings.Contains(gotQueries[0], "offset=0") {
		t.Errorf("unexpected first query %q", gotQueries[0])
	}
}

func TestListSubscriptionPayments_ErrorIsWrapped(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = io.WriteString(w, `{"code":"NOT_FOUND","message":"no such subscription"}`)
	}))
	defer srv.Close()

	c, _ := newTestClient(t, srv, testProfile())

	_, err := c.ListSubscriptionPayments(t.Context(), "sub-1", 0, 0)
	if err == nil || !strings.Contains(err.Error(), "list payments of subscription sub-1") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestListPlans_SetsTheAccountAndPages(t *testing.T) {
	var gotQuery, gotPath string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery, gotPath = r.URL.RawQuery, r.URL.EscapedPath()
		_, _ = io.WriteString(w, `{"data":[{"id":"plan-1"}],"pagination":{"has_more":false}}`)
	}))
	defer srv.Close()

	c, _ := newTestClient(t, srv, testProfile())

	plans, err := c.ListPlans(t.Context(), "acc-1", 0, 5)
	if err != nil {
		t.Fatalf("ListPlans: %v", err)
	}

	if gotPath != "/subscriptions/plans" {
		t.Errorf("unexpected path %s", gotPath)
	}

	for _, want := range []string{"account_id=acc-1", "limit=5", "offset=0"} {
		if !strings.Contains(gotQuery, want) {
			t.Errorf("expected the query to contain %q, got %q", want, gotQuery)
		}
	}

	if len(plans) != 1 || plans[0].ID != "plan-1" {
		t.Errorf("unexpected plans: %+v", plans)
	}
}

func TestListPlans_ErrorIsWrapped(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = io.WriteString(w, `{"code":"UNAUTHORIZED","message":"bad keys"}`)
	}))
	defer srv.Close()

	c, _ := newTestClient(t, srv, testProfile())

	_, err := c.ListPlans(t.Context(), "", 0, 0)
	if err == nil || !errors.Is(err, ErrUnauthorized) || !strings.Contains(err.Error(), "list plans") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestPlanCreateGetAndStatus(t *testing.T) {
	var gotMethod, gotPath, gotBody string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		gotMethod, gotPath, gotBody = r.Method, r.URL.EscapedPath(), string(body)
		_, _ = io.WriteString(w, `{"id":"plan-1","status":"CANCELED","affected_subscriptions":2}`)
	}))
	defer srv.Close()

	c, _ := newTestClient(t, srv, testProfile())

	if _, err := c.CreatePlan(t.Context(), map[string]any{"name": "Gold"}); err != nil {
		t.Fatalf("CreatePlan: %v", err)
	}

	if gotMethod != http.MethodPost || gotPath != "/subscriptions/plans" || !strings.Contains(gotBody, "Gold") {
		t.Errorf("unexpected create request: %s %s %s", gotMethod, gotPath, gotBody)
	}

	if _, err := c.GetPlan(t.Context(), "plan-1"); err != nil {
		t.Fatalf("GetPlan: %v", err)
	}

	if gotMethod != http.MethodGet || gotPath != "/subscriptions/plans/plan-1" {
		t.Errorf("unexpected get request: %s %s", gotMethod, gotPath)
	}

	status, err := c.UpdatePlanStatus(t.Context(), "plan-1", map[string]any{"status": "CANCELED"})
	if err != nil {
		t.Fatalf("UpdatePlanStatus: %v", err)
	}

	if gotMethod != http.MethodPost || gotPath != "/subscriptions/plans/plan-1/status" {
		t.Errorf("unexpected status request: %s %s", gotMethod, gotPath)
	}

	if status.AffectedSubscriptions == nil || *status.AffectedSubscriptions != 2 {
		t.Errorf("unexpected status: %+v", status)
	}
}

func TestPlanErrorsAreWrapped(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = io.WriteString(w, `{"code":"VALIDATION_ERROR","message":"bad plan"}`)
	}))
	defer srv.Close()

	c, _ := newTestClient(t, srv, testProfile())

	if _, err := c.CreatePlan(t.Context(), map[string]any{}); err == nil ||
		!strings.Contains(err.Error(), "create plan") {
		t.Errorf("unexpected create error: %v", err)
	}

	if _, err := c.GetPlan(t.Context(), "plan-1"); err == nil ||
		!strings.Contains(err.Error(), "get plan plan-1") {
		t.Errorf("unexpected get error: %v", err)
	}

	if _, err := c.UpdatePlanStatus(t.Context(), "plan-1", map[string]any{}); err == nil ||
		!strings.Contains(err.Error(), "update status of plan plan-1") {
		t.Errorf("unexpected status error: %v", err)
	}
}
