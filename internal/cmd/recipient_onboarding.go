package cmd

import (
	"github.com/spf13/cobra"

	"github.com/pvarganov/yuno-cli/internal/model"
)

// onboardingFields are the named body fields of `recipient onboarding create`,
// `update` and `continue`. The documentation, the withdrawal methods and the
// legal representatives are arrays of objects, so they come from --file or from
// their JSON flags.
var onboardingFields = []FieldFlag{
	{Flag: "account-id", Field: "account_id", Kind: FieldString, Usage: "account the onboarding belongs to"},
	{Flag: "type", Field: "type", Kind: FieldString, Usage: "onboarding type, e.g. INDIVIDUAL"},
	{Flag: "workflow", Field: "workflow", Kind: FieldString, Usage: "onboarding workflow, e.g. AUTOMATIC"},
	{Flag: "description", Field: "description", Kind: FieldString, Usage: "description of the onboarding"},
	{Flag: "callback-url", Field: "callback_url", Kind: FieldString, Usage: "url Yuno calls back when it finishes"},
	{Flag: "provider-id", Field: "provider.id", Kind: FieldString, Usage: "provider the recipient is onboarded with"},
	{
		Flag: "connection-id", Field: "provider.connection_id", Kind: FieldString,
		Usage: "connection the onboarding is sent through",
	},
	{
		Flag: "provider-recipient-id", Field: "provider.recipient_id", Kind: FieldString,
		Usage: "id the recipient already has at the provider",
	},
	{
		Flag: "provider-recipient-type", Field: "provider.recipient_type", Kind: FieldString,
		Usage: "recipient type at the provider",
	},
	{
		Flag: "terms-accepted", Field: "terms_of_service.acceptance", Kind: FieldBool,
		Usage: "whether the recipient accepted the terms of service",
	},
	{
		Flag: "terms-date", Field: "terms_of_service.date", Kind: FieldString,
		Usage: "date the terms of service were accepted",
	},
	{
		Flag: "terms-ip", Field: "terms_of_service.ip", Kind: FieldString,
		Usage: "ip the terms of service were accepted from",
	},
	{
		Flag: "withdrawal-methods", Field: "withdrawal_methods", Kind: FieldJSON,
		Usage: "withdrawal methods as a JSON object",
	},
	{
		Flag: "legal-representatives", Field: "legal_representatives", Kind: FieldJSON,
		Usage: "legal representatives as a JSON array",
	},
	{Flag: "documentation", Field: "documentation", Kind: FieldJSON, Usage: "documentation as a JSON array"},
}

// onboardingActions are the bodyless lifecycle actions of an onboarding.
var onboardingActions = []struct {
	Name  string
	Short string
}{
	{Name: "cancel", Short: "Cancel an onboarding"},
	{Name: "block", Short: "Block an onboarded recipient"},
	{Name: "unblock", Short: "Unblock a blocked recipient"},
}

// newRecipientOnboardingCommand builds the `recipient onboarding` subtree.
func newRecipientOnboardingCommand() *cobra.Command {
	onboarding := &cobra.Command{
		Use:     "onboarding",
		Short:   "Run the onboarding flow of a marketplace recipient",
		Aliases: []string{"onboardings"},
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return cmd.Help()
		},
	}

	onboarding.AddCommand(
		newOnboardingCreateCommand(),
		newOnboardingGetCommand(),
		newOnboardingUpdateCommand(),
		newOnboardingContinueCommand(),
		newOnboardingTransfersCommand(),
	)

	for _, action := range onboardingActions {
		onboarding.AddCommand(newOnboardingActionCommand(action.Name, action.Short))
	}

	return onboarding
}

func newOnboardingCreateCommand() *cobra.Command {
	command := &cobra.Command{
		Annotations: apiOperations("POST /recipients/{recipient_id}/onboardings"),
		Use:         "create <recipient_id>",
		Short:       "Start an onboarding for a recipient",
		Example: "  yuno-cli recipient onboarding create rec-1 --account-id acc-1 " +
			"--type INDIVIDUAL --workflow AUTOMATIC --provider-id NUVEI",
		Args: cobra.ExactArgs(1),
		RunE: runOnboardingCreate,
	}

	registerWriteFlags(command, onboardingFields)

	return command
}

func runOnboardingCreate(cmd *cobra.Command, args []string) error {
	body, err := requireBody(cmd, onboardingFields)
	if err != nil {
		return err
	}

	client, err := newClientFromFlags(cmd)
	if err != nil {
		return err
	}

	recipient, err := client.CreateOnboarding(cmd.Context(), args[0], body)
	if err != nil {
		return scopeHint(err, recipientScopes)
	}

	return printRecipient(cmd, recipient)
}

func newOnboardingGetCommand() *cobra.Command {
	return &cobra.Command{
		Annotations: apiOperations("GET /recipients/{recipient_id}/onboardings/{onboarding_id}"),
		Use:         "get <recipient_id> <onboarding_id>",
		Short:       "Retrieve one onboarding of a recipient",
		Example:     "  yuno-cli recipient onboarding get rec-1 onb-1 --json",
		Args:        cobra.ExactArgs(2),
		RunE:        runOnboardingGet,
	}
}

func runOnboardingGet(cmd *cobra.Command, args []string) error {
	client, err := newClientFromFlags(cmd)
	if err != nil {
		return err
	}

	onboarding, err := client.GetOnboarding(cmd.Context(), args[0], args[1])
	if err != nil {
		return scopeHint(err, recipientScopes)
	}

	return printResult(cmd, onboarding, []model.RecipientOnboardingView{onboarding.View()})
}

func newOnboardingUpdateCommand() *cobra.Command {
	command := &cobra.Command{
		Annotations: apiOperations("PATCH /recipients/{recipient_id}/onboardings/{onboarding_id}"),
		Use:         "update <recipient_id> <onboarding_id>",
		Short:       "Update an onboarding of a recipient",
		Example:     "  yuno-cli recipient onboarding update rec-1 onb-1 --callback-url https://example.com/hook --yes",
		Args:        cobra.ExactArgs(2),
		RunE:        runOnboardingUpdate,
	}

	registerWriteFlags(command, onboardingFields)

	return command
}

func runOnboardingUpdate(cmd *cobra.Command, args []string) error {
	body, err := requireBody(cmd, onboardingFields)
	if err != nil {
		return err
	}

	client, err := newClientFromFlags(cmd)
	if err != nil {
		return err
	}

	recipient, err := client.UpdateOnboarding(cmd.Context(), args[0], args[1], body)
	if err != nil {
		return scopeHint(err, recipientScopes)
	}

	return printRecipient(cmd, recipient)
}

func newOnboardingContinueCommand() *cobra.Command {
	command := &cobra.Command{
		Annotations: apiOperations("POST /recipients/{recipient_id}/onboardings/{onboarding_id}/continue"),
		Use:         "continue <recipient_id> <onboarding_id>",
		Short:       "Continue an onboarding that is waiting for more data",
		Long: "Continue an onboarding that is waiting for more data.\n\n" +
			"This is the only onboarding action that takes a body: it carries the documentation, " +
			"the withdrawal methods or the legal representatives the provider asked for.",
		Example: "  yuno-cli recipient onboarding continue rec-1 onb-1 " +
			`--withdrawal-methods '{"bank":{"code":"001","account":"123"}}' --yes`,
		Args: cobra.ExactArgs(2),
		RunE: runOnboardingContinue,
	}

	registerWriteFlags(command, onboardingFields)

	return command
}

func runOnboardingContinue(cmd *cobra.Command, args []string) error {
	body, err := requireBody(cmd, onboardingFields)
	if err != nil {
		return err
	}

	client, err := newClientFromFlags(cmd)
	if err != nil {
		return err
	}

	recipient, err := client.OnboardingAction(cmd.Context(), args[0], args[1], "continue", body)
	if err != nil {
		return scopeHint(err, recipientScopes)
	}

	return printRecipient(cmd, recipient)
}

// newOnboardingActionCommand builds one of the bodyless lifecycle actions.
func newOnboardingActionCommand(name, short string) *cobra.Command {
	command := &cobra.Command{
		Annotations: apiOperations("POST /recipients/{recipient_id}/onboardings/{onboarding_id}/" + name),
		Use:         name + " <recipient_id> <onboarding_id>",
		Short:       short,
		Example:     "  yuno-cli recipient onboarding " + name + " rec-1 onb-1 --yes",
		Args:        cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runOnboardingAction(cmd, name, args)
		},
	}

	command.Flags().String("idempotency-key", "", "pin the X-Idempotency-Key of the request")

	return command
}

func runOnboardingAction(cmd *cobra.Command, action string, args []string) error {
	client, err := newClientFromFlags(cmd)
	if err != nil {
		return err
	}

	recipient, err := client.OnboardingAction(cmd.Context(), args[0], args[1], action, nil)
	if err != nil {
		return scopeHint(err, recipientScopes)
	}

	return printRecipient(cmd, recipient)
}

func newOnboardingTransfersCommand() *cobra.Command {
	return &cobra.Command{
		Annotations: apiOperations("GET /onboardings/{onboarding_id}/transfers"),
		Use:         "transfers <onboarding_id>",
		Short:       "List the transfers of one onboarding",
		Example:     "  yuno-cli recipient onboarding transfers onb-1",
		Args:        cobra.ExactArgs(1),
		RunE:        runOnboardingTransfers,
	}
}

func runOnboardingTransfers(cmd *cobra.Command, args []string) error {
	client, err := newClientFromFlags(cmd)
	if err != nil {
		return err
	}

	transfers, err := client.ListOnboardingTransfers(cmd.Context(), args[0])
	if err != nil {
		return scopeHint(err, recipientScopes)
	}

	return printResult(cmd, transfers, model.RecipientTransferViews(transfers))
}

// transferReversalFields are the named body fields of
// `recipient transfer reversal`. An amount is optional: leaving it out reverses
// the whole split of the transaction.
var transferReversalFields = []FieldFlag{
	{Flag: "currency", Field: "amount.currency", Kind: FieldString, Usage: "currency of the reversed amount"},
	{Flag: "amount", Field: "amount.value", Kind: FieldNumber, Usage: "amount to reverse, omit to reverse it all"},
	{Flag: "description", Field: "description", Kind: FieldString, Usage: "description of the reversal"},
}

// newRecipientTransferCommand builds the `recipient transfer` subtree: the
// transfers that move a recipient from one onboarding to another, plus the
// reversal of a split marketplace transfer of a payment.
func newRecipientTransferCommand() *cobra.Command {
	transfer := &cobra.Command{
		Use:     "transfer",
		Short:   "Move a recipient between onboardings and reverse split transfers",
		Aliases: []string{"transfers"},
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return cmd.Help()
		},
	}

	transfer.AddCommand(
		newRecipientTransferRequestCommand(),
		newRecipientTransferListCommand(),
		newRecipientTransferGetCommand(),
		newRecipientTransferReverseCommand(),
		newRecipientTransferReversalCommand(),
	)

	return transfer
}

func newRecipientTransferRequestCommand() *cobra.Command {
	command := &cobra.Command{
		Annotations: apiOperations("POST /recipients/{recipient_id}/onboardings/{onboarding_id}/transfer"),
		Use:         "request <recipient_id> <onboarding_id>",
		Short:       "Transfer a recipient onto another onboarding",
		Example:     "  yuno-cli recipient transfer request rec-1 onb-2 --yes",
		Args:        cobra.ExactArgs(2),
		RunE:        runRecipientTransferRequest,
	}

	command.Flags().String("idempotency-key", "", "pin the X-Idempotency-Key of the request")

	return command
}

func runRecipientTransferRequest(cmd *cobra.Command, args []string) error {
	client, err := newClientFromFlags(cmd)
	if err != nil {
		return err
	}

	transfer, err := client.CreateOnboardingTransfer(cmd.Context(), args[0], args[1])
	if err != nil {
		return scopeHint(err, recipientScopes)
	}

	return printRecipientTransfer(cmd, transfer)
}

func newRecipientTransferListCommand() *cobra.Command {
	return &cobra.Command{
		Annotations: apiOperations("GET /recipients/{recipient_id}/transfers"),
		Use:         "list <recipient_id>",
		Short:       "List the onboarding transfers of one recipient",
		Example:     "  yuno-cli recipient transfer list rec-1",
		Args:        cobra.ExactArgs(1),
		RunE:        runRecipientTransferList,
	}
}

func runRecipientTransferList(cmd *cobra.Command, args []string) error {
	client, err := newClientFromFlags(cmd)
	if err != nil {
		return err
	}

	transfers, err := client.ListRecipientTransfers(cmd.Context(), args[0])
	if err != nil {
		return scopeHint(err, recipientScopes)
	}

	return printResult(cmd, transfers, model.RecipientTransferViews(transfers))
}

func newRecipientTransferGetCommand() *cobra.Command {
	return &cobra.Command{
		Annotations: apiOperations("GET /transfers/{transfer_id}"),
		Use:         "get <transfer_id>",
		Short:       "Retrieve one onboarding transfer",
		Example:     "  yuno-cli recipient transfer get tr-1 --json",
		Args:        cobra.ExactArgs(1),
		RunE:        runRecipientTransferGet,
	}
}

func runRecipientTransferGet(cmd *cobra.Command, args []string) error {
	client, err := newClientFromFlags(cmd)
	if err != nil {
		return err
	}

	transfer, err := client.GetRecipientTransfer(cmd.Context(), args[0])
	if err != nil {
		return scopeHint(err, recipientScopes)
	}

	return printRecipientTransfer(cmd, transfer)
}

func newRecipientTransferReverseCommand() *cobra.Command {
	command := &cobra.Command{
		Annotations: apiOperations("POST /recipients/onboardings/reverse-transfer/{transfer_id}"),
		Use:         "reverse <transfer_id>",
		Short:       "Reverse an onboarding transfer",
		Example:     "  yuno-cli recipient transfer reverse tr-1 --yes",
		Args:        cobra.ExactArgs(1),
		RunE:        runRecipientTransferReverse,
	}

	command.Flags().String("idempotency-key", "", "pin the X-Idempotency-Key of the request")

	return command
}

func runRecipientTransferReverse(cmd *cobra.Command, args []string) error {
	client, err := newClientFromFlags(cmd)
	if err != nil {
		return err
	}

	transfer, err := client.ReverseOnboardingTransfer(cmd.Context(), args[0])
	if err != nil {
		return scopeHint(err, recipientScopes)
	}

	return printRecipientTransfer(cmd, transfer)
}

func newRecipientTransferReversalCommand() *cobra.Command {
	command := &cobra.Command{
		Annotations: apiOperations("POST /payments/{payment_id}/transactions/{transaction_id}/split-marketplace/transfer-reversal"),
		Use:         "reversal <payment_id> <transaction_id>",
		Short:       "Reverse the split marketplace transfer of a payment transaction",
		Example: "  yuno-cli recipient transfer reversal pay-1 txn-1 " +
			"--currency USD --amount 10 --yes",
		Args: cobra.ExactArgs(2),
		RunE: runRecipientTransferReversal,
	}

	registerWriteFlags(command, transferReversalFields)

	return command
}

func runRecipientTransferReversal(cmd *cobra.Command, args []string) error {
	body, err := bodyFromFlags(cmd, transferReversalFields)
	if err != nil {
		return err
	}

	client, err := newClientFromFlags(cmd)
	if err != nil {
		return err
	}

	data, err := client.ReversePaymentTransfer(cmd.Context(), args[0], args[1], body)
	if err != nil {
		return scopeHint(err, recipientScopes)
	}

	return printJSON(cmd, data)
}

// printRecipientTransfer renders one onboarding transfer: the whole response as
// JSON, a single table row otherwise.
func printRecipientTransfer(cmd *cobra.Command, transfer *model.RecipientTransfer) error {
	return printResult(cmd, transfer, []model.RecipientTransferView{transfer.View()})
}
