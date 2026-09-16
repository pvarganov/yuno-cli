package api

import (
	"context"
	"fmt"
	"net/http"
	"net/url"

	"github.com/pvarganov/yuno-cli/internal/model"
)

// CreateCheckoutSession opens a checkout session for an order.
func (c *Client) CreateCheckoutSession(ctx context.Context, body any) (*model.CheckoutSession, error) {
	session, err := Do[model.CheckoutSession](ctx, c, Request{
		Method: http.MethodPost,
		Path:   "/checkout/sessions",
		Body:   body,
	})
	if err != nil {
		return nil, fmt.Errorf("create checkout session: %w", err)
	}

	return &session, nil
}

// GetCheckoutSession retrieves one checkout session.
func (c *Client) GetCheckoutSession(ctx context.Context, checkoutSession string) (*model.CheckoutSession, error) {
	session, err := Do[model.CheckoutSession](ctx, c, Request{
		Method: http.MethodGet,
		Path:   "/checkout/sessions/" + url.PathEscape(checkoutSession),
	})
	if err != nil {
		return nil, fmt.Errorf("get checkout session %s: %w", checkoutSession, err)
	}

	return &session, nil
}

// UpdateCheckoutSession patches a checkout session that has not been used yet.
func (c *Client) UpdateCheckoutSession(ctx context.Context, checkoutSession string, body any) (*model.CheckoutSession, error) {
	session, err := Do[model.CheckoutSession](ctx, c, Request{
		Method: http.MethodPatch,
		Path:   "/checkout/sessions/" + url.PathEscape(checkoutSession),
		Body:   body,
	})
	if err != nil {
		return nil, fmt.Errorf("update checkout session %s: %w", checkoutSession, err)
	}

	return &session, nil
}

// ListCheckoutSessionPaymentMethods returns the payment methods available for a
// checkout session, optionally narrowed to one category.
func (c *Client) ListCheckoutSessionPaymentMethods(
	ctx context.Context, checkoutSession, category string,
) ([]model.CheckoutPaymentMethod, error) {
	query := url.Values{}
	if category != "" {
		query.Set("category", category)
	}

	methods, err := Do[[]model.CheckoutPaymentMethod](ctx, c, Request{
		Method: http.MethodGet,
		Path:   "/checkout/sessions/" + url.PathEscape(checkoutSession) + "/payment-methods",
		Query:  query,
	})
	if err != nil {
		return nil, fmt.Errorf("list payment methods of checkout session %s: %w", checkoutSession, err)
	}

	return methods, nil
}

// ListEnrollablePaymentMethods returns the payment methods a customer session
// can enroll. Yuno wraps them in a `payment_methods` object here.
func (c *Client) ListEnrollablePaymentMethods(
	ctx context.Context, customerSession string,
) ([]model.CheckoutPaymentMethod, error) {
	list, err := Do[model.CheckoutPaymentMethodList](ctx, c, Request{
		Method: http.MethodGet,
		Path:   "/checkout/customers/sessions/" + url.PathEscape(customerSession) + "/payment-methods",
	})
	if err != nil {
		return nil, fmt.Errorf("list enrollable payment methods of customer session %s: %w", customerSession, err)
	}

	return list.PaymentMethods, nil
}

// EnrollPaymentMethod starts the enrollment of a payment method in a customer
// session.
func (c *Client) EnrollPaymentMethod(
	ctx context.Context, customerSession string, body any,
) (*model.CustomerPaymentMethod, error) {
	method, err := Do[model.CustomerPaymentMethod](ctx, c, Request{
		Method: http.MethodPost,
		Path:   "/customers/sessions/" + url.PathEscape(customerSession) + "/payment-methods",
		Body:   body,
	})
	if err != nil {
		return nil, fmt.Errorf("enroll payment method in customer session %s: %w", customerSession, err)
	}

	return &method, nil
}

// GetPaymentMethod retrieves one payment method by its Yuno id.
func (c *Client) GetPaymentMethod(ctx context.Context, paymentMethodID string) (*model.CustomerPaymentMethod, error) {
	method, err := Do[model.CustomerPaymentMethod](ctx, c, Request{
		Method: http.MethodGet,
		Path:   "/payment-methods/" + url.PathEscape(paymentMethodID),
	})
	if err != nil {
		return nil, fmt.Errorf("get payment method %s: %w", paymentMethodID, err)
	}

	return &method, nil
}

// UnenrollPaymentMethod unenrolls a payment method of a customer.
func (c *Client) UnenrollPaymentMethod(ctx context.Context, paymentMethodID string) (*model.CustomerPaymentMethod, error) {
	method, err := Do[model.CustomerPaymentMethod](ctx, c, Request{
		Method: http.MethodPost,
		Path:   "/customers/payment-methods/" + url.PathEscape(paymentMethodID) + "/unenroll",
	})
	if err != nil {
		return nil, fmt.Errorf("unenroll payment method %s: %w", paymentMethodID, err)
	}

	return &method, nil
}
