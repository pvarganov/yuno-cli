package api

import (
	"context"
	"fmt"
	"net/http"
)

// reportingTransactionsPath ingests transactions processed outside Yuno.
const reportingTransactionsPath = "/reporting/transactions"

// ReportTransactions ingests off-Yuno transactions. Yuno answers 202 with a
// per-event result, which the caller prints verbatim: one invalid event never
// rejects the whole batch, so the answer is worth reading in full.
func (c *Client) ReportTransactions(ctx context.Context, body any) ([]byte, error) {
	data, err := c.DoRaw(ctx, Request{
		Method: http.MethodPost,
		Path:   reportingTransactionsPath,
		Body:   body,
	})
	if err != nil {
		return nil, fmt.Errorf("report transactions: %w", err)
	}

	return data, nil
}
