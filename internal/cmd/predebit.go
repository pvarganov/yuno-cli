package cmd

import (
	"github.com/spf13/cobra"

	"github.com/pvarganov/yuno-cli/internal/model"
)

// preDebitScopes are the scopes a 403 on the pre-debit commands usually asks for.
const preDebitScopes = "predebit-notify:read/predebit-notify:write"

// preDebitFields are the named body fields of `pre-debit create`.
var preDebitFields = []FieldFlag{
	{Flag: "account-id", Field: "account_id", Kind: FieldString, Usage: "account the debit is collected for"},
	{
		Flag: "merchant-reference", Field: "merchant_reference", Kind: FieldString,
		Usage: "your own reference for the notification",
	},
	{Flag: "description", Field: "description", Kind: FieldString, Usage: "description of the upcoming debit"},
	{Flag: "currency", Field: "amount.currency", Kind: FieldString, Usage: "currency of the amount"},
	{Flag: "amount", Field: "amount.value", Kind: FieldString, Usage: "amount of the upcoming debit"},
	{Flag: "billing-date", Field: "billing_date", Kind: FieldString, Usage: "date the debit is collected on"},
	{
		Flag: "billing-sequence-number", Field: "billing_sequence_number", Kind: FieldString,
		Usage: "position of the debit in the recurring series",
	},
	{
		Flag: "origin-payment-id", Field: "origin_payment_id", Kind: FieldString,
		Usage: "payment the recurring series started with",
	},
	{
		Flag: "customer-id", Field: "customer_payer.id", Kind: FieldString,
		Usage: "Yuno id of the customer being debited",
	},
	{
		Flag: "merchant-customer-id", Field: "customer_payer.merchant_customer_id", Kind: FieldString,
		Usage: "your own id of the customer being debited",
	},
	{
		Flag: "payment-method-type", Field: "payment_method.type", Kind: FieldString,
		Usage: "payment method the debit is collected with",
	},
	{
		Flag: "vaulted-token", Field: "payment_method.vaulted_token", Kind: FieldString,
		Usage: "vaulted token of the payment method",
	},
	{Flag: "provider-id", Field: "provider_data.id", Kind: FieldString, Usage: "provider carrying the notification"},
	{
		Flag: "connection-id", Field: "provider_data.connection_id", Kind: FieldString,
		Usage: "connection carrying the notification",
	},
}

// newPreDebitCommand builds the `pre-debit` command tree.
func newPreDebitCommand() *cobra.Command {
	preDebit := &cobra.Command{
		Use:     "pre-debit",
		Short:   "Create and inspect pre-debit notifications",
		Aliases: []string{"predebit", "predebit-notify"},
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return cmd.Help()
		},
	}

	preDebit.AddCommand(
		newPreDebitCreateCommand(),
		newPreDebitListCommand(),
		newPreDebitGetCommand(),
	)

	return preDebit
}

func newPreDebitCreateCommand() *cobra.Command {
	command := &cobra.Command{
		Annotations: apiOperations("POST /predebit-notify"),
		Use:         "create",
		Short:       "Create a pre-debit notification",
		Example: "  yuno-cli pre-debit create --account-id acc-1 --merchant-reference ref-1 " +
			"--currency INR --amount 1000 --billing-date 2026-10-01",
		Args: cobra.NoArgs,
		RunE: runPreDebitCreate,
	}

	registerWriteFlags(command, preDebitFields)

	return command
}

func runPreDebitCreate(cmd *cobra.Command, _ []string) error {
	body, err := requireBody(cmd, preDebitFields)
	if err != nil {
		return err
	}

	client, err := newClientFromFlags(cmd)
	if err != nil {
		return err
	}

	notification, err := client.CreatePreDebitNotification(cmd.Context(), body)
	if err != nil {
		return scopeHint(err, preDebitScopes)
	}

	return printPreDebit(cmd, notification)
}

func newPreDebitListCommand() *cobra.Command {
	command := &cobra.Command{
		Annotations: apiOperations("GET /predebit-notify"),
		Use:         "list",
		Short:       "Look a pre-debit notification up by merchant reference",
		Long: "Look a pre-debit notification up by merchant reference.\n\n" +
			"Yuno has no unfiltered list on this resource: GET /predebit-notify requires\n" +
			"merchant_reference and answers with the single matching notification.",
		Example: "  yuno-cli pre-debit list --merchant-reference ref-1",
		Args:    cobra.NoArgs,
		RunE:    runPreDebitList,
	}

	command.Flags().String("merchant-reference", "", "merchant reference the notification was created with")
	_ = command.MarkFlagRequired("merchant-reference")

	return command
}

func runPreDebitList(cmd *cobra.Command, _ []string) error {
	client, err := newClientFromFlags(cmd)
	if err != nil {
		return err
	}

	notification, err := client.GetPreDebitNotificationByReference(cmd.Context(), flagString(cmd, "merchant-reference"))
	if err != nil {
		return scopeHint(err, preDebitScopes)
	}

	return printPreDebit(cmd, notification)
}

func newPreDebitGetCommand() *cobra.Command {
	return &cobra.Command{
		Annotations: apiOperations("GET /predebit-notify/{notification_id}"),
		Use:         "get <pre_debit_notification_id>",
		Short:       "Retrieve one pre-debit notification",
		Example:     "  yuno-cli pre-debit get pdn-1 --json",
		Args:        cobra.ExactArgs(1),
		RunE:        runPreDebitGet,
	}
}

func runPreDebitGet(cmd *cobra.Command, args []string) error {
	client, err := newClientFromFlags(cmd)
	if err != nil {
		return err
	}

	notification, err := client.GetPreDebitNotification(cmd.Context(), args[0])
	if err != nil {
		return scopeHint(err, preDebitScopes)
	}

	return printPreDebit(cmd, notification)
}

// printPreDebit renders one pre-debit notification.
func printPreDebit(cmd *cobra.Command, notification *model.PreDebitNotification) error {
	return printResult(cmd, notification, []model.PreDebitNotificationView{notification.View()})
}
