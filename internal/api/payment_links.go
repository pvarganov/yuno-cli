package api

import (
	"context"
	"fmt"
	"net/http"
	"net/url"

	"github.com/pvarganov/yuno-cli/internal/model"
)

// paymentLinkPath is the base path of the payment link resource.
const paymentLinkPath = "/payment-links"

// CreatePaymentLink creates a payment link from an already built JSON body.
func (c *Client) CreatePaymentLink(ctx context.Context, body any) (*model.PaymentLink, error) {
	link, err := Do[model.PaymentLink](ctx, c, Request{
		Method: http.MethodPost,
		Path:   paymentLinkPath,
		Body:   body,
	})
	if err != nil {
		return nil, fmt.Errorf("create payment link: %w", err)
	}

	return &link, nil
}

// GetPaymentLink retrieves one payment link by its code.
func (c *Client) GetPaymentLink(ctx context.Context, code string) (*model.PaymentLink, error) {
	link, err := Do[model.PaymentLink](ctx, c, Request{
		Method: http.MethodGet,
		Path:   paymentLinkPath + "/" + url.PathEscape(code),
	})
	if err != nil {
		return nil, fmt.Errorf("get payment link %s: %w", code, err)
	}

	return &link, nil
}

// CancelPaymentLink cancels a payment link so it can no longer be paid.
func (c *Client) CancelPaymentLink(ctx context.Context, code string) (*model.PaymentLink, error) {
	link, err := Do[model.PaymentLink](ctx, c, Request{
		Method: http.MethodPost,
		Path:   paymentLinkPath + "/" + url.PathEscape(code) + "/cancel",
	})
	if err != nil {
		return nil, fmt.Errorf("cancel payment link %s: %w", code, err)
	}

	return &link, nil
}

// GetConversionRate quotes the currency conversion of an amount for a provider.
func (c *Client) GetConversionRate(ctx context.Context, body any) (*model.ConversionRate, error) {
	rate, err := Do[model.ConversionRate](ctx, c, Request{
		Method: http.MethodPost,
		Path:   "/currency-conversion",
		Body:   body,
	})
	if err != nil {
		return nil, fmt.Errorf("get conversion rate: %w", err)
	}

	return &rate, nil
}
