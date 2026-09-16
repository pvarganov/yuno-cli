package cmd_test

import (
	"net/http"
	"strings"
	"testing"
)

func TestOrgUserList_SendsTheFilters(t *testing.T) {
	isolateConfig(t)
	seedCredentials(t)

	got := startAPI(t, http.StatusOK, `{"data":[`+orgUserResponse+`],"pagination":{"has_next":false}}`)

	out, err := runCLI(t, "", "org", "user", "list", "--account-id", "acc-1", "--account-group-id", "grp-1")
	if err != nil {
		t.Fatalf("org user list failed: %v (%s)", err, out)
	}

	if got.Method != http.MethodGet || got.Path != "/v1/organizations/users" {
		t.Errorf("expected GET /v1/organizations/users, got %s %s", got.Method, got.Path)
	}

	for _, want := range []string{"account_id=acc-1", "account_group_id=grp-1"} {
		if !strings.Contains(got.Query, want) {
			t.Errorf("expected the query to contain %q, got %s", want, got.Query)
		}
	}

	for _, want := range []string{"usr-1", "ops@example.com"} {
		if !strings.Contains(out, want) {
			t.Errorf("expected table output to contain %q, got:\n%s", want, out)
		}
	}
}

func TestOrgUserFindByEmail_FiltersLocally(t *testing.T) {
	isolateConfig(t)
	seedCredentials(t)

	response := `{"data":[` + orgUserResponse +
		`,{"id":"usr-2","email":"other@example.com"}],"pagination":{"has_next":false}}`

	got := startAPI(t, http.StatusOK, response)

	out, err := runCLI(t, "", "org", "user", "find-by-email", "OPS@example.com")
	if err != nil {
		t.Fatalf("find-by-email failed: %v (%s)", err, out)
	}

	if got.Method != http.MethodGet || got.Path != "/v1/organizations/users" {
		t.Errorf("expected GET /v1/organizations/users, got %s %s", got.Method, got.Path)
	}

	if !strings.Contains(out, "usr-1") || strings.Contains(out, "usr-2") {
		t.Errorf("expected only the matching user, got:\n%s", out)
	}
}

func TestOrgUserCRUD_HitsTheRightPaths(t *testing.T) {
	tests := []struct {
		name   string
		args   []string
		method string
		path   string
	}{
		{
			name:   "get",
			args:   []string{"org", "user", "get", "usr-1"},
			method: http.MethodGet,
			path:   "/v1/organizations/users/usr-1",
		},
		{
			name:   "create",
			args:   []string{"org", "user", "create", "--email", "ops@example.com", "--yes"},
			method: http.MethodPost,
			path:   "/v1/organizations/users",
		},
		{
			name:   "update",
			args:   []string{"org", "user", "update", "usr-1", "--first-name", "Ops", "--yes"},
			method: http.MethodPatch,
			path:   "/v1/organizations/users/usr-1",
		},
		{
			name:   "delete",
			args:   []string{"org", "user", "delete", "usr-1", "--yes"},
			method: http.MethodDelete,
			path:   "/v1/organizations/users/usr-1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isolateConfig(t)
			seedCredentials(t)

			got := startAPI(t, http.StatusOK, orgUserResponse)

			if out, err := runCLI(t, "", tt.args...); err != nil {
				t.Fatalf("%s failed: %v (%s)", tt.name, err, out)
			}

			if got.Method != tt.method || got.Path != tt.path {
				t.Errorf("expected %s %s, got %s %s", tt.method, tt.path, got.Method, got.Path)
			}
		})
	}
}

func TestOrgUserCreate_SendsThePermissionArrays(t *testing.T) {
	isolateConfig(t)
	seedCredentials(t)

	got := startAPI(t, http.StatusCreated, orgUserResponse)

	out, err := runCLI(t, "", "org", "user", "create", "--yes",
		"--email", "ops@example.com", "--first-name", "Ops", "--last-name", "Team",
		"--account-permissions", `[{"account_id":"acc-1","role_id":"role-1"}]`)
	if err != nil {
		t.Fatalf("org user create failed: %v (%s)", err, out)
	}

	body := decodeBody(t, got.Body)

	permissions, ok := body["account_permissions"].([]any)
	if !ok || len(permissions) != 1 {
		t.Fatalf("unexpected body: %s", got.Body)
	}

	first, ok := permissions[0].(map[string]any)
	if !ok || first["account_id"] != "acc-1" || first["role_id"] != "role-1" {
		t.Errorf("unexpected account permissions in body: %s", got.Body)
	}
}

func TestOrgUserCreate_RequiresABody(t *testing.T) {
	isolateConfig(t)
	seedCredentials(t)
	startAPI(t, http.StatusCreated, orgUserResponse)

	_, err := runCLI(t, "", "org", "user", "create", "--yes")
	if err == nil || !strings.Contains(err.Error(), "request body is required") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestUserAccountPermissions_HitTheRightPaths(t *testing.T) {
	tests := []struct {
		name   string
		args   []string
		method string
		path   string
	}{
		{
			name:   "list",
			args:   []string{"org", "user", "account-permission", "list", "usr-1"},
			method: http.MethodGet,
			path:   "/v1/organizations/users/usr-1/account-permissions",
		},
		{
			name:   "set",
			args:   []string{"org", "user", "account-permission", "set", "usr-1", "--grant", "acc-1=role-1", "--yes"},
			method: http.MethodPut,
			path:   "/v1/organizations/users/usr-1/account-permissions",
		},
		{
			name:   "update",
			args:   []string{"org", "user", "account-permission", "update", "usr-1", "--grant", "acc-1=role-1", "--yes"},
			method: http.MethodPatch,
			path:   "/v1/organizations/users/usr-1/account-permissions",
		},
		{
			name:   "delete",
			args:   []string{"org", "user", "account-permission", "delete", "usr-1", "acc-1", "--yes"},
			method: http.MethodDelete,
			path:   "/v1/organizations/users/usr-1/account-permissions/acc-1",
		},
		{
			name:   "group list",
			args:   []string{"org", "user", "account-group-permission", "list", "usr-1"},
			method: http.MethodGet,
			path:   "/v1/organizations/users/usr-1/account-group-permissions",
		},
		{
			name: "group set",
			args: []string{
				"org", "user", "account-group-permission", "set", "usr-1", "--grant", "grp-1=role-1", "--yes",
			},
			method: http.MethodPut,
			path:   "/v1/organizations/users/usr-1/account-group-permissions",
		},
		{
			name: "group update",
			args: []string{
				"org", "user", "account-group-permission", "update", "usr-1", "--grant", "grp-1=role-1", "--yes",
			},
			method: http.MethodPatch,
			path:   "/v1/organizations/users/usr-1/account-group-permissions",
		},
		{
			name:   "group delete",
			args:   []string{"org", "user", "account-group-permission", "delete", "usr-1", "grp-1", "--yes"},
			method: http.MethodDelete,
			path:   "/v1/organizations/users/usr-1/account-group-permissions/grp-1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isolateConfig(t)
			seedCredentials(t)

			got := startAPI(t, http.StatusOK, `{"permissions":[],"status":"ok"}`)

			if out, err := runCLI(t, "", tt.args...); err != nil {
				t.Fatalf("%s failed: %v (%s)", tt.name, err, out)
			}

			if got.Method != tt.method || got.Path != tt.path {
				t.Errorf("expected %s %s, got %s %s", tt.method, tt.path, got.Method, got.Path)
			}
		})
	}
}

func TestUserAccountPermissionList_RendersTheResolvedRows(t *testing.T) {
	isolateConfig(t)
	seedCredentials(t)

	startAPI(t, http.StatusOK,
		`{"permissions":[{"account_id":"acc-1","account_name":"BR","role_id":"role-1","role_name":"Support"}]}`)

	out, err := runCLI(t, "", "org", "user", "account-permission", "list", "usr-1")
	if err != nil {
		t.Fatalf("account-permission list failed: %v (%s)", err, out)
	}

	for _, want := range []string{"acc-1", "BR", "Support"} {
		if !strings.Contains(out, want) {
			t.Errorf("expected table output to contain %q, got:\n%s", want, out)
		}
	}
}

func TestUserAccountGroupPermissionList_RendersTheAccountCount(t *testing.T) {
	isolateConfig(t)
	seedCredentials(t)

	startAPI(t, http.StatusOK, `{"permissions":[{"account_group_id":"grp-1","account_group_name":"Overgear",
		"role_id":"role-1","role_name":"Admin","accounts_count":3}]}`)

	out, err := runCLI(t, "", "org", "user", "account-group-permission", "list", "usr-1")
	if err != nil {
		t.Fatalf("account-group-permission list failed: %v (%s)", err, out)
	}

	for _, want := range []string{"grp-1", "Overgear", "Admin", "3"} {
		if !strings.Contains(out, want) {
			t.Errorf("expected table output to contain %q, got:\n%s", want, out)
		}
	}
}

func TestUserPermissionGrant_BuildsThePermissionArray(t *testing.T) {
	isolateConfig(t)
	seedCredentials(t)

	got := startAPI(t, http.StatusOK, `{"status":"ok"}`)

	out, err := runCLI(t, "", "org", "user", "account-permission", "set", "usr-1", "--yes",
		"--grant", "acc-1=role-1", "--grant", "acc-2=role-2")
	if err != nil {
		t.Fatalf("account-permission set failed: %v (%s)", err, out)
	}

	body := decodeBody(t, got.Body)

	permissions, ok := body["account_permissions"].([]any)
	if !ok || len(permissions) != 2 {
		t.Fatalf("unexpected body: %s", got.Body)
	}

	second, ok := permissions[1].(map[string]any)
	if !ok || second["account_id"] != "acc-2" || second["role_id"] != "role-2" {
		t.Errorf("unexpected second grant: %s", got.Body)
	}
}

func TestUserPermissionGrant_UsesTheGroupKeyForGroups(t *testing.T) {
	isolateConfig(t)
	seedCredentials(t)

	got := startAPI(t, http.StatusOK, `{"permissions":[]}`)

	out, err := runCLI(t, "", "org", "user", "account-group-permission", "update", "usr-1", "--yes",
		"--grant", "grp-1=role-1")
	if err != nil {
		t.Fatalf("account-group-permission update failed: %v (%s)", err, out)
	}

	body := decodeBody(t, got.Body)

	permissions, ok := body["account_group_permissions"].([]any)
	if !ok || len(permissions) != 1 {
		t.Fatalf("unexpected body: %s", got.Body)
	}

	first, ok := permissions[0].(map[string]any)
	if !ok || first["account_group_id"] != "grp-1" {
		t.Errorf("unexpected grant: %s", got.Body)
	}
}

func TestUserPermissionGrant_RejectsAMalformedPair(t *testing.T) {
	isolateConfig(t)
	seedCredentials(t)
	startAPI(t, http.StatusOK, `{"status":"ok"}`)

	_, err := runCLI(t, "", "org", "user", "account-permission", "set", "usr-1", "--grant", "acc-1", "--yes")
	if err == nil || !strings.Contains(err.Error(), "want <account_id>=<role_id>") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestUserPermissionSet_AcceptsAWholeJSONArray(t *testing.T) {
	isolateConfig(t)
	seedCredentials(t)

	got := startAPI(t, http.StatusOK, `{"status":"ok"}`)

	out, err := runCLI(t, "", "org", "user", "account-permission", "set", "usr-1", "--yes",
		"--permissions", `[{"account_id":"acc-9","role_id":"role-9"}]`)
	if err != nil {
		t.Fatalf("account-permission set failed: %v (%s)", err, out)
	}

	if !strings.Contains(got.Body, `"acc-9"`) {
		t.Errorf("unexpected body: %s", got.Body)
	}
}

func TestUserPermissionSet_RequiresABody(t *testing.T) {
	isolateConfig(t)
	seedCredentials(t)
	startAPI(t, http.StatusOK, `{"status":"ok"}`)

	_, err := runCLI(t, "", "org", "user", "account-permission", "set", "usr-1", "--yes")
	if err == nil || !strings.Contains(err.Error(), "request body is required") {
		t.Errorf("unexpected error: %v", err)
	}
}
