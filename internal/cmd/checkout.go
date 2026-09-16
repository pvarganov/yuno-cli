package cmd

import (
	"github.com/spf13/cobra"

	"github.com/pvarganov/yuno-cli/internal/model"
)

// checkoutScopes are the scopes a 403 on the checkout commands usually asks for.
const checkoutScopes = "checkout:read/checkout:write"

// checkoutSessionFields are the named body fields of `checkout-session create`
// and `checkout-session update`. Metadata, installments and the recurring
// payment block are too nested for flags and come from --file.
var checkoutSessionFields = []FieldFlag{
	{Flag: "account-id", Field: "account_id", Kind: FieldString, Usage: "account the session belongs to"},
	{Flag: "customer-id", Field: "customer_id", Kind: FieldString, Usage: "customer the session belongs to"},
	{Flag: "merchant-order-id", Field: "merchant_order_id", Kind: FieldString, Usage: "your own id for the order"},
	{Flag: "payment-description", Field: "payment_description", Kind: FieldString, Usage: "description shown to the customer"},
	{Flag: "callback-url", Field: "callback_url", Kind: FieldString, Usage: "url the customer returns to"},
	{Flag: "country", Field: "country", Kind: FieldString, Usage: "country the transaction is processed in, e.g. CO"},
	{Flag: "currency", Field: "amount.currency", Kind: FieldString, Usage: "currency of the payment, e.g. USD"},
	{Flag: "amount", Field: "amount.value", Kind: FieldNumber, Usage: "value of the payment"},
	{Flag: "checkout-id", Field: "checkout_id", Kind: FieldString, Usage: "checkout the session is opened for"},
	{Flag: "workflow", Field: "workflow", Kind: FieldString, Usage: "checkout workflow, e.g. SDK_CHECKOUT"},
	{Flag: "metadata", Field: "metadata", Kind: FieldJSON, Usage: "metadata as a JSON array"},
	{Flag: "installments", Field: "installments", Kind: FieldJSON, Usage: "installments as a JSON object"},
}

// enrollFields are the body fields of `checkout payment-method enroll`.
var enrollFields = []FieldFlag{
	{Flag: "account-id", Field: "account_id", Kind: FieldString, Usage: "account the payment method is enrolled for"},
	{Flag: "payment-method-type", Field: "payment_method_type", Kind: FieldString, Usage: "payment method type, e.g. CARD"},
	{Flag: "country", Field: "country", Kind: FieldString, Usage: "country of the enrollment, e.g. CO"},
	{Flag: "account-updater", Field: "account_updater", Kind: FieldBool, Usage: "include this card in the card account updater"},
	{Flag: "vault-on-success", Field: "verify.vault_on_success", Kind: FieldBool, Usage: "vault the card only when the verification succeeds"},
}

// newCheckoutSessionCommand builds the `checkout-session` command tree.
func newCheckoutSessionCommand() *cobra.Command {
	session := &cobra.Command{
		Use:     "checkout-session",
		Short:   "Open, inspect and update checkout sessions",
		Aliases: []string{"checkout-sessions"},
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return cmd.Help()
		},
	}

	session.AddCommand(
		newCheckoutSessionCreateCommand(),
		newCheckoutSessionGetCommand(),
		newCheckoutSessionUpdateCommand(),
		newCheckoutSessionPaymentMethodsCommand(),
	)

	return session
}

func newCheckoutSessionCreateCommand() *cobra.Command {
	command := &cobra.Command{
		Annotations: apiOperations("POST /checkout/sessions"),
		Use:         "create",
		Short:       "Open a checkout session",
		Example:     "  yuno-cli checkout-session create --account-id acc-1 --merchant-order-id order-42 --country CO",
		Args:        cobra.NoArgs,
		RunE:        runCheckoutSessionCreate,
	}

	registerWriteFlags(command, checkoutSessionFields)

	return command
}

func runCheckoutSessionCreate(cmd *cobra.Command, _ []string) error {
	body, err := requireBody(cmd, checkoutSessionFields)
	if err != nil {
		return err
	}

	client, err := newClientFromFlags(cmd)
	if err != nil {
		return err
	}

	session, err := client.CreateCheckoutSession(cmd.Context(), body)
	if err != nil {
		return scopeHint(err, checkoutScopes)
	}

	return printCheckoutSession(cmd, session)
}

func newCheckoutSessionGetCommand() *cobra.Command {
	return &cobra.Command{
		Annotations: apiOperations("GET /checkout/sessions/{checkout_session}"),
		Use:         "get <checkout_session>",
		Short:       "Retrieve one checkout session",
		Example:     "  yuno-cli checkout-session get d313047b --json",
		Args:        cobra.ExactArgs(1),
		RunE:        runCheckoutSessionGet,
	}
}

func runCheckoutSessionGet(cmd *cobra.Command, args []string) error {
	client, err := newClientFromFlags(cmd)
	if err != nil {
		return err
	}

	session, err := client.GetCheckoutSession(cmd.Context(), args[0])
	if err != nil {
		return scopeHint(err, checkoutScopes)
	}

	return printCheckoutSession(cmd, session)
}

func newCheckoutSessionUpdateCommand() *cobra.Command {
	command := &cobra.Command{
		Annotations: apiOperations("PATCH /checkout/sessions/{checkout_session}"),
		Use:         "update <checkout_session>",
		Short:       "Update a checkout session that has not been used yet",
		Example:     "  yuno-cli checkout-session update d313047b --amount 520 --currency USD",
		Args:        cobra.ExactArgs(1),
		RunE:        runCheckoutSessionUpdate,
	}

	registerWriteFlags(command, checkoutSessionFields)

	return command
}

func runCheckoutSessionUpdate(cmd *cobra.Command, args []string) error {
	body, err := requireBody(cmd, checkoutSessionFields)
	if err != nil {
		return err
	}

	client, err := newClientFromFlags(cmd)
	if err != nil {
		return err
	}

	session, err := client.UpdateCheckoutSession(cmd.Context(), args[0], body)
	if err != nil {
		return scopeHint(err, checkoutScopes)
	}

	return printCheckoutSession(cmd, session)
}

func newCheckoutSessionPaymentMethodsCommand() *cobra.Command {
	command := &cobra.Command{
		Annotations: apiOperations("GET /checkout/sessions/{checkout_session}/payment-methods"),
		Use:         "payment-methods <checkout_session>",
		Short:       "List the payment methods available for a checkout session",
		Example:     "  yuno-cli checkout-session payment-methods d313047b --category CARD",
		Args:        cobra.ExactArgs(1),
		RunE:        runCheckoutSessionPaymentMethods,
	}

	command.Flags().String("category", "", "only list payment methods of this category, e.g. CARD")

	return command
}

func runCheckoutSessionPaymentMethods(cmd *cobra.Command, args []string) error {
	client, err := newClientFromFlags(cmd)
	if err != nil {
		return err
	}

	methods, err := client.ListCheckoutSessionPaymentMethods(cmd.Context(), args[0], flagString(cmd, "category"))
	if err != nil {
		return scopeHint(err, checkoutScopes)
	}

	return printResult(cmd, methods, model.CheckoutPaymentMethodViews(methods))
}

// newCheckoutCommand builds the `checkout` command tree, which holds the
// payment method operations of the checkout workflow.
func newCheckoutCommand() *cobra.Command {
	checkout := &cobra.Command{
		Use:   "checkout",
		Short: "Work with the payment methods of the checkout workflow",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return cmd.Help()
		},
	}

	paymentMethod := &cobra.Command{
		Use:     "payment-method",
		Short:   "Enroll, inspect and unenroll payment methods of a customer session",
		Aliases: []string{"payment-methods"},
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return cmd.Help()
		},
	}

	paymentMethod.AddCommand(
		newCheckoutPaymentMethodListCommand(),
		newCheckoutPaymentMethodEnrollCommand(),
		newCheckoutPaymentMethodGetCommand(),
		newCheckoutPaymentMethodUnenrollCommand(),
	)
	checkout.AddCommand(paymentMethod)

	return checkout
}

func newCheckoutPaymentMethodListCommand() *cobra.Command {
	return &cobra.Command{
		Annotations: apiOperations("GET /checkout/customers/sessions/{customer_session}/payment-methods"),
		Use:         "list <customer_session>",
		Short:       "List the payment methods a customer session can enroll",
		Example:     "  yuno-cli checkout payment-method list 6641e30d",
		Args:        cobra.ExactArgs(1),
		RunE:        runCheckoutPaymentMethodList,
	}
}

func runCheckoutPaymentMethodList(cmd *cobra.Command, args []string) error {
	client, err := newClientFromFlags(cmd)
	if err != nil {
		return err
	}

	methods, err := client.ListEnrollablePaymentMethods(cmd.Context(), args[0])
	if err != nil {
		return scopeHint(err, checkoutScopes)
	}

	return printResult(cmd, methods, model.CheckoutPaymentMethodViews(methods))
}

func newCheckoutPaymentMethodEnrollCommand() *cobra.Command {
	command := &cobra.Command{
		Annotations: apiOperations("POST /customers/sessions/{customer_session}/payment-methods"),
		Use:         "enroll <customer_session>",
		Short:       "Enroll a payment method in a customer session",
		Example:     "  yuno-cli checkout payment-method enroll 6641e30d --account-id acc-1 --payment-method-type CARD --country US",
		Args:        cobra.ExactArgs(1),
		RunE:        runCheckoutPaymentMethodEnroll,
	}

	registerWriteFlags(command, enrollFields)

	return command
}

func runCheckoutPaymentMethodEnroll(cmd *cobra.Command, args []string) error {
	body, err := requireBody(cmd, enrollFields)
	if err != nil {
		return err
	}

	client, err := newClientFromFlags(cmd)
	if err != nil {
		return err
	}

	method, err := client.EnrollPaymentMethod(cmd.Context(), args[0], body)
	if err != nil {
		return scopeHint(err, checkoutScopes)
	}

	return printCustomerPaymentMethod(cmd, method)
}

func newCheckoutPaymentMethodGetCommand() *cobra.Command {
	return &cobra.Command{
		Annotations: apiOperations("GET /payment-methods/{payment_method_id}"),
		Use:         "get <payment_method_id>",
		Short:       "Retrieve one payment method by id",
		Example:     "  yuno-cli checkout payment-method get 0395199e --json",
		Args:        cobra.ExactArgs(1),
		RunE:        runCheckoutPaymentMethodGet,
	}
}

func runCheckoutPaymentMethodGet(cmd *cobra.Command, args []string) error {
	client, err := newClientFromFlags(cmd)
	if err != nil {
		return err
	}

	method, err := client.GetPaymentMethod(cmd.Context(), args[0])
	if err != nil {
		return scopeHint(err, checkoutScopes)
	}

	return printCustomerPaymentMethod(cmd, method)
}

func newCheckoutPaymentMethodUnenrollCommand() *cobra.Command {
	command := &cobra.Command{
		Annotations: apiOperations("POST /customers/payment-methods/{payment_method_id}/unenroll"),
		Use:         "unenroll <payment_method_id>",
		Short:       "Unenroll a payment method of a customer",
		Example:     "  yuno-cli checkout payment-method unenroll 77ee4a02 --yes",
		Args:        cobra.ExactArgs(1),
		RunE:        runCheckoutPaymentMethodUnenroll,
	}

	command.Flags().String("idempotency-key", "", "pin the X-Idempotency-Key of the request")

	return command
}

func runCheckoutPaymentMethodUnenroll(cmd *cobra.Command, args []string) error {
	client, err := newClientFromFlags(cmd)
	if err != nil {
		return err
	}

	method, err := client.UnenrollPaymentMethod(cmd.Context(), args[0])
	if err != nil {
		return scopeHint(err, checkoutScopes)
	}

	return printCustomerPaymentMethod(cmd, method)
}

// printCheckoutSession renders one checkout session.
func printCheckoutSession(cmd *cobra.Command, session *model.CheckoutSession) error {
	return printResult(cmd, session, []model.CheckoutSessionView{session.View()})
}

// printCustomerPaymentMethod renders one enrolled payment method.
func printCustomerPaymentMethod(cmd *cobra.Command, method *model.CustomerPaymentMethod) error {
	return printResult(cmd, method, []model.CustomerPaymentMethodView{method.View()})
}
