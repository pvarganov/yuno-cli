package cmd

import (
	"net/url"

	"github.com/spf13/cobra"

	"github.com/pvarganov/yuno-cli/internal/model"
)

// subscriptionScopes are the scopes a 403 on the subscription commands usually
// asks for.
const subscriptionScopes = "subscriptions:read/subscriptions:write"

// subscriptionFields are the named body fields of `subscription create` and
// `subscription update`. Phases, retries and metadata come from --file.
var subscriptionFields = []FieldFlag{
	{Flag: "account-id", Field: "account_id", Kind: FieldString, Usage: "account the subscription belongs to"},
	{Flag: "name", Field: "name", Kind: FieldString, Usage: "human readable subscription name"},
	{Flag: "description", Field: "description", Kind: FieldString, Usage: "description of the subscription"},
	{Flag: "country", Field: "country", Kind: FieldString, Usage: "country of the subscription, e.g. CO"},
	{Flag: "merchant-reference", Field: "merchant_reference", Kind: FieldString, Usage: "your own reference for the subscription"},
	{Flag: "soft-descriptor", Field: "soft_descriptor", Kind: FieldString, Usage: "statement descriptor of the charges"},
	{Flag: "plan-id", Field: "plan_id", Kind: FieldString, Usage: "plan the subscription is created from"},
	{Flag: "currency", Field: "amount.currency", Kind: FieldString, Usage: "currency of the subscription, e.g. USD"},
	{Flag: "amount", Field: "amount.value", Kind: FieldNumber, Usage: "amount billed every cycle"},
	{Flag: "frequency-type", Field: "frequency.type", Kind: FieldString, Usage: "billing frequency unit, e.g. MONTH"},
	{Flag: "frequency-value", Field: "frequency.value", Kind: FieldInt, Usage: "billing frequency value, e.g. 1"},
	{Flag: "billing-cycles", Field: "billing_cycles.total", Kind: FieldInt, Usage: "how many cycles the subscription runs for"},
	{Flag: "customer-id", Field: "customer_payer.id", Kind: FieldString, Usage: "customer the subscription bills"},
	{Flag: "payment-method-type", Field: "payment_method.type", Kind: FieldString, Usage: "payment method type, e.g. CARD"},
	{Flag: "vaulted-token", Field: "payment_method.vaulted_token", Kind: FieldString, Usage: "vaulted token the subscription charges"},
	{Flag: "payment-method", Field: "payment_method", Kind: FieldJSON, Usage: "payment method as a JSON object"},
	{Flag: "trial-period", Field: "trial_period", Kind: FieldJSON, Usage: "trial period as a JSON object"},
	{Flag: "availability", Field: "availability", Kind: FieldJSON, Usage: "availability window as a JSON object"},
	{Flag: "retries", Field: "retries", Kind: FieldJSON, Usage: "retry policy as a JSON object"},
	{Flag: "metadata", Field: "metadata", Kind: FieldJSON, Usage: "metadata as a JSON array"},
}

// subscriptionFilters maps the list flags onto the query parameters of
// `GET /subscriptions`.
var subscriptionFilters = []struct {
	Flag  string
	Query string
	Usage string
}{
	{Flag: "customer-id", Query: "customer_id", Usage: "only list subscriptions of this customer"},
	{Flag: "status", Query: "status", Usage: "only list subscriptions in this status"},
	{Flag: "plan-id", Query: "plan_id", Usage: "only list subscriptions of this plan"},
	{Flag: "created-after", Query: "created_at_from", Usage: "only list subscriptions created at or after this date"},
	{Flag: "created-before", Query: "created_at_to", Usage: "only list subscriptions created at or before this date"},
	{Flag: "payment-method-type", Query: "payment_method_type", Usage: "only list subscriptions paid with this method"},
}

// newSubscriptionCommand builds the `subscription` command tree.
func newSubscriptionCommand() *cobra.Command {
	subscription := &cobra.Command{
		Use:     "subscription",
		Short:   "Create, inspect and steer subscriptions",
		Aliases: []string{"subscriptions"},
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return cmd.Help()
		},
	}

	subscription.AddCommand(
		newSubscriptionCreateCommand(),
		newSubscriptionListCommand(),
		newSubscriptionGetCommand(),
		newSubscriptionUpdateCommand(),
		newSubscriptionPaymentsCommand(),
	)
	subscription.AddCommand(newSubscriptionActionCommands()...)

	return subscription
}

func newSubscriptionCreateCommand() *cobra.Command {
	command := &cobra.Command{
		Annotations: apiOperations("POST /subscriptions"),
		Use:         "create",
		Short:       "Create a subscription",
		Example:     "  yuno-cli subscription create --account-id acc-1 --name Gold --country CO --file subscription.json",
		Args:        cobra.NoArgs,
		RunE:        runSubscriptionCreate,
	}

	registerWriteFlags(command, subscriptionFields)

	return command
}

func runSubscriptionCreate(cmd *cobra.Command, _ []string) error {
	body, err := requireBody(cmd, subscriptionFields)
	if err != nil {
		return err
	}

	client, err := newClientFromFlags(cmd)
	if err != nil {
		return err
	}

	subscription, err := client.CreateSubscription(cmd.Context(), body)
	if err != nil {
		return scopeHint(err, subscriptionScopes)
	}

	return printSubscription(cmd, subscription)
}

func newSubscriptionListCommand() *cobra.Command {
	command := &cobra.Command{
		Annotations: apiOperations("GET /subscriptions"),
		Use:         "list",
		Short:       "List subscriptions",
		Example:     "  yuno-cli subscription list --status ACTIVE --limit 50",
		Args:        cobra.NoArgs,
		RunE:        runSubscriptionList,
	}

	for _, filter := range subscriptionFilters {
		command.Flags().String(filter.Flag, "", filter.Usage)
	}

	registerPagingFlags(command)

	return command
}

func runSubscriptionList(cmd *cobra.Command, _ []string) error {
	client, err := newClientFromFlags(cmd)
	if err != nil {
		return err
	}

	filters := url.Values{}

	for _, filter := range subscriptionFilters {
		if value := flagString(cmd, filter.Flag); value != "" {
			filters.Set(filter.Query, value)
		}
	}

	limit, pageSize := pagingFlags(cmd)

	subscriptions, err := client.ListSubscriptions(cmd.Context(), filters, limit, pageSize)
	if err != nil {
		return scopeHint(err, subscriptionScopes)
	}

	return printResult(cmd, subscriptions, model.SubscriptionViews(subscriptions))
}

func newSubscriptionGetCommand() *cobra.Command {
	return &cobra.Command{
		Annotations: apiOperations("GET /subscriptions/{subscription_id}"),
		Use:         "get <subscription_id>",
		Short:       "Retrieve one subscription",
		Example:     "  yuno-cli subscription get sub-1 --json",
		Args:        cobra.ExactArgs(1),
		RunE:        runSubscriptionGet,
	}
}

func runSubscriptionGet(cmd *cobra.Command, args []string) error {
	client, err := newClientFromFlags(cmd)
	if err != nil {
		return err
	}

	subscription, err := client.GetSubscription(cmd.Context(), args[0])
	if err != nil {
		return scopeHint(err, subscriptionScopes)
	}

	return printSubscription(cmd, subscription)
}

func newSubscriptionUpdateCommand() *cobra.Command {
	command := &cobra.Command{
		Annotations: apiOperations("PATCH /subscriptions/{subscription_id}"),
		Use:         "update <subscription_id>",
		Short:       "Update a subscription",
		Example:     "  yuno-cli subscription update sub-1 --amount 12.5",
		Args:        cobra.ExactArgs(1),
		RunE:        runSubscriptionUpdate,
	}

	registerWriteFlags(command, subscriptionFields)

	return command
}

func runSubscriptionUpdate(cmd *cobra.Command, args []string) error {
	body, err := requireBody(cmd, subscriptionFields)
	if err != nil {
		return err
	}

	client, err := newClientFromFlags(cmd)
	if err != nil {
		return err
	}

	subscription, err := client.UpdateSubscription(cmd.Context(), args[0], body)
	if err != nil {
		return scopeHint(err, subscriptionScopes)
	}

	return printSubscription(cmd, subscription)
}

func newSubscriptionPaymentsCommand() *cobra.Command {
	command := &cobra.Command{
		Annotations: apiOperations("GET /subscriptions/{subscription_id}/payments"),
		Use:         "payments <subscription_id>",
		Short:       "List the payments of a subscription",
		Example:     "  yuno-cli subscription payments sub-1 --limit 10",
		Args:        cobra.ExactArgs(1),
		RunE:        runSubscriptionPayments,
	}

	registerPagingFlags(command)

	return command
}

func runSubscriptionPayments(cmd *cobra.Command, args []string) error {
	client, err := newClientFromFlags(cmd)
	if err != nil {
		return err
	}

	limit, pageSize := pagingFlags(cmd)

	payments, err := client.ListSubscriptionPayments(cmd.Context(), args[0], limit, pageSize)
	if err != nil {
		return scopeHint(err, subscriptionScopes)
	}

	return printResult(cmd, payments, model.SubscriptionPaymentViews(payments))
}

// printSubscription renders one subscription: the whole response as JSON, a
// single table row otherwise.
func printSubscription(cmd *cobra.Command, subscription *model.Subscription) error {
	return printResult(cmd, subscription, []model.SubscriptionView{subscription.View()})
}
