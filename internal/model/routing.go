// Package model holds the typed shapes of the Yuno API resources and the flat
// views the table formatter renders.
package model

import (
	"fmt"
	"strings"
)

// Routing is one routing rule of an account, as returned by `GET /routing`.
type Routing struct {
	ID            string         `json:"id"`
	AccountID     string         `json:"account_id"`
	PaymentMethod string         `json:"payment_method"`
	Name          string         `json:"name"`
	DefaultRoute  Route          `json:"default_route"`
	ConditionSets []ConditionSet `json:"condition_sets,omitempty"`
	CreatedAt     string         `json:"created_at,omitempty"`
	UpdatedAt     string         `json:"updated_at,omitempty"`
}

// Route is an ordered chain of providers a payment is attempted against.
type Route struct {
	Steps []RouteStep `json:"steps"`
}

// RouteStep is one provider attempt of a route.
type RouteStep struct {
	Index        int    `json:"index"`
	ProviderID   string `json:"provider_id"`
	ConnectionID string `json:"connection_id"`
}

// ConditionSet is a conditional override of the default route.
type ConditionSet struct {
	SortNumber int              `json:"sort_number"`
	Name       string           `json:"name"`
	Conditions []map[string]any `json:"conditions,omitempty"`
	Route      Route            `json:"route"`
}

// RoutingView is the flat table row of a routing rule.
type RoutingView struct {
	ID            string `json:"id"`
	PaymentMethod string `json:"payment_method"`
	Name          string `json:"name"`
	Providers     string `json:"providers"`
	Conditions    int    `json:"conditions"`
}

// View flattens a routing rule into a table row.
func (r *Routing) View() RoutingView {
	return RoutingView{
		ID:            r.ID,
		PaymentMethod: r.PaymentMethod,
		Name:          r.Name,
		Providers:     r.DefaultRoute.Chain(),
		Conditions:    len(r.ConditionSets),
	}
}

// RoutingViews flattens a list of routing rules.
func RoutingViews(routings []Routing) []RoutingView {
	views := make([]RoutingView, 0, len(routings))
	for i := range routings {
		views = append(views, routings[i].View())
	}

	return views
}

// Chain renders the provider chain of a route as `STRIPE > ADYEN`, in step
// order as the API returned it.
func (r *Route) Chain() string {
	names := make([]string, 0, len(r.Steps))

	for _, step := range r.Steps {
		name := step.ProviderID
		if name == "" {
			name = step.ConnectionID
		}

		names = append(names, name)
	}

	return strings.Join(names, " > ")
}

// RoutingRecommendation is the answer of `POST /routing/recommendations`.
type RoutingRecommendation struct {
	RecommendationID string               `json:"recommendation_id"`
	MerchantOrderID  *string              `json:"merchant_order_id,omitempty"`
	OptimizedFor     string               `json:"optimized_for"`
	DecisionSource   string               `json:"decision_source"`
	Recommended      RecommendedCandidate `json:"recommended"`
	Ranking          []RankedCandidate    `json:"ranking,omitempty"`
}

// RecommendedCandidate is the provider the recommendation engine picked.
type RecommendedCandidate struct {
	ProviderID           string  `json:"provider_id"`
	MerchantConnectionID *string `json:"merchant_connection_id,omitempty"`
}

// RankedCandidate is one scored candidate of a recommendation.
type RankedCandidate struct {
	ProviderID           string   `json:"provider_id"`
	MerchantConnectionID *string  `json:"merchant_connection_id,omitempty"`
	ApprovalRate         *float64 `json:"approval_rate,omitempty"`
	AvgLatencyMs         *int     `json:"avg_latency_ms,omitempty"`
	SampleSize           int      `json:"sample_size"`
}

// RecommendationView is the flat table row of a ranked candidate.
type RecommendationView struct {
	ProviderID           string `json:"provider_id"`
	MerchantConnectionID string `json:"merchant_connection_id"`
	ApprovalRate         string `json:"approval_rate"`
	AvgLatencyMs         string `json:"avg_latency_ms"`
	SampleSize           int    `json:"sample_size"`
	Recommended          bool   `json:"recommended"`
}

// Views flattens a recommendation into one row per ranked candidate, falling
// back to the recommended provider when the API returned no ranking.
func (r *RoutingRecommendation) Views() []RecommendationView {
	if len(r.Ranking) == 0 {
		return []RecommendationView{{
			ProviderID:           r.Recommended.ProviderID,
			MerchantConnectionID: deref(r.Recommended.MerchantConnectionID),
			Recommended:          true,
		}}
	}

	views := make([]RecommendationView, 0, len(r.Ranking))

	for _, c := range r.Ranking {
		views = append(views, RecommendationView{
			ProviderID:           c.ProviderID,
			MerchantConnectionID: deref(c.MerchantConnectionID),
			ApprovalRate:         formatFloat(c.ApprovalRate),
			AvgLatencyMs:         formatInt(c.AvgLatencyMs),
			SampleSize:           c.SampleSize,
			Recommended:          c.ProviderID == r.Recommended.ProviderID,
		})
	}

	return views
}

// deref renders an optional string, with nil becoming an empty cell.
func deref(s *string) string {
	if s == nil {
		return ""
	}

	return *s
}

// formatFloat renders an optional number, with nil becoming an empty cell.
func formatFloat(f *float64) string {
	if f == nil {
		return ""
	}

	return strings.TrimRight(strings.TrimRight(fmt.Sprintf("%.4f", *f), "0"), ".")
}

// formatInt renders an optional integer, with nil becoming an empty cell.
func formatInt(i *int) string {
	if i == nil {
		return ""
	}

	return fmt.Sprintf("%d", *i)
}
