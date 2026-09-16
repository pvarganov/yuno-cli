package cmd

import (
	"github.com/spf13/cobra"

	"github.com/pvarganov/yuno-cli/internal/model"
)

// refundFields are the body fields of the refund and cancel-or-refund actions.
var refundFields = []FieldFlag{
	{Flag: "merchant-reference", Field: "merchant_reference", Kind: FieldString, Usage: "your own reference for the refund"},
	{Flag: "reason", Field: "reason", Kind: FieldString, Usage: "reason of the refund, e.g. REQUESTED_BY_CUSTOMER"},
	{Flag: "description", Field: "description", Kind: FieldString, Usage: "free text description"},
	{Flag: "currency", Field: "amount.currency", Kind: FieldString, Usage: "currency of a partial refund"},
	{Flag: "amount", Field: "amount.value", Kind: FieldNumber, Usage: "amount of a partial refund"},
}

// cancelFields are the body fields of the cancel action.
var cancelFields = []FieldFlag{
	{Flag: "merchant-reference", Field: "merchant_reference", Kind: FieldString, Usage: "your own reference for the cancellation"},
	{Flag: "reason", Field: "reason", Kind: FieldString, Usage: "reason of the cancellation"},
	{Flag: "description", Field: "description", Kind: FieldString, Usage: "free text description"},
}

// captureFields are the body fields of the capture action.
var captureFields = []FieldFlag{
	{Flag: "merchant-reference", Field: "merchant_reference", Kind: FieldString, Usage: "your own reference for the capture"},
	{Flag: "reason", Field: "reason", Kind: FieldString, Usage: "reason of the capture"},
	{Flag: "currency", Field: "amount.currency", Kind: FieldString, Usage: "currency of the captured amount"},
	{Flag: "amount", Field: "amount.value", Kind: FieldNumber, Usage: "amount to capture"},
}

// disputeFields are the body fields of the dispute actions.
var disputeFields = []FieldFlag{
	{Flag: "account-id", Field: "account_id", Kind: FieldString, Usage: "account the disputed payment belongs to"},
	{Flag: "evidence", Field: "evidence", Kind: FieldJSON, Usage: "dispute evidence as a JSON object"},
}

// fulfillmentFields are the body fields of the fulfillments action.
var fulfillmentFields = []FieldFlag{
	{Flag: "status", Field: "status", Kind: FieldString, Usage: "fulfillment status, e.g. FULFILLED"},
	{Flag: "fulfillments", Field: "fulfillments", Kind: FieldJSON, Usage: "fulfillments as a JSON array"},
	{Flag: "carriers", Field: "carriers", Kind: FieldJSON, Usage: "carriers as a JSON array"},
}

func newPaymentRefundCommand() *cobra.Command {
	command := &cobra.Command{
		Use:     "refund <payment_id> <transaction_id>",
		Short:   "Refund one transaction of a payment",
		Example: "  yuno-cli payment refund p-1 t-1 --merchant-reference ref-1",
		Args:    cobra.ExactArgs(2),
		RunE:    runPaymentRefund,
	}

	registerWriteFlags(command, refundFields)

	return command
}

func runPaymentRefund(cmd *cobra.Command, args []string) error {
	body, err := bodyFromFlags(cmd, refundFields)
	if err != nil {
		return err
	}

	client, err := newClientFromFlags(cmd)
	if err != nil {
		return err
	}

	payment, err := client.RefundPayment(cmd.Context(), args[0], args[1], body)
	if err != nil {
		return scopeHint(err, paymentScopes)
	}

	return printPayment(cmd, payment)
}

func newPaymentCancelCommand() *cobra.Command {
	command := &cobra.Command{
		Use:     "cancel <payment_id> <transaction_id>",
		Short:   "Cancel one transaction of a payment",
		Example: "  yuno-cli payment cancel p-1 t-1 --merchant-reference ref-1",
		Args:    cobra.ExactArgs(2),
		RunE:    runPaymentCancel,
	}

	registerWriteFlags(command, cancelFields)

	return command
}

func runPaymentCancel(cmd *cobra.Command, args []string) error {
	body, err := bodyFromFlags(cmd, cancelFields)
	if err != nil {
		return err
	}

	client, err := newClientFromFlags(cmd)
	if err != nil {
		return err
	}

	payment, err := client.CancelTransaction(cmd.Context(), args[0], args[1], body)
	if err != nil {
		return scopeHint(err, paymentScopes)
	}

	return printPayment(cmd, payment)
}

func newPaymentCaptureCommand() *cobra.Command {
	command := &cobra.Command{
		Use:     "capture <payment_id> <transaction_id>",
		Short:   "Capture an authorized transaction of a payment",
		Example: "  yuno-cli payment capture p-1 t-1 --amount 150 --currency USD --reason CAPTURE",
		Args:    cobra.ExactArgs(2),
		RunE:    runPaymentCapture,
	}

	registerWriteFlags(command, captureFields)

	return command
}

func runPaymentCapture(cmd *cobra.Command, args []string) error {
	body, err := bodyFromFlags(cmd, captureFields)
	if err != nil {
		return err
	}

	client, err := newClientFromFlags(cmd)
	if err != nil {
		return err
	}

	payment, err := client.CaptureTransaction(cmd.Context(), args[0], args[1], body)
	if err != nil {
		return scopeHint(err, paymentScopes)
	}

	return printPayment(cmd, payment)
}

// newPaymentCancelOrRefundCommand covers both variants of the operation: with
// a transaction id it targets one transaction, without it the whole payment.
func newPaymentCancelOrRefundCommand() *cobra.Command {
	command := &cobra.Command{
		Use:     "cancel-or-refund <payment_id> [transaction_id]",
		Short:   "Let Yuno choose between cancelling and refunding",
		Example: "  yuno-cli payment cancel-or-refund p-1 --reason REQUESTED_BY_CUSTOMER",
		Args:    cobra.RangeArgs(1, 2),
		RunE:    runPaymentCancelOrRefund,
	}

	registerWriteFlags(command, refundFields)

	return command
}

func runPaymentCancelOrRefund(cmd *cobra.Command, args []string) error {
	body, err := bodyFromFlags(cmd, refundFields)
	if err != nil {
		return err
	}

	client, err := newClientFromFlags(cmd)
	if err != nil {
		return err
	}

	var payment *model.Payment

	if len(args) == 2 {
		payment, err = client.CancelOrRefundTransaction(cmd.Context(), args[0], args[1], body)
	} else {
		payment, err = client.CancelOrRefundPayment(cmd.Context(), args[0], body)
	}

	if err != nil {
		return scopeHint(err, paymentScopes)
	}

	return printPayment(cmd, payment)
}

func newPaymentDisputeCommand() *cobra.Command {
	dispute := &cobra.Command{
		Use:     "dispute",
		Short:   "Submit and update the evidence of a disputed transaction",
		Aliases: []string{"disputes"},
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return cmd.Help()
		},
	}

	create := &cobra.Command{
		Use:     "create <payment_id> <transaction_id>",
		Short:   "Submit the evidence of a disputed transaction",
		Example: "  yuno-cli payment dispute create p-1 t-1 --file evidence.json",
		Args:    cobra.ExactArgs(2),
		RunE:    runPaymentDisputeCreate,
	}
	registerWriteFlags(create, disputeFields)

	update := &cobra.Command{
		Use:     "update <payment_id> <transaction_id>",
		Short:   "Replace the evidence of a disputed transaction",
		Example: "  yuno-cli payment dispute update p-1 t-1 --file evidence.json",
		Args:    cobra.ExactArgs(2),
		RunE:    runPaymentDisputeUpdate,
	}
	registerWriteFlags(update, disputeFields)

	dispute.AddCommand(create, update)

	return dispute
}

func runPaymentDisputeCreate(cmd *cobra.Command, args []string) error {
	body, err := requireBody(cmd, disputeFields)
	if err != nil {
		return err
	}

	client, err := newClientFromFlags(cmd)
	if err != nil {
		return err
	}

	data, err := client.CreateDispute(cmd.Context(), args[0], args[1], body)
	if err != nil {
		return scopeHint(err, paymentScopes)
	}

	return printJSON(cmd, data)
}

func runPaymentDisputeUpdate(cmd *cobra.Command, args []string) error {
	body, err := requireBody(cmd, disputeFields)
	if err != nil {
		return err
	}

	client, err := newClientFromFlags(cmd)
	if err != nil {
		return err
	}

	data, err := client.UpdateDispute(cmd.Context(), args[0], args[1], body)
	if err != nil {
		return scopeHint(err, paymentScopes)
	}

	return printJSON(cmd, data)
}

func newPaymentFulfillmentsCommand() *cobra.Command {
	command := &cobra.Command{
		Use:     "fulfillments <payment_id>",
		Short:   "Report the fulfillment status of a payment",
		Aliases: []string{"fulfillment"},
		Example: "  yuno-cli payment fulfillments p-1 --file fulfillments.json",
		Args:    cobra.ExactArgs(1),
		RunE:    runPaymentFulfillments,
	}

	registerWriteFlags(command, fulfillmentFields)

	return command
}

func runPaymentFulfillments(cmd *cobra.Command, args []string) error {
	body, err := requireBody(cmd, fulfillmentFields)
	if err != nil {
		return err
	}

	client, err := newClientFromFlags(cmd)
	if err != nil {
		return err
	}

	data, err := client.CreateFulfillment(cmd.Context(), args[0], body)
	if err != nil {
		return scopeHint(err, paymentScopes)
	}

	return printJSON(cmd, data)
}
