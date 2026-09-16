package api

import (
	"context"
	"fmt"
	"net/http"
	"net/url"

	"github.com/pvarganov/yuno-cli/internal/model"
)

// CreateCustomer creates a customer from an already built JSON body.
func (c *Client) CreateCustomer(ctx context.Context, body any) (*model.Customer, error) {
	customer, err := Do[model.Customer](ctx, c, Request{
		Method: http.MethodPost,
		Path:   "/customers",
		Body:   body,
	})
	if err != nil {
		return nil, fmt.Errorf("create customer: %w", err)
	}

	return &customer, nil
}

// GetCustomer retrieves a customer by its Yuno id.
func (c *Client) GetCustomer(ctx context.Context, customerID string) (*model.Customer, error) {
	customer, err := Do[model.Customer](ctx, c, Request{
		Method: http.MethodGet,
		Path:   "/customers/" + url.PathEscape(customerID),
	})
	if err != nil {
		return nil, fmt.Errorf("get customer %s: %w", customerID, err)
	}

	return &customer, nil
}

// GetCustomerByMerchantCustomerID retrieves a customer by the id you gave it.
func (c *Client) GetCustomerByMerchantCustomerID(ctx context.Context, merchantCustomerID string) (*model.Customer, error) {
	query := url.Values{}
	query.Set("merchant_customer_id", merchantCustomerID)

	customer, err := Do[model.Customer](ctx, c, Request{
		Method: http.MethodGet,
		Path:   "/customers",
		Query:  query,
	})
	if err != nil {
		return nil, fmt.Errorf("get customer by merchant customer id %s: %w", merchantCustomerID, err)
	}

	return &customer, nil
}

// UpdateCustomer patches a customer.
func (c *Client) UpdateCustomer(ctx context.Context, customerID string, body any) (*model.Customer, error) {
	customer, err := Do[model.Customer](ctx, c, Request{
		Method: http.MethodPatch,
		Path:   "/customers/" + url.PathEscape(customerID),
		Body:   body,
	})
	if err != nil {
		return nil, fmt.Errorf("update customer %s: %w", customerID, err)
	}

	return &customer, nil
}

// DeleteCustomer deletes a customer. Yuno answers 202 with an empty body, so
// the raw response is returned as is.
func (c *Client) DeleteCustomer(ctx context.Context, customerID string) ([]byte, error) {
	data, err := c.DoRaw(ctx, Request{
		Method: http.MethodDelete,
		Path:   "/customers/" + url.PathEscape(customerID),
	})
	if err != nil {
		return nil, fmt.Errorf("delete customer %s: %w", customerID, err)
	}

	return data, nil
}

// CreateCustomerSession opens an enrollment session for a customer.
func (c *Client) CreateCustomerSession(ctx context.Context, body any) (*model.CustomerSession, error) {
	session, err := Do[model.CustomerSession](ctx, c, Request{
		Method: http.MethodPost,
		Path:   "/customers/sessions",
		Body:   body,
	})
	if err != nil {
		return nil, fmt.Errorf("create customer session: %w", err)
	}

	return &session, nil
}

// GenerateNetworkTokenCryptogram generates a cryptogram for the network token
// of a vaulted card.
func (c *Client) GenerateNetworkTokenCryptogram(ctx context.Context, body any) (*model.NetworkTokenCryptogram, error) {
	cryptogram, err := Do[model.NetworkTokenCryptogram](ctx, c, Request{
		Method: http.MethodPost,
		Path:   "/network-tokens/cryptograms",
		Body:   body,
	})
	if err != nil {
		return nil, fmt.Errorf("generate network token cryptogram: %w", err)
	}

	return &cryptogram, nil
}
