package api

import (
	"context"
	"fmt"
	"net/http"
	"net/url"

	"github.com/pvarganov/yuno-cli/internal/model"
)

// subscriptionPath is the base path of the subscription resource.
const subscriptionPath = "/subscriptions"

// ListSubscriptions returns the subscriptions matching the filters. The
// endpoint pages with `page` / `size`; a limit of zero fetches everything.
func (c *Client) ListSubscriptions(
	ctx context.Context, filters url.Values, limit, pageSize int,
) ([]model.Subscription, error) {
	subscriptions, err := PaginateAll[model.Subscription](ctx, c, Request{
		Method: http.MethodGet,
		Path:   subscriptionPath,
		Query:  filters,
	}, NewSizePager(pageSize), limit)
	if err != nil {
		return nil, fmt.Errorf("list subscriptions: %w", err)
	}

	return subscriptions, nil
}

// GetSubscription retrieves one subscription by id.
func (c *Client) GetSubscription(ctx context.Context, subscriptionID string) (*model.Subscription, error) {
	subscription, err := Do[model.Subscription](ctx, c, Request{
		Method: http.MethodGet,
		Path:   subscriptionPath + "/" + url.PathEscape(subscriptionID),
	})
	if err != nil {
		return nil, fmt.Errorf("get subscription %s: %w", subscriptionID, err)
	}

	return &subscription, nil
}

// CreateSubscription creates a subscription from an already built JSON body.
func (c *Client) CreateSubscription(ctx context.Context, body any) (*model.Subscription, error) {
	subscription, err := Do[model.Subscription](ctx, c, Request{
		Method: http.MethodPost,
		Path:   subscriptionPath,
		Body:   body,
	})
	if err != nil {
		return nil, fmt.Errorf("create subscription: %w", err)
	}

	return &subscription, nil
}

// UpdateSubscription patches a subscription.
func (c *Client) UpdateSubscription(ctx context.Context, subscriptionID string, body any) (*model.Subscription, error) {
	subscription, err := Do[model.Subscription](ctx, c, Request{
		Method: http.MethodPatch,
		Path:   subscriptionPath + "/" + url.PathEscape(subscriptionID),
		Body:   body,
	})
	if err != nil {
		return nil, fmt.Errorf("update subscription %s: %w", subscriptionID, err)
	}

	return &subscription, nil
}

// SubscriptionAction posts one of the lifecycle actions of a subscription
// (`cancel`, `pause`, `resume`, `retry`, `plan`). The body may be nil for the
// actions that take none.
func (c *Client) SubscriptionAction(
	ctx context.Context, subscriptionID, action string, body any,
) (*model.Subscription, error) {
	subscription, err := Do[model.Subscription](ctx, c, Request{
		Method: http.MethodPost,
		Path:   subscriptionPath + "/" + url.PathEscape(subscriptionID) + "/" + action,
		Body:   body,
	})
	if err != nil {
		return nil, fmt.Errorf("%s subscription %s: %w", action, subscriptionID, err)
	}

	return &subscription, nil
}

// ListSubscriptionPayments returns the payments of a subscription. The endpoint
// pages with `limit` / `offset`.
func (c *Client) ListSubscriptionPayments(
	ctx context.Context, subscriptionID string, limit, pageSize int,
) ([]model.SubscriptionPayment, error) {
	payments, err := PaginateAll[model.SubscriptionPayment](ctx, c, Request{
		Method: http.MethodGet,
		Path:   subscriptionPath + "/" + url.PathEscape(subscriptionID) + "/payments",
	}, NewOffsetPager(pageSize), limit)
	if err != nil {
		return nil, fmt.Errorf("list payments of subscription %s: %w", subscriptionID, err)
	}

	return payments, nil
}

// ListPlans returns the subscription plans of an account. The endpoint pages
// with `limit` / `offset`.
func (c *Client) ListPlans(ctx context.Context, accountID string, limit, pageSize int) ([]model.Plan, error) {
	query := url.Values{}
	if accountID != "" {
		query.Set("account_id", accountID)
	}

	plans, err := PaginateAll[model.Plan](ctx, c, Request{
		Method: http.MethodGet,
		Path:   subscriptionPath + "/plans",
		Query:  query,
	}, NewOffsetPager(pageSize), limit)
	if err != nil {
		return nil, fmt.Errorf("list plans: %w", err)
	}

	return plans, nil
}

// GetPlan retrieves one subscription plan by id.
func (c *Client) GetPlan(ctx context.Context, planID string) (*model.Plan, error) {
	plan, err := Do[model.Plan](ctx, c, Request{
		Method: http.MethodGet,
		Path:   subscriptionPath + "/plans/" + url.PathEscape(planID),
	})
	if err != nil {
		return nil, fmt.Errorf("get plan %s: %w", planID, err)
	}

	return &plan, nil
}

// CreatePlan creates a subscription plan from an already built JSON body.
func (c *Client) CreatePlan(ctx context.Context, body any) (*model.Plan, error) {
	plan, err := Do[model.Plan](ctx, c, Request{
		Method: http.MethodPost,
		Path:   subscriptionPath + "/plans",
		Body:   body,
	})
	if err != nil {
		return nil, fmt.Errorf("create plan: %w", err)
	}

	return &plan, nil
}

// UpdatePlanStatus changes the status of a plan, which today means cancelling it.
func (c *Client) UpdatePlanStatus(ctx context.Context, planID string, body any) (*model.PlanStatus, error) {
	status, err := Do[model.PlanStatus](ctx, c, Request{
		Method: http.MethodPost,
		Path:   subscriptionPath + "/plans/" + url.PathEscape(planID) + "/status",
		Body:   body,
	})
	if err != nil {
		return nil, fmt.Errorf("update status of plan %s: %w", planID, err)
	}

	return &status, nil
}
