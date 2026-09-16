package api

import (
	"context"
	"fmt"
	"net/http"
	"net/url"

	"github.com/pvarganov/yuno-cli/internal/model"
)

// webhookPath is the base path of the webhook resource.
const webhookPath = "/webhooks"

// ListWebhooks lists the webhooks of an account. `GET /webhooks` requires
// `account_id`, takes an optional `state` filter and answers with a bare array:
// there is no pagination on this endpoint.
func (c *Client) ListWebhooks(ctx context.Context, accountID, state string) ([]model.Webhook, error) {
	query := url.Values{"account_id": []string{accountID}}
	if state != "" {
		query.Set("state", state)
	}

	webhooks, err := Do[[]model.Webhook](ctx, c, Request{
		Method: http.MethodGet,
		Path:   webhookPath,
		Query:  query,
	})
	if err != nil {
		return nil, fmt.Errorf("list webhooks of account %s: %w", accountID, err)
	}

	return webhooks, nil
}

// GetWebhook retrieves one webhook of an account.
func (c *Client) GetWebhook(ctx context.Context, webhookID, accountID string) (*model.Webhook, error) {
	webhook, err := Do[model.Webhook](ctx, c, Request{
		Method: http.MethodGet,
		Path:   webhookPath + "/" + url.PathEscape(webhookID),
		Query:  url.Values{"account_id": []string{accountID}},
	})
	if err != nil {
		return nil, fmt.Errorf("get webhook %s: %w", webhookID, err)
	}

	return &webhook, nil
}

// CreateWebhook registers a webhook from an already built JSON body.
func (c *Client) CreateWebhook(ctx context.Context, body any) (*model.Webhook, error) {
	webhook, err := Do[model.Webhook](ctx, c, Request{
		Method: http.MethodPost,
		Path:   webhookPath,
		Body:   body,
	})
	if err != nil {
		return nil, fmt.Errorf("create webhook: %w", err)
	}

	return &webhook, nil
}

// UpdateWebhook patches a webhook. The body carries the account id, which Yuno
// requires even on the update.
func (c *Client) UpdateWebhook(ctx context.Context, webhookID string, body any) (*model.Webhook, error) {
	webhook, err := Do[model.Webhook](ctx, c, Request{
		Method: http.MethodPatch,
		Path:   webhookPath + "/" + url.PathEscape(webhookID),
		Body:   body,
	})
	if err != nil {
		return nil, fmt.Errorf("update webhook %s: %w", webhookID, err)
	}

	return &webhook, nil
}

// DeleteWebhook removes a webhook and returns the raw confirmation body.
func (c *Client) DeleteWebhook(ctx context.Context, webhookID, accountID string) ([]byte, error) {
	data, err := c.DoRaw(ctx, Request{
		Method: http.MethodDelete,
		Path:   webhookPath + "/" + url.PathEscape(webhookID),
		Query:  url.Values{"account_id": []string{accountID}},
	})
	if err != nil {
		return nil, fmt.Errorf("delete webhook %s: %w", webhookID, err)
	}

	return data, nil
}
