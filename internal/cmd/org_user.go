package cmd

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"github.com/spf13/cobra"

	"github.com/pvarganov/yuno-cli/internal/api"
	"github.com/pvarganov/yuno-cli/internal/model"
)

// orgUserFields are the named body fields of `org user create` and `update`.
// The two permission lists are arrays of objects, so they come from --file or
// from their JSON flags.
var orgUserFields = []FieldFlag{
	{Flag: "email", Field: "email", Kind: FieldString, Usage: "email of the user"},
	{Flag: "first-name", Field: "first_name", Kind: FieldString, Usage: "first name of the user"},
	{Flag: "last-name", Field: "last_name", Kind: FieldString, Usage: "last name of the user"},
	{
		Flag: "account-permissions", Field: "account_permissions", Kind: FieldJSON,
		Usage: "account permissions as a JSON array",
	},
	{
		Flag: "account-group-permissions", Field: "account_group_permissions", Kind: FieldJSON,
		Usage: "account group permissions as a JSON array",
	},
}

// orgUserFilters are the query filters of `org user list`.
var orgUserFilters = []struct {
	Flag  string
	Query string
	Usage string
}{
	{Flag: "account-id", Query: "account_id", Usage: "only the users with a permission on this account"},
	{Flag: "account-group-id", Query: "account_group_id", Usage: "only the users with a permission on this group"},
}

// newOrgUserCommand builds the `org user` subtree, permissions included.
func newOrgUserCommand() *cobra.Command {
	user := &cobra.Command{
		Use:     "user",
		Short:   "Manage the users of the organization",
		Aliases: []string{"users"},
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return cmd.Help()
		},
	}

	user.AddCommand(
		newOrgUserListCommand(),
		newOrgUserGetCommand(),
		newOrgUserCreateCommand(),
		newOrgUserUpdateCommand(),
		newOrgUserDeleteCommand(),
		newOrgUserFindByEmailCommand(),
		newUserPermissionCommand(accountPermissionKind()),
		newUserPermissionCommand(accountGroupPermissionKind()),
	)

	return user
}

func newOrgUserListCommand() *cobra.Command {
	command := &cobra.Command{
		Use:     "list",
		Short:   "List the users of the organization",
		Example: "  yuno-cli org user list --account-id acc-1",
		Args:    cobra.NoArgs,
		RunE:    runOrgUserList,
	}

	for _, filter := range orgUserFilters {
		command.Flags().String(filter.Flag, "", filter.Usage)
	}

	registerPagingFlags(command)

	return command
}

func runOrgUserList(cmd *cobra.Command, _ []string) error {
	users, err := listOrgUsers(cmd)
	if err != nil {
		return err
	}

	return printResult(cmd, users, model.OrgUserViews(users))
}

// listOrgUsers fetches the users a list command asked for.
func listOrgUsers(cmd *cobra.Command) ([]model.OrgUser, error) {
	client, err := newClientFromFlags(cmd)
	if err != nil {
		return nil, err
	}

	filters := url.Values{}

	for _, filter := range orgUserFilters {
		if value := flagString(cmd, filter.Flag); value != "" {
			filters.Set(filter.Query, value)
		}
	}

	limit, pageSize := pagingFlags(cmd)

	users, err := client.ListOrgUsers(cmd.Context(), filters, limit, pageSize)
	if err != nil {
		return nil, scopeHint(err, orgScopes)
	}

	return users, nil
}

// newOrgUserFindByEmailCommand filters the users client-side: Yuno has no
// lookup endpoint for an email, but the list response carries one.
func newOrgUserFindByEmailCommand() *cobra.Command {
	command := &cobra.Command{
		Use:   "find-by-email <email>",
		Short: "Find the user with this email",
		Long: "Find the user with this email.\n\n" +
			"Yuno has no lookup endpoint for an email, so this lists the users and filters " +
			"them locally.",
		Example: "  yuno-cli org user find-by-email ops@example.com",
		Args:    cobra.ExactArgs(1),
		RunE:    runOrgUserFindByEmail,
	}

	for _, filter := range orgUserFilters {
		command.Flags().String(filter.Flag, "", filter.Usage)
	}

	registerPagingFlags(command)

	return command
}

func runOrgUserFindByEmail(cmd *cobra.Command, args []string) error {
	users, err := listOrgUsers(cmd)
	if err != nil {
		return err
	}

	matched := make([]model.OrgUser, 0, len(users))

	for i := range users {
		if strings.EqualFold(users[i].Email, args[0]) {
			matched = append(matched, users[i])
		}
	}

	return printResult(cmd, matched, model.OrgUserViews(matched))
}

func newOrgUserGetCommand() *cobra.Command {
	return &cobra.Command{
		Use:     "get <user_id>",
		Short:   "Retrieve one user",
		Example: "  yuno-cli org user get usr-1 --json",
		Args:    cobra.ExactArgs(1),
		RunE:    runOrgUserGet,
	}
}

func runOrgUserGet(cmd *cobra.Command, args []string) error {
	client, err := newClientFromFlags(cmd)
	if err != nil {
		return err
	}

	user, err := client.GetOrgUser(cmd.Context(), args[0])
	if err != nil {
		return scopeHint(err, orgScopes)
	}

	return printOrgUser(cmd, user)
}

func newOrgUserCreateCommand() *cobra.Command {
	command := &cobra.Command{
		Use:   "create",
		Short: "Invite a user into the organization",
		Example: "  yuno-cli org user create --email ops@example.com --first-name Ops " +
			`--account-permissions '[{"account_id":"acc-1","role_id":"role-1"}]' --yes`,
		Args: cobra.NoArgs,
		RunE: runOrgUserCreate,
	}

	registerWriteFlags(command, orgUserFields)

	return command
}

func runOrgUserCreate(cmd *cobra.Command, _ []string) error {
	body, err := requireBody(cmd, orgUserFields)
	if err != nil {
		return err
	}

	client, err := newClientFromFlags(cmd)
	if err != nil {
		return err
	}

	user, err := client.CreateOrgUser(cmd.Context(), body)
	if err != nil {
		return scopeHint(err, orgScopes)
	}

	return printOrgUser(cmd, user)
}

func newOrgUserUpdateCommand() *cobra.Command {
	command := &cobra.Command{
		Use:     "update <user_id>",
		Short:   "Update a user",
		Example: "  yuno-cli org user update usr-1 --first-name Ops --yes",
		Args:    cobra.ExactArgs(1),
		RunE:    runOrgUserUpdate,
	}

	registerWriteFlags(command, orgUserFields)

	return command
}

func runOrgUserUpdate(cmd *cobra.Command, args []string) error {
	body, err := requireBody(cmd, orgUserFields)
	if err != nil {
		return err
	}

	client, err := newClientFromFlags(cmd)
	if err != nil {
		return err
	}

	user, err := client.UpdateOrgUser(cmd.Context(), args[0], body)
	if err != nil {
		return scopeHint(err, orgScopes)
	}

	return printOrgUser(cmd, user)
}

func newOrgUserDeleteCommand() *cobra.Command {
	command := &cobra.Command{
		Use:     "delete <user_id>",
		Short:   "Remove a user from the organization",
		Example: "  yuno-cli org user delete usr-1 --yes",
		Args:    cobra.ExactArgs(1),
		RunE:    runOrgUserDelete,
	}

	command.Flags().String("idempotency-key", "", "pin the X-Idempotency-Key of the request")

	return command
}

func runOrgUserDelete(cmd *cobra.Command, args []string) error {
	client, err := newClientFromFlags(cmd)
	if err != nil {
		return err
	}

	data, err := client.DeleteOrgUser(cmd.Context(), args[0])
	if err != nil {
		return scopeHint(err, orgScopes)
	}

	return printJSON(cmd, data)
}

// printOrgUser renders one user: the whole response as JSON, a single table row
// otherwise.
func printOrgUser(cmd *cobra.Command, user *model.OrgUser) error {
	return printResult(cmd, user, []model.OrgUserView{user.View()})
}

// permissionKind describes one of the two permission sub-resources of a user.
// Both speak the same four verbs, so the subtree is built once and the typed
// client calls are supplied per kind.
type permissionKind struct {
	use       string
	aliases   []string
	short     string
	idArg     string
	bodyField string
	grantKey  string
	get       func(ctx context.Context, c *api.Client, userID string) (payload, views any, err error)
	update    func(ctx context.Context, c *api.Client, userID string, body any) (payload, views any, err error)
	replace   func(ctx context.Context, c *api.Client, userID string, body any) ([]byte, error)
	remove    func(ctx context.Context, c *api.Client, userID, id string) ([]byte, error)
}

// accountPermissionKind wires the `account-permission` subtree.
func accountPermissionKind() permissionKind {
	return permissionKind{
		use:       "account-permission",
		aliases:   []string{"account-permissions"},
		short:     "Manage the roles a user holds on single accounts",
		idArg:     "account_id",
		bodyField: "account_permissions",
		grantKey:  "account_id",
		get: func(ctx context.Context, c *api.Client, userID string) (any, any, error) {
			permissions, err := c.GetUserAccountPermissions(ctx, userID)
			if err != nil {
				return nil, nil, err
			}

			views := permissions.Views()

			return permissions, views, nil
		},
		update: func(ctx context.Context, c *api.Client, userID string, body any) (any, any, error) {
			permissions, err := c.UpdateUserAccountPermissions(ctx, userID, body)
			if err != nil {
				return nil, nil, err
			}

			views := permissions.Views()

			return permissions, views, nil
		},
		replace: func(ctx context.Context, c *api.Client, userID string, body any) ([]byte, error) {
			return c.ReplaceUserAccountPermissions(ctx, userID, body)
		},
		remove: func(ctx context.Context, c *api.Client, userID, id string) ([]byte, error) {
			return c.DeleteUserAccountPermission(ctx, userID, id)
		},
	}
}

// accountGroupPermissionKind wires the `account-group-permission` subtree.
func accountGroupPermissionKind() permissionKind {
	return permissionKind{
		use:       "account-group-permission",
		aliases:   []string{"account-group-permissions"},
		short:     "Manage the roles a user holds on account groups",
		idArg:     "account_group_id",
		bodyField: "account_group_permissions",
		grantKey:  "account_group_id",
		get: func(ctx context.Context, c *api.Client, userID string) (any, any, error) {
			permissions, err := c.GetUserAccountGroupPermissions(ctx, userID)
			if err != nil {
				return nil, nil, err
			}

			views := permissions.Views()

			return permissions, views, nil
		},
		update: func(ctx context.Context, c *api.Client, userID string, body any) (any, any, error) {
			permissions, err := c.UpdateUserAccountGroupPermissions(ctx, userID, body)
			if err != nil {
				return nil, nil, err
			}

			views := permissions.Views()

			return permissions, views, nil
		},
		replace: func(ctx context.Context, c *api.Client, userID string, body any) ([]byte, error) {
			return c.ReplaceUserAccountGroupPermissions(ctx, userID, body)
		},
		remove: func(ctx context.Context, c *api.Client, userID, id string) ([]byte, error) {
			return c.DeleteUserAccountGroupPermission(ctx, userID, id)
		},
	}
}

// newUserPermissionCommand builds the four verbs of one permission kind.
//
//nolint:gocritic // hugeParam: the kind is a small descriptor built once per subtree
func newUserPermissionCommand(kind permissionKind) *cobra.Command {
	permission := &cobra.Command{
		Use:     kind.use,
		Short:   kind.short,
		Aliases: kind.aliases,
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return cmd.Help()
		},
	}

	list := &cobra.Command{
		Use:     "list <user_id>",
		Short:   "List the permissions of a user",
		Example: "  yuno-cli org user " + kind.use + " list usr-1",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runPermissionList(cmd, &kind, args)
		},
	}

	remove := &cobra.Command{
		Use:     "delete <user_id> <" + kind.idArg + ">",
		Short:   "Revoke the permission of a user on one resource",
		Example: "  yuno-cli org user " + kind.use + " delete usr-1 res-1 --yes",
		Args:    cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runPermissionDelete(cmd, &kind, args)
		},
	}

	remove.Flags().String("idempotency-key", "", "pin the X-Idempotency-Key of the request")

	permission.AddCommand(list, newPermissionSetCommand(&kind), newPermissionUpdateCommand(&kind), remove)

	return permission
}

// newPermissionSetCommand builds the PUT verb, which replaces the whole set.
func newPermissionSetCommand(kind *permissionKind) *cobra.Command {
	command := &cobra.Command{
		Use:   "set <user_id>",
		Short: "Replace every permission of a user",
		Example: "  yuno-cli org user " + kind.use + " set usr-1 " +
			"--grant res-1=role-1 --yes",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runPermissionSet(cmd, kind, args)
		},
	}

	registerPermissionFlags(command, kind)

	return command
}

// newPermissionUpdateCommand builds the PATCH verb, which merges into the set.
func newPermissionUpdateCommand(kind *permissionKind) *cobra.Command {
	command := &cobra.Command{
		Use:   "update <user_id>",
		Short: "Add to the permissions of a user",
		Example: "  yuno-cli org user " + kind.use + " update usr-1 " +
			"--grant res-1=role-1 --yes",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runPermissionUpdate(cmd, kind, args)
		},
	}

	registerPermissionFlags(command, kind)

	return command
}

// registerPermissionFlags declares the body flags of a permission write: the
// repeatable `--grant <id>=<role_id>` shorthand plus the usual body sources.
func registerPermissionFlags(cmd *cobra.Command, kind *permissionKind) {
	registerWriteFlags(cmd, permissionFields(kind))
	cmd.Flags().StringArray("grant", nil,
		"permission as <"+kind.idArg+">=<role_id>, repeat for more")
}

// permissionFields is the whole-array flag of a permission write.
func permissionFields(kind *permissionKind) []FieldFlag {
	return []FieldFlag{
		{
			Flag: "permissions", Field: kind.bodyField, Kind: FieldJSON,
			Usage: "whole permission list as a JSON array",
		},
	}
}

// permissionBody builds the payload of a permission write. The `--grant` pairs
// win when they are given; otherwise the usual body sources apply.
func permissionBody(cmd *cobra.Command, kind *permissionKind) (any, error) {
	grants, err := parseGrants(cmd, kind.grantKey)
	if err != nil {
		return nil, err
	}

	if len(grants) > 0 {
		return map[string]any{kind.bodyField: grants}, nil
	}

	body, err := requireBody(cmd, permissionFields(kind))
	if err != nil {
		return nil, err
	}

	return body, nil
}

// parseGrants turns the `--grant <id>=<role_id>` pairs into permission objects.
func parseGrants(cmd *cobra.Command, idKey string) ([]any, error) {
	values := flagStringArray(cmd, "grant")

	grants := make([]any, 0, len(values))

	for _, value := range values {
		id, roleID, ok := strings.Cut(value, "=")

		id = strings.TrimSpace(id)
		roleID = strings.TrimSpace(roleID)

		if !ok || id == "" || roleID == "" {
			return nil, fmt.Errorf("--grant %q: want <%s>=<role_id>", value, idKey)
		}

		grants = append(grants, map[string]any{idKey: id, "role_id": roleID})
	}

	return grants, nil
}

func runPermissionList(cmd *cobra.Command, kind *permissionKind, args []string) error {
	client, err := newClientFromFlags(cmd)
	if err != nil {
		return err
	}

	payload, views, err := kind.get(cmd.Context(), client, args[0])
	if err != nil {
		return scopeHint(err, orgScopes)
	}

	return printResult(cmd, payload, views)
}

func runPermissionSet(cmd *cobra.Command, kind *permissionKind, args []string) error {
	body, err := permissionBody(cmd, kind)
	if err != nil {
		return err
	}

	client, err := newClientFromFlags(cmd)
	if err != nil {
		return err
	}

	data, err := kind.replace(cmd.Context(), client, args[0], body)
	if err != nil {
		return scopeHint(err, orgScopes)
	}

	return printJSON(cmd, data)
}

func runPermissionUpdate(cmd *cobra.Command, kind *permissionKind, args []string) error {
	body, err := permissionBody(cmd, kind)
	if err != nil {
		return err
	}

	client, err := newClientFromFlags(cmd)
	if err != nil {
		return err
	}

	payload, views, err := kind.update(cmd.Context(), client, args[0], body)
	if err != nil {
		return scopeHint(err, orgScopes)
	}

	return printResult(cmd, payload, views)
}

func runPermissionDelete(cmd *cobra.Command, kind *permissionKind, args []string) error {
	client, err := newClientFromFlags(cmd)
	if err != nil {
		return err
	}

	data, err := kind.remove(cmd.Context(), client, args[0], args[1])
	if err != nil {
		return scopeHint(err, orgScopes)
	}

	return printJSON(cmd, data)
}
