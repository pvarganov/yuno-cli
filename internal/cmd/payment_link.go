package cmd

import (
	"github.com/spf13/cobra"

	"github.com/pvarganov/yuno-cli/internal/model"
)

// paymentLinkScopes are the scopes a 403 on the payment link commands usually
// asks for.
const paymentLinkScopes = "payment-links:read/payment-links:write"

// paymentLinkFields are the named body fields of `payment-link create`. The
// nested objects (customer payer, taxes, additional data) come from --file.
var paymentLinkFields = []FieldFlag{
	{Flag: "account-id", Field: "account_id", Kind: FieldString, Usage: "account the link belongs to"},
	{Flag: "description", Field: "description", Kind: FieldString, Usage: "description of the payment link"},
	{Flag: "country", Field: "country", Kind: FieldString, Usage: "country of the shopper, ISO 3166-1 alpha-2"},
	{
		Flag: "merchant-order-id", Field: "merchant_order_id", Kind: FieldString,
		Usage: "your own reference for the link",
	},
	{Flag: "currency", Field: "amount.currency", Kind: FieldString, Usage: "currency of the amount, e.g. USD"},
	{Flag: "amount", Field: "amount.value", Kind: FieldNumber, Usage: "amount to charge"},
	{Flag: "capture", Field: "capture", Kind: FieldBool, Usage: "capture the payment instead of only authorizing it"},
	{
		Flag: "payment-method-type", Field: "payment_method_types", Kind: FieldStringSlice,
		Usage: "payment method the link accepts, repeatable",
	},
	{Flag: "one-time-use", Field: "one_time_use", Kind: FieldBool, Usage: "let the link be paid only once"},
	{
		Flag: "split-payment-methods", Field: "split_payment_methods", Kind: FieldBool,
		Usage: "allow the shopper to split the amount across payment methods",
	},
	{Flag: "callback-url", Field: "callback_url", Kind: FieldString, Usage: "url the shopper returns to"},
	{
		Flag: "start-at", Field: "availability.start_at", Kind: FieldString,
		Usage: "when the link becomes payable, RFC 3339",
	},
	{
		Flag: "finish-at", Field: "availability.finish_at", Kind: FieldString,
		Usage: "when the link expires, RFC 3339",
	},
	{Flag: "timezone", Field: "timezone", Kind: FieldString, Usage: "timezone of the availability window"},
	{Flag: "payments-number", Field: "payments_number", Kind: FieldInt, Usage: "how many payments the link accepts"},
	{
		Flag: "customer-payer", Field: "customer_payer", Kind: FieldJSON,
		Usage: "customer paying the link as a JSON object",
	},
	{
		Flag: "installments-plan", Field: "installments_plan", Kind: FieldJSON,
		Usage: "installments plan configuration as a JSON object",
	},
	{Flag: "metadata", Field: "metadata", Kind: FieldJSON, Usage: "metadata as a JSON array"},
}

// newPaymentLinkCommand builds the `payment-link` command tree.
func newPaymentLinkCommand() *cobra.Command {
	link := &cobra.Command{
		Use:     "payment-link",
		Short:   "Create, inspect and cancel payment links",
		Aliases: []string{"payment-links"},
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return cmd.Help()
		},
	}

	link.AddCommand(
		newPaymentLinkCreateCommand(),
		newPaymentLinkGetCommand(),
		newPaymentLinkCancelCommand(),
	)

	return link
}

func newPaymentLinkCreateCommand() *cobra.Command {
	command := &cobra.Command{
		Annotations: apiOperations("POST /payment-links"),
		Use:         "create",
		Short:       "Create a payment link",
		Example: "  yuno-cli payment-link create --account-id acc-1 --country US " +
			"--currency USD --amount 50 --payment-method-type CARD",
		Args: cobra.NoArgs,
		RunE: runPaymentLinkCreate,
	}

	registerWriteFlags(command, paymentLinkFields)

	return command
}

func runPaymentLinkCreate(cmd *cobra.Command, _ []string) error {
	body, err := requireBody(cmd, paymentLinkFields)
	if err != nil {
		return err
	}

	client, err := newClientFromFlags(cmd)
	if err != nil {
		return err
	}

	link, err := client.CreatePaymentLink(cmd.Context(), body)
	if err != nil {
		return scopeHint(err, paymentLinkScopes)
	}

	return printPaymentLink(cmd, link)
}

func newPaymentLinkGetCommand() *cobra.Command {
	return &cobra.Command{
		Annotations: apiOperations("GET /payment-links/{code}"),
		Use:         "get <code>",
		Short:       "Retrieve one payment link",
		Example:     "  yuno-cli payment-link get aace3f6d --json",
		Args:        cobra.ExactArgs(1),
		RunE:        runPaymentLinkGet,
	}
}

func runPaymentLinkGet(cmd *cobra.Command, args []string) error {
	client, err := newClientFromFlags(cmd)
	if err != nil {
		return err
	}

	link, err := client.GetPaymentLink(cmd.Context(), args[0])
	if err != nil {
		return scopeHint(err, paymentLinkScopes)
	}

	return printPaymentLink(cmd, link)
}

func newPaymentLinkCancelCommand() *cobra.Command {
	command := &cobra.Command{
		Annotations: apiOperations("POST /payment-links/{code}/cancel"),
		Use:         "cancel <code>",
		Short:       "Cancel a payment link so it can no longer be paid",
		Example:     "  yuno-cli payment-link cancel aace3f6d --yes",
		Args:        cobra.ExactArgs(1),
		RunE:        runPaymentLinkCancel,
	}

	command.Flags().String("idempotency-key", "", "pin the X-Idempotency-Key of the request")

	return command
}

func runPaymentLinkCancel(cmd *cobra.Command, args []string) error {
	client, err := newClientFromFlags(cmd)
	if err != nil {
		return err
	}

	link, err := client.CancelPaymentLink(cmd.Context(), args[0])
	if err != nil {
		return scopeHint(err, paymentLinkScopes)
	}

	return printPaymentLink(cmd, link)
}

// printPaymentLink renders one payment link: the whole response as JSON, a
// single table row otherwise.
func printPaymentLink(cmd *cobra.Command, link *model.PaymentLink) error {
	return printResult(cmd, link, []model.PaymentLinkView{link.View()})
}
