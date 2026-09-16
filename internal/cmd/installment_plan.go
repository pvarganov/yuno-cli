package cmd

import (
	"net/url"

	"github.com/spf13/cobra"

	"github.com/pvarganov/yuno-cli/internal/model"
)

// installmentPlanScopes are the scopes a 403 on the installment plan commands
// usually asks for.
const installmentPlanScopes = "installments-plans:read/installments-plans:write"

// installmentPlanFields are the named body fields of `installment-plan
// create|update`. The installment options themselves are a nested array, so
// they come from --installments or --file.
var installmentPlanFields = []FieldFlag{
	{Flag: "name", Field: "name", Kind: FieldString, Usage: "name of the installments plan"},
	{
		Flag: "account-id", Field: "account_id", Kind: FieldStringSlice,
		Usage: "account the plan applies to, repeatable",
	},
	{
		Flag: "merchant-reference", Field: "merchant_reference", Kind: FieldString,
		Usage: "your own reference for the plan",
	},
	{
		Flag: "installments", Field: "installments_plan", Kind: FieldJSON,
		Usage: `installment options as a JSON array, e.g. '[{"installment":3,"rate":1.2}]'`,
	},
	{Flag: "country-code", Field: "country_code", Kind: FieldString, Usage: "issuer country, ISO 3166-1 alpha-2"},
	{Flag: "brand", Field: "brand", Kind: FieldStringSlice, Usage: "card brand the plan applies to, repeatable"},
	{Flag: "issuer", Field: "issuer", Kind: FieldString, Usage: "card issuer the plan applies to"},
	{Flag: "iin", Field: "iin", Kind: FieldStringSlice, Usage: "card IIN the plan applies to, repeatable"},
	{
		Flag: "first-installment-deferral", Field: "first_installment_deferral", Kind: FieldInt,
		Usage: "months before the first installment is charged",
	},
	{Flag: "currency", Field: "amount.currency", Kind: FieldString, Usage: "currency of the amount range"},
	{Flag: "min-amount", Field: "amount.min_value", Kind: FieldNumber, Usage: "lowest amount the plan applies to"},
	{Flag: "max-amount", Field: "amount.max_value", Kind: FieldNumber, Usage: "highest amount the plan applies to"},
	{
		Flag: "start-at", Field: "availability.start_at", Kind: FieldString,
		Usage: "when the plan becomes available, RFC 3339",
	},
	{
		Flag: "finish-at", Field: "availability.finish_at", Kind: FieldString,
		Usage: "when the plan stops being available, RFC 3339",
	},
	{
		Flag: "payment-method-type", Field: "payment_method_type", Kind: FieldString,
		Usage: "payment method type the plan applies to, e.g. CARD",
	},
}

// installmentPlanFilters maps the list flags onto the query parameters of
// `GET /installments-plans`, minus `account_id`, which falls back to the
// profile account.
var installmentPlanFilters = []struct {
	Flag  string
	Query string
	Usage string
}{
	{Flag: "currency", Query: "currency", Usage: "only list plans in this currency"},
	{Flag: "iin", Query: "iin", Usage: "only list plans matching this card IIN"},
	{Flag: "amount", Query: "amount", Usage: "only list plans covering this amount"},
	{Flag: "payment-method-type", Query: "payment_method_type", Usage: "only list plans of this payment method type"},
}

// newInstallmentPlanCommand builds the `installment-plan` command tree.
func newInstallmentPlanCommand() *cobra.Command {
	plan := &cobra.Command{
		Use:     "installment-plan",
		Short:   "Manage installments plans",
		Aliases: []string{"installment-plans", "installments-plan"},
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return cmd.Help()
		},
	}

	plan.AddCommand(
		newInstallmentPlanCreateCommand(),
		newInstallmentPlanListCommand(),
		newInstallmentPlanGetCommand(),
		newInstallmentPlanUpdateCommand(),
		newInstallmentPlanDeleteCommand(),
	)

	return plan
}

func newInstallmentPlanCreateCommand() *cobra.Command {
	command := &cobra.Command{
		Use:   "create",
		Short: "Create an installments plan",
		Example: "  yuno-cli installment-plan create --name plan-007 --account-id acc-1 " +
			`--merchant-reference ref-1 --installments '[{"installment":3,"rate":1.2}]'`,
		Args: cobra.NoArgs,
		RunE: runInstallmentPlanCreate,
	}

	registerWriteFlags(command, installmentPlanFields)

	return command
}

func runInstallmentPlanCreate(cmd *cobra.Command, _ []string) error {
	body, err := requireBody(cmd, installmentPlanFields)
	if err != nil {
		return err
	}

	client, err := newClientFromFlags(cmd)
	if err != nil {
		return err
	}

	plan, err := client.CreateInstallmentPlan(cmd.Context(), body)
	if err != nil {
		return scopeHint(err, installmentPlanScopes)
	}

	return printResult(cmd, plan, []model.InstallmentPlanView{plan.View()})
}

func newInstallmentPlanListCommand() *cobra.Command {
	command := &cobra.Command{
		Use:     "list",
		Short:   "List the installments plans of an account",
		Example: "  yuno-cli installment-plan list --account-id acc-1 --currency USD",
		Args:    cobra.NoArgs,
		RunE:    runInstallmentPlanList,
	}

	command.Flags().String("account-id", "", "account to list the plans of (defaults to the profile account_id)")

	for _, filter := range installmentPlanFilters {
		command.Flags().String(filter.Flag, "", filter.Usage)
	}

	return command
}

func runInstallmentPlanList(cmd *cobra.Command, _ []string) error {
	client, err := newClientFromFlags(cmd)
	if err != nil {
		return err
	}

	accountID := flagString(cmd, "account-id")
	if accountID == "" {
		accountID = client.AccountID()
	}

	filters := url.Values{}
	if accountID != "" {
		filters.Set("account_id", accountID)
	}

	for _, filter := range installmentPlanFilters {
		if value := flagString(cmd, filter.Flag); value != "" {
			filters.Set(filter.Query, value)
		}
	}

	plans, err := client.ListInstallmentPlans(cmd.Context(), filters)
	if err != nil {
		return scopeHint(err, installmentPlanScopes)
	}

	return printResult(cmd, plans, model.InstallmentPlanViews(plans))
}

func newInstallmentPlanGetCommand() *cobra.Command {
	return &cobra.Command{
		Use:     "get <installment_code>",
		Short:   "Retrieve one installments plan",
		Example: "  yuno-cli installment-plan get 4d573425 --json",
		Args:    cobra.ExactArgs(1),
		RunE:    runInstallmentPlanGet,
	}
}

func runInstallmentPlanGet(cmd *cobra.Command, args []string) error {
	client, err := newClientFromFlags(cmd)
	if err != nil {
		return err
	}

	plans, err := client.GetInstallmentPlan(cmd.Context(), args[0])
	if err != nil {
		return scopeHint(err, installmentPlanScopes)
	}

	return printResult(cmd, plans, model.InstallmentPlanViews(plans))
}

func newInstallmentPlanUpdateCommand() *cobra.Command {
	command := &cobra.Command{
		Use:     "update <installment_code>",
		Short:   "Update an installments plan",
		Example: "  yuno-cli installment-plan update 4d573425 --name plan-008 --yes",
		Args:    cobra.ExactArgs(1),
		RunE:    runInstallmentPlanUpdate,
	}

	registerWriteFlags(command, installmentPlanFields)

	return command
}

func runInstallmentPlanUpdate(cmd *cobra.Command, args []string) error {
	body, err := requireBody(cmd, installmentPlanFields)
	if err != nil {
		return err
	}

	client, err := newClientFromFlags(cmd)
	if err != nil {
		return err
	}

	plan, err := client.UpdateInstallmentPlan(cmd.Context(), args[0], body)
	if err != nil {
		return scopeHint(err, installmentPlanScopes)
	}

	return printResult(cmd, plan, []model.InstallmentPlanView{plan.View()})
}

func newInstallmentPlanDeleteCommand() *cobra.Command {
	command := &cobra.Command{
		Use:     "delete <installment_code>",
		Short:   "Delete an installments plan",
		Example: "  yuno-cli installment-plan delete 4d573425 --yes",
		Args:    cobra.ExactArgs(1),
		RunE:    runInstallmentPlanDelete,
	}

	command.Flags().String("idempotency-key", "", "pin the X-Idempotency-Key of the request")

	return command
}

func runInstallmentPlanDelete(cmd *cobra.Command, args []string) error {
	client, err := newClientFromFlags(cmd)
	if err != nil {
		return err
	}

	data, err := client.DeleteInstallmentPlan(cmd.Context(), args[0])
	if err != nil {
		return scopeHint(err, installmentPlanScopes)
	}

	return printJSON(cmd, data)
}
