package cmd

import (
	"strings"

	"github.com/spf13/cobra"

	"github.com/pvarganov/yuno-cli/internal/model"
)

// orgAccountFields are the named body fields of the account commands. An
// account only carries a name; everything else is derived from its group.
var orgAccountFields = []FieldFlag{
	{Flag: "name", Field: "name", Kind: FieldString, Usage: "name of the account"},
}

// orgAccountGroupFields are the named body fields of the account group commands.
var orgAccountGroupFields = []FieldFlag{
	{Flag: "name", Field: "name", Kind: FieldString, Usage: "name of the account group"},
	{Flag: "merchant-id", Field: "merchant_id", Kind: FieldString, Usage: "merchant the group belongs to"},
}

// newOrgAccountCommand builds the `org account` subtree.
func newOrgAccountCommand() *cobra.Command {
	account := &cobra.Command{
		Use:     "account",
		Short:   "Manage the accounts of the organization",
		Aliases: []string{"accounts"},
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return cmd.Help()
		},
	}

	account.AddCommand(
		newOrgAccountListCommand(),
		newOrgAccountGetCommand(),
		newOrgAccountCreateCommand(),
		newOrgAccountUpdateCommand(),
		newOrgAccountDeleteCommand(),
	)

	return account
}

func newOrgAccountListCommand() *cobra.Command {
	command := &cobra.Command{
		Annotations: apiOperations("GET /organizations/accounts"),
		Use:         "list",
		Short:       "List the accounts of the organization",
		Example:     "  yuno-cli org account list --limit 20",
		Args:        cobra.NoArgs,
		RunE:        runOrgAccountList,
	}

	registerPagingFlags(command)

	return command
}

func runOrgAccountList(cmd *cobra.Command, _ []string) error {
	client, err := newClientFromFlags(cmd)
	if err != nil {
		return err
	}

	limit, pageSize := pagingFlags(cmd)

	accounts, err := client.ListOrgAccounts(cmd.Context(), limit, pageSize)
	if err != nil {
		return scopeHint(err, orgScopes)
	}

	return printResult(cmd, accounts, model.OrgAccountViews(accounts))
}

func newOrgAccountGetCommand() *cobra.Command {
	return &cobra.Command{
		Annotations: apiOperations("GET /organizations/accounts/{account_id}"),
		Use:         "get <account_id>",
		Short:       "Retrieve one account",
		Example:     "  yuno-cli org account get acc-1 --json",
		Args:        cobra.ExactArgs(1),
		RunE:        runOrgAccountGet,
	}
}

func runOrgAccountGet(cmd *cobra.Command, args []string) error {
	client, err := newClientFromFlags(cmd)
	if err != nil {
		return err
	}

	account, err := client.GetOrgAccount(cmd.Context(), args[0])
	if err != nil {
		return scopeHint(err, orgScopes)
	}

	return printOrgAccount(cmd, account)
}

// newOrgAccountCreateCommand creates an account. Yuno only creates accounts
// inside an account group, which is why the group id is a positional argument.
func newOrgAccountCreateCommand() *cobra.Command {
	command := &cobra.Command{
		Annotations: apiOperations("POST /organizations/account-groups/{group_id}/accounts"),
		Use:         "create <account_group_id>",
		Short:       "Create an account inside an account group",
		Example:     "  yuno-cli org account create grp-1 --name Overgear-BR --yes",
		Args:        cobra.ExactArgs(1),
		RunE:        runOrgAccountCreate,
	}

	registerWriteFlags(command, orgAccountFields)

	return command
}

func runOrgAccountCreate(cmd *cobra.Command, args []string) error {
	body, err := requireBody(cmd, orgAccountFields)
	if err != nil {
		return err
	}

	client, err := newClientFromFlags(cmd)
	if err != nil {
		return err
	}

	account, err := client.CreateOrgAccount(cmd.Context(), args[0], body)
	if err != nil {
		return scopeHint(err, orgScopes)
	}

	return printOrgAccount(cmd, account)
}

func newOrgAccountUpdateCommand() *cobra.Command {
	command := &cobra.Command{
		Annotations: apiOperations("PATCH /organizations/accounts/{account_id}"),
		Use:         "update <account_id>",
		Short:       "Update an account",
		Example:     "  yuno-cli org account update acc-1 --name Overgear-BR --yes",
		Args:        cobra.ExactArgs(1),
		RunE:        runOrgAccountUpdate,
	}

	registerWriteFlags(command, orgAccountFields)

	return command
}

func runOrgAccountUpdate(cmd *cobra.Command, args []string) error {
	body, err := requireBody(cmd, orgAccountFields)
	if err != nil {
		return err
	}

	client, err := newClientFromFlags(cmd)
	if err != nil {
		return err
	}

	account, err := client.UpdateOrgAccount(cmd.Context(), args[0], body)
	if err != nil {
		return scopeHint(err, orgScopes)
	}

	return printOrgAccount(cmd, account)
}

func newOrgAccountDeleteCommand() *cobra.Command {
	command := &cobra.Command{
		Annotations: apiOperations("DELETE /organizations/accounts/{account_id}"),
		Use:         "delete <account_id>",
		Short:       "Delete an account",
		Example:     "  yuno-cli org account delete acc-1 --yes",
		Args:        cobra.ExactArgs(1),
		RunE:        runOrgAccountDelete,
	}

	return command
}

func runOrgAccountDelete(cmd *cobra.Command, args []string) error {
	client, err := newClientFromFlags(cmd)
	if err != nil {
		return err
	}

	data, err := client.DeleteOrgAccount(cmd.Context(), args[0])
	if err != nil {
		return scopeHint(err, orgScopes)
	}

	return printJSON(cmd, data)
}

// newOrgAccountGroupCommand builds the `org account-group` subtree.
func newOrgAccountGroupCommand() *cobra.Command {
	group := &cobra.Command{
		Use:     "account-group",
		Short:   "Manage the account groups of the organization",
		Aliases: []string{"account-groups", "group", "groups"},
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return cmd.Help()
		},
	}

	group.AddCommand(
		newAccountGroupListCommand(),
		newAccountGroupGetCommand(),
		newAccountGroupCreateCommand(),
		newAccountGroupUpdateCommand(),
		newAccountGroupDeleteCommand(),
		newAccountGroupAccountsCommand(),
		newAccountGroupAddAccountCommand(),
		newAccountGroupFindByMerchantCommand(),
	)

	return group
}

func newAccountGroupListCommand() *cobra.Command {
	command := &cobra.Command{
		Annotations: apiOperations("GET /organizations/account-groups"),
		Use:         "list",
		Short:       "List the account groups of the organization",
		Example:     "  yuno-cli org account-group list",
		Args:        cobra.NoArgs,
		RunE:        runAccountGroupList,
	}

	registerPagingFlags(command)

	return command
}

func runAccountGroupList(cmd *cobra.Command, _ []string) error {
	groups, err := listAccountGroups(cmd)
	if err != nil {
		return err
	}

	return printResult(cmd, groups, model.OrgAccountGroupViews(groups))
}

// listAccountGroups fetches the account groups a list command asked for.
func listAccountGroups(cmd *cobra.Command) ([]model.OrgAccountGroup, error) {
	client, err := newClientFromFlags(cmd)
	if err != nil {
		return nil, err
	}

	limit, pageSize := pagingFlags(cmd)

	groups, err := client.ListAccountGroups(cmd.Context(), limit, pageSize)
	if err != nil {
		return nil, scopeHint(err, orgScopes)
	}

	return groups, nil
}

func newAccountGroupGetCommand() *cobra.Command {
	return &cobra.Command{
		Annotations: apiOperations("GET /organizations/account-groups/{group_id}"),
		Use:         "get <account_group_id>",
		Short:       "Retrieve one account group",
		Example:     "  yuno-cli org account-group get grp-1 --json",
		Args:        cobra.ExactArgs(1),
		RunE:        runAccountGroupGet,
	}
}

func runAccountGroupGet(cmd *cobra.Command, args []string) error {
	client, err := newClientFromFlags(cmd)
	if err != nil {
		return err
	}

	group, err := client.GetAccountGroup(cmd.Context(), args[0])
	if err != nil {
		return scopeHint(err, orgScopes)
	}

	return printAccountGroup(cmd, group)
}

// newAccountGroupFindByMerchantCommand filters the groups client-side: Yuno has
// no lookup endpoint for a merchant id, but the list response carries one.
func newAccountGroupFindByMerchantCommand() *cobra.Command {
	command := &cobra.Command{
		Annotations: apiOperations("GET /organizations/account-groups"),
		Use:         "find-by-merchant-id <merchant_id>",
		Short:       "Find the account groups of one merchant",
		Long: "Find the account groups of one merchant.\n\n" +
			"Yuno has no lookup endpoint for a merchant id, so this lists the account groups " +
			"and filters them locally.",
		Example: "  yuno-cli org account-group find-by-merchant-id mer-1",
		Args:    cobra.ExactArgs(1),
		RunE:    runAccountGroupFindByMerchant,
	}

	registerPagingFlags(command)

	return command
}

func runAccountGroupFindByMerchant(cmd *cobra.Command, args []string) error {
	groups, err := listAccountGroups(cmd)
	if err != nil {
		return err
	}

	matched := make([]model.OrgAccountGroup, 0, len(groups))

	for i := range groups {
		if strings.EqualFold(groups[i].MerchantID, args[0]) {
			matched = append(matched, groups[i])
		}
	}

	return printResult(cmd, matched, model.OrgAccountGroupViews(matched))
}

func newAccountGroupCreateCommand() *cobra.Command {
	command := &cobra.Command{
		Annotations: apiOperations("POST /organizations/account-groups"),
		Use:         "create",
		Short:       "Create an account group",
		Example:     "  yuno-cli org account-group create --name Overgear --merchant-id mer-1 --yes",
		Args:        cobra.NoArgs,
		RunE:        runAccountGroupCreate,
	}

	registerWriteFlags(command, orgAccountGroupFields)

	return command
}

func runAccountGroupCreate(cmd *cobra.Command, _ []string) error {
	body, err := requireBody(cmd, orgAccountGroupFields)
	if err != nil {
		return err
	}

	client, err := newClientFromFlags(cmd)
	if err != nil {
		return err
	}

	group, err := client.CreateAccountGroup(cmd.Context(), body)
	if err != nil {
		return scopeHint(err, orgScopes)
	}

	return printAccountGroup(cmd, group)
}

func newAccountGroupUpdateCommand() *cobra.Command {
	command := &cobra.Command{
		Annotations: apiOperations("PATCH /organizations/account-groups/{group_id}"),
		Use:         "update <account_group_id>",
		Short:       "Update an account group",
		Example:     "  yuno-cli org account-group update grp-1 --name Overgear --yes",
		Args:        cobra.ExactArgs(1),
		RunE:        runAccountGroupUpdate,
	}

	registerWriteFlags(command, orgAccountGroupFields)

	return command
}

func runAccountGroupUpdate(cmd *cobra.Command, args []string) error {
	body, err := requireBody(cmd, orgAccountGroupFields)
	if err != nil {
		return err
	}

	client, err := newClientFromFlags(cmd)
	if err != nil {
		return err
	}

	group, err := client.UpdateAccountGroup(cmd.Context(), args[0], body)
	if err != nil {
		return scopeHint(err, orgScopes)
	}

	return printAccountGroup(cmd, group)
}

func newAccountGroupDeleteCommand() *cobra.Command {
	command := &cobra.Command{
		Annotations: apiOperations("DELETE /organizations/account-groups/{group_id}"),
		Use:         "delete <account_group_id>",
		Short:       "Delete an account group",
		Example:     "  yuno-cli org account-group delete grp-1 --yes",
		Args:        cobra.ExactArgs(1),
		RunE:        runAccountGroupDelete,
	}

	return command
}

func runAccountGroupDelete(cmd *cobra.Command, args []string) error {
	client, err := newClientFromFlags(cmd)
	if err != nil {
		return err
	}

	data, err := client.DeleteAccountGroup(cmd.Context(), args[0])
	if err != nil {
		return scopeHint(err, orgScopes)
	}

	return printJSON(cmd, data)
}

func newAccountGroupAccountsCommand() *cobra.Command {
	command := &cobra.Command{
		Annotations: apiOperations("GET /organizations/account-groups/{group_id}/accounts"),
		Use:         "accounts <account_group_id>",
		Short:       "List the accounts of one account group",
		Example:     "  yuno-cli org account-group accounts grp-1",
		Args:        cobra.ExactArgs(1),
		RunE:        runAccountGroupAccounts,
	}

	registerPagingFlags(command)

	return command
}

func runAccountGroupAccounts(cmd *cobra.Command, args []string) error {
	client, err := newClientFromFlags(cmd)
	if err != nil {
		return err
	}

	limit, pageSize := pagingFlags(cmd)

	accounts, err := client.ListAccountGroupAccounts(cmd.Context(), args[0], limit, pageSize)
	if err != nil {
		return scopeHint(err, orgScopes)
	}

	return printResult(cmd, accounts, model.OrgAccountViews(accounts))
}

// newAccountGroupAddAccountCommand is the group-side spelling of
// `org account create`: both send POST /organizations/account-groups/{id}/accounts.
func newAccountGroupAddAccountCommand() *cobra.Command {
	command := &cobra.Command{
		Annotations: apiOperations("POST /organizations/account-groups/{group_id}/accounts"),
		Use:         "add-account <account_group_id>",
		Short:       "Create an account inside this account group",
		Example:     "  yuno-cli org account-group add-account grp-1 --name Overgear-BR --yes",
		Args:        cobra.ExactArgs(1),
		RunE:        runOrgAccountCreate,
	}

	registerWriteFlags(command, orgAccountFields)

	return command
}

// printOrgAccount renders one account: the whole response as JSON, a single
// table row otherwise.
func printOrgAccount(cmd *cobra.Command, account *model.OrgAccount) error {
	return printResult(cmd, account, []model.OrgAccountView{account.View()})
}

// printAccountGroup renders one account group.
func printAccountGroup(cmd *cobra.Command, group *model.OrgAccountGroup) error {
	return printResult(cmd, group, []model.OrgAccountGroupView{group.View()})
}
