package api

import (
	"context"
	"fmt"
	"net/http"
	"net/url"

	"github.com/pvarganov/yuno-cli/internal/model"
)

// Base paths of the organization resources.
const (
	orgAccountPath      = "/organizations/accounts"
	orgAccountGroupPath = "/organizations/account-groups"
	orgUserPath         = "/organizations/users"
	orgRolePath         = "/organizations/roles"
	orgCatalogPath      = "/organizations/permissions-catalog"
	orgAuthenticatePath = "/organizations/authenticate"
)

// Permission sub-resources of a user. They are spelled out here because every
// permission method addresses one of the two.
const (
	AccountPermissions      = "account-permissions"
	AccountGroupPermissions = "account-group-permissions"
)

// ListOrgAccounts returns the accounts of the organization. The endpoint pages
// with `page` / `page_size`, counting from one.
func (c *Client) ListOrgAccounts(ctx context.Context, limit, pageSize int) ([]model.OrgAccount, error) {
	accounts, err := PaginateAll[model.OrgAccount](ctx, c, Request{
		Method: http.MethodGet,
		Path:   orgAccountPath,
	}, NewPageNumberPager(pageSize), limit)
	if err != nil {
		return nil, fmt.Errorf("list organization accounts: %w", err)
	}

	return accounts, nil
}

// GetOrgAccount retrieves one account by id.
func (c *Client) GetOrgAccount(ctx context.Context, accountID string) (*model.OrgAccount, error) {
	account, err := Do[model.OrgAccount](ctx, c, Request{
		Method: http.MethodGet,
		Path:   orgAccountPath + "/" + url.PathEscape(accountID),
	})
	if err != nil {
		return nil, fmt.Errorf("get organization account %s: %w", accountID, err)
	}

	return &account, nil
}

// UpdateOrgAccount patches one account.
func (c *Client) UpdateOrgAccount(ctx context.Context, accountID string, body any) (*model.OrgAccount, error) {
	account, err := Do[model.OrgAccount](ctx, c, Request{
		Method: http.MethodPatch,
		Path:   orgAccountPath + "/" + url.PathEscape(accountID),
		Body:   body,
	})
	if err != nil {
		return nil, fmt.Errorf("update organization account %s: %w", accountID, err)
	}

	return &account, nil
}

// DeleteOrgAccount deletes one account. Yuno answers with a short
// acknowledgement, so the raw response is returned as is.
func (c *Client) DeleteOrgAccount(ctx context.Context, accountID string) ([]byte, error) {
	data, err := c.DoRaw(ctx, Request{
		Method: http.MethodDelete,
		Path:   orgAccountPath + "/" + url.PathEscape(accountID),
	})
	if err != nil {
		return nil, fmt.Errorf("delete organization account %s: %w", accountID, err)
	}

	return data, nil
}

// accountGroupAccountsPath builds the accounts collection path of one group.
func accountGroupAccountsPath(groupID string) string {
	return orgAccountGroupPath + "/" + url.PathEscape(groupID) + "/accounts"
}

// CreateOrgAccount creates an account inside an account group: Yuno has no
// top-level account creation endpoint.
func (c *Client) CreateOrgAccount(ctx context.Context, groupID string, body any) (*model.OrgAccount, error) {
	account, err := Do[model.OrgAccount](ctx, c, Request{
		Method: http.MethodPost,
		Path:   accountGroupAccountsPath(groupID),
		Body:   body,
	})
	if err != nil {
		return nil, fmt.Errorf("create account in group %s: %w", groupID, err)
	}

	return &account, nil
}

// ListAccountGroupAccounts returns the accounts of one account group.
func (c *Client) ListAccountGroupAccounts(
	ctx context.Context, groupID string, limit, pageSize int,
) ([]model.OrgAccount, error) {
	accounts, err := PaginateAll[model.OrgAccount](ctx, c, Request{
		Method: http.MethodGet,
		Path:   accountGroupAccountsPath(groupID),
	}, NewPageNumberPager(pageSize), limit)
	if err != nil {
		return nil, fmt.Errorf("list accounts of group %s: %w", groupID, err)
	}

	return accounts, nil
}

// ListAccountGroups returns the account groups of the organization.
func (c *Client) ListAccountGroups(ctx context.Context, limit, pageSize int) ([]model.OrgAccountGroup, error) {
	groups, err := PaginateAll[model.OrgAccountGroup](ctx, c, Request{
		Method: http.MethodGet,
		Path:   orgAccountGroupPath,
	}, NewPageNumberPager(pageSize), limit)
	if err != nil {
		return nil, fmt.Errorf("list account groups: %w", err)
	}

	return groups, nil
}

// GetAccountGroup retrieves one account group by id.
func (c *Client) GetAccountGroup(ctx context.Context, groupID string) (*model.OrgAccountGroup, error) {
	group, err := Do[model.OrgAccountGroup](ctx, c, Request{
		Method: http.MethodGet,
		Path:   orgAccountGroupPath + "/" + url.PathEscape(groupID),
	})
	if err != nil {
		return nil, fmt.Errorf("get account group %s: %w", groupID, err)
	}

	return &group, nil
}

// CreateAccountGroup creates an account group.
func (c *Client) CreateAccountGroup(ctx context.Context, body any) (*model.OrgAccountGroup, error) {
	group, err := Do[model.OrgAccountGroup](ctx, c, Request{
		Method: http.MethodPost,
		Path:   orgAccountGroupPath,
		Body:   body,
	})
	if err != nil {
		return nil, fmt.Errorf("create account group: %w", err)
	}

	return &group, nil
}

// UpdateAccountGroup patches an account group.
func (c *Client) UpdateAccountGroup(ctx context.Context, groupID string, body any) (*model.OrgAccountGroup, error) {
	group, err := Do[model.OrgAccountGroup](ctx, c, Request{
		Method: http.MethodPatch,
		Path:   orgAccountGroupPath + "/" + url.PathEscape(groupID),
		Body:   body,
	})
	if err != nil {
		return nil, fmt.Errorf("update account group %s: %w", groupID, err)
	}

	return &group, nil
}

// DeleteAccountGroup deletes an account group.
func (c *Client) DeleteAccountGroup(ctx context.Context, groupID string) ([]byte, error) {
	data, err := c.DoRaw(ctx, Request{
		Method: http.MethodDelete,
		Path:   orgAccountGroupPath + "/" + url.PathEscape(groupID),
	})
	if err != nil {
		return nil, fmt.Errorf("delete account group %s: %w", groupID, err)
	}

	return data, nil
}

// ListOrgUsers returns the users of the organization matching the filters.
func (c *Client) ListOrgUsers(
	ctx context.Context, filters url.Values, limit, pageSize int,
) ([]model.OrgUser, error) {
	users, err := PaginateAll[model.OrgUser](ctx, c, Request{
		Method: http.MethodGet,
		Path:   orgUserPath,
		Query:  filters,
	}, NewPageNumberPager(pageSize), limit)
	if err != nil {
		return nil, fmt.Errorf("list organization users: %w", err)
	}

	return users, nil
}

// GetOrgUser retrieves one user by id.
func (c *Client) GetOrgUser(ctx context.Context, userID string) (*model.OrgUser, error) {
	user, err := Do[model.OrgUser](ctx, c, Request{
		Method: http.MethodGet,
		Path:   orgUserPath + "/" + url.PathEscape(userID),
	})
	if err != nil {
		return nil, fmt.Errorf("get organization user %s: %w", userID, err)
	}

	return &user, nil
}

// CreateOrgUser invites a user into the organization.
func (c *Client) CreateOrgUser(ctx context.Context, body any) (*model.OrgUser, error) {
	user, err := Do[model.OrgUser](ctx, c, Request{
		Method: http.MethodPost,
		Path:   orgUserPath,
		Body:   body,
	})
	if err != nil {
		return nil, fmt.Errorf("create organization user: %w", err)
	}

	return &user, nil
}

// UpdateOrgUser patches a user.
func (c *Client) UpdateOrgUser(ctx context.Context, userID string, body any) (*model.OrgUser, error) {
	user, err := Do[model.OrgUser](ctx, c, Request{
		Method: http.MethodPatch,
		Path:   orgUserPath + "/" + url.PathEscape(userID),
		Body:   body,
	})
	if err != nil {
		return nil, fmt.Errorf("update organization user %s: %w", userID, err)
	}

	return &user, nil
}

// DeleteOrgUser removes a user from the organization.
func (c *Client) DeleteOrgUser(ctx context.Context, userID string) ([]byte, error) {
	data, err := c.DoRaw(ctx, Request{
		Method: http.MethodDelete,
		Path:   orgUserPath + "/" + url.PathEscape(userID),
	})
	if err != nil {
		return nil, fmt.Errorf("delete organization user %s: %w", userID, err)
	}

	return data, nil
}

// AuthenticateOrgUser exchanges a user id for a whitelabel access token.
func (c *Client) AuthenticateOrgUser(ctx context.Context, body any) (*model.OrgAuthToken, error) {
	token, err := Do[model.OrgAuthToken](ctx, c, Request{
		Method: http.MethodPost,
		Path:   orgAuthenticatePath,
		Body:   body,
	})
	if err != nil {
		return nil, fmt.Errorf("authenticate organization user: %w", err)
	}

	return &token, nil
}

// ListOrgRoles returns the roles of the organization. The endpoint answers with
// a bare array, so there is nothing to paginate.
func (c *Client) ListOrgRoles(ctx context.Context) ([]model.OrgRole, error) {
	roles, err := Do[[]model.OrgRole](ctx, c, Request{
		Method: http.MethodGet,
		Path:   orgRolePath,
	})
	if err != nil {
		return nil, fmt.Errorf("list organization roles: %w", err)
	}

	return roles, nil
}

// CreateOrgRole creates a role.
func (c *Client) CreateOrgRole(ctx context.Context, body any) (*model.OrgRole, error) {
	role, err := Do[model.OrgRole](ctx, c, Request{
		Method: http.MethodPost,
		Path:   orgRolePath,
		Body:   body,
	})
	if err != nil {
		return nil, fmt.Errorf("create organization role: %w", err)
	}

	return &role, nil
}

// UpdateOrgRole patches a role.
func (c *Client) UpdateOrgRole(ctx context.Context, roleID string, body any) (*model.OrgRole, error) {
	role, err := Do[model.OrgRole](ctx, c, Request{
		Method: http.MethodPatch,
		Path:   orgRolePath + "/" + url.PathEscape(roleID),
		Body:   body,
	})
	if err != nil {
		return nil, fmt.Errorf("update organization role %s: %w", roleID, err)
	}

	return &role, nil
}

// DeleteOrgRole deletes a role.
func (c *Client) DeleteOrgRole(ctx context.Context, roleID string) ([]byte, error) {
	data, err := c.DoRaw(ctx, Request{
		Method: http.MethodDelete,
		Path:   orgRolePath + "/" + url.PathEscape(roleID),
	})
	if err != nil {
		return nil, fmt.Errorf("delete organization role %s: %w", roleID, err)
	}

	return data, nil
}

// ListPermissionsCatalog returns every permission a role may grant, grouped by
// section. The endpoint answers with a bare array.
func (c *Client) ListPermissionsCatalog(ctx context.Context) ([]model.OrgPermissionGroup, error) {
	groups, err := Do[[]model.OrgPermissionGroup](ctx, c, Request{
		Method: http.MethodGet,
		Path:   orgCatalogPath,
	})
	if err != nil {
		return nil, fmt.Errorf("list permissions catalog: %w", err)
	}

	return groups, nil
}

// userPermissionsPath builds the permissions path of one user, for either of
// the two permission kinds.
func userPermissionsPath(userID, kind string) string {
	return orgUserPath + "/" + url.PathEscape(userID) + "/" + kind
}

// GetUserAccountPermissions returns the roles a user holds on single accounts.
func (c *Client) GetUserAccountPermissions(
	ctx context.Context, userID string,
) (*model.OrgAccountPermissions, error) {
	permissions, err := Do[model.OrgAccountPermissions](ctx, c, Request{
		Method: http.MethodGet,
		Path:   userPermissionsPath(userID, AccountPermissions),
	})
	if err != nil {
		return nil, fmt.Errorf("get account permissions of user %s: %w", userID, err)
	}

	return &permissions, nil
}

// ReplaceUserAccountPermissions overwrites the account permissions of a user.
// Yuno answers with a bare status, so the raw response is returned as is.
func (c *Client) ReplaceUserAccountPermissions(ctx context.Context, userID string, body any) ([]byte, error) {
	data, err := c.DoRaw(ctx, Request{
		Method: http.MethodPut,
		Path:   userPermissionsPath(userID, AccountPermissions),
		Body:   body,
	})
	if err != nil {
		return nil, fmt.Errorf("replace account permissions of user %s: %w", userID, err)
	}

	return data, nil
}

// UpdateUserAccountPermissions adds to the account permissions of a user and
// answers with the resulting set.
func (c *Client) UpdateUserAccountPermissions(
	ctx context.Context, userID string, body any,
) (*model.OrgAccountPermissions, error) {
	permissions, err := Do[model.OrgAccountPermissions](ctx, c, Request{
		Method: http.MethodPatch,
		Path:   userPermissionsPath(userID, AccountPermissions),
		Body:   body,
	})
	if err != nil {
		return nil, fmt.Errorf("update account permissions of user %s: %w", userID, err)
	}

	return &permissions, nil
}

// DeleteUserAccountPermission revokes the role a user holds on one account.
func (c *Client) DeleteUserAccountPermission(ctx context.Context, userID, accountID string) ([]byte, error) {
	data, err := c.DoRaw(ctx, Request{
		Method: http.MethodDelete,
		Path:   userPermissionsPath(userID, AccountPermissions) + "/" + url.PathEscape(accountID),
	})
	if err != nil {
		return nil, fmt.Errorf("delete account permission of user %s on account %s: %w", userID, accountID, err)
	}

	return data, nil
}

// GetUserAccountGroupPermissions returns the roles a user holds on groups.
func (c *Client) GetUserAccountGroupPermissions(
	ctx context.Context, userID string,
) (*model.OrgAccountGroupPermissions, error) {
	permissions, err := Do[model.OrgAccountGroupPermissions](ctx, c, Request{
		Method: http.MethodGet,
		Path:   userPermissionsPath(userID, AccountGroupPermissions),
	})
	if err != nil {
		return nil, fmt.Errorf("get account group permissions of user %s: %w", userID, err)
	}

	return &permissions, nil
}

// ReplaceUserAccountGroupPermissions overwrites the group permissions of a user.
func (c *Client) ReplaceUserAccountGroupPermissions(ctx context.Context, userID string, body any) ([]byte, error) {
	data, err := c.DoRaw(ctx, Request{
		Method: http.MethodPut,
		Path:   userPermissionsPath(userID, AccountGroupPermissions),
		Body:   body,
	})
	if err != nil {
		return nil, fmt.Errorf("replace account group permissions of user %s: %w", userID, err)
	}

	return data, nil
}

// UpdateUserAccountGroupPermissions adds to the group permissions of a user and
// answers with the resulting set.
func (c *Client) UpdateUserAccountGroupPermissions(
	ctx context.Context, userID string, body any,
) (*model.OrgAccountGroupPermissions, error) {
	permissions, err := Do[model.OrgAccountGroupPermissions](ctx, c, Request{
		Method: http.MethodPatch,
		Path:   userPermissionsPath(userID, AccountGroupPermissions),
		Body:   body,
	})
	if err != nil {
		return nil, fmt.Errorf("update account group permissions of user %s: %w", userID, err)
	}

	return &permissions, nil
}

// DeleteUserAccountGroupPermission revokes the role a user holds on one group.
func (c *Client) DeleteUserAccountGroupPermission(ctx context.Context, userID, groupID string) ([]byte, error) {
	data, err := c.DoRaw(ctx, Request{
		Method: http.MethodDelete,
		Path:   userPermissionsPath(userID, AccountGroupPermissions) + "/" + url.PathEscape(groupID),
	})
	if err != nil {
		return nil, fmt.Errorf("delete account group permission of user %s on group %s: %w", userID, groupID, err)
	}

	return data, nil
}
