package cmd

import (
	"github.com/spf13/cobra"
)

// subscriptionCancelFields are the optional body fields of
// `subscription cancel`.
var subscriptionCancelFields = []FieldFlag{
	{Flag: "refund", Field: "refund", Kind: FieldBool, Usage: "refund the last charge when cancelling"},
	{Flag: "refund-amount", Field: "refund_amount", Kind: FieldNumber, Usage: "amount to refund when cancelling"},
	{Flag: "schedule", Field: "schedule", Kind: FieldString, Usage: "when the cancellation applies, e.g. END_OF_CYCLE"},
}

// subscriptionChangePlanFields are the body fields of
// `subscription change-plan`.
var subscriptionChangePlanFields = []FieldFlag{
	{Flag: "plan-id", Field: "plan_id", Kind: FieldString, Usage: "plan the subscription moves to"},
	{Flag: "skip-trial", Field: "skip_trial", Kind: FieldBool, Usage: "skip the trial period of the new plan"},
}

// newSubscriptionActionCommands builds the lifecycle actions of a subscription.
// They all post to `/subscriptions/{id}/<action>`; only cancel and change-plan
// take a body.
func newSubscriptionActionCommands() []*cobra.Command {
	return []*cobra.Command{
		newSubscriptionCancelCommand(),
		newSubscriptionSimpleActionCommand(
			"pause", "Pause a subscription", "  yuno-cli subscription pause sub-1 --yes"),
		newSubscriptionSimpleActionCommand(
			"resume", "Resume a paused subscription", "  yuno-cli subscription resume sub-1 --yes"),
		newSubscriptionSimpleActionCommand(
			"retry", "Retry the last failed charge of a subscription", "  yuno-cli subscription retry sub-1 --yes"),
		newSubscriptionChangePlanCommand(),
	}
}

// newSubscriptionSimpleActionCommand builds an action that takes no body.
func newSubscriptionSimpleActionCommand(action, short, example string) *cobra.Command {
	command := &cobra.Command{
		Annotations: apiOperations("POST /subscriptions/{subscription_id}/" + action),
		Use:         action + " <subscription_id>",
		Short:       short,
		Example:     example,
		Args:        cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runSubscriptionAction(cmd, args[0], action, nil)
		},
	}

	command.Flags().String("idempotency-key", "", "pin the X-Idempotency-Key of the request")

	return command
}

func newSubscriptionCancelCommand() *cobra.Command {
	command := &cobra.Command{
		Annotations: apiOperations("POST /subscriptions/{subscription_id}/cancel"),
		Use:         "cancel <subscription_id>",
		Short:       "Cancel a subscription",
		Example:     "  yuno-cli subscription cancel sub-1 --refund --yes",
		Args:        cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			body, err := bodyFromFlags(cmd, subscriptionCancelFields)
			if err != nil {
				return err
			}

			return runSubscriptionAction(cmd, args[0], "cancel", body)
		},
	}

	registerWriteFlags(command, subscriptionCancelFields)

	return command
}

func newSubscriptionChangePlanCommand() *cobra.Command {
	command := &cobra.Command{
		Annotations: apiOperations("POST /subscriptions/{subscription_id}/plan"),
		Use:         "change-plan <subscription_id>",
		Short:       "Move a subscription to another plan",
		Example:     "  yuno-cli subscription change-plan sub-1 --plan-id plan-2 --yes",
		Args:        cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			body, err := requireBody(cmd, subscriptionChangePlanFields)
			if err != nil {
				return err
			}

			return runSubscriptionAction(cmd, args[0], "plan", body)
		},
	}

	registerWriteFlags(command, subscriptionChangePlanFields)

	return command
}

// runSubscriptionAction posts one lifecycle action and prints the subscription
// the API answers with.
func runSubscriptionAction(cmd *cobra.Command, subscriptionID, action string, body any) error {
	client, err := newClientFromFlags(cmd)
	if err != nil {
		return err
	}

	subscription, err := client.SubscriptionAction(cmd.Context(), subscriptionID, action, body)
	if err != nil {
		return scopeHint(err, subscriptionScopes)
	}

	return printSubscription(cmd, subscription)
}
