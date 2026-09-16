package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/pvarganov/yuno-cli/internal/model"
)

// routingScopes are the scopes Yuno has to grant before these commands work.
const routingScopes = "routing:read/routing:write"

// routingFields are the frequently used body fields of the routing commands.
// Everything else (condition sets, nested routes) comes from --file.
var routingFields = []FieldFlag{
	{Flag: "account-id", Field: "account_id", Kind: FieldString, Usage: "account the routing belongs to"},
	{Flag: "payment-method", Field: "payment_method", Kind: FieldString, Usage: "payment method the routing applies to"},
	{Flag: "name", Field: "name", Kind: FieldString, Usage: "human readable routing name"},
	{Flag: "default-route", Field: "default_route", Kind: FieldJSON, Usage: "default route as a JSON object"},
}

// recommendFields are declared separately because the recommendation body
// shares no fields with the routing resource itself; the payment and the
// candidates are too nested for flags and come from --file.
var recommendFields = []FieldFlag{
	{Flag: "account-id", Field: "account_id", Kind: FieldString, Usage: "account the recommendation is requested for"},
	{Flag: "merchant-order-id", Field: "merchant_order_id", Kind: FieldString, Usage: "merchant order the recommendation is requested for"},
}

// newRoutingCommand builds the `routing` command tree.
func newRoutingCommand() *cobra.Command {
	routing := &cobra.Command{
		Use:     "routing",
		Short:   "Inspect and manage payment routing rules",
		Aliases: []string{"routings"},
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return cmd.Help()
		},
	}

	routing.AddCommand(
		newRoutingListCommand(),
		newRoutingGetCommand(),
		newRoutingCreateCommand(),
		newRoutingUpdateCommand(),
		newRoutingRecommendCommand(),
	)

	return routing
}

func newRoutingListCommand() *cobra.Command {
	command := &cobra.Command{
		Use:     "list",
		Short:   "List the routing rules of an account",
		Example: "  yuno-cli routing list --account-id <uuid> --payment-method CARD",
		Args:    cobra.NoArgs,
		RunE:    runRoutingList,
	}

	command.Flags().String("account-id", "", "account to list the routings of (defaults to the profile account_id)")
	command.Flags().String("payment-method", "", "only list routings of this payment method")

	return command
}

func runRoutingList(cmd *cobra.Command, _ []string) error {
	client, err := newClientFromFlags(cmd)
	if err != nil {
		return err
	}

	accountID := flagString(cmd, "account-id")
	if accountID == "" {
		accountID = client.AccountID()
	}

	if accountID == "" {
		return fmt.Errorf("--account-id is required: set it on the command line or store account_id in the profile")
	}

	routings, err := client.ListRoutings(cmd.Context(), accountID, flagString(cmd, "payment-method"))
	if err != nil {
		return scopeHint(err, routingScopes)
	}

	return printResult(cmd, routings, model.RoutingViews(routings))
}

func newRoutingGetCommand() *cobra.Command {
	return &cobra.Command{
		Use:     "get <routing_id>",
		Short:   "Retrieve one routing rule",
		Example: "  yuno-cli routing get r_8f2c1d3e --json",
		Args:    cobra.ExactArgs(1),
		RunE:    runRoutingGet,
	}
}

func runRoutingGet(cmd *cobra.Command, args []string) error {
	client, err := newClientFromFlags(cmd)
	if err != nil {
		return err
	}

	routing, err := client.GetRouting(cmd.Context(), args[0])
	if err != nil {
		return scopeHint(err, routingScopes)
	}

	return printResult(cmd, routing, []model.RoutingView{routing.View()})
}

func newRoutingCreateCommand() *cobra.Command {
	command := &cobra.Command{
		Use:     "create",
		Short:   "Create a routing rule",
		Example: "  yuno-cli routing create --file routing.json",
		Args:    cobra.NoArgs,
		RunE:    runRoutingCreate,
	}

	registerWriteFlags(command, routingFields)

	return command
}

func runRoutingCreate(cmd *cobra.Command, _ []string) error {
	body, err := requireBody(cmd, routingFields)
	if err != nil {
		return err
	}

	client, err := newClientFromFlags(cmd)
	if err != nil {
		return err
	}

	routing, err := client.CreateRouting(cmd.Context(), body)
	if err != nil {
		return scopeHint(err, routingScopes)
	}

	return printResult(cmd, routing, []model.RoutingView{routing.View()})
}

func newRoutingUpdateCommand() *cobra.Command {
	command := &cobra.Command{
		Use:     "update <routing_id>",
		Short:   "Update a routing rule",
		Example: "  yuno-cli routing update r_8f2c1d3e --name 'Card routing'",
		Args:    cobra.ExactArgs(1),
		RunE:    runRoutingUpdate,
	}

	registerWriteFlags(command, routingFields)

	return command
}

func runRoutingUpdate(cmd *cobra.Command, args []string) error {
	body, err := requireBody(cmd, routingFields)
	if err != nil {
		return err
	}

	client, err := newClientFromFlags(cmd)
	if err != nil {
		return err
	}

	routing, err := client.UpdateRouting(cmd.Context(), args[0], body)
	if err != nil {
		return scopeHint(err, routingScopes)
	}

	return printResult(cmd, routing, []model.RoutingView{routing.View()})
}

func newRoutingRecommendCommand() *cobra.Command {
	command := &cobra.Command{
		Use:     "recommend",
		Short:   "Ask Yuno which candidate provider to route a payment to",
		Example: "  yuno-cli routing recommend --file recommendation.json",
		Args:    cobra.NoArgs,
		RunE:    runRoutingRecommend,
	}

	registerWriteFlags(command, recommendFields)

	return command
}

func runRoutingRecommend(cmd *cobra.Command, _ []string) error {
	body, err := requireBody(cmd, recommendFields)
	if err != nil {
		return err
	}

	client, err := newClientFromFlags(cmd)
	if err != nil {
		return err
	}

	recommendation, err := client.RecommendRouting(cmd.Context(), body)
	if err != nil {
		return scopeHint(err, routingScopes)
	}

	return printResult(cmd, recommendation, recommendation.Views())
}
