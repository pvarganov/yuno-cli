package api

import (
	"context"
	"fmt"
	"net/http"
	"net/url"

	"github.com/pvarganov/yuno-cli/internal/model"
)

// Banking connectivity paths.
const (
	bankingEntityPath   = "/banking/entities"
	bankingAccountPath  = "/banking/accounts"
	bankingTransferPath = "/banking/transfers"
)

// CreateBankingEntity registers a legal entity for banking connectivity.
func (c *Client) CreateBankingEntity(ctx context.Context, body any) (*model.BankingEntity, error) {
	entity, err := Do[model.BankingEntity](ctx, c, Request{
		Method: http.MethodPost,
		Path:   bankingEntityPath,
		Body:   body,
	})
	if err != nil {
		return nil, fmt.Errorf("create banking entity: %w", err)
	}

	return &entity, nil
}

// GetBankingEntity retrieves one entity.
func (c *Client) GetBankingEntity(ctx context.Context, entityID string) (*model.BankingEntity, error) {
	entity, err := Do[model.BankingEntity](ctx, c, Request{
		Method: http.MethodGet,
		Path:   bankingEntityPath + "/" + url.PathEscape(entityID),
	})
	if err != nil {
		return nil, fmt.Errorf("get banking entity %s: %w", entityID, err)
	}

	return &entity, nil
}

// UpdateBankingEntity patches an entity.
func (c *Client) UpdateBankingEntity(ctx context.Context, entityID string, body any) (*model.BankingEntity, error) {
	entity, err := Do[model.BankingEntity](ctx, c, Request{
		Method: http.MethodPatch,
		Path:   bankingEntityPath + "/" + url.PathEscape(entityID),
		Body:   body,
	})
	if err != nil {
		return nil, fmt.Errorf("update banking entity %s: %w", entityID, err)
	}

	return &entity, nil
}

// bankingOnboardingPath builds the onboarding collection path of an entity.
func bankingOnboardingPath(entityID string) string {
	return bankingEntityPath + "/" + url.PathEscape(entityID) + "/onboardings"
}

// CreateBankingOnboarding starts the onboarding of an entity with a provider.
func (c *Client) CreateBankingOnboarding(
	ctx context.Context, entityID string, body any,
) (*model.BankingOnboarding, error) {
	onboarding, err := Do[model.BankingOnboarding](ctx, c, Request{
		Method: http.MethodPost,
		Path:   bankingOnboardingPath(entityID),
		Body:   body,
	})
	if err != nil {
		return nil, fmt.Errorf("create onboarding of banking entity %s: %w", entityID, err)
	}

	return &onboarding, nil
}

// GetBankingOnboarding retrieves the status of one onboarding.
func (c *Client) GetBankingOnboarding(
	ctx context.Context, entityID, onboardingID string,
) (*model.BankingOnboarding, error) {
	onboarding, err := Do[model.BankingOnboarding](ctx, c, Request{
		Method: http.MethodGet,
		Path:   bankingOnboardingPath(entityID) + "/" + url.PathEscape(onboardingID),
	})
	if err != nil {
		return nil, fmt.Errorf("get banking onboarding %s: %w", onboardingID, err)
	}

	return &onboarding, nil
}

// UpdateBankingOnboarding patches an onboarding, usually to satisfy a pending
// requirement.
func (c *Client) UpdateBankingOnboarding(
	ctx context.Context, entityID, onboardingID string, body any,
) (*model.BankingOnboarding, error) {
	onboarding, err := Do[model.BankingOnboarding](ctx, c, Request{
		Method: http.MethodPatch,
		Path:   bankingOnboardingPath(entityID) + "/" + url.PathEscape(onboardingID),
		Body:   body,
	})
	if err != nil {
		return nil, fmt.Errorf("update banking onboarding %s: %w", onboardingID, err)
	}

	return &onboarding, nil
}

// CancelBankingOnboarding stops an onboarding that has not completed yet.
func (c *Client) CancelBankingOnboarding(
	ctx context.Context, entityID, onboardingID string,
) (*model.BankingOnboarding, error) {
	onboarding, err := Do[model.BankingOnboarding](ctx, c, Request{
		Method: http.MethodPost,
		Path:   bankingOnboardingPath(entityID) + "/" + url.PathEscape(onboardingID) + "/cancel",
	})
	if err != nil {
		return nil, fmt.Errorf("cancel banking onboarding %s: %w", onboardingID, err)
	}

	return &onboarding, nil
}

// CreateBankingAccount opens a bank account for a completed onboarding.
func (c *Client) CreateBankingAccount(ctx context.Context, body any) (*model.BankingAccount, error) {
	account, err := Do[model.BankingAccount](ctx, c, Request{
		Method: http.MethodPost,
		Path:   bankingAccountPath,
		Body:   body,
	})
	if err != nil {
		return nil, fmt.Errorf("create banking account: %w", err)
	}

	return &account, nil
}

// GetBankingAccount retrieves one bank account with its balance.
func (c *Client) GetBankingAccount(ctx context.Context, accountID string) (*model.BankingAccount, error) {
	account, err := Do[model.BankingAccount](ctx, c, Request{
		Method: http.MethodGet,
		Path:   bankingAccountPath + "/" + url.PathEscape(accountID),
	})
	if err != nil {
		return nil, fmt.Errorf("get banking account %s: %w", accountID, err)
	}

	return &account, nil
}

// UpdateBankingAccount patches a bank account.
func (c *Client) UpdateBankingAccount(
	ctx context.Context, accountID string, body any,
) (*model.BankingAccount, error) {
	account, err := Do[model.BankingAccount](ctx, c, Request{
		Method: http.MethodPatch,
		Path:   bankingAccountPath + "/" + url.PathEscape(accountID),
		Body:   body,
	})
	if err != nil {
		return nil, fmt.Errorf("update banking account %s: %w", accountID, err)
	}

	return &account, nil
}

// CloseBankingAccount closes a bank account. Yuno exposes it as a DELETE but
// answers with the closed account, not an empty body.
func (c *Client) CloseBankingAccount(ctx context.Context, accountID string) (*model.BankingAccount, error) {
	account, err := Do[model.BankingAccount](ctx, c, Request{
		Method: http.MethodDelete,
		Path:   bankingAccountPath + "/" + url.PathEscape(accountID),
	})
	if err != nil {
		return nil, fmt.Errorf("close banking account %s: %w", accountID, err)
	}

	return &account, nil
}

// CreateBankingTransfer initiates a transfer between banking accounts.
func (c *Client) CreateBankingTransfer(ctx context.Context, body any) (*model.BankingTransfer, error) {
	transfer, err := Do[model.BankingTransfer](ctx, c, Request{
		Method: http.MethodPost,
		Path:   bankingTransferPath,
		Body:   body,
	})
	if err != nil {
		return nil, fmt.Errorf("create banking transfer: %w", err)
	}

	return &transfer, nil
}

// bankingAccountTransferPath builds the transfer path of one account. The read
// side of a transfer hangs off the source account, not off /banking/transfers.
func bankingAccountTransferPath(accountID, transferID string) string {
	return bankingAccountPath + "/" + url.PathEscape(accountID) + "/transfers/" + url.PathEscape(transferID)
}

// GetBankingTransfer retrieves the status of one transfer.
func (c *Client) GetBankingTransfer(
	ctx context.Context, accountID, transferID string,
) (*model.BankingTransfer, error) {
	transfer, err := Do[model.BankingTransfer](ctx, c, Request{
		Method: http.MethodGet,
		Path:   bankingAccountTransferPath(accountID, transferID),
	})
	if err != nil {
		return nil, fmt.Errorf("get banking transfer %s: %w", transferID, err)
	}

	return &transfer, nil
}

// CancelBankingTransfer cancels a transfer that has not settled yet.
func (c *Client) CancelBankingTransfer(
	ctx context.Context, accountID, transferID string,
) (*model.BankingTransfer, error) {
	transfer, err := Do[model.BankingTransfer](ctx, c, Request{
		Method: http.MethodPost,
		Path:   bankingAccountTransferPath(accountID, transferID) + "/cancel",
	})
	if err != nil {
		return nil, fmt.Errorf("cancel banking transfer %s: %w", transferID, err)
	}

	return &transfer, nil
}
