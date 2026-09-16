package api

import (
	"context"
	"fmt"
	"net/http"
	"net/url"

	"github.com/pvarganov/yuno-cli/internal/model"
)

// paymentTransactionPath builds the transaction scoped path of a payment
// action, e.g. `/payments/p-1/transactions/t-1/capture`.
func paymentTransactionPath(paymentID, transactionID, action string) string {
	return "/payments/" + url.PathEscape(paymentID) +
		"/transactions/" + url.PathEscape(transactionID) + "/" + action
}

// CreatePayment creates a payment from an already built JSON body.
func (c *Client) CreatePayment(ctx context.Context, body any) (*model.Payment, error) {
	payment, err := Do[model.Payment](ctx, c, Request{
		Method: http.MethodPost,
		Path:   "/payments",
		Body:   body,
	})
	if err != nil {
		return nil, fmt.Errorf("create payment: %w", err)
	}

	return &payment, nil
}

// GetPayment retrieves a payment by id, optionally with the provider raw
// responses and the full transaction history.
func (c *Client) GetPayment(ctx context.Context, paymentID string, rawResponse, history bool) (*model.Payment, error) {
	query := url.Values{}

	if rawResponse {
		query.Set("raw_response", "true")
	}

	if history {
		query.Set("transactions_history", "true")
	}

	payment, err := Do[model.Payment](ctx, c, Request{
		Method: http.MethodGet,
		Path:   "/payments/" + url.PathEscape(paymentID),
		Query:  query,
	})
	if err != nil {
		return nil, fmt.Errorf("get payment %s: %w", paymentID, err)
	}

	return &payment, nil
}

// GetPaymentByMerchantOrderID retrieves the payments of a merchant order.
func (c *Client) GetPaymentByMerchantOrderID(ctx context.Context, merchantOrderID string) ([]model.Payment, error) {
	query := url.Values{}
	query.Set("merchant_order_id", merchantOrderID)

	payments, err := Do[[]model.Payment](ctx, c, Request{
		Method: http.MethodGet,
		Path:   "/payments",
		Query:  query,
	})
	if err != nil {
		return nil, fmt.Errorf("get payment by merchant order id %s: %w", merchantOrderID, err)
	}

	return payments, nil
}

// ListIssuers returns the banks available for a payment method in a country.
func (c *Client) ListIssuers(ctx context.Context, countryCode, paymentMethod, checkoutSession string) (*model.IssuerList, error) {
	query := url.Values{}

	for param, value := range map[string]string{
		"country_code":     countryCode,
		"payment_method":   paymentMethod,
		"checkout_session": checkoutSession,
	} {
		if value != "" {
			query.Set(param, value)
		}
	}

	issuers, err := Do[model.IssuerList](ctx, c, Request{
		Method: http.MethodGet,
		Path:   "/issuers",
		Query:  query,
	})
	if err != nil {
		return nil, fmt.Errorf("list issuers: %w", err)
	}

	return &issuers, nil
}

// RefundPayment refunds one transaction of a payment.
func (c *Client) RefundPayment(ctx context.Context, paymentID, transactionID string, body any) (*model.Payment, error) {
	// The refund operation is the only one of the group that names the
	// payment `{id}` in the spec; the path shape is the same.
	payment, err := c.paymentAction(ctx, paymentTransactionPath(paymentID, transactionID, "refund"), body)
	if err != nil {
		return nil, fmt.Errorf("refund payment %s transaction %s: %w", paymentID, transactionID, err)
	}

	return payment, nil
}

// CancelTransaction cancels one transaction of a payment.
func (c *Client) CancelTransaction(ctx context.Context, paymentID, transactionID string, body any) (*model.Payment, error) {
	payment, err := c.paymentAction(ctx, paymentTransactionPath(paymentID, transactionID, "cancel"), body)
	if err != nil {
		return nil, fmt.Errorf("cancel payment %s transaction %s: %w", paymentID, transactionID, err)
	}

	return payment, nil
}

// CaptureTransaction captures an authorized transaction of a payment.
func (c *Client) CaptureTransaction(ctx context.Context, paymentID, transactionID string, body any) (*model.Payment, error) {
	payment, err := c.paymentAction(ctx, paymentTransactionPath(paymentID, transactionID, "capture"), body)
	if err != nil {
		return nil, fmt.Errorf("capture payment %s transaction %s: %w", paymentID, transactionID, err)
	}

	return payment, nil
}

// CancelOrRefundPayment lets Yuno decide between a cancellation and a refund
// for the whole payment.
func (c *Client) CancelOrRefundPayment(ctx context.Context, paymentID string, body any) (*model.Payment, error) {
	payment, err := c.paymentAction(ctx, "/payments/"+url.PathEscape(paymentID)+"/cancel-or-refund", body)
	if err != nil {
		return nil, fmt.Errorf("cancel or refund payment %s: %w", paymentID, err)
	}

	return payment, nil
}

// CancelOrRefundTransaction lets Yuno decide between a cancellation and a
// refund for one transaction of a payment.
func (c *Client) CancelOrRefundTransaction(ctx context.Context, paymentID, transactionID string, body any) (*model.Payment, error) {
	payment, err := c.paymentAction(ctx, paymentTransactionPath(paymentID, transactionID, "cancel-or-refund"), body)
	if err != nil {
		return nil, fmt.Errorf("cancel or refund payment %s transaction %s: %w", paymentID, transactionID, err)
	}

	return payment, nil
}

// paymentAction posts a payment action and decodes the returned payment.
func (c *Client) paymentAction(ctx context.Context, path string, body any) (*model.Payment, error) {
	payment, err := Do[model.Payment](ctx, c, Request{
		Method: http.MethodPost,
		Path:   path,
		Body:   body,
	})
	if err != nil {
		return nil, err
	}

	return &payment, nil
}

// CreateDispute submits the evidence of a disputed transaction.
func (c *Client) CreateDispute(ctx context.Context, paymentID, transactionID string, body any) ([]byte, error) {
	data, err := c.DoRaw(ctx, Request{
		Method: http.MethodPost,
		Path:   paymentTransactionPath(paymentID, transactionID, "dispute"),
		Body:   body,
	})
	if err != nil {
		return nil, fmt.Errorf("create dispute for payment %s transaction %s: %w", paymentID, transactionID, err)
	}

	return data, nil
}

// UpdateDispute replaces the evidence of a disputed transaction.
func (c *Client) UpdateDispute(ctx context.Context, paymentID, transactionID string, body any) ([]byte, error) {
	data, err := c.DoRaw(ctx, Request{
		Method: http.MethodPatch,
		Path:   paymentTransactionPath(paymentID, transactionID, "dispute"),
		Body:   body,
	})
	if err != nil {
		return nil, fmt.Errorf("update dispute for payment %s transaction %s: %w", paymentID, transactionID, err)
	}

	return data, nil
}

// CreateFulfillment reports the fulfillment status of a payment.
func (c *Client) CreateFulfillment(ctx context.Context, paymentID string, body any) ([]byte, error) {
	data, err := c.DoRaw(ctx, Request{
		Method: http.MethodPost,
		Path:   "/payments/" + url.PathEscape(paymentID) + "/fulfillments",
		Body:   body,
	})
	if err != nil {
		return nil, fmt.Errorf("create fulfillment for payment %s: %w", paymentID, err)
	}

	return data, nil
}
