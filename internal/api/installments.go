package api

import (
	"context"
	"fmt"
	"net/http"
	"net/url"

	"github.com/pvarganov/yuno-cli/internal/model"
)

// installmentPlanPath is the base path of the installments plan resource.
const installmentPlanPath = "/installments-plans"

// ListInstallmentPlans returns the installments plans matching the filters.
// The endpoint answers with a bare array and does not page.
func (c *Client) ListInstallmentPlans(ctx context.Context, filters url.Values) ([]model.InstallmentPlan, error) {
	data, err := c.DoRaw(ctx, Request{
		Method: http.MethodGet,
		Path:   installmentPlanPath,
		Query:  filters,
	})
	if err != nil {
		return nil, fmt.Errorf("list installment plans: %w", err)
	}

	plans, err := model.DecodeInstallmentPlans(data)
	if err != nil {
		return nil, fmt.Errorf("list installment plans: %w", err)
	}

	return plans, nil
}

// GetInstallmentPlan retrieves one installments plan by its code. Yuno answers
// with an array on this endpoint, so every plan it returns is passed through.
func (c *Client) GetInstallmentPlan(ctx context.Context, code string) ([]model.InstallmentPlan, error) {
	data, err := c.DoRaw(ctx, Request{
		Method: http.MethodGet,
		Path:   installmentPlanPath + "/" + url.PathEscape(code),
	})
	if err != nil {
		return nil, fmt.Errorf("get installment plan %s: %w", code, err)
	}

	plans, err := model.DecodeInstallmentPlans(data)
	if err != nil {
		return nil, fmt.Errorf("get installment plan %s: %w", code, err)
	}

	return plans, nil
}

// CreateInstallmentPlan creates an installments plan from an already built body.
func (c *Client) CreateInstallmentPlan(ctx context.Context, body any) (*model.InstallmentPlan, error) {
	plan, err := Do[model.InstallmentPlan](ctx, c, Request{
		Method: http.MethodPost,
		Path:   installmentPlanPath,
		Body:   body,
	})
	if err != nil {
		return nil, fmt.Errorf("create installment plan: %w", err)
	}

	return &plan, nil
}

// UpdateInstallmentPlan patches an installments plan.
func (c *Client) UpdateInstallmentPlan(ctx context.Context, code string, body any) (*model.InstallmentPlan, error) {
	plan, err := Do[model.InstallmentPlan](ctx, c, Request{
		Method: http.MethodPatch,
		Path:   installmentPlanPath + "/" + url.PathEscape(code),
		Body:   body,
	})
	if err != nil {
		return nil, fmt.Errorf("update installment plan %s: %w", code, err)
	}

	return &plan, nil
}

// DeleteInstallmentPlan deletes an installments plan. Yuno answers with an
// empty body, so the raw response is returned as is.
func (c *Client) DeleteInstallmentPlan(ctx context.Context, code string) ([]byte, error) {
	data, err := c.DoRaw(ctx, Request{
		Method: http.MethodDelete,
		Path:   installmentPlanPath + "/" + url.PathEscape(code),
	})
	if err != nil {
		return nil, fmt.Errorf("delete installment plan %s: %w", code, err)
	}

	return data, nil
}
