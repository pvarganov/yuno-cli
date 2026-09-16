package api

import (
	"context"
	"fmt"
	"net/http"
	"net/url"

	"github.com/pvarganov/yuno-cli/internal/model"
)

// preDebitPath is the base path of the pre-debit notification resource.
const preDebitPath = "/predebit-notify"

// CreatePreDebitNotification creates a pre-debit notification.
func (c *Client) CreatePreDebitNotification(ctx context.Context, body any) (*model.PreDebitNotification, error) {
	notification, err := Do[model.PreDebitNotification](ctx, c, Request{
		Method: http.MethodPost,
		Path:   preDebitPath,
		Body:   body,
	})
	if err != nil {
		return nil, fmt.Errorf("create pre-debit notification: %w", err)
	}

	return &notification, nil
}

// GetPreDebitNotificationByReference looks a pre-debit notification up by
// merchant reference. Yuno has no unfiltered list on this resource.
func (c *Client) GetPreDebitNotificationByReference(
	ctx context.Context, merchantReference string,
) (*model.PreDebitNotification, error) {
	notification, err := Do[model.PreDebitNotification](ctx, c, Request{
		Method: http.MethodGet,
		Path:   preDebitPath,
		Query:  url.Values{"merchant_reference": []string{merchantReference}},
	})
	if err != nil {
		return nil, fmt.Errorf("get pre-debit notification by merchant reference %s: %w", merchantReference, err)
	}

	return &notification, nil
}

// GetPreDebitNotification retrieves one pre-debit notification by id.
func (c *Client) GetPreDebitNotification(
	ctx context.Context, notificationID string,
) (*model.PreDebitNotification, error) {
	notification, err := Do[model.PreDebitNotification](ctx, c, Request{
		Method: http.MethodGet,
		Path:   preDebitPath + "/" + url.PathEscape(notificationID),
	})
	if err != nil {
		return nil, fmt.Errorf("get pre-debit notification %s: %w", notificationID, err)
	}

	return &notification, nil
}
