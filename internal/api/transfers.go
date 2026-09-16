package api

import (
	"context"
	"fmt"
	"net/http"
	"net/url"

	"github.com/pvarganov/yuno-cli/internal/model"
)

// transferPath is the base path of the standalone split-marketplace transfers.
const transferPath = "/split-marketplace/transfers"

// CreateTransfer creates a standalone transfer to a recipient.
func (c *Client) CreateTransfer(ctx context.Context, body any) (*model.Transfer, error) {
	transfer, err := Do[model.Transfer](ctx, c, Request{
		Method: http.MethodPost,
		Path:   transferPath,
		Body:   body,
	})
	if err != nil {
		return nil, fmt.Errorf("create transfer: %w", err)
	}

	return &transfer, nil
}

// GetTransfer retrieves one standalone transfer by id.
func (c *Client) GetTransfer(ctx context.Context, transferID string) (*model.Transfer, error) {
	transfer, err := Do[model.Transfer](ctx, c, Request{
		Method: http.MethodGet,
		Path:   transferPath + "/" + url.PathEscape(transferID),
	})
	if err != nil {
		return nil, fmt.Errorf("get transfer %s: %w", transferID, err)
	}

	return &transfer, nil
}

// ReverseTransfer reverses a standalone transfer, fully or partially.
func (c *Client) ReverseTransfer(ctx context.Context, transferID string, body any) (*model.Transfer, error) {
	transfer, err := Do[model.Transfer](ctx, c, Request{
		Method: http.MethodPost,
		Path:   transferPath + "/" + url.PathEscape(transferID) + "/reverse",
		Body:   body,
	})
	if err != nil {
		return nil, fmt.Errorf("reverse transfer %s: %w", transferID, err)
	}

	return &transfer, nil
}
