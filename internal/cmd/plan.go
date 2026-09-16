package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/pvarganov/yuno-cli/internal/model"
)

// planScopes are the scopes a 403 on the plan commands usually asks for.
const planScopes = "subscriptions:read/subscriptions:write"

// planFields are the named body fields of `plan create`. Phases and country
// prices come from --file.
var planFields = []FieldFlag{
	{Flag: "account-id", Field: "account_id", Kind: FieldString, Usage: "account the plan belongs to"},
	{Flag: "name", Field: "name", Kind: FieldString, Usage: "human readable plan name"},
	{Flag: "description", Field: "description", Kind: FieldString, Usage: "description of the plan"},
	{Flag: "merchant-reference", Field: "merchant_reference", Kind: FieldString, Usage: "your own reference for the plan"},
	{Flag: "currency", Field: "base_amount.currency", Kind: FieldString, Usage: "currency of the base amount, e.g. USD"},
	{Flag: "amount", Field: "base_amount.value", Kind: FieldNumber, Usage: "base amount billed every cycle"},
	{Flag: "frequency-type", Field: "frequency.type", Kind: FieldString, Usage: "billing frequency unit, e.g. MONTH"},
	{Flag: "frequency-value", Field: "frequency.value", Kind: FieldInt, Usage: "billing frequency value, e.g. 1"},
	{
		Flag: "payment-method", Field: "allowed_payment_methods", Kind: FieldStringSlice,
		Usage: "payment method the plan accepts, repeatable",
	},
	{Flag: "country-prices", Field: "country_prices", Kind: FieldJSON, Usage: "per country prices as a JSON array"},
	{Flag: "phases", Field: "phases", Kind: FieldJSON, Usage: "plan phases as a JSON array"},
	{Flag: "metadata", Field: "metadata", Kind: FieldJSON, Usage: "metadata as a JSON array"},
}

// planStatusFields are the body fields of `plan status`, which today only
// cancels a plan.
var planStatusFields = []FieldFlag{
	{Flag: "status", Field: "status", Kind: FieldString, Usage: "new status of the plan, CANCELED"},
}

// newPlanCommand builds the `plan` command tree.
func newPlanCommand() *cobra.Command {
	plan := &cobra.Command{
		Use:     "plan",
		Short:   "Create and inspect subscription plans",
		Aliases: []string{"plans"},
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return cmd.Help()
		},
	}

	plan.AddCommand(
		newPlanCreateCommand(),
		newPlanListCommand(),
		newPlanGetCommand(),
		newPlanStatusCommand(),
	)

	return plan
}

func newPlanCreateCommand() *cobra.Command {
	command := &cobra.Command{
		Annotations: apiOperations("POST /subscriptions/plans"),
		Use:         "create",
		Short:       "Create a subscription plan",
		Example:     "  yuno-cli plan create --account-id acc-1 --name Gold --currency USD --amount 9.99",
		Args:        cobra.NoArgs,
		RunE:        runPlanCreate,
	}

	registerWriteFlags(command, planFields)

	return command
}

func runPlanCreate(cmd *cobra.Command, _ []string) error {
	body, err := requireBody(cmd, planFields)
	if err != nil {
		return err
	}

	client, err := newClientFromFlags(cmd)
	if err != nil {
		return err
	}

	plan, err := client.CreatePlan(cmd.Context(), body)
	if err != nil {
		return scopeHint(err, planScopes)
	}

	return printPlan(cmd, plan)
}

func newPlanListCommand() *cobra.Command {
	command := &cobra.Command{
		Annotations: apiOperations("GET /subscriptions/plans"),
		Use:         "list",
		Short:       "List the subscription plans of an account",
		Example:     "  yuno-cli plan list --account-id acc-1 --limit 20",
		Args:        cobra.NoArgs,
		RunE:        runPlanList,
	}

	command.Flags().String("account-id", "", "account to list the plans of (defaults to the profile account_id)")
	registerPagingFlags(command)

	return command
}

func runPlanList(cmd *cobra.Command, _ []string) error {
	client, err := newClientFromFlags(cmd)
	if err != nil {
		return err
	}

	accountID := flagString(cmd, "account-id")
	if accountID == "" {
		accountID = client.AccountID()
	}

	limit, pageSize := pagingFlags(cmd)

	plans, err := client.ListPlans(cmd.Context(), accountID, limit, pageSize)
	if err != nil {
		return scopeHint(err, planScopes)
	}

	return printResult(cmd, plans, model.PlanViews(plans))
}

func newPlanGetCommand() *cobra.Command {
	return &cobra.Command{
		Annotations: apiOperations("GET /subscriptions/plans/{plan_id}"),
		Use:         "get <plan_id>",
		Short:       "Retrieve one subscription plan",
		Example:     "  yuno-cli plan get plan-1 --json",
		Args:        cobra.ExactArgs(1),
		RunE:        runPlanGet,
	}
}

func runPlanGet(cmd *cobra.Command, args []string) error {
	client, err := newClientFromFlags(cmd)
	if err != nil {
		return err
	}

	plan, err := client.GetPlan(cmd.Context(), args[0])
	if err != nil {
		return scopeHint(err, planScopes)
	}

	return printPlan(cmd, plan)
}

func newPlanStatusCommand() *cobra.Command {
	command := &cobra.Command{
		Annotations: apiOperations("POST /subscriptions/plans/{plan_id}/status"),
		Use:         "status <plan_id>",
		Short:       "Change the status of a plan, which cancels it",
		Example:     "  yuno-cli plan status plan-1 --status CANCELED --yes",
		Args:        cobra.ExactArgs(1),
		RunE:        runPlanStatus,
	}

	registerWriteFlags(command, planStatusFields)

	return command
}

func runPlanStatus(cmd *cobra.Command, args []string) error {
	body, err := bodyFromFlags(cmd, planStatusFields)
	if err != nil {
		return err
	}

	if body == nil {
		return fmt.Errorf("--status is required (the only status a plan accepts is CANCELED)")
	}

	client, err := newClientFromFlags(cmd)
	if err != nil {
		return err
	}

	status, err := client.UpdatePlanStatus(cmd.Context(), args[0], body)
	if err != nil {
		return scopeHint(err, planScopes)
	}

	return printResult(cmd, status, []model.PlanStatusView{status.View()})
}

// printPlan renders one plan: the whole response as JSON, a single table row
// otherwise.
func printPlan(cmd *cobra.Command, plan *model.Plan) error {
	return printResult(cmd, plan, []model.PlanView{plan.View()})
}
