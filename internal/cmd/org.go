package cmd

import (
	"github.com/spf13/cobra"

	"github.com/pvarganov/yuno-cli/internal/model"
)

// orgScopes are the scopes a 403 on the organization commands usually asks for.
const orgScopes = "organizations:read/organizations:write"

// orgRoleFields are the named body fields of `org role create` and `update`.
var orgRoleFields = []FieldFlag{
	{Flag: "name", Field: "name", Kind: FieldString, Usage: "name of the role"},
	{Flag: "description", Field: "description", Kind: FieldString, Usage: "description of the role"},
	{
		Flag: "permission", Field: "permission_ids", Kind: FieldStringSlice,
		Usage: "permission the role grants, repeat for more",
	},
	{
		Flag: "testing-permission", Field: "testing_permission_ids", Kind: FieldStringSlice,
		Usage: "permission the role grants in testing mode, repeat for more",
	},
	{Flag: "role-type", Field: "role_type", Kind: FieldString, Usage: "role type, e.g. ACCOUNT or ACCOUNT_GROUP"},
}

// newOrgCommand builds the `org` command tree: the accounts, account groups,
// users, roles and permissions of the organization.
func newOrgCommand() *cobra.Command {
	org := &cobra.Command{
		Use:     "org",
		Short:   "Manage the accounts, users, roles and permissions of the organization",
		Aliases: []string{"organization", "organizations"},
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return cmd.Help()
		},
	}

	org.AddCommand(
		newOrgAccountCommand(),
		newOrgAccountGroupCommand(),
		newOrgUserCommand(),
		newOrgRoleCommand(),
		newOrgPermissionsCatalogCommand(),
		newOrgAuthenticateCommand(),
	)

	return org
}

// newOrgRoleCommand builds the `org role` subtree. Yuno has no endpoint for a
// single role, so there is no `get`: `list` is the only way to read one.
func newOrgRoleCommand() *cobra.Command {
	role := &cobra.Command{
		Use:     "role",
		Short:   "Manage the roles of the organization",
		Aliases: []string{"roles"},
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return cmd.Help()
		},
	}

	role.AddCommand(
		newOrgRoleListCommand(),
		newOrgRoleCreateCommand(),
		newOrgRoleUpdateCommand(),
		newOrgRoleDeleteCommand(),
	)

	return role
}

func newOrgRoleListCommand() *cobra.Command {
	return &cobra.Command{
		Annotations: apiOperations("GET /organizations/roles"),
		Use:         "list",
		Short:       "List the roles of the organization",
		Example:     "  yuno-cli org role list",
		Args:        cobra.NoArgs,
		RunE:        runOrgRoleList,
	}
}

func runOrgRoleList(cmd *cobra.Command, _ []string) error {
	client, err := newClientFromFlags(cmd)
	if err != nil {
		return err
	}

	roles, err := client.ListOrgRoles(cmd.Context())
	if err != nil {
		return scopeHint(err, orgScopes)
	}

	return printResult(cmd, roles, model.OrgRoleViews(roles))
}

func newOrgRoleCreateCommand() *cobra.Command {
	command := &cobra.Command{
		Annotations: apiOperations("POST /organizations/roles"),
		Use:         "create",
		Short:       "Create a role",
		Example:     "  yuno-cli org role create --name Support --permission payments.read --yes",
		Args:        cobra.NoArgs,
		RunE:        runOrgRoleCreate,
	}

	registerWriteFlags(command, orgRoleFields)

	return command
}

func runOrgRoleCreate(cmd *cobra.Command, _ []string) error {
	body, err := requireBody(cmd, orgRoleFields)
	if err != nil {
		return err
	}

	client, err := newClientFromFlags(cmd)
	if err != nil {
		return err
	}

	role, err := client.CreateOrgRole(cmd.Context(), body)
	if err != nil {
		return scopeHint(err, orgScopes)
	}

	return printOrgRole(cmd, role)
}

func newOrgRoleUpdateCommand() *cobra.Command {
	command := &cobra.Command{
		Annotations: apiOperations("PATCH /organizations/roles/{role_id}"),
		Use:         "update <role_id>",
		Short:       "Update a role",
		Example:     "  yuno-cli org role update role-1 --name Support --yes",
		Args:        cobra.ExactArgs(1),
		RunE:        runOrgRoleUpdate,
	}

	registerWriteFlags(command, orgRoleFields)

	return command
}

func runOrgRoleUpdate(cmd *cobra.Command, args []string) error {
	body, err := requireBody(cmd, orgRoleFields)
	if err != nil {
		return err
	}

	client, err := newClientFromFlags(cmd)
	if err != nil {
		return err
	}

	role, err := client.UpdateOrgRole(cmd.Context(), args[0], body)
	if err != nil {
		return scopeHint(err, orgScopes)
	}

	return printOrgRole(cmd, role)
}

func newOrgRoleDeleteCommand() *cobra.Command {
	command := &cobra.Command{
		Annotations: apiOperations("DELETE /organizations/roles/{role_id}"),
		Use:         "delete <role_id>",
		Short:       "Delete a role",
		Example:     "  yuno-cli org role delete role-1 --yes",
		Args:        cobra.ExactArgs(1),
		RunE:        runOrgRoleDelete,
	}

	return command
}

func runOrgRoleDelete(cmd *cobra.Command, args []string) error {
	client, err := newClientFromFlags(cmd)
	if err != nil {
		return err
	}

	data, err := client.DeleteOrgRole(cmd.Context(), args[0])
	if err != nil {
		return scopeHint(err, orgScopes)
	}

	return printJSON(cmd, data)
}

func newOrgPermissionsCatalogCommand() *cobra.Command {
	return &cobra.Command{
		Annotations: apiOperations("GET /organizations/permissions-catalog"),
		Use:         "permissions-catalog",
		Short:       "List every permission a role may grant",
		Aliases:     []string{"permissions"},
		Example:     "  yuno-cli org permissions-catalog --json",
		Args:        cobra.NoArgs,
		RunE:        runOrgPermissionsCatalog,
	}
}

func runOrgPermissionsCatalog(cmd *cobra.Command, _ []string) error {
	client, err := newClientFromFlags(cmd)
	if err != nil {
		return err
	}

	groups, err := client.ListPermissionsCatalog(cmd.Context())
	if err != nil {
		return scopeHint(err, orgScopes)
	}

	return printResult(cmd, groups, model.OrgPermissionViews(groups))
}

func newOrgAuthenticateCommand() *cobra.Command {
	command := &cobra.Command{
		Annotations: apiOperations("POST /organizations/authenticate"),
		Use:         "authenticate <user_id>",
		Short:       "Issue a whitelabel access token for one user",
		Long: "Issue a whitelabel access token for one user.\n\n" +
			"The token is masked in the table output; use --json to read it in full.",
		Example: "  yuno-cli org authenticate usr-1 --json",
		Args:    cobra.ExactArgs(1),
		RunE:    runOrgAuthenticate,
	}

	command.Flags().String("idempotency-key", "", "pin the X-Idempotency-Key of the request")

	return command
}

func runOrgAuthenticate(cmd *cobra.Command, args []string) error {
	client, err := newClientFromFlags(cmd)
	if err != nil {
		return err
	}

	token, err := client.AuthenticateOrgUser(cmd.Context(), map[string]any{"user_id": args[0]})
	if err != nil {
		return scopeHint(err, orgScopes)
	}

	return printResult(cmd, token, []model.OrgAuthTokenView{token.View()})
}

// printOrgRole renders one role: the whole response as JSON, a single table row
// otherwise.
func printOrgRole(cmd *cobra.Command, role *model.OrgRole) error {
	return printResult(cmd, role, []model.OrgRoleView{role.View()})
}
