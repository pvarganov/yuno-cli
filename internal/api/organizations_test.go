package api

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

// orgRequest records what the fake organization API received.
type orgRequest struct {
	Method string
	Path   string
	Query  string
	Body   string
}

// startOrgServer serves one canned response and records the last request.
func startOrgServer(t *testing.T, status int, response string) (*Client, *orgRequest) {
	t.Helper()

	got := &orgRequest{}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)

		got.Method = r.Method
		got.Path = r.URL.Path
		got.Query = r.URL.RawQuery
		got.Body = string(body)

		w.WriteHeader(status)
		_, _ = io.WriteString(w, response)
	}))
	t.Cleanup(srv.Close)

	c, _ := newTestClient(t, srv, testProfile())

	return c, got
}

func TestListOrgAccounts_PagesFromPageOne(t *testing.T) {
	var queries []string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		queries = append(queries, r.URL.RawQuery)

		if len(queries) == 1 {
			_, _ = io.WriteString(w, `{"data":[{"id":"acc-1"},{"id":"acc-2"}],"pagination":{"has_next":true}}`)

			return
		}

		_, _ = io.WriteString(w, `{"data":[{"id":"acc-3"}],"pagination":{"has_next":false}}`)
	}))
	defer srv.Close()

	c, _ := newTestClient(t, srv, testProfile())

	accounts, err := c.ListOrgAccounts(t.Context(), 0, 2)
	if err != nil {
		t.Fatalf("ListOrgAccounts: %v", err)
	}

	if len(accounts) != 3 || accounts[2].ID != "acc-3" {
		t.Errorf("unexpected accounts: %+v", accounts)
	}

	if len(queries) != 2 {
		t.Fatalf("expected two requests, got %v", queries)
	}

	if !strings.Contains(queries[0], "page=1") || !strings.Contains(queries[0], "page_size=2") {
		t.Errorf("expected the first page to be page 1, got %s", queries[0])
	}

	if !strings.Contains(queries[1], "page=2") {
		t.Errorf("expected the second request to ask for page 2, got %s", queries[1])
	}
}

func TestListOrgAccounts_StopsOnTheTotal(t *testing.T) {
	var pages int

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		pages++

		_, _ = io.WriteString(w, `{"data":[{"id":"acc-1"},{"id":"acc-2"}],"pagination":{"total_items":2}}`)
	}))
	defer srv.Close()

	c, _ := newTestClient(t, srv, testProfile())

	accounts, err := c.ListOrgAccounts(t.Context(), 0, 2)
	if err != nil {
		t.Fatalf("ListOrgAccounts: %v", err)
	}

	if len(accounts) != 2 || pages != 1 {
		t.Errorf("expected one page of two accounts, got %d accounts in %d pages", len(accounts), pages)
	}
}

func TestListOrgAccounts_ErrorIsWrapped(t *testing.T) {
	c, _ := startOrgServer(t, http.StatusForbidden, `{"code":"INSUFFICIENT_SCOPE","message":"nope"}`)

	_, err := c.ListOrgAccounts(t.Context(), 0, 0)
	if err == nil || !errors.Is(err, ErrForbidden) || !strings.Contains(err.Error(), "list organization accounts") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestOrgAccountMethods_HitTheRightPaths(t *testing.T) {
	tests := []struct {
		name   string
		call   func(c *Client) error
		method string
		path   string
	}{
		{
			name:   "get",
			call:   func(c *Client) error { _, err := c.GetOrgAccount(t.Context(), "acc-1"); return err },
			method: http.MethodGet,
			path:   "/organizations/accounts/acc-1",
		},
		{
			name: "update",
			call: func(c *Client) error {
				_, err := c.UpdateOrgAccount(t.Context(), "acc-1", map[string]any{"name": "BR"})

				return err
			},
			method: http.MethodPatch,
			path:   "/organizations/accounts/acc-1",
		},
		{
			name:   "delete",
			call:   func(c *Client) error { _, err := c.DeleteOrgAccount(t.Context(), "acc-1"); return err },
			method: http.MethodDelete,
			path:   "/organizations/accounts/acc-1",
		},
		{
			name: "create",
			call: func(c *Client) error {
				_, err := c.CreateOrgAccount(t.Context(), "grp-1", map[string]any{"name": "BR"})

				return err
			},
			method: http.MethodPost,
			path:   "/organizations/account-groups/grp-1/accounts",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, got := startOrgServer(t, http.StatusOK, `{"id":"acc-1"}`)

			if err := tt.call(c); err != nil {
				t.Fatalf("%s: %v", tt.name, err)
			}

			if got.Method != tt.method || got.Path != tt.path {
				t.Errorf("expected %s %s, got %s %s", tt.method, tt.path, got.Method, got.Path)
			}
		})
	}
}

func TestListAccountGroupAccounts_HitsTheGroupPath(t *testing.T) {
	c, got := startOrgServer(t, http.StatusOK, `{"data":[{"id":"acc-1"}],"pagination":{"has_next":false}}`)

	accounts, err := c.ListAccountGroupAccounts(t.Context(), "grp-1", 0, 0)
	if err != nil {
		t.Fatalf("ListAccountGroupAccounts: %v", err)
	}

	if len(accounts) != 1 || got.Path != "/organizations/account-groups/grp-1/accounts" {
		t.Errorf("unexpected result: %+v at %s", accounts, got.Path)
	}
}

func TestAccountGroupMethods_HitTheRightPaths(t *testing.T) {
	tests := []struct {
		name   string
		call   func(c *Client) error
		method string
		path   string
	}{
		{
			name:   "list",
			call:   func(c *Client) error { _, err := c.ListAccountGroups(t.Context(), 0, 0); return err },
			method: http.MethodGet,
			path:   "/organizations/account-groups",
		},
		{
			name:   "get",
			call:   func(c *Client) error { _, err := c.GetAccountGroup(t.Context(), "grp-1"); return err },
			method: http.MethodGet,
			path:   "/organizations/account-groups/grp-1",
		},
		{
			name: "create",
			call: func(c *Client) error {
				_, err := c.CreateAccountGroup(t.Context(), map[string]any{"name": "Overgear"})

				return err
			},
			method: http.MethodPost,
			path:   "/organizations/account-groups",
		},
		{
			name: "update",
			call: func(c *Client) error {
				_, err := c.UpdateAccountGroup(t.Context(), "grp-1", map[string]any{"name": "Overgear"})

				return err
			},
			method: http.MethodPatch,
			path:   "/organizations/account-groups/grp-1",
		},
		{
			name:   "delete",
			call:   func(c *Client) error { _, err := c.DeleteAccountGroup(t.Context(), "grp-1"); return err },
			method: http.MethodDelete,
			path:   "/organizations/account-groups/grp-1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, got := startOrgServer(t, http.StatusOK,
				`{"id":"grp-1","data":[{"id":"grp-1"}],"pagination":{"has_next":false}}`)

			if err := tt.call(c); err != nil {
				t.Fatalf("%s: %v", tt.name, err)
			}

			if got.Method != tt.method || got.Path != tt.path {
				t.Errorf("expected %s %s, got %s %s", tt.method, tt.path, got.Method, got.Path)
			}
		})
	}
}

func TestListOrgUsers_SendsTheFilters(t *testing.T) {
	c, got := startOrgServer(t, http.StatusOK, `{"data":[{"id":"usr-1"}],"pagination":{"has_next":false}}`)

	users, err := c.ListOrgUsers(t.Context(), url.Values{"account_id": {"acc-1"}}, 0, 10)
	if err != nil {
		t.Fatalf("ListOrgUsers: %v", err)
	}

	if len(users) != 1 || got.Path != "/organizations/users" {
		t.Errorf("unexpected result: %+v at %s", users, got.Path)
	}

	for _, want := range []string{"account_id=acc-1", "page=1", "page_size=10"} {
		if !strings.Contains(got.Query, want) {
			t.Errorf("expected the query to contain %q, got %s", want, got.Query)
		}
	}
}

func TestOrgUserMethods_HitTheRightPaths(t *testing.T) {
	tests := []struct {
		name   string
		call   func(c *Client) error
		method string
		path   string
	}{
		{
			name:   "get",
			call:   func(c *Client) error { _, err := c.GetOrgUser(t.Context(), "usr-1"); return err },
			method: http.MethodGet,
			path:   "/organizations/users/usr-1",
		},
		{
			name: "create",
			call: func(c *Client) error {
				_, err := c.CreateOrgUser(t.Context(), map[string]any{"email": "ops@example.com"})

				return err
			},
			method: http.MethodPost,
			path:   "/organizations/users",
		},
		{
			name: "update",
			call: func(c *Client) error {
				_, err := c.UpdateOrgUser(t.Context(), "usr-1", map[string]any{"first_name": "Ops"})

				return err
			},
			method: http.MethodPatch,
			path:   "/organizations/users/usr-1",
		},
		{
			name:   "delete",
			call:   func(c *Client) error { _, err := c.DeleteOrgUser(t.Context(), "usr-1"); return err },
			method: http.MethodDelete,
			path:   "/organizations/users/usr-1",
		},
		{
			name: "authenticate",
			call: func(c *Client) error {
				_, err := c.AuthenticateOrgUser(t.Context(), map[string]any{"user_id": "usr-1"})

				return err
			},
			method: http.MethodPost,
			path:   "/organizations/authenticate",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, got := startOrgServer(t, http.StatusOK, `{"id":"usr-1","access_token":"tok-1"}`)

			if err := tt.call(c); err != nil {
				t.Fatalf("%s: %v", tt.name, err)
			}

			if got.Method != tt.method || got.Path != tt.path {
				t.Errorf("expected %s %s, got %s %s", tt.method, tt.path, got.Method, got.Path)
			}
		})
	}
}

func TestAuthenticateOrgUser_DecodesTheToken(t *testing.T) {
	c, got := startOrgServer(t, http.StatusOK, `{"access_token":"tok-1","token_type":"Bearer","expires_in":3600}`)

	token, err := c.AuthenticateOrgUser(t.Context(), map[string]any{"user_id": "usr-1"})
	if err != nil {
		t.Fatalf("AuthenticateOrgUser: %v", err)
	}

	if token.AccessToken != "tok-1" || token.ExpiresIn != 3600 {
		t.Errorf("unexpected token: %+v", token)
	}

	if !strings.Contains(got.Body, `"user_id":"usr-1"`) {
		t.Errorf("unexpected body: %s", got.Body)
	}
}

func TestOrgRoleMethods_HitTheRightPaths(t *testing.T) {
	tests := []struct {
		name     string
		call     func(c *Client) error
		method   string
		path     string
		response string
	}{
		{
			name:     "list",
			call:     func(c *Client) error { _, err := c.ListOrgRoles(t.Context()); return err },
			method:   http.MethodGet,
			path:     "/organizations/roles",
			response: `[]`,
		},
		{
			name: "create",
			call: func(c *Client) error {
				_, err := c.CreateOrgRole(t.Context(), map[string]any{"name": "Support"})

				return err
			},
			method:   http.MethodPost,
			path:     "/organizations/roles",
			response: `{"id":"role-1"}`,
		},
		{
			name: "update",
			call: func(c *Client) error {
				_, err := c.UpdateOrgRole(t.Context(), "role-1", map[string]any{"name": "Support"})

				return err
			},
			method:   http.MethodPatch,
			path:     "/organizations/roles/role-1",
			response: `{"id":"role-1"}`,
		},
		{
			name:     "delete",
			call:     func(c *Client) error { _, err := c.DeleteOrgRole(t.Context(), "role-1"); return err },
			method:   http.MethodDelete,
			path:     "/organizations/roles/role-1",
			response: ``,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, got := startOrgServer(t, http.StatusOK, tt.response)

			if err := tt.call(c); err != nil {
				t.Fatalf("%s: %v", tt.name, err)
			}

			if got.Method != tt.method || got.Path != tt.path {
				t.Errorf("expected %s %s, got %s %s", tt.method, tt.path, got.Method, got.Path)
			}
		})
	}
}

func TestListOrgRoles_DecodesTheBareArray(t *testing.T) {
	c, _ := startOrgServer(t, http.StatusOK, `[{"id":"role-1","name":"Support","admin":true}]`)

	roles, err := c.ListOrgRoles(t.Context())
	if err != nil {
		t.Fatalf("ListOrgRoles: %v", err)
	}

	if len(roles) != 1 || !roles[0].Admin {
		t.Errorf("unexpected roles: %+v", roles)
	}
}

func TestListPermissionsCatalog_DecodesTheSections(t *testing.T) {
	c, got := startOrgServer(t, http.StatusOK,
		`[{"title":"Payments","permissions_catalog":[{"id":"payments.read"}]}]`)

	groups, err := c.ListPermissionsCatalog(t.Context())
	if err != nil {
		t.Fatalf("ListPermissionsCatalog: %v", err)
	}

	if len(groups) != 1 || groups[0].PermissionsCatalog[0].ID != "payments.read" {
		t.Errorf("unexpected catalog: %+v", groups)
	}

	if got.Path != "/organizations/permissions-catalog" {
		t.Errorf("unexpected path: %s", got.Path)
	}
}

func TestListPermissionsCatalog_ErrorIsWrapped(t *testing.T) {
	c, _ := startOrgServer(t, http.StatusInternalServerError, `{"code":"BOOM","message":"nope"}`)

	_, err := c.ListPermissionsCatalog(t.Context())
	if err == nil || !strings.Contains(err.Error(), "list permissions catalog") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestUserPermissionMethods_HitTheRightPaths(t *testing.T) {
	body := map[string]any{"account_permissions": []any{map[string]any{"account_id": "acc-1"}}}

	tests := []struct {
		name   string
		call   func(c *Client) error
		method string
		path   string
	}{
		{
			name:   "account get",
			call:   func(c *Client) error { _, err := c.GetUserAccountPermissions(t.Context(), "usr-1"); return err },
			method: http.MethodGet,
			path:   "/organizations/users/usr-1/account-permissions",
		},
		{
			name: "account replace",
			call: func(c *Client) error {
				_, err := c.ReplaceUserAccountPermissions(t.Context(), "usr-1", body)

				return err
			},
			method: http.MethodPut,
			path:   "/organizations/users/usr-1/account-permissions",
		},
		{
			name: "account update",
			call: func(c *Client) error {
				_, err := c.UpdateUserAccountPermissions(t.Context(), "usr-1", body)

				return err
			},
			method: http.MethodPatch,
			path:   "/organizations/users/usr-1/account-permissions",
		},
		{
			name: "account delete",
			call: func(c *Client) error {
				_, err := c.DeleteUserAccountPermission(t.Context(), "usr-1", "acc-1")

				return err
			},
			method: http.MethodDelete,
			path:   "/organizations/users/usr-1/account-permissions/acc-1",
		},
		{
			name: "group get",
			call: func(c *Client) error {
				_, err := c.GetUserAccountGroupPermissions(t.Context(), "usr-1")

				return err
			},
			method: http.MethodGet,
			path:   "/organizations/users/usr-1/account-group-permissions",
		},
		{
			name: "group replace",
			call: func(c *Client) error {
				_, err := c.ReplaceUserAccountGroupPermissions(t.Context(), "usr-1", body)

				return err
			},
			method: http.MethodPut,
			path:   "/organizations/users/usr-1/account-group-permissions",
		},
		{
			name: "group update",
			call: func(c *Client) error {
				_, err := c.UpdateUserAccountGroupPermissions(t.Context(), "usr-1", body)

				return err
			},
			method: http.MethodPatch,
			path:   "/organizations/users/usr-1/account-group-permissions",
		},
		{
			name: "group delete",
			call: func(c *Client) error {
				_, err := c.DeleteUserAccountGroupPermission(t.Context(), "usr-1", "grp-1")

				return err
			},
			method: http.MethodDelete,
			path:   "/organizations/users/usr-1/account-group-permissions/grp-1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, got := startOrgServer(t, http.StatusOK, `{"permissions":[],"status":"ok"}`)

			if err := tt.call(c); err != nil {
				t.Fatalf("%s: %v", tt.name, err)
			}

			if got.Method != tt.method || got.Path != tt.path {
				t.Errorf("expected %s %s, got %s %s", tt.method, tt.path, got.Method, got.Path)
			}
		})
	}
}

func TestGetUserAccountPermissions_DecodesTheEnvelope(t *testing.T) {
	c, _ := startOrgServer(t, http.StatusOK,
		`{"permissions":[{"account_id":"acc-1","account_name":"BR","role_id":"role-1","role_name":"Support"}]}`)

	permissions, err := c.GetUserAccountPermissions(t.Context(), "usr-1")
	if err != nil {
		t.Fatalf("GetUserAccountPermissions: %v", err)
	}

	if len(permissions.Permissions) != 1 || permissions.Permissions[0].RoleName != "Support" {
		t.Errorf("unexpected permissions: %+v", permissions)
	}
}

func TestDeleteUserAccountPermission_ErrorIsWrapped(t *testing.T) {
	c, _ := startOrgServer(t, http.StatusNotFound, `{"code":"NOT_FOUND","message":"nope"}`)

	_, err := c.DeleteUserAccountPermission(t.Context(), "usr-1", "acc-1")
	if err == nil || !strings.Contains(err.Error(), "delete account permission of user usr-1") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestOrgPaths_EscapeTheIdentifiers(t *testing.T) {
	c, got := startOrgServer(t, http.StatusOK, `{"id":"acc-1"}`)

	if _, err := c.GetOrgAccount(t.Context(), "acc 1/2"); err != nil {
		t.Fatalf("GetOrgAccount: %v", err)
	}

	if got.Path != "/organizations/accounts/acc 1/2" {
		t.Errorf("unexpected path: %s", got.Path)
	}
}
