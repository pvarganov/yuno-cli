package model_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/pvarganov/yuno-cli/internal/model"
)

func TestOrgAccountViewFlattensTheAccount(t *testing.T) {
	t.Parallel()

	var account model.OrgAccount
	if err := json.Unmarshal([]byte(`{"id":"acc-1","name":"BR","account_group_id":"grp-1",
		"created_at":"2026-09-16T10:00:00Z","unknown":"kept"}`), &account); err != nil {
		t.Fatalf("decode account: %v", err)
	}

	view := account.View()
	if view.ID != "acc-1" || view.Name != "BR" || view.AccountGroupID != "grp-1" {
		t.Errorf("unexpected view: %+v", view)
	}

	data, err := json.Marshal(account)
	if err != nil {
		t.Fatalf("marshal account: %v", err)
	}

	if !strings.Contains(string(data), `"unknown"`) {
		t.Errorf("marshalling should replay the raw response, got: %s", data)
	}
}

func TestOrgAccountMarshalsWithoutARawBody(t *testing.T) {
	t.Parallel()

	data, err := json.Marshal(model.OrgAccount{ID: "acc-1", Name: "BR"})
	if err != nil {
		t.Fatalf("marshal account: %v", err)
	}

	if !strings.Contains(string(data), `"acc-1"`) {
		t.Errorf("unexpected json: %s", data)
	}
}

func TestOrgAccountGroupViewFlattensTheGroup(t *testing.T) {
	t.Parallel()

	var group model.OrgAccountGroup
	if err := json.Unmarshal([]byte(`{"id":"grp-1","name":"Overgear","merchant_id":"mer-1"}`), &group); err != nil {
		t.Fatalf("decode group: %v", err)
	}

	views := model.OrgAccountGroupViews([]model.OrgAccountGroup{group})
	if len(views) != 1 || views[0].MerchantID != "mer-1" || views[0].Name != "Overgear" {
		t.Errorf("unexpected views: %+v", views)
	}
}

func TestOrgUserViewCountsThePermissions(t *testing.T) {
	t.Parallel()

	var user model.OrgUser
	if err := json.Unmarshal([]byte(`{"id":"usr-1","email":"ops@example.com","first_name":"Ops",
		"account_permissions":[{"account_id":"acc-1","role_id":"role-1"},{"account_id":"acc-2","role_id":"role-1"}],
		"account_group_permissions":[{"account_group_id":"grp-1","role_id":"role-2"}]}`), &user); err != nil {
		t.Fatalf("decode user: %v", err)
	}

	view := user.View()
	if view.Accounts != "2" || view.Groups != "1" || view.Email != "ops@example.com" {
		t.Errorf("unexpected view: %+v", view)
	}
}

func TestOrgRoleViewRendersTheNullableDescription(t *testing.T) {
	t.Parallel()

	var withNull, withText model.OrgRole

	if err := json.Unmarshal([]byte(`{"id":"role-1","name":"Support","description":null,
		"admin":true,"permission_ids":["a","b"],"role_type":"ACCOUNT"}`), &withNull); err != nil {
		t.Fatalf("decode role: %v", err)
	}

	if err := json.Unmarshal([]byte(`{"id":"role-2","description":"read only"}`), &withText); err != nil {
		t.Fatalf("decode role: %v", err)
	}

	views := model.OrgRoleViews([]model.OrgRole{withNull, withText})
	if views[0].Description != "" || views[0].Admin != "true" || views[0].Permissions != "2" {
		t.Errorf("unexpected view: %+v", views[0])
	}

	if views[1].Description != "read only" {
		t.Errorf("unexpected view: %+v", views[1])
	}
}

func TestOrgPermissionViewsFlattenEveryGroup(t *testing.T) {
	t.Parallel()

	var groups []model.OrgPermissionGroup
	if err := json.Unmarshal([]byte(`[{"title":"Payments","organization_only":false,
		"permissions_catalog":[{"id":"payments.read","title":"Read","action_type":"READ"},
		{"id":"payments.write","title":"Write","action_type":"WRITE"}]},
		{"title":"Empty","permissions_catalog":[]}]`), &groups); err != nil {
		t.Fatalf("decode catalog: %v", err)
	}

	views := model.OrgPermissionViews(groups)
	if len(views) != 2 || views[0].Group != "Payments" || views[1].ID != "payments.write" {
		t.Errorf("unexpected views: %+v", views)
	}
}

func TestOrgAccountPermissionsViews(t *testing.T) {
	t.Parallel()

	var permissions model.OrgAccountPermissions
	if err := json.Unmarshal([]byte(`{"permissions":[{"account_id":"acc-1","account_name":"BR",
		"role_id":"role-1","role_name":"Support"}]}`), &permissions); err != nil {
		t.Fatalf("decode permissions: %v", err)
	}

	views := permissions.Views()
	if len(views) != 1 || views[0].AccountName != "BR" || views[0].RoleName != "Support" {
		t.Errorf("unexpected views: %+v", views)
	}
}

func TestOrgAccountGroupPermissionsViewsRenderTheAccountCount(t *testing.T) {
	t.Parallel()

	var permissions model.OrgAccountGroupPermissions
	if err := json.Unmarshal([]byte(`{"permissions":[{"account_group_id":"grp-1",
		"account_group_name":"Overgear","role_id":"role-1","role_name":"Admin","accounts_count":3}]}`),
		&permissions); err != nil {
		t.Fatalf("decode permissions: %v", err)
	}

	views := permissions.Views()
	if len(views) != 1 || views[0].Accounts != "3" || views[0].AccountGroupName != "Overgear" {
		t.Errorf("unexpected views: %+v", views)
	}
}

func TestOrgAuthTokenView(t *testing.T) {
	t.Parallel()

	token := model.OrgAuthToken{AccessToken: "tok-123", TokenType: "Bearer", ExpiresIn: 3600}

	view := token.View()
	if view.ExpiresIn != "3600" || view.AccessToken != "tok-123" || view.TokenType != "Bearer" {
		t.Errorf("unexpected view: %+v", view)
	}
}
