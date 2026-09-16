package api

import (
	"context"
	"fmt"
	"net/http"

	"github.com/pvarganov/yuno-cli/internal/model"
)

// CreateThreeDSecureSetup creates a 3-D Secure setup for a browser.
func (c *Client) CreateThreeDSecureSetup(ctx context.Context, body any) (*model.ThreeDSecureSetup, error) {
	setup, err := Do[model.ThreeDSecureSetup](ctx, c, Request{
		Method: http.MethodPost,
		Path:   "/three-d-secure/setups",
		Body:   body,
	})
	if err != nil {
		return nil, fmt.Errorf("create three-d-secure setup: %w", err)
	}

	return &setup, nil
}

// CreateDryRunProviderEvent registers a simulated provider event.
func (c *Client) CreateDryRunProviderEvent(ctx context.Context, body any) (*model.DryRunProviderEvent, error) {
	event, err := Do[model.DryRunProviderEvent](ctx, c, Request{
		Method: http.MethodPost,
		Path:   "/dry-run/provider-events",
		Body:   body,
	})
	if err != nil {
		return nil, fmt.Errorf("create dry run provider event: %w", err)
	}

	return &event, nil
}
