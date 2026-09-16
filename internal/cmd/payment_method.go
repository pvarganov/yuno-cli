package cmd

import (
	"github.com/spf13/cobra"

	"github.com/pvarganov/yuno-cli/internal/model"
)

// paymentMethodScopes are the scopes a 403 on the payment method commands
// usually asks for.
const paymentMethodScopes = "payment-method:read/payment-method:write"

// paymentMethodEnrollFields are the body fields of `payment-method enroll`, the
// direct workflow enrollment.
var paymentMethodEnrollFields = []FieldFlag{
	{Flag: "account-id", Field: "account_id", Kind: FieldString, Usage: "account the payment method is enrolled for"},
	{Flag: "type", Field: "type", Kind: FieldString, Usage: "payment method type, e.g. CARD"},
	{Flag: "country", Field: "country", Kind: FieldString, Usage: "country of the enrollment, e.g. US"},
	{Flag: "workflow", Field: "workflow", Kind: FieldString, Usage: "enrollment workflow, DIRECT for this command"},
	{Flag: "callback-url", Field: "callback_url", Kind: FieldString, Usage: "url the customer returns to"},
	{Flag: "token", Field: "token", Kind: FieldString, Usage: "pre-generated token of the payment method"},
	{Flag: "parent-type", Field: "parent_type", Kind: FieldString, Usage: "type of the parent payment method"},
	{Flag: "account-updater", Field: "account_updater", Kind: FieldBool, Usage: "include this card in the card account updater"},
	{Flag: "vault-on-success", Field: "verify.vault_on_success", Kind: FieldBool, Usage: "vault the card only when the verification succeeds"},
	{Flag: "card", Field: "card", Kind: FieldJSON, Usage: "card data as a JSON object"},
}

// paymentMethodUpdateFields are the body fields of `payment-method update`,
// which reassigns a payment method to another customer.
var paymentMethodUpdateFields = []FieldFlag{
	{Flag: "customer-id", Field: "customer_id", Kind: FieldString, Usage: "customer the payment method is reassigned to"},
}

// accountUpdaterFields are the body fields of `payment-method account-updater`.
var accountUpdaterFields = []FieldFlag{
	{
		Flag: "payment-method-id", Field: "payment_method_ids", Kind: FieldStringSlice,
		Usage: "stored payment method id (vaulted_token) to register, repeatable",
	},
}

// newPaymentMethodCommand builds the `payment-method` command tree of the
// direct workflow, scoped to a customer.
func newPaymentMethodCommand() *cobra.Command {
	paymentMethod := &cobra.Command{
		Use:     "payment-method",
		Short:   "Enroll, inspect and reassign the payment methods of a customer",
		Aliases: []string{"payment-methods"},
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return cmd.Help()
		},
	}

	paymentMethod.AddCommand(
		newPaymentMethodListCommand(),
		newPaymentMethodGetCommand(),
		newPaymentMethodEnrollCommand(),
		newPaymentMethodUnenrollCommand(),
		newPaymentMethodUpdateCommand(),
		newPaymentMethodAccountUpdaterCommand(),
	)

	return paymentMethod
}

func newPaymentMethodListCommand() *cobra.Command {
	return &cobra.Command{
		Use:     "list <customer_id>",
		Short:   "List the payment methods enrolled for a customer",
		Example: "  yuno-cli payment-method list cus-1",
		Args:    cobra.ExactArgs(1),
		RunE:    runPaymentMethodList,
	}
}

func runPaymentMethodList(cmd *cobra.Command, args []string) error {
	client, err := newClientFromFlags(cmd)
	if err != nil {
		return err
	}

	methods, err := client.ListCustomerPaymentMethods(cmd.Context(), args[0])
	if err != nil {
		return scopeHint(err, paymentMethodScopes)
	}

	return printResult(cmd, methods, model.CustomerPaymentMethodViews(methods))
}

func newPaymentMethodGetCommand() *cobra.Command {
	return &cobra.Command{
		Use:     "get <customer_id> <payment_method_id>",
		Short:   "Retrieve one payment method enrolled for a customer",
		Example: "  yuno-cli payment-method get cus-1 pm-1 --json",
		Args:    cobra.ExactArgs(2),
		RunE:    runPaymentMethodGet,
	}
}

func runPaymentMethodGet(cmd *cobra.Command, args []string) error {
	client, err := newClientFromFlags(cmd)
	if err != nil {
		return err
	}

	method, err := client.GetCustomerPaymentMethod(cmd.Context(), args[0], args[1])
	if err != nil {
		return scopeHint(err, paymentMethodScopes)
	}

	return printCustomerPaymentMethod(cmd, method)
}

func newPaymentMethodEnrollCommand() *cobra.Command {
	command := &cobra.Command{
		Use:     "enroll <customer_id>",
		Short:   "Enroll a payment method for a customer through the direct workflow",
		Example: "  yuno-cli payment-method enroll cus-1 --account-id acc-1 --type CARD --country US",
		Args:    cobra.ExactArgs(1),
		RunE:    runPaymentMethodEnroll,
	}

	registerWriteFlags(command, paymentMethodEnrollFields)

	return command
}

func runPaymentMethodEnroll(cmd *cobra.Command, args []string) error {
	body, err := requireBody(cmd, paymentMethodEnrollFields)
	if err != nil {
		return err
	}

	client, err := newClientFromFlags(cmd)
	if err != nil {
		return err
	}

	method, err := client.EnrollCustomerPaymentMethod(cmd.Context(), args[0], body)
	if err != nil {
		return scopeHint(err, paymentMethodScopes)
	}

	return printCustomerPaymentMethod(cmd, method)
}

func newPaymentMethodUnenrollCommand() *cobra.Command {
	command := &cobra.Command{
		Use:     "unenroll <customer_id> <payment_method_id>",
		Short:   "Unenroll a payment method of a customer",
		Example: "  yuno-cli payment-method unenroll cus-1 pm-1 --yes",
		Args:    cobra.ExactArgs(2),
		RunE:    runPaymentMethodUnenroll,
	}

	command.Flags().String("idempotency-key", "", "pin the X-Idempotency-Key of the request")

	return command
}

func runPaymentMethodUnenroll(cmd *cobra.Command, args []string) error {
	client, err := newClientFromFlags(cmd)
	if err != nil {
		return err
	}

	method, err := client.UnenrollCustomerPaymentMethod(cmd.Context(), args[0], args[1])
	if err != nil {
		return scopeHint(err, paymentMethodScopes)
	}

	return printCustomerPaymentMethod(cmd, method)
}

func newPaymentMethodUpdateCommand() *cobra.Command {
	command := &cobra.Command{
		Use:   "update <payment_method_id>",
		Short: "Reassign a payment method to another customer",
		Long: "Reassign a payment method to another customer. Yuno accepts the reassignment only " +
			"while the current owner has no data; otherwise the request fails with 409 CONFLICT.",
		Example: "  yuno-cli payment-method update pm-1 --customer-id cus-2 --account-code acc-code",
		Args:    cobra.ExactArgs(1),
		RunE:    runPaymentMethodUpdate,
	}

	registerWriteFlags(command, paymentMethodUpdateFields)
	command.Flags().String("account-code", "", "X-Account-Code of the request, overriding the profile one")

	return command
}

func runPaymentMethodUpdate(cmd *cobra.Command, args []string) error {
	body, err := requireBody(cmd, paymentMethodUpdateFields)
	if err != nil {
		return err
	}

	client, err := newClientFromFlags(cmd)
	if err != nil {
		return err
	}

	method, err := client.ReassignPaymentMethod(cmd.Context(), args[0], flagString(cmd, "account-code"), body)
	if err != nil {
		return scopeHint(err, paymentMethodScopes)
	}

	return printCustomerPaymentMethod(cmd, method)
}

func newPaymentMethodAccountUpdaterCommand() *cobra.Command {
	command := &cobra.Command{
		Use:     "account-updater",
		Short:   "Register stored cards for the card account updater",
		Example: "  yuno-cli payment-method account-updater --payment-method-id pm-1 --payment-method-id pm-2",
		Args:    cobra.NoArgs,
		RunE:    runPaymentMethodAccountUpdater,
	}

	registerWriteFlags(command, accountUpdaterFields)

	return command
}

func runPaymentMethodAccountUpdater(cmd *cobra.Command, _ []string) error {
	body, err := requireBody(cmd, accountUpdaterFields)
	if err != nil {
		return err
	}

	client, err := newClientFromFlags(cmd)
	if err != nil {
		return err
	}

	result, err := client.RegisterCardsForAccountUpdater(cmd.Context(), body)
	if err != nil {
		return scopeHint(err, paymentMethodScopes)
	}

	return printResult(cmd, result, []model.AccountUpdaterView{result.View()})
}
