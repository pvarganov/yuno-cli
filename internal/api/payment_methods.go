package api

import (
	"context"
	"fmt"
	"net/http"
	"net/url"

	"github.com/pvarganov/yuno-cli/internal/model"
)

// ListCustomerPaymentMethods returns the payment methods enrolled for a
// customer. Yuno wraps them in a `payment_methods` object.
func (c *Client) ListCustomerPaymentMethods(
	ctx context.Context, customerID string,
) ([]model.CustomerPaymentMethod, error) {
	list, err := Do[model.CustomerPaymentMethodList](ctx, c, Request{
		Method: http.MethodGet,
		Path:   "/customers/" + url.PathEscape(customerID) + "/payment-methods",
	})
	if err != nil {
		return nil, fmt.Errorf("list payment methods of customer %s: %w", customerID, err)
	}

	return list.PaymentMethods, nil
}

// GetCustomerPaymentMethod retrieves one payment method enrolled for a customer.
func (c *Client) GetCustomerPaymentMethod(
	ctx context.Context, customerID, paymentMethodID string,
) (*model.CustomerPaymentMethod, error) {
	method, err := Do[model.CustomerPaymentMethod](ctx, c, Request{
		Method: http.MethodGet,
		Path: "/customers/" + url.PathEscape(customerID) +
			"/payment-methods/" + url.PathEscape(paymentMethodID),
	})
	if err != nil {
		return nil, fmt.Errorf("get payment method %s of customer %s: %w", paymentMethodID, customerID, err)
	}

	return &method, nil
}

// EnrollCustomerPaymentMethod enrolls a payment method for a customer through
// the direct workflow.
func (c *Client) EnrollCustomerPaymentMethod(
	ctx context.Context, customerID string, body any,
) (*model.CustomerPaymentMethod, error) {
	method, err := Do[model.CustomerPaymentMethod](ctx, c, Request{
		Method: http.MethodPost,
		Path:   "/customers/" + url.PathEscape(customerID) + "/payment-methods",
		Body:   body,
	})
	if err != nil {
		return nil, fmt.Errorf("enroll payment method for customer %s: %w", customerID, err)
	}

	return &method, nil
}

// UnenrollCustomerPaymentMethod unenrolls a payment method of a customer.
func (c *Client) UnenrollCustomerPaymentMethod(
	ctx context.Context, customerID, paymentMethodID string,
) (*model.CustomerPaymentMethod, error) {
	method, err := Do[model.CustomerPaymentMethod](ctx, c, Request{
		Method: http.MethodPost,
		Path: "/customers/" + url.PathEscape(customerID) +
			"/payment-methods/" + url.PathEscape(paymentMethodID) + "/unenroll",
	})
	if err != nil {
		return nil, fmt.Errorf("unenroll payment method %s of customer %s: %w", paymentMethodID, customerID, err)
	}

	return &method, nil
}

// ReassignPaymentMethod moves a payment method to another customer. The
// operation is scoped by X-Account-Code; accountCode overrides the one the
// profile pins, and an empty value leaves the profile header in place.
func (c *Client) ReassignPaymentMethod(
	ctx context.Context, paymentMethodID, accountCode string, body any,
) (*model.CustomerPaymentMethod, error) {
	var headers map[string]string
	if accountCode != "" {
		headers = map[string]string{HeaderAccountCode: accountCode}
	}

	method, err := Do[model.CustomerPaymentMethod](ctx, c, Request{
		Method:  http.MethodPatch,
		Path:    "/payment-methods/" + url.PathEscape(paymentMethodID),
		Body:    body,
		Headers: headers,
	})
	if err != nil {
		return nil, fmt.Errorf("reassign payment method %s: %w", paymentMethodID, err)
	}

	return &method, nil
}

// RegisterCardsForAccountUpdater registers stored cards for the card account
// updater.
func (c *Client) RegisterCardsForAccountUpdater(ctx context.Context, body any) (*model.AccountUpdaterResult, error) {
	result, err := Do[model.AccountUpdaterResult](ctx, c, Request{
		Method: http.MethodPost,
		Path:   "/payment-methods/account-updater",
		Body:   body,
	})
	if err != nil {
		return nil, fmt.Errorf("register cards for account updater: %w", err)
	}

	return &result, nil
}
