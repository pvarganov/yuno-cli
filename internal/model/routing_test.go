package model_test

import (
	"testing"

	"github.com/pvarganov/yuno-cli/internal/model"
)

func TestRouteChain(t *testing.T) {
	tests := []struct {
		name  string
		steps []model.RouteStep
		want  string
	}{
		{name: "empty route", steps: nil, want: ""},
		{
			name:  "provider chain in step order",
			steps: []model.RouteStep{{ProviderID: "STRIPE"}, {ProviderID: "ADYEN"}},
			want:  "STRIPE > ADYEN",
		},
		{
			name:  "falls back to the connection id",
			steps: []model.RouteStep{{ConnectionID: "c-1"}},
			want:  "c-1",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			route := model.Route{Steps: tc.steps}
			if got := route.Chain(); got != tc.want {
				t.Errorf("Chain() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestRoutingViews(t *testing.T) {
	routings := []model.Routing{{
		ID:            "r-1",
		PaymentMethod: "CARD",
		Name:          "Card routing",
		DefaultRoute:  model.Route{Steps: []model.RouteStep{{ProviderID: "STRIPE"}}},
		ConditionSets: []model.ConditionSet{{Name: "High value"}},
	}}

	views := model.RoutingViews(routings)
	if len(views) != 1 {
		t.Fatalf("expected one view, got %d", len(views))
	}

	want := model.RoutingView{
		ID: "r-1", PaymentMethod: "CARD", Name: "Card routing",
		Providers: "STRIPE", Conditions: 1,
	}

	if views[0] != want {
		t.Errorf("View() = %+v, want %+v", views[0], want)
	}
}

func TestRecommendationViews(t *testing.T) {
	rate := 0.9987
	latency := 180
	connection := "acct-adyen-us-1"

	rec := model.RoutingRecommendation{
		Recommended: model.RecommendedCandidate{ProviderID: "ADYEN", MerchantConnectionID: &connection},
		Ranking: []model.RankedCandidate{
			{ProviderID: "ADYEN", ApprovalRate: &rate, AvgLatencyMs: &latency, SampleSize: 767},
			{ProviderID: "STRIPE", SampleSize: 12},
		},
	}

	views := rec.Views()
	if len(views) != 2 {
		t.Fatalf("expected two views, got %d", len(views))
	}

	if !views[0].Recommended || views[0].ApprovalRate != "0.9987" || views[0].AvgLatencyMs != "180" {
		t.Errorf("unexpected first row: %+v", views[0])
	}

	if views[1].Recommended || views[1].ApprovalRate != "" || views[1].AvgLatencyMs != "" {
		t.Errorf("expected nil metrics to render as empty cells: %+v", views[1])
	}
}

func TestRecommendationViews_WithoutRanking(t *testing.T) {
	rec := model.RoutingRecommendation{
		Recommended: model.RecommendedCandidate{ProviderID: "ADYEN"},
	}

	views := rec.Views()
	if len(views) != 1 || views[0].ProviderID != "ADYEN" || !views[0].Recommended {
		t.Errorf("expected the recommended provider as the only row, got %+v", views)
	}
}

func TestConnectionView(t *testing.T) {
	connection := model.Connection{
		ConnectionID:         "c-1",
		MerchantConnectionID: "adyen-us-001",
		ProviderID:           "ADYEN",
		Status:               "ACTIVE",
		FlowType:             "PAYIN",
		PaymentMethods:       []string{"CARD", "GOOGLE_PAY"},
	}

	if got := connection.View().PaymentMethods; got != "CARD,GOOGLE_PAY" {
		t.Errorf("PaymentMethods = %q, want CARD,GOOGLE_PAY", got)
	}
}

func TestProviderCatalogViews(t *testing.T) {
	catalog := model.ProviderCatalog{Params: []model.CatalogParam{
		{ParamID: "API_KEY", FieldType: "string", Secret: true, Description: "Secret key"},
	}}

	views := catalog.Views()
	if len(views) != 1 || views[0].ParamID != "API_KEY" || !views[0].Secret {
		t.Errorf("unexpected catalog views: %+v", views)
	}
}

func TestProviderCatalogViews_Empty(t *testing.T) {
	var catalog model.ProviderCatalog

	if views := catalog.Views(); len(views) != 0 {
		t.Errorf("expected no rows for an empty catalog, got %+v", views)
	}
}

func TestRouteChain_FollowsDeclinedBranches(t *testing.T) {
	next := func(i int) *int { return &i }

	route := model.Route{Steps: []model.RouteStep{
		{Index: 1, ProviderID: "NETCETERA_3DS", Output: []model.RouteOutput{
			{Status: "APPROVED", Next: next(2)},
			{Status: "DECLINED", Next: next(3)},
			{Status: "INTERNAL_ERROR", Next: next(4)},
		}},
		{Index: 2, ProviderID: "APPROVED_BRANCH"},
		{Index: 3, ProviderID: "ECOMMPAY", Output: []model.RouteOutput{
			{Status: "DECLINED", Next: next(7)},
		}},
		{Index: 4, ProviderID: "ERROR_BRANCH"},
		{Index: 7, ProviderID: "CHECKOUT"},
	}}

	if got, want := route.Chain(), "NETCETERA_3DS > ECOMMPAY > CHECKOUT"; got != want {
		t.Errorf("Chain() = %q, want %q", got, want)
	}
}

func TestRouteChain_StopsOnALoop(t *testing.T) {
	next := func(i int) *int { return &i }

	route := model.Route{Steps: []model.RouteStep{
		{Index: 1, ProviderID: "A", Output: []model.RouteOutput{{Status: "DECLINED", Next: next(2)}}},
		{Index: 2, ProviderID: "B", Output: []model.RouteOutput{{Status: "DECLINED", Next: next(1)}}},
	}}

	if got, want := route.Chain(), "A > B"; got != want {
		t.Errorf("Chain() = %q, want %q", got, want)
	}
}
