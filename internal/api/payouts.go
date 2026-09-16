package api

import (
	"context"
	"fmt"
	"net/http"
	"net/url"

	"github.com/pvarganov/yuno-cli/internal/model"
)

// payoutPath is the base path of the payout resource.
const payoutPath = "/payouts"

// ListPayouts looks payouts up by merchant reference. Yuno has no unfiltered
// payout list: `GET /payouts` requires `merchant_reference` and answers with an
// array of the payouts created under it.
func (c *Client) ListPayouts(ctx context.Context, merchantReference string) ([]model.Payout, error) {
	data, err := c.DoRaw(ctx, Request{
		Method: http.MethodGet,
		Path:   payoutPath,
		Query:  url.Values{"merchant_reference": []string{merchantReference}},
	})
	if err != nil {
		return nil, fmt.Errorf("list payouts by merchant reference %s: %w", merchantReference, err)
	}

	payouts, err := model.DecodePayouts(data)
	if err != nil {
		return nil, fmt.Errorf("list payouts by merchant reference %s: %w", merchantReference, err)
	}

	return payouts, nil
}

// GetPayout retrieves one payout by id.
func (c *Client) GetPayout(ctx context.Context, payoutID string) (*model.Payout, error) {
	payout, err := Do[model.Payout](ctx, c, Request{
		Method: http.MethodGet,
		Path:   payoutPath + "/" + url.PathEscape(payoutID),
	})
	if err != nil {
		return nil, fmt.Errorf("get payout %s: %w", payoutID, err)
	}

	return &payout, nil
}

// CreatePayout creates a payout from an already built JSON body.
func (c *Client) CreatePayout(ctx context.Context, body any) (*model.Payout, error) {
	payout, err := Do[model.Payout](ctx, c, Request{
		Method: http.MethodPost,
		Path:   payoutPath,
		Body:   body,
	})
	if err != nil {
		return nil, fmt.Errorf("create payout: %w", err)
	}

	return &payout, nil
}
