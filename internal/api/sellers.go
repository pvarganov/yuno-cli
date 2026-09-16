package api

import (
	"context"
	"fmt"
	"net/http"
	"net/url"

	"github.com/pvarganov/yuno-cli/internal/model"
)

// sellerPath is the base path of the seller resource.
const sellerPath = "/sellers"

// CreateSeller registers a marketplace seller from an already built JSON body.
func (c *Client) CreateSeller(ctx context.Context, body any) (*model.Seller, error) {
	seller, err := Do[model.Seller](ctx, c, Request{
		Method: http.MethodPost,
		Path:   sellerPath,
		Body:   body,
	})
	if err != nil {
		return nil, fmt.Errorf("create seller: %w", err)
	}

	return &seller, nil
}

// GetSeller retrieves one seller by the merchant's own identifier.
func (c *Client) GetSeller(ctx context.Context, merchantSellerID string) (*model.Seller, error) {
	seller, err := Do[model.Seller](ctx, c, Request{
		Method: http.MethodGet,
		Path:   sellerPath + "/" + url.PathEscape(merchantSellerID),
	})
	if err != nil {
		return nil, fmt.Errorf("get seller %s: %w", merchantSellerID, err)
	}

	return &seller, nil
}

// UpdateSeller replaces a seller. Yuno exposes the update as a PUT, so the body
// has to carry every field that should survive the call.
func (c *Client) UpdateSeller(ctx context.Context, merchantSellerID string, body any) (*model.Seller, error) {
	seller, err := Do[model.Seller](ctx, c, Request{
		Method: http.MethodPut,
		Path:   sellerPath + "/" + url.PathEscape(merchantSellerID),
		Body:   body,
	})
	if err != nil {
		return nil, fmt.Errorf("update seller %s: %w", merchantSellerID, err)
	}

	return &seller, nil
}

// DeleteSeller removes a seller. Yuno answers 204, so the body is empty.
func (c *Client) DeleteSeller(ctx context.Context, merchantSellerID string) ([]byte, error) {
	data, err := c.DoRaw(ctx, Request{
		Method: http.MethodDelete,
		Path:   sellerPath + "/" + url.PathEscape(merchantSellerID),
	})
	if err != nil {
		return nil, fmt.Errorf("delete seller %s: %w", merchantSellerID, err)
	}

	return data, nil
}
