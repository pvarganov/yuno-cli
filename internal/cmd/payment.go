package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/pvarganov/yuno-cli/internal/model"
)

// paymentScopes are the scopes a 403 on the payment commands usually asks for.
const paymentScopes = "payments:read/payments:write"

// paymentFields are the frequently used body fields of `payment create`.
// Everything else (fraud screening, split marketplace, metadata) comes from
// --file.
var paymentFields = []FieldFlag{
	{Flag: "account-id", Field: "account_id", Kind: FieldString, Usage: "account the payment belongs to"},
	{Flag: "merchant-order-id", Field: "merchant_order_id", Kind: FieldString, Usage: "your own order id"},
	{Flag: "description", Field: "description", Kind: FieldString, Usage: "payment description"},
	{Flag: "country", Field: "country", Kind: FieldString, Usage: "country of the payment, e.g. CO"},
	{Flag: "currency", Field: "amount.currency", Kind: FieldString, Usage: "currency of the payment, e.g. USD"},
	{Flag: "amount", Field: "amount.value", Kind: FieldNumber, Usage: "amount of the payment"},
	{Flag: "checkout-session", Field: "checkout.session", Kind: FieldString, Usage: "checkout session the payment belongs to"},
	{Flag: "workflow", Field: "workflow", Kind: FieldString, Usage: "payment workflow, e.g. SDK_CHECKOUT or DIRECT"},
	{Flag: "payment-method-type", Field: "payment_method.type", Kind: FieldString, Usage: "payment method type, e.g. CARD"},
	{Flag: "payment-method-token", Field: "payment_method.token", Kind: FieldString, Usage: "one time token of the payment method"},
	{Flag: "vaulted-token", Field: "payment_method.vaulted_token", Kind: FieldString, Usage: "vaulted token of the payment method"},
	{Flag: "customer-id", Field: "customer_payer.id", Kind: FieldString, Usage: "customer the payment is made by"},
	{Flag: "callback-url", Field: "callback_url", Kind: FieldString, Usage: "url the customer returns to"},
}

// newPaymentCommand builds the `payment` command tree.
func newPaymentCommand() *cobra.Command {
	payment := &cobra.Command{
		Use:     "payment",
		Short:   "Create, inspect and settle payments",
		Aliases: []string{"payments"},
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return cmd.Help()
		},
	}

	payment.AddCommand(
		newPaymentCreateCommand(),
		newPaymentGetCommand(),
		newPaymentListCommand(),
		newPaymentByOrderIDCommand(),
		newPaymentIssuersCommand(),
		newPaymentRefundCommand(),
		newPaymentCancelCommand(),
		newPaymentCaptureCommand(),
		newPaymentCancelOrRefundCommand(),
		newPaymentDisputeCommand(),
		newPaymentFulfillmentsCommand(),
	)

	return payment
}

func newPaymentCreateCommand() *cobra.Command {
	command := &cobra.Command{
		Annotations: apiOperations("POST /payments"),
		Use:         "create",
		Short:       "Create a payment",
		Example:     "  yuno-cli payment create --file payment.json --idempotency-key <uuid>",
		Args:        cobra.NoArgs,
		RunE:        runPaymentCreate,
	}

	registerWriteFlags(command, paymentFields)

	return command
}

func runPaymentCreate(cmd *cobra.Command, _ []string) error {
	body, err := requireBody(cmd, paymentFields)
	if err != nil {
		return err
	}

	client, err := newClientFromFlags(cmd)
	if err != nil {
		return err
	}

	payment, err := client.CreatePayment(cmd.Context(), body)
	if err != nil {
		return scopeHint(err, paymentScopes)
	}

	return printPayment(cmd, payment)
}

func newPaymentGetCommand() *cobra.Command {
	command := &cobra.Command{
		Annotations: apiOperations("GET /payments/{payment_id}"),
		Use:         "get <payment_id>",
		Short:       "Retrieve one payment",
		Example:     "  yuno-cli payment get e3f397ed --json | jq '.transactions[].provider_data'",
		Args:        cobra.ExactArgs(1),
		RunE:        runPaymentGet,
	}

	command.Flags().Bool("raw-response", false, "include the raw provider responses")
	command.Flags().Bool("transactions-history", false, "include the full transaction history")

	return command
}

func runPaymentGet(cmd *cobra.Command, args []string) error {
	client, err := newClientFromFlags(cmd)
	if err != nil {
		return err
	}

	payment, err := client.GetPayment(cmd.Context(), args[0],
		flagBool(cmd, "raw-response"), flagBool(cmd, "transactions-history"))
	if err != nil {
		return scopeHint(err, paymentScopes)
	}

	return printPayment(cmd, payment)
}

// newPaymentListCommand exposes `GET /payments`, which Yuno only serves scoped
// to one merchant order id - there is no unfiltered payment list.
func newPaymentListCommand() *cobra.Command {
	command := &cobra.Command{
		Annotations: apiOperations("GET /payments"),
		Use:         "list",
		Short:       "List the payments of a merchant order",
		Example:     "  yuno-cli payment list --merchant-order-id order-42",
		Args:        cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runPaymentByOrderID(cmd, flagString(cmd, "merchant-order-id"))
		},
	}

	command.Flags().String("merchant-order-id", "", "merchant order to list the payments of")
	_ = command.MarkFlagRequired("merchant-order-id")

	return command
}

func newPaymentByOrderIDCommand() *cobra.Command {
	return &cobra.Command{
		Annotations: apiOperations("GET /payments"),
		Use:         "get-by-order-id <merchant_order_id>",
		Short:       "Retrieve the payments of a merchant order",
		Example:     "  yuno-cli payment get-by-order-id order-42",
		Args:        cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runPaymentByOrderID(cmd, args[0])
		},
	}
}

func runPaymentByOrderID(cmd *cobra.Command, merchantOrderID string) error {
	if merchantOrderID == "" {
		return fmt.Errorf("merchant order id must not be empty")
	}

	client, err := newClientFromFlags(cmd)
	if err != nil {
		return err
	}

	payments, err := client.GetPaymentByMerchantOrderID(cmd.Context(), merchantOrderID)
	if err != nil {
		return scopeHint(err, paymentScopes)
	}

	return printResult(cmd, payments, model.PaymentViews(payments))
}

func newPaymentIssuersCommand() *cobra.Command {
	command := &cobra.Command{
		Annotations: apiOperations("GET /issuers"),
		Use:         "issuers",
		Short:       "List the banks available for a payment method",
		Example:     "  yuno-cli payment issuers --country-code CO --payment-method PSE",
		Args:        cobra.NoArgs,
		RunE:        runPaymentIssuers,
	}

	flags := command.Flags()
	flags.String("country-code", "", "country to list the issuers of, e.g. CO")
	flags.String("payment-method", "", "payment method to list the issuers of, e.g. PSE")
	flags.String("checkout-session", "", "checkout session to scope the issuers to")

	return command
}

func runPaymentIssuers(cmd *cobra.Command, _ []string) error {
	client, err := newClientFromFlags(cmd)
	if err != nil {
		return err
	}

	issuers, err := client.ListIssuers(cmd.Context(),
		flagString(cmd, "country-code"),
		flagString(cmd, "payment-method"),
		flagString(cmd, "checkout-session"))
	if err != nil {
		return scopeHint(err, paymentScopes)
	}

	return printResult(cmd, issuers, issuers.Views())
}

// printPayment renders one payment: the whole response as JSON, a single table
// row otherwise.
func printPayment(cmd *cobra.Command, payment *model.Payment) error {
	return printResult(cmd, payment, []model.PaymentView{payment.View()})
}
