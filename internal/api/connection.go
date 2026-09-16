package api

import (
	"context"
	"fmt"
	"net/http"
	"net/url"

	"github.com/pvarganov/yuno-cli/internal/model"
)

// GetProviderCatalog returns the payment methods and connection parameters a
// provider supports.
func (c *Client) GetProviderCatalog(ctx context.Context, providerID string) (*model.ProviderCatalog, error) {
	catalog, err := Do[model.ProviderCatalog](ctx, c, Request{
		Method: http.MethodGet,
		Path:   "/connections/catalog/" + url.PathEscape(providerID),
	})
	if err != nil {
		return nil, fmt.Errorf("get provider catalog %s: %w", providerID, err)
	}

	return &catalog, nil
}

// GetConnection returns one provider connection by id.
func (c *Client) GetConnection(ctx context.Context, connectionID string) (*model.Connection, error) {
	connection, err := Do[model.Connection](ctx, c, Request{
		Method: http.MethodGet,
		Path:   "/connections/" + url.PathEscape(connectionID),
	})
	if err != nil {
		return nil, fmt.Errorf("get connection %s: %w", connectionID, err)
	}

	return &connection, nil
}

// CreateConnection creates a provider connection from an already built body.
func (c *Client) CreateConnection(ctx context.Context, body any) (*model.Connection, error) {
	connection, err := Do[model.Connection](ctx, c, Request{
		Method: http.MethodPost,
		Path:   "/connections",
		Body:   body,
	})
	if err != nil {
		return nil, fmt.Errorf("create connection: %w", err)
	}

	return &connection, nil
}
