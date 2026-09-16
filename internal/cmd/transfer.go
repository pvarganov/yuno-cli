package cmd

import (
	"github.com/spf13/cobra"

	"github.com/pvarganov/yuno-cli/internal/model"
)

// transferScopes are the scopes a 403 on the transfer commands usually asks for.
const transferScopes = "split-marketplace:read/split-marketplace:write"

// transferFields are the named body fields of `transfer create`.
var transferFields = []FieldFlag{
	{Flag: "account-id", Field: "account_id", Kind: FieldString, Usage: "account the transfer is sent from"},
	{Flag: "recipient-id", Field: "recipient_id", Kind: FieldString, Usage: "recipient the money is sent to"},
	{Flag: "provider-id", Field: "provider_id", Kind: FieldString, Usage: "provider the transfer is sent through"},
	{Flag: "currency", Field: "amount.currency", Kind: FieldString, Usage: "currency of the amount"},
	{Flag: "amount", Field: "amount.value", Kind: FieldNumber, Usage: "amount to transfer"},
	{Flag: "description", Field: "description", Kind: FieldString, Usage: "description of the transfer"},
	{
		Flag: "merchant-reference", Field: "merchant_reference", Kind: FieldString,
		Usage: "your own reference for the transfer",
	},
	{Flag: "metadata", Field: "metadata", Kind: FieldJSON, Usage: "metadata as a JSON array"},
}

// transferReverseFields are the named body fields of `transfer reverse`. An
// amount is optional: leaving it out reverses the whole transfer.
var transferReverseFields = []FieldFlag{
	{Flag: "account-id", Field: "account_id", Kind: FieldString, Usage: "account the transfer was sent from"},
	{Flag: "currency", Field: "amount.currency", Kind: FieldString, Usage: "currency of the reversed amount"},
	{Flag: "amount", Field: "amount.value", Kind: FieldNumber, Usage: "amount to reverse, omit to reverse it all"},
	{Flag: "reason", Field: "reason", Kind: FieldString, Usage: "reason of the reversal"},
	{Flag: "description", Field: "description", Kind: FieldString, Usage: "description of the reversal"},
	{
		Flag: "merchant-reference", Field: "merchant_reference", Kind: FieldString,
		Usage: "your own reference for the reversal",
	},
}

// newTransferCommand builds the `transfer` command tree.
func newTransferCommand() *cobra.Command {
	transfer := &cobra.Command{
		Use:     "transfer",
		Short:   "Create, inspect and reverse split marketplace transfers",
		Aliases: []string{"transfers", "split-marketplace"},
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return cmd.Help()
		},
	}

	transfer.AddCommand(
		newTransferCreateCommand(),
		newTransferGetCommand(),
		newTransferReverseCommand(),
	)

	return transfer
}

func newTransferCreateCommand() *cobra.Command {
	command := &cobra.Command{
		Annotations: apiOperations("POST /split-marketplace/transfers"),
		Use:         "create",
		Short:       "Create a standalone transfer to a recipient",
		Example: "  yuno-cli transfer create --account-id acc-1 --recipient-id rec-1 " +
			"--provider-id NUVEI --currency USD --amount 25",
		Args: cobra.NoArgs,
		RunE: runTransferCreate,
	}

	registerWriteFlags(command, transferFields)

	return command
}

func runTransferCreate(cmd *cobra.Command, _ []string) error {
	body, err := requireBody(cmd, transferFields)
	if err != nil {
		return err
	}

	client, err := newClientFromFlags(cmd)
	if err != nil {
		return err
	}

	transfer, err := client.CreateTransfer(cmd.Context(), body)
	if err != nil {
		return scopeHint(err, transferScopes)
	}

	return printTransfer(cmd, transfer)
}

func newTransferGetCommand() *cobra.Command {
	return &cobra.Command{
		Annotations: apiOperations("GET /split-marketplace/transfers/{transfer_id}"),
		Use:         "get <transfer_id>",
		Short:       "Retrieve one standalone transfer",
		Example:     "  yuno-cli transfer get tr-1 --json",
		Args:        cobra.ExactArgs(1),
		RunE:        runTransferGet,
	}
}

func runTransferGet(cmd *cobra.Command, args []string) error {
	client, err := newClientFromFlags(cmd)
	if err != nil {
		return err
	}

	transfer, err := client.GetTransfer(cmd.Context(), args[0])
	if err != nil {
		return scopeHint(err, transferScopes)
	}

	return printTransfer(cmd, transfer)
}

func newTransferReverseCommand() *cobra.Command {
	command := &cobra.Command{
		Annotations: apiOperations("POST /split-marketplace/transfers/{transfer_id}/reverse"),
		Use:         "reverse <transfer_id>",
		Short:       "Reverse a standalone transfer, fully or partially",
		Example:     "  yuno-cli transfer reverse tr-1 --account-id acc-1 --currency USD --amount 10 --yes",
		Args:        cobra.ExactArgs(1),
		RunE:        runTransferReverse,
	}

	registerWriteFlags(command, transferReverseFields)

	return command
}

func runTransferReverse(cmd *cobra.Command, args []string) error {
	body, err := bodyFromFlags(cmd, transferReverseFields)
	if err != nil {
		return err
	}

	client, err := newClientFromFlags(cmd)
	if err != nil {
		return err
	}

	transfer, err := client.ReverseTransfer(cmd.Context(), args[0], body)
	if err != nil {
		return scopeHint(err, transferScopes)
	}

	return printTransfer(cmd, transfer)
}

// printTransfer renders one transfer: the whole response as JSON, a single
// table row otherwise.
func printTransfer(cmd *cobra.Command, transfer *model.Transfer) error {
	return printResult(cmd, transfer, []model.TransferView{transfer.View()})
}
