package api

import (
	"context"
	"fmt"
	"net/http"
	"net/url"

	"github.com/pvarganov/yuno-cli/internal/model"
)

// Base paths of the marketplace recipient resource. The transfers of a
// recipient live under their own top-level paths, which is why they are spelled
// out separately here.
const (
	recipientPath           = "/recipients"
	onboardingTransfersPath = "/onboardings"
	recipientTransferPath   = "/transfers"
	reverseTransferPath     = "/recipients/onboardings/reverse-transfer"
)

// ListRecipients returns the recipients matching the filters. The endpoint
// pages with `limit` / `offset`; a limit of zero fetches everything.
func (c *Client) ListRecipients(
	ctx context.Context, filters url.Values, limit, pageSize int,
) ([]model.Recipient, error) {
	recipients, err := PaginateAll[model.Recipient](ctx, c, Request{
		Method: http.MethodGet,
		Path:   recipientPath,
		Query:  filters,
	}, NewOffsetPager(pageSize), limit)
	if err != nil {
		return nil, fmt.Errorf("list recipients: %w", err)
	}

	return recipients, nil
}

// GetRecipient retrieves one recipient by id.
func (c *Client) GetRecipient(ctx context.Context, recipientID string) (*model.Recipient, error) {
	recipient, err := Do[model.Recipient](ctx, c, Request{
		Method: http.MethodGet,
		Path:   recipientPath + "/" + url.PathEscape(recipientID),
	})
	if err != nil {
		return nil, fmt.Errorf("get recipient %s: %w", recipientID, err)
	}

	return &recipient, nil
}

// CreateRecipient creates a recipient from an already built JSON body.
func (c *Client) CreateRecipient(ctx context.Context, body any) (*model.Recipient, error) {
	recipient, err := Do[model.Recipient](ctx, c, Request{
		Method: http.MethodPost,
		Path:   recipientPath,
		Body:   body,
	})
	if err != nil {
		return nil, fmt.Errorf("create recipient: %w", err)
	}

	return &recipient, nil
}

// UpdateRecipient patches a recipient.
func (c *Client) UpdateRecipient(ctx context.Context, recipientID string, body any) (*model.Recipient, error) {
	recipient, err := Do[model.Recipient](ctx, c, Request{
		Method: http.MethodPatch,
		Path:   recipientPath + "/" + url.PathEscape(recipientID),
		Body:   body,
	})
	if err != nil {
		return nil, fmt.Errorf("update recipient %s: %w", recipientID, err)
	}

	return &recipient, nil
}

// DeleteRecipient deletes a recipient. Yuno answers with an empty body, so the
// raw response is returned as is.
func (c *Client) DeleteRecipient(ctx context.Context, recipientID string) ([]byte, error) {
	data, err := c.DoRaw(ctx, Request{
		Method: http.MethodDelete,
		Path:   recipientPath + "/" + url.PathEscape(recipientID),
	})
	if err != nil {
		return nil, fmt.Errorf("delete recipient %s: %w", recipientID, err)
	}

	return data, nil
}

// onboardingsPath builds the onboardings collection path of one recipient.
func onboardingsPath(recipientID string) string {
	return recipientPath + "/" + url.PathEscape(recipientID) + "/onboardings"
}

// onboardingPath builds the path of one onboarding of one recipient.
func onboardingPath(recipientID, onboardingID string) string {
	return onboardingsPath(recipientID) + "/" + url.PathEscape(onboardingID)
}

// CreateOnboarding starts an onboarding for a recipient. Yuno answers with the
// whole recipient, onboardings included.
func (c *Client) CreateOnboarding(ctx context.Context, recipientID string, body any) (*model.Recipient, error) {
	recipient, err := Do[model.Recipient](ctx, c, Request{
		Method: http.MethodPost,
		Path:   onboardingsPath(recipientID),
		Body:   body,
	})
	if err != nil {
		return nil, fmt.Errorf("create onboarding for recipient %s: %w", recipientID, err)
	}

	return &recipient, nil
}

// GetOnboarding retrieves one onboarding of a recipient.
func (c *Client) GetOnboarding(
	ctx context.Context, recipientID, onboardingID string,
) (*model.RecipientOnboarding, error) {
	onboarding, err := Do[model.RecipientOnboarding](ctx, c, Request{
		Method: http.MethodGet,
		Path:   onboardingPath(recipientID, onboardingID),
	})
	if err != nil {
		return nil, fmt.Errorf("get onboarding %s: %w", onboardingID, err)
	}

	return &onboarding, nil
}

// UpdateOnboarding patches an onboarding. Yuno answers with the whole recipient.
func (c *Client) UpdateOnboarding(
	ctx context.Context, recipientID, onboardingID string, body any,
) (*model.Recipient, error) {
	recipient, err := Do[model.Recipient](ctx, c, Request{
		Method: http.MethodPatch,
		Path:   onboardingPath(recipientID, onboardingID),
		Body:   body,
	})
	if err != nil {
		return nil, fmt.Errorf("update onboarding %s: %w", onboardingID, err)
	}

	return &recipient, nil
}

// OnboardingAction posts one of the lifecycle actions of an onboarding
// (`continue`, `cancel`, `block`, `unblock`). Only `continue` takes a body; the
// others are posted with none. Every action answers with the whole recipient.
func (c *Client) OnboardingAction(
	ctx context.Context, recipientID, onboardingID, action string, body any,
) (*model.Recipient, error) {
	recipient, err := Do[model.Recipient](ctx, c, Request{
		Method: http.MethodPost,
		Path:   onboardingPath(recipientID, onboardingID) + "/" + action,
		Body:   body,
	})
	if err != nil {
		return nil, fmt.Errorf("%s onboarding %s: %w", action, onboardingID, err)
	}

	return &recipient, nil
}

// CreateOnboardingTransfer moves a recipient from the onboarding it is on today
// onto a new one, transferring the balance with it.
func (c *Client) CreateOnboardingTransfer(
	ctx context.Context, recipientID, onboardingID string,
) (*model.RecipientTransfer, error) {
	transfer, err := Do[model.RecipientTransfer](ctx, c, Request{
		Method: http.MethodPost,
		Path:   onboardingPath(recipientID, onboardingID) + "/transfer",
	})
	if err != nil {
		return nil, fmt.Errorf("transfer onboarding %s: %w", onboardingID, err)
	}

	return &transfer, nil
}

// ReverseOnboardingTransfer undoes an onboarding transfer, moving the recipient
// back to the onboarding it came from.
func (c *Client) ReverseOnboardingTransfer(
	ctx context.Context, transferID string,
) (*model.RecipientTransfer, error) {
	transfer, err := Do[model.RecipientTransfer](ctx, c, Request{
		Method: http.MethodPost,
		Path:   reverseTransferPath + "/" + url.PathEscape(transferID),
	})
	if err != nil {
		return nil, fmt.Errorf("reverse onboarding transfer %s: %w", transferID, err)
	}

	return &transfer, nil
}

// GetRecipientTransfer retrieves one onboarding transfer by id.
func (c *Client) GetRecipientTransfer(ctx context.Context, transferID string) (*model.RecipientTransfer, error) {
	transfer, err := Do[model.RecipientTransfer](ctx, c, Request{
		Method: http.MethodGet,
		Path:   recipientTransferPath + "/" + url.PathEscape(transferID),
	})
	if err != nil {
		return nil, fmt.Errorf("get recipient transfer %s: %w", transferID, err)
	}

	return &transfer, nil
}

// ListRecipientTransfers returns the onboarding transfers of one recipient.
// The endpoint answers with a bare array and does not page.
func (c *Client) ListRecipientTransfers(ctx context.Context, recipientID string) ([]model.RecipientTransfer, error) {
	transfers, err := Do[[]model.RecipientTransfer](ctx, c, Request{
		Method: http.MethodGet,
		Path:   recipientPath + "/" + url.PathEscape(recipientID) + "/transfers",
	})
	if err != nil {
		return nil, fmt.Errorf("list transfers of recipient %s: %w", recipientID, err)
	}

	return transfers, nil
}

// ListOnboardingTransfers returns the transfers of one onboarding. The endpoint
// answers with a bare array and does not page.
func (c *Client) ListOnboardingTransfers(ctx context.Context, onboardingID string) ([]model.RecipientTransfer, error) {
	transfers, err := Do[[]model.RecipientTransfer](ctx, c, Request{
		Method: http.MethodGet,
		Path:   onboardingTransfersPath + "/" + url.PathEscape(onboardingID) + "/transfers",
	})
	if err != nil {
		return nil, fmt.Errorf("list transfers of onboarding %s: %w", onboardingID, err)
	}

	return transfers, nil
}

// ReversePaymentTransfer reverses the split marketplace transfer of one payment
// transaction, fully or partially. The response is a plain acknowledgement, so
// the raw body is returned as is.
func (c *Client) ReversePaymentTransfer(
	ctx context.Context, paymentID, transactionID string, body any,
) ([]byte, error) {
	data, err := c.DoRaw(ctx, Request{
		Method: http.MethodPost,
		Path:   paymentTransactionPath(paymentID, transactionID, "split-marketplace/transfer-reversal"),
		Body:   body,
	})
	if err != nil {
		return nil, fmt.Errorf("reverse transfer of payment %s: %w", paymentID, err)
	}

	return data, nil
}
