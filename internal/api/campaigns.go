package api

import (
	"context"
	"fmt"
	"net/http"
	"net/url"

	"github.com/pvarganov/yuno-cli/internal/model"
)

// campaignPath is the base path of the communications campaign resource.
const campaignPath = "/campaigns"

// CreateCampaign creates a communications campaign from an already built body.
func (c *Client) CreateCampaign(ctx context.Context, body any) (*model.Campaign, error) {
	campaign, err := Do[model.Campaign](ctx, c, Request{
		Method: http.MethodPost,
		Path:   campaignPath,
		Body:   body,
	})
	if err != nil {
		return nil, fmt.Errorf("create campaign: %w", err)
	}

	return &campaign, nil
}

// ListCampaigns walks `GET /campaigns`, which pages with `limit` and `offset`.
func (c *Client) ListCampaigns(
	ctx context.Context, query url.Values, limit, pageSize int,
) ([]model.Campaign, error) {
	campaigns, err := PaginateAll[model.Campaign](ctx, c, Request{
		Method: http.MethodGet,
		Path:   campaignPath,
		Query:  query,
	}, NewOffsetPager(pageSize), limit)
	if err != nil {
		return nil, fmt.Errorf("list campaigns: %w", err)
	}

	return campaigns, nil
}

// GetCampaign retrieves one campaign, its rules included.
func (c *Client) GetCampaign(ctx context.Context, campaignID string) (*model.Campaign, error) {
	campaign, err := Do[model.Campaign](ctx, c, Request{
		Method: http.MethodGet,
		Path:   campaignPath + "/" + url.PathEscape(campaignID),
	})
	if err != nil {
		return nil, fmt.Errorf("get campaign %s: %w", campaignID, err)
	}

	return &campaign, nil
}

// UpdateCampaign changes the status of a campaign.
func (c *Client) UpdateCampaign(ctx context.Context, campaignID string, body any) (*model.Campaign, error) {
	campaign, err := Do[model.Campaign](ctx, c, Request{
		Method: http.MethodPatch,
		Path:   campaignPath + "/" + url.PathEscape(campaignID),
		Body:   body,
	})
	if err != nil {
		return nil, fmt.Errorf("update campaign %s: %w", campaignID, err)
	}

	return &campaign, nil
}

// campaignRulesPath is the rules collection of one campaign.
func campaignRulesPath(campaignID string) string {
	return campaignPath + "/" + url.PathEscape(campaignID) + "/rules"
}

// CreateCampaignRules adds targeting rules to a campaign. The endpoint takes a
// `rules` array and answers with every rule it created.
func (c *Client) CreateCampaignRules(ctx context.Context, campaignID string, body any) ([]model.CampaignRule, error) {
	created, err := Do[model.CampaignRuleList](ctx, c, Request{
		Method: http.MethodPost,
		Path:   campaignRulesPath(campaignID),
		Body:   body,
	})
	if err != nil {
		return nil, fmt.Errorf("create rules of campaign %s: %w", campaignID, err)
	}

	return created.Data, nil
}

// GetCampaignRule retrieves one rule of a campaign.
func (c *Client) GetCampaignRule(ctx context.Context, campaignID, ruleID string) (*model.CampaignRule, error) {
	rule, err := Do[model.CampaignRule](ctx, c, Request{
		Method: http.MethodGet,
		Path:   campaignRulesPath(campaignID) + "/" + url.PathEscape(ruleID),
	})
	if err != nil {
		return nil, fmt.Errorf("get rule %s of campaign %s: %w", ruleID, campaignID, err)
	}

	return &rule, nil
}

// UpdateCampaignRule changes the definition of one rule of a campaign.
func (c *Client) UpdateCampaignRule(
	ctx context.Context, campaignID, ruleID string, body any,
) (*model.CampaignRule, error) {
	rule, err := Do[model.CampaignRule](ctx, c, Request{
		Method: http.MethodPatch,
		Path:   campaignRulesPath(campaignID) + "/" + url.PathEscape(ruleID),
		Body:   body,
	})
	if err != nil {
		return nil, fmt.Errorf("update rule %s of campaign %s: %w", ruleID, campaignID, err)
	}

	return &rule, nil
}

// UpdateCampaignRuleStatus enables or disables one rule of a campaign without
// touching its definition.
func (c *Client) UpdateCampaignRuleStatus(
	ctx context.Context, campaignID, ruleID string, body any,
) (*model.CampaignRule, error) {
	rule, err := Do[model.CampaignRule](ctx, c, Request{
		Method: http.MethodPatch,
		Path:   campaignRulesPath(campaignID) + "/" + url.PathEscape(ruleID) + "/status",
		Body:   body,
	})
	if err != nil {
		return nil, fmt.Errorf("update status of rule %s of campaign %s: %w", ruleID, campaignID, err)
	}

	return &rule, nil
}
