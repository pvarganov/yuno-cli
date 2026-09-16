package cmd_test

import (
	"net/http"
	"strings"
	"testing"
)

// orgAccountResponse is the account payload the fake API returns.
const orgAccountResponse = `{"id":"acc-1","name":"Overgear-BR","account_group_id":"grp-1",
	"created_at":"2026-09-16T10:00:00Z"}`

// orgAccountGroupResponse is the account group payload the fake API returns.
const orgAccountGroupResponse = `{"id":"grp-1","name":"Overgear","merchant_id":"mer-1",
	"created_at":"2026-09-16T10:00:00Z"}`

// orgUserResponse is the user payload the fake API returns.
const orgUserResponse = `{"id":"usr-1","email":"ops@example.com","first_name":"Ops","last_name":"Team",
	"account_permissions":[{"account_id":"acc-1","role_id":"role-1"}],
	"created_at":"2026-09-16T10:00:00Z"}`

// orgRoleResponse is the role payload the fake API returns.
const orgRoleResponse = `{"id":"role-1","name":"Support","description":"read only","admin":false,
	"permission_ids":["payments.read"],"role_type":"ACCOUNT"}`

func TestOrgAccountList_PagesTheAccounts(t *testing.T) {
	isolateConfig(t)
	seedCredentials(t)

	got := startAPI(t, http.StatusOK, `{"data":[`+orgAccountResponse+`],"pagination":{"has_next":false}}`)

	out, err := runCLI(t, "", "org", "account", "list", "--page-size", "5")
	if err != nil {
		t.Fatalf("org account list failed: %v (%s)", err, out)
	}

	if got.Method != http.MethodGet || got.Path != "/v1/organizations/accounts" {
		t.Errorf("expected GET /v1/organizations/accounts, got %s %s", got.Method, got.Path)
	}

	for _, want := range []string{"page=1", "page_size=5"} {
		if !strings.Contains(got.Query, want) {
			t.Errorf("expected the query to contain %q, got %s", want, got.Query)
		}
	}

	for _, want := range []string{"acc-1", "Overgear-BR", "grp-1"} {
		if !strings.Contains(out, want) {
			t.Errorf("expected table output to contain %q, got:\n%s", want, out)
		}
	}
}

func TestOrgAccountGet_HitsTheIDPath(t *testing.T) {
	isolateConfig(t)
	seedCredentials(t)

	got := startAPI(t, http.StatusOK, orgAccountResponse)

	out, err := runCLI(t, "", "org", "account", "get", "acc-1", "--json")
	if err != nil {
		t.Fatalf("org account get failed: %v (%s)", err, out)
	}

	if got.Method != http.MethodGet || got.Path != "/v1/organizations/accounts/acc-1" {
		t.Errorf("expected GET /v1/organizations/accounts/acc-1, got %s %s", got.Method, got.Path)
	}

	if !strings.Contains(out, `"account_group_id"`) {
		t.Errorf("expected the whole payload as json, got:\n%s", out)
	}
}

func TestOrgAccountCreate_PostsUnderTheAccountGroup(t *testing.T) {
	isolateConfig(t)
	seedCredentials(t)

	got := startAPI(t, http.StatusCreated, orgAccountResponse)

	out, err := runCLI(t, "", "org", "account", "create", "grp-1", "--name", "Overgear-BR", "--yes")
	if err != nil {
		t.Fatalf("org account create failed: %v (%s)", err, out)
	}

	if got.Method != http.MethodPost || got.Path != "/v1/organizations/account-groups/grp-1/accounts" {
		t.Errorf("expected POST /v1/organizations/account-groups/grp-1/accounts, got %s %s", got.Method, got.Path)
	}

	if body := decodeBody(t, got.Body); body["name"] != "Overgear-BR" {
		t.Errorf("unexpected body: %s", got.Body)
	}
}

func TestOrgAccountCreate_RequiresABody(t *testing.T) {
	isolateConfig(t)
	seedCredentials(t)
	startAPI(t, http.StatusCreated, orgAccountResponse)

	_, err := runCLI(t, "", "org", "account", "create", "grp-1", "--yes")
	if err == nil || !strings.Contains(err.Error(), "request body is required") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestOrgAccountUpdate_PatchesTheAccount(t *testing.T) {
	isolateConfig(t)
	seedCredentials(t)

	got := startAPI(t, http.StatusOK, orgAccountResponse)

	out, err := runCLI(t, "", "org", "account", "update", "acc-1", "--name", "Overgear-BR", "--yes")
	if err != nil {
		t.Fatalf("org account update failed: %v (%s)", err, out)
	}

	if got.Method != http.MethodPatch || got.Path != "/v1/organizations/accounts/acc-1" {
		t.Errorf("expected PATCH /v1/organizations/accounts/acc-1, got %s %s", got.Method, got.Path)
	}
}

func TestOrgAccountDelete_DeletesTheAccount(t *testing.T) {
	isolateConfig(t)
	seedCredentials(t)

	got := startAPI(t, http.StatusOK, `{"message":"deleted"}`)

	out, err := runCLI(t, "", "org", "account", "delete", "acc-1", "--yes")
	if err != nil {
		t.Fatalf("org account delete failed: %v (%s)", err, out)
	}

	if got.Method != http.MethodDelete || got.Path != "/v1/organizations/accounts/acc-1" {
		t.Errorf("expected DELETE /v1/organizations/accounts/acc-1, got %s %s", got.Method, got.Path)
	}

	if !strings.Contains(out, "deleted") {
		t.Errorf("expected the acknowledgement in the output, got:\n%s", out)
	}
}

func TestOrgAccountGroupCRUD_HitsTheRightPaths(t *testing.T) {
	tests := []struct {
		name   string
		args   []string
		method string
		path   string
	}{
		{
			name:   "list",
			args:   []string{"org", "account-group", "list"},
			method: http.MethodGet,
			path:   "/v1/organizations/account-groups",
		},
		{
			name:   "get",
			args:   []string{"org", "account-group", "get", "grp-1"},
			method: http.MethodGet,
			path:   "/v1/organizations/account-groups/grp-1",
		},
		{
			name:   "create",
			args:   []string{"org", "account-group", "create", "--name", "Overgear", "--merchant-id", "mer-1", "--yes"},
			method: http.MethodPost,
			path:   "/v1/organizations/account-groups",
		},
		{
			name:   "update",
			args:   []string{"org", "account-group", "update", "grp-1", "--name", "Overgear", "--yes"},
			method: http.MethodPatch,
			path:   "/v1/organizations/account-groups/grp-1",
		},
		{
			name:   "delete",
			args:   []string{"org", "account-group", "delete", "grp-1", "--yes"},
			method: http.MethodDelete,
			path:   "/v1/organizations/account-groups/grp-1",
		},
		{
			name:   "accounts",
			args:   []string{"org", "account-group", "accounts", "grp-1"},
			method: http.MethodGet,
			path:   "/v1/organizations/account-groups/grp-1/accounts",
		},
		{
			name:   "add-account",
			args:   []string{"org", "account-group", "add-account", "grp-1", "--name", "BR", "--yes"},
			method: http.MethodPost,
			path:   "/v1/organizations/account-groups/grp-1/accounts",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isolateConfig(t)
			seedCredentials(t)

			got := startAPI(t, http.StatusOK,
				`{"id":"grp-1","name":"Overgear","data":[`+orgAccountGroupResponse+`],"pagination":{"has_next":false}}`)

			if out, err := runCLI(t, "", tt.args...); err != nil {
				t.Fatalf("%s failed: %v (%s)", tt.name, err, out)
			}

			if got.Method != tt.method || got.Path != tt.path {
				t.Errorf("expected %s %s, got %s %s", tt.method, tt.path, got.Method, got.Path)
			}
		})
	}
}

func TestOrgAccountGroupFindByMerchantID_FiltersLocally(t *testing.T) {
	isolateConfig(t)
	seedCredentials(t)

	response := `{"data":[` + orgAccountGroupResponse +
		`,{"id":"grp-2","name":"Other","merchant_id":"mer-2"}],"pagination":{"has_next":false}}`

	got := startAPI(t, http.StatusOK, response)

	out, err := runCLI(t, "", "org", "account-group", "find-by-merchant-id", "mer-1")
	if err != nil {
		t.Fatalf("find-by-merchant-id failed: %v (%s)", err, out)
	}

	if got.Method != http.MethodGet || got.Path != "/v1/organizations/account-groups" {
		t.Errorf("expected GET /v1/organizations/account-groups, got %s %s", got.Method, got.Path)
	}

	if !strings.Contains(out, "grp-1") || strings.Contains(out, "grp-2") {
		t.Errorf("expected only the matching group, got:\n%s", out)
	}
}

func TestOrgRoleList_RendersTheRoleRows(t *testing.T) {
	isolateConfig(t)
	seedCredentials(t)

	got := startAPI(t, http.StatusOK, `[`+orgRoleResponse+`]`)

	out, err := runCLI(t, "", "org", "role", "list")
	if err != nil {
		t.Fatalf("org role list failed: %v (%s)", err, out)
	}

	if got.Method != http.MethodGet || got.Path != "/v1/organizations/roles" {
		t.Errorf("expected GET /v1/organizations/roles, got %s %s", got.Method, got.Path)
	}

	for _, want := range []string{"role-1", "Support", "ACCOUNT", "read only"} {
		if !strings.Contains(out, want) {
			t.Errorf("expected table output to contain %q, got:\n%s", want, out)
		}
	}
}

func TestOrgRoleCreate_SendsThePermissionList(t *testing.T) {
	isolateConfig(t)
	seedCredentials(t)

	got := startAPI(t, http.StatusCreated, orgRoleResponse)

	out, err := runCLI(t, "", "org", "role", "create", "--yes",
		"--name", "Support", "--role-type", "ACCOUNT",
		"--permission", "payments.read", "--permission", "payments.write")
	if err != nil {
		t.Fatalf("org role create failed: %v (%s)", err, out)
	}

	if got.Method != http.MethodPost || got.Path != "/v1/organizations/roles" {
		t.Errorf("expected POST /v1/organizations/roles, got %s %s", got.Method, got.Path)
	}

	body := decodeBody(t, got.Body)

	permissions, ok := body["permission_ids"].([]any)
	if !ok || len(permissions) != 2 || permissions[1] != "payments.write" {
		t.Errorf("unexpected permission list in body: %s", got.Body)
	}
}

func TestOrgRoleUpdateAndDelete_HitTheRoleIDPath(t *testing.T) {
	isolateConfig(t)
	seedCredentials(t)

	got := startAPI(t, http.StatusOK, orgRoleResponse)

	if out, err := runCLI(t, "", "org", "role", "update", "role-1", "--name", "Support", "--yes"); err != nil {
		t.Fatalf("org role update failed: %v (%s)", err, out)
	}

	if got.Method != http.MethodPatch || got.Path != "/v1/organizations/roles/role-1" {
		t.Errorf("expected PATCH /v1/organizations/roles/role-1, got %s %s", got.Method, got.Path)
	}

	if out, err := runCLI(t, "", "org", "role", "delete", "role-1", "--yes"); err != nil {
		t.Fatalf("org role delete failed: %v (%s)", err, out)
	}

	if got.Method != http.MethodDelete || got.Path != "/v1/organizations/roles/role-1" {
		t.Errorf("expected DELETE /v1/organizations/roles/role-1, got %s %s", got.Method, got.Path)
	}
}

func TestOrgPermissionsCatalog_FlattensTheSections(t *testing.T) {
	isolateConfig(t)
	seedCredentials(t)

	got := startAPI(t, http.StatusOK,
		`[{"title":"Payments","permissions_catalog":[{"id":"payments.read","title":"Read","action_type":"READ"}]}]`)

	out, err := runCLI(t, "", "org", "permissions-catalog")
	if err != nil {
		t.Fatalf("org permissions-catalog failed: %v (%s)", err, out)
	}

	if got.Method != http.MethodGet || got.Path != "/v1/organizations/permissions-catalog" {
		t.Errorf("expected GET /v1/organizations/permissions-catalog, got %s %s", got.Method, got.Path)
	}

	for _, want := range []string{"Payments", "payments.read", "READ"} {
		if !strings.Contains(out, want) {
			t.Errorf("expected table output to contain %q, got:\n%s", want, out)
		}
	}
}

func TestOrgAuthenticate_PostsTheUserIDAndMasksTheToken(t *testing.T) {
	isolateConfig(t)
	seedCredentials(t)

	got := startAPI(t, http.StatusOK, `{"access_token":"tok-abcdef123456","token_type":"Bearer","expires_in":3600}`)

	out, err := runCLI(t, "", "org", "authenticate", "usr-1", "--yes")
	if err != nil {
		t.Fatalf("org authenticate failed: %v (%s)", err, out)
	}

	if got.Method != http.MethodPost || got.Path != "/v1/organizations/authenticate" {
		t.Errorf("expected POST /v1/organizations/authenticate, got %s %s", got.Method, got.Path)
	}

	if body := decodeBody(t, got.Body); body["user_id"] != "usr-1" {
		t.Errorf("unexpected body: %s", got.Body)
	}

	if strings.Contains(out, "tok-abcdef123456") {
		t.Errorf("the access token must be masked in the table output, got:\n%s", out)
	}

	if !strings.Contains(out, "3600") {
		t.Errorf("expected the lifetime in the output, got:\n%s", out)
	}
}

func TestOrgAuthenticate_ShowsTheTokenAsJSON(t *testing.T) {
	isolateConfig(t)
	seedCredentials(t)

	startAPI(t, http.StatusOK, `{"access_token":"tok-abcdef123456","token_type":"Bearer","expires_in":3600}`)

	out, err := runCLI(t, "", "org", "authenticate", "usr-1", "--json", "--yes")
	if err != nil {
		t.Fatalf("org authenticate failed: %v (%s)", err, out)
	}

	if !strings.Contains(out, "tok-abcdef123456") {
		t.Errorf("--json must print the token verbatim, got:\n%s", out)
	}
}

func TestOrgCommands_ExplainAMissingScope(t *testing.T) {
	isolateConfig(t)
	seedCredentials(t)
	startAPI(t, http.StatusForbidden, `{"code":"INSUFFICIENT_SCOPE","message":"nope"}`)

	_, err := runCLI(t, "", "org", "account", "list")
	if err == nil || !strings.Contains(err.Error(), "organizations:read") {
		t.Errorf("expected a scope hint, got: %v", err)
	}
}
