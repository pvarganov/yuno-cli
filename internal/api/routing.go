package api

import (
	"context"
	"fmt"
	"net/http"
	"net/url"

	"github.com/pvarganov/yuno-cli/internal/model"
)

// ListRoutings returns the routing rules of an account, optionally narrowed to
// one payment method.
func (c *Client) ListRoutings(ctx context.Context, accountID, paymentMethod string) ([]model.Routing, error) {
	query := url.Values{}
	query.Set("account_id", accountID)

	if paymentMethod != "" {
		query.Set("payment_method", paymentMethod)
	}

	routings, err := Do[[]model.Routing](ctx, c, Request{
		Method: http.MethodGet,
		Path:   "/routing",
		Query:  query,
	})
	if err != nil {
		return nil, fmt.Errorf("list routings: %w", err)
	}

	return routings, nil
}

// GetRouting returns one routing rule by id.
func (c *Client) GetRouting(ctx context.Context, routingID string) (*model.Routing, error) {
	routing, err := Do[model.Routing](ctx, c, Request{
		Method: http.MethodGet,
		Path:   "/routing/" + url.PathEscape(routingID),
	})
	if err != nil {
		return nil, fmt.Errorf("get routing %s: %w", routingID, err)
	}

	return &routing, nil
}

// CreateRouting creates a routing rule from an already built JSON body.
func (c *Client) CreateRouting(ctx context.Context, body any) (*model.Routing, error) {
	routing, err := Do[model.Routing](ctx, c, Request{
		Method: http.MethodPost,
		Path:   "/routing",
		Body:   body,
	})
	if err != nil {
		return nil, fmt.Errorf("create routing: %w", err)
	}

	return &routing, nil
}

// UpdateRouting patches a routing rule.
func (c *Client) UpdateRouting(ctx context.Context, routingID string, body any) (*model.Routing, error) {
	routing, err := Do[model.Routing](ctx, c, Request{
		Method: http.MethodPatch,
		Path:   "/routing/" + url.PathEscape(routingID),
		Body:   body,
	})
	if err != nil {
		return nil, fmt.Errorf("update routing %s: %w", routingID, err)
	}

	return &routing, nil
}

// RecommendRouting asks Yuno which candidate provider to route a payment to.
func (c *Client) RecommendRouting(ctx context.Context, body any) (*model.RoutingRecommendation, error) {
	recommendation, err := Do[model.RoutingRecommendation](ctx, c, Request{
		Method: http.MethodPost,
		Path:   "/routing/recommendations",
		Body:   body,
	})
	if err != nil {
		return nil, fmt.Errorf("recommend routing: %w", err)
	}

	return &recommendation, nil
}
