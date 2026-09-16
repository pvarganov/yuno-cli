package api

import (
	"context"
	"fmt"
	"net/http"
	"net/url"

	"github.com/pvarganov/yuno-cli/internal/model"
)

// checkoutBuilderPath is the base path of the checkout builder resource.
const checkoutBuilderPath = "/checkouts"

// CreateCheckout creates an empty checkout configuration. The id it answers with
// is the checkout_code every other call takes.
func (c *Client) CreateCheckout(ctx context.Context, body any) (*model.Checkout, error) {
	checkout, err := Do[model.Checkout](ctx, c, Request{
		Method: http.MethodPost,
		Path:   checkoutBuilderPath,
		Body:   body,
	})
	if err != nil {
		return nil, fmt.Errorf("create checkout: %w", err)
	}

	return &checkout, nil
}

// ListCheckouts walks `GET /checkouts`, which pages with `page` and `size`
// starting at page one and answers with a `data` / `pagination` envelope.
func (c *Client) ListCheckouts(
	ctx context.Context, query url.Values, limit, pageSize int,
) ([]model.Checkout, error) {
	checkouts, err := PaginateAll[model.Checkout](ctx, c, Request{
		Method: http.MethodGet,
		Path:   checkoutBuilderPath,
		Query:  query,
	}, NewSizeNumberPager(pageSize), limit)
	if err != nil {
		return nil, fmt.Errorf("list checkouts: %w", err)
	}

	return checkouts, nil
}

// GetCheckout retrieves one checkout configuration, styling included.
func (c *Client) GetCheckout(ctx context.Context, checkoutCode string) (*model.Checkout, error) {
	checkout, err := Do[model.Checkout](ctx, c, Request{
		Method: http.MethodGet,
		Path:   checkoutBuilderPath + "/" + url.PathEscape(checkoutCode),
	})
	if err != nil {
		return nil, fmt.Errorf("get checkout %s: %w", checkoutCode, err)
	}

	return &checkout, nil
}

// PublishCheckout writes the configuration and the styling of a checkout. Yuno
// exposes this as a PUT: the body replaces what it touches, so send every
// payment method that should survive the call, not only the ones that change.
func (c *Client) PublishCheckout(ctx context.Context, checkoutCode string, body any) ([]byte, error) {
	data, err := c.DoRaw(ctx, Request{
		Method: http.MethodPut,
		Path:   checkoutBuilderPath + "/" + url.PathEscape(checkoutCode),
		Body:   body,
	})
	if err != nil {
		return nil, fmt.Errorf("publish checkout %s: %w", checkoutCode, err)
	}

	return data, nil
}

// UpdateCheckout drives the lifecycle of a checkout: its name, its description,
// its status and whether it is the account default.
func (c *Client) UpdateCheckout(ctx context.Context, checkoutCode string, body any) (*model.Checkout, error) {
	checkout, err := Do[model.Checkout](ctx, c, Request{
		Method: http.MethodPatch,
		Path:   checkoutBuilderPath + "/" + url.PathEscape(checkoutCode),
		Body:   body,
	})
	if err != nil {
		return nil, fmt.Errorf("update checkout %s: %w", checkoutCode, err)
	}

	return &checkout, nil
}
