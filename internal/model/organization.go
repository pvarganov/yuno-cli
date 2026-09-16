package model

import (
	"encoding/json"
	"strconv"
)

// OrgAccount is one account of the organization. The untouched response stays
// in Raw so `--json` never loses a field the typed struct does not know.
type OrgAccount struct {
	ID             string `json:"id,omitempty"`
	Name           string `json:"name,omitempty"`
	AccountGroupID string `json:"account_group_id,omitempty"`
	CreatedAt      string `json:"created_at,omitempty"`
	UpdatedAt      string `json:"updated_at,omitempty"`

	// Raw is the verbatim response body, set when the account was decoded from one.
	Raw json.RawMessage `json:"-"`
}

// UnmarshalJSON decodes the typed fields and keeps the original body.
func (a *OrgAccount) UnmarshalJSON(data []byte) error {
	type alias OrgAccount

	var decoded alias
	if err := json.Unmarshal(data, &decoded); err != nil {
		return err
	}

	*a = OrgAccount(decoded)
	a.Raw = append(json.RawMessage(nil), data...)

	return nil
}

// MarshalJSON replays the original response when there is one.
//
//nolint:gocritic // hugeParam: a value receiver is required by json.Marshaler here
func (a OrgAccount) MarshalJSON() ([]byte, error) {
	if len(a.Raw) > 0 {
		return a.Raw, nil
	}

	type alias OrgAccount

	return json.Marshal(alias(a))
}

// OrgAccountView is the flat table row of an account.
type OrgAccountView struct {
	ID             string `json:"id"`
	Name           string `json:"name"`
	AccountGroupID string `json:"account_group_id"`
	CreatedAt      string `json:"created_at"`
}

// View flattens an account into a table row.
func (a *OrgAccount) View() OrgAccountView {
	return OrgAccountView{
		ID:             a.ID,
		Name:           a.Name,
		AccountGroupID: a.AccountGroupID,
		CreatedAt:      a.CreatedAt,
	}
}

// OrgAccountViews flattens a list of accounts.
func OrgAccountViews(accounts []OrgAccount) []OrgAccountView {
	views := make([]OrgAccountView, 0, len(accounts))
	for i := range accounts {
		views = append(views, accounts[i].View())
	}

	return views
}

// OrgAccountGroup groups several accounts of the organization under one
// merchant.
type OrgAccountGroup struct {
	ID         string `json:"id,omitempty"`
	Name       string `json:"name,omitempty"`
	MerchantID string `json:"merchant_id,omitempty"`
	CreatedAt  string `json:"created_at,omitempty"`
	UpdatedAt  string `json:"updated_at,omitempty"`

	// Raw is the verbatim response body, set when the group was decoded from one.
	Raw json.RawMessage `json:"-"`
}

// UnmarshalJSON decodes the typed fields and keeps the original body.
func (g *OrgAccountGroup) UnmarshalJSON(data []byte) error {
	type alias OrgAccountGroup

	var decoded alias
	if err := json.Unmarshal(data, &decoded); err != nil {
		return err
	}

	*g = OrgAccountGroup(decoded)
	g.Raw = append(json.RawMessage(nil), data...)

	return nil
}

// MarshalJSON replays the original response when there is one.
//
//nolint:gocritic // hugeParam: a value receiver is required by json.Marshaler here
func (g OrgAccountGroup) MarshalJSON() ([]byte, error) {
	if len(g.Raw) > 0 {
		return g.Raw, nil
	}

	type alias OrgAccountGroup

	return json.Marshal(alias(g))
}

// OrgAccountGroupView is the flat table row of an account group.
type OrgAccountGroupView struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	MerchantID string `json:"merchant_id"`
	CreatedAt  string `json:"created_at"`
}

// View flattens an account group into a table row.
func (g *OrgAccountGroup) View() OrgAccountGroupView {
	return OrgAccountGroupView{
		ID:         g.ID,
		Name:       g.Name,
		MerchantID: g.MerchantID,
		CreatedAt:  g.CreatedAt,
	}
}

// OrgAccountGroupViews flattens a list of account groups.
func OrgAccountGroupViews(groups []OrgAccountGroup) []OrgAccountGroupView {
	views := make([]OrgAccountGroupView, 0, len(groups))
	for i := range groups {
		views = append(views, groups[i].View())
	}

	return views
}

// OrgUserAccountPermission binds one user to one role on one account, as the
// user payload spells it out.
type OrgUserAccountPermission struct {
	AccountID string `json:"account_id,omitempty"`
	RoleID    string `json:"role_id,omitempty"`
}

// OrgUserAccountGroupPermission binds one user to one role on one account group.
type OrgUserAccountGroupPermission struct {
	AccountGroupID string `json:"account_group_id,omitempty"`
	RoleID         string `json:"role_id,omitempty"`
}

// OrgUser is one member of the organization.
type OrgUser struct {
	ID                      string                          `json:"id,omitempty"`
	Email                   string                          `json:"email,omitempty"`
	FirstName               string                          `json:"first_name,omitempty"`
	LastName                string                          `json:"last_name,omitempty"`
	AccountPermissions      []OrgUserAccountPermission      `json:"account_permissions,omitempty"`
	AccountGroupPermissions []OrgUserAccountGroupPermission `json:"account_group_permissions,omitempty"`
	CreatedAt               string                          `json:"created_at,omitempty"`
	UpdatedAt               string                          `json:"updated_at,omitempty"`

	// Raw is the verbatim response body, set when the user was decoded from one.
	Raw json.RawMessage `json:"-"`
}

// UnmarshalJSON decodes the typed fields and keeps the original body.
func (u *OrgUser) UnmarshalJSON(data []byte) error {
	type alias OrgUser

	var decoded alias
	if err := json.Unmarshal(data, &decoded); err != nil {
		return err
	}

	*u = OrgUser(decoded)
	u.Raw = append(json.RawMessage(nil), data...)

	return nil
}

// MarshalJSON replays the original response when there is one.
//
//nolint:gocritic // hugeParam: a value receiver is required by json.Marshaler here
func (u OrgUser) MarshalJSON() ([]byte, error) {
	if len(u.Raw) > 0 {
		return u.Raw, nil
	}

	type alias OrgUser

	return json.Marshal(alias(u))
}

// OrgUserView is the flat table row of a user. The two permission lists are
// reduced to their sizes: the details belong to the permission commands.
type OrgUserView struct {
	ID        string `json:"id"`
	Email     string `json:"email"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Accounts  string `json:"accounts"`
	Groups    string `json:"groups"`
	CreatedAt string `json:"created_at"`
}

// View flattens a user into a table row.
func (u *OrgUser) View() OrgUserView {
	return OrgUserView{
		ID:        u.ID,
		Email:     u.Email,
		FirstName: u.FirstName,
		LastName:  u.LastName,
		Accounts:  strconv.Itoa(len(u.AccountPermissions)),
		Groups:    strconv.Itoa(len(u.AccountGroupPermissions)),
		CreatedAt: u.CreatedAt,
	}
}

// OrgUserViews flattens a list of users.
func OrgUserViews(users []OrgUser) []OrgUserView {
	views := make([]OrgUserView, 0, len(users))
	for i := range users {
		views = append(views, users[i].View())
	}

	return views
}

// OrgRole is one role of the organization: a named bundle of permissions.
type OrgRole struct {
	ID                   string   `json:"id,omitempty"`
	Name                 string   `json:"name,omitempty"`
	Description          *string  `json:"description,omitempty"`
	Admin                bool     `json:"admin,omitempty"`
	PermissionIDs        []string `json:"permission_ids,omitempty"`
	TestingPermissionIDs []string `json:"testing_permission_ids,omitempty"`
	RoleType             string   `json:"role_type,omitempty"`
	CreatedAt            string   `json:"created_at,omitempty"`
	UpdatedAt            string   `json:"updated_at,omitempty"`

	// Raw is the verbatim response body, set when the role was decoded from one.
	Raw json.RawMessage `json:"-"`
}

// UnmarshalJSON decodes the typed fields and keeps the original body.
func (r *OrgRole) UnmarshalJSON(data []byte) error {
	type alias OrgRole

	var decoded alias
	if err := json.Unmarshal(data, &decoded); err != nil {
		return err
	}

	*r = OrgRole(decoded)
	r.Raw = append(json.RawMessage(nil), data...)

	return nil
}

// MarshalJSON replays the original response when there is one.
//
//nolint:gocritic // hugeParam: a value receiver is required by json.Marshaler here
func (r OrgRole) MarshalJSON() ([]byte, error) {
	if len(r.Raw) > 0 {
		return r.Raw, nil
	}

	type alias OrgRole

	return json.Marshal(alias(r))
}

// OrgRoleView is the flat table row of a role.
type OrgRoleView struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	RoleType    string `json:"role_type"`
	Admin       string `json:"admin"`
	Permissions string `json:"permissions"`
	Description string `json:"description"`
}

// View flattens a role into a table row.
func (r *OrgRole) View() OrgRoleView {
	view := OrgRoleView{
		ID:          r.ID,
		Name:        r.Name,
		RoleType:    r.RoleType,
		Admin:       strconv.FormatBool(r.Admin),
		Permissions: strconv.Itoa(len(r.PermissionIDs)),
	}

	if r.Description != nil {
		view.Description = *r.Description
	}

	return view
}

// OrgRoleViews flattens a list of roles.
func OrgRoleViews(roles []OrgRole) []OrgRoleView {
	views := make([]OrgRoleView, 0, len(roles))
	for i := range roles {
		views = append(views, roles[i].View())
	}

	return views
}

// OrgPermission is one entry of the permissions catalog.
type OrgPermission struct {
	ID          string `json:"id,omitempty"`
	Title       string `json:"title,omitempty"`
	Description string `json:"description,omitempty"`
	ActionType  string `json:"action_type,omitempty"`
}

// OrgPermissionGroup is one section of the permissions catalog.
type OrgPermissionGroup struct {
	Title              string          `json:"title,omitempty"`
	OrganizationOnly   bool            `json:"organization_only,omitempty"`
	PermissionsCatalog []OrgPermission `json:"permissions_catalog,omitempty"`
}

// OrgPermissionView is the flat table row of one catalog permission, carrying
// the section it belongs to so the table reads on its own.
type OrgPermissionView struct {
	Group      string `json:"group"`
	ID         string `json:"id"`
	Title      string `json:"title"`
	ActionType string `json:"action_type"`
}

// OrgPermissionViews flattens the whole catalog into one table.
func OrgPermissionViews(groups []OrgPermissionGroup) []OrgPermissionView {
	var views []OrgPermissionView

	for i := range groups {
		group := &groups[i]

		for j := range group.PermissionsCatalog {
			permission := &group.PermissionsCatalog[j]

			views = append(views, OrgPermissionView{
				Group:      group.Title,
				ID:         permission.ID,
				Title:      permission.Title,
				ActionType: permission.ActionType,
			})
		}
	}

	return views
}

// OrgAccountPermission is one resolved account permission of a user: the role
// the user holds on one account, with both names spelled out.
type OrgAccountPermission struct {
	AccountID   string `json:"account_id,omitempty"`
	AccountName string `json:"account_name,omitempty"`
	RoleID      string `json:"role_id,omitempty"`
	RoleName    string `json:"role_name,omitempty"`
}

// OrgAccountPermissions is the envelope of the account permissions endpoints.
type OrgAccountPermissions struct {
	Permissions []OrgAccountPermission `json:"permissions,omitempty"`
}

// OrgAccountPermissionView is the flat table row of an account permission.
type OrgAccountPermissionView struct {
	AccountID   string `json:"account_id"`
	AccountName string `json:"account_name"`
	RoleID      string `json:"role_id"`
	RoleName    string `json:"role_name"`
}

// Views flattens the account permissions of a user.
func (p *OrgAccountPermissions) Views() []OrgAccountPermissionView {
	views := make([]OrgAccountPermissionView, 0, len(p.Permissions))

	for i := range p.Permissions {
		permission := &p.Permissions[i]

		views = append(views, OrgAccountPermissionView{
			AccountID:   permission.AccountID,
			AccountName: permission.AccountName,
			RoleID:      permission.RoleID,
			RoleName:    permission.RoleName,
		})
	}

	return views
}

// OrgAccountGroupPermission is one resolved account group permission of a user.
type OrgAccountGroupPermission struct {
	AccountGroupID   string `json:"account_group_id,omitempty"`
	AccountGroupName string `json:"account_group_name,omitempty"`
	RoleID           string `json:"role_id,omitempty"`
	RoleName         string `json:"role_name,omitempty"`
	AccountsCount    int    `json:"accounts_count,omitempty"`
}

// OrgAccountGroupPermissions is the envelope of the account group permissions
// endpoints.
type OrgAccountGroupPermissions struct {
	Permissions []OrgAccountGroupPermission `json:"permissions,omitempty"`
}

// OrgAccountGroupPermissionView is the flat table row of a group permission.
type OrgAccountGroupPermissionView struct {
	AccountGroupID   string `json:"account_group_id"`
	AccountGroupName string `json:"account_group_name"`
	RoleID           string `json:"role_id"`
	RoleName         string `json:"role_name"`
	Accounts         string `json:"accounts"`
}

// Views flattens the account group permissions of a user.
func (p *OrgAccountGroupPermissions) Views() []OrgAccountGroupPermissionView {
	views := make([]OrgAccountGroupPermissionView, 0, len(p.Permissions))

	for i := range p.Permissions {
		permission := &p.Permissions[i]

		views = append(views, OrgAccountGroupPermissionView{
			AccountGroupID:   permission.AccountGroupID,
			AccountGroupName: permission.AccountGroupName,
			RoleID:           permission.RoleID,
			RoleName:         permission.RoleName,
			Accounts:         strconv.Itoa(permission.AccountsCount),
		})
	}

	return views
}

// OrgAuthToken is the whitelabel access token returned by the authenticate
// endpoint.
type OrgAuthToken struct {
	AccessToken string `json:"access_token,omitempty"`
	TokenType   string `json:"token_type,omitempty"`
	ExpiresIn   int    `json:"expires_in,omitempty"`
}

// OrgAuthTokenView is the flat table row of an access token. The token itself
// is masked in the table output; `--json` still prints it verbatim.
type OrgAuthTokenView struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   string `json:"expires_in"`
}

// View flattens an access token into a table row.
func (t *OrgAuthToken) View() OrgAuthTokenView {
	return OrgAuthTokenView{
		AccessToken: t.AccessToken,
		TokenType:   t.TokenType,
		ExpiresIn:   strconv.Itoa(t.ExpiresIn),
	}
}
