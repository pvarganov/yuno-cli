package api

import (
	"context"
	"fmt"
	"net/http"

	"github.com/pvarganov/yuno-cli/internal/model"
)

// AI caller paths. The endpoint names read backwards: `/payments` is the
// abandoned-flow outreach and `/payments/recover` is the declined payment call,
// as the spec's operation ids spell out. The command names follow the spec, not
// the paths.
const (
	aiCallerAbandonedPath = "/smart-support/external/payments"
	aiCallerDeclinedPath  = "/smart-support/external/payments/recover"
)

// AICallerDeclinedPayments asks the AI caller to reach out to the payer of a
// declined payment and recover it.
func (c *Client) AICallerDeclinedPayments(ctx context.Context, body any) (*model.AICallerOutreach, error) {
	outreach, err := Do[model.AICallerOutreach](ctx, c, Request{
		Method: http.MethodPost,
		Path:   aiCallerDeclinedPath,
		Body:   body,
	})
	if err != nil {
		return nil, fmt.Errorf("start declined payment outreach: %w", err)
	}

	return &outreach, nil
}

// AICallerRecover asks the AI caller to reach out to a customer who abandoned a
// checkout flow. Yuno answers 200 with no body, so the raw response is returned.
func (c *Client) AICallerRecover(ctx context.Context, body any) ([]byte, error) {
	data, err := c.DoRaw(ctx, Request{
		Method: http.MethodPost,
		Path:   aiCallerAbandonedPath,
		Body:   body,
	})
	if err != nil {
		return nil, fmt.Errorf("start abandoned flow outreach: %w", err)
	}

	return data, nil
}
