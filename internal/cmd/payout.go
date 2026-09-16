package cmd

import (
	"github.com/spf13/cobra"

	"github.com/pvarganov/yuno-cli/internal/model"
)

// payoutScopes are the scopes a 403 on the payout commands usually asks for.
const payoutScopes = "payouts:read/payouts:write"

// payoutFields are the named body fields of `payout create`. The nested
// beneficiary document, phone and address and the withdrawal method detail are
// objects, so they come from --file or from their JSON flags.
var payoutFields = []FieldFlag{
	{Flag: "account-id", Field: "account_id", Kind: FieldString, Usage: "account the payout is sent from"},
	{
		Flag: "merchant-reference", Field: "merchant_reference", Kind: FieldString,
		Usage: "your own reference for the payout",
	},
	{Flag: "description", Field: "description", Kind: FieldString, Usage: "description of the payout"},
	{Flag: "country", Field: "country", Kind: FieldString, Usage: "country of the payout, ISO 3166-1 alpha-2"},
	{Flag: "purpose", Field: "purpose", Kind: FieldString, Usage: "purpose of the payout, e.g. SALARY"},
	{Flag: "currency", Field: "amount.currency", Kind: FieldString, Usage: "currency of the amount"},
	{Flag: "amount", Field: "amount.value", Kind: FieldNumber, Usage: "amount to pay out"},
	{
		Flag: "beneficiary-id", Field: "beneficiary.merchant_beneficiary_id", Kind: FieldString,
		Usage: "your own reference for the beneficiary",
	},
	{
		Flag: "beneficiary-country", Field: "beneficiary.country", Kind: FieldString,
		Usage: "country of the beneficiary, ISO 3166-1 alpha-2",
	},
	{
		Flag: "beneficiary-first-name", Field: "beneficiary.first_name", Kind: FieldString,
		Usage: "first name of the beneficiary",
	},
	{
		Flag: "beneficiary-last-name", Field: "beneficiary.last_name", Kind: FieldString,
		Usage: "last name of the beneficiary",
	},
	{
		Flag: "beneficiary-legal-name", Field: "beneficiary.legal_name", Kind: FieldString,
		Usage: "legal name of a company beneficiary",
	},
	{
		Flag: "beneficiary-email", Field: "beneficiary.email", Kind: FieldString,
		Usage: "email of the beneficiary",
	},
	{
		Flag: "beneficiary", Field: "beneficiary", Kind: FieldJSON,
		Usage: "whole beneficiary as a JSON object",
	},
	{
		Flag: "withdrawal-type", Field: "withdrawal_method.type", Kind: FieldString,
		Usage: "withdrawal method type, e.g. BANK_TRANSFER",
	},
	{
		Flag: "provider-id", Field: "withdrawal_method.provider_id", Kind: FieldString,
		Usage: "provider the payout is sent through",
	},
	{
		Flag: "vaulted-token", Field: "withdrawal_method.vaulted_token", Kind: FieldString,
		Usage: "vaulted token of the withdrawal method",
	},
	{
		Flag: "withdrawal-method", Field: "withdrawal_method", Kind: FieldJSON,
		Usage: "whole withdrawal method as a JSON object",
	},
	{Flag: "metadata", Field: "metadata", Kind: FieldJSON, Usage: "metadata as a JSON array"},
}

// newPayoutCommand builds the `payout` command tree.
func newPayoutCommand() *cobra.Command {
	payout := &cobra.Command{
		Use:     "payout",
		Short:   "Create and inspect payouts",
		Aliases: []string{"payouts"},
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return cmd.Help()
		},
	}

	payout.AddCommand(
		newPayoutCreateCommand(),
		newPayoutListCommand(),
		newPayoutGetCommand(),
	)

	return payout
}

func newPayoutCreateCommand() *cobra.Command {
	command := &cobra.Command{
		Use:   "create",
		Short: "Create a payout",
		Example: "  yuno-cli payout create --account-id acc-1 --merchant-reference ref-1 --country US " +
			"--purpose SALARY --currency USD --amount 100 --beneficiary-id ben-1 --beneficiary-country US " +
			"--withdrawal-type BANK_TRANSFER --provider-id NUVEI",
		Args: cobra.NoArgs,
		RunE: runPayoutCreate,
	}

	registerWriteFlags(command, payoutFields)

	return command
}

func runPayoutCreate(cmd *cobra.Command, _ []string) error {
	body, err := requireBody(cmd, payoutFields)
	if err != nil {
		return err
	}

	client, err := newClientFromFlags(cmd)
	if err != nil {
		return err
	}

	payout, err := client.CreatePayout(cmd.Context(), body)
	if err != nil {
		return scopeHint(err, payoutScopes)
	}

	return printResult(cmd, payout, []model.PayoutView{payout.View()})
}

func newPayoutListCommand() *cobra.Command {
	command := &cobra.Command{
		Use:   "list",
		Short: "List the payouts created under a merchant reference",
		Long: "List the payouts created under a merchant reference.\n\n" +
			"Yuno has no unfiltered payout list: GET /payouts requires merchant_reference.",
		Example: "  yuno-cli payout list --merchant-reference ref-1",
		Args:    cobra.NoArgs,
		RunE:    runPayoutList,
	}

	command.Flags().String("merchant-reference", "", "merchant reference the payouts were created with")
	_ = command.MarkFlagRequired("merchant-reference")

	return command
}

func runPayoutList(cmd *cobra.Command, _ []string) error {
	client, err := newClientFromFlags(cmd)
	if err != nil {
		return err
	}

	payouts, err := client.ListPayouts(cmd.Context(), flagString(cmd, "merchant-reference"))
	if err != nil {
		return scopeHint(err, payoutScopes)
	}

	return printResult(cmd, payouts, model.PayoutViews(payouts))
}

func newPayoutGetCommand() *cobra.Command {
	return &cobra.Command{
		Use:     "get <payout_id>",
		Short:   "Retrieve one payout",
		Example: "  yuno-cli payout get 565c9733-000d-4066-a4bf-b084908dc74c --json",
		Args:    cobra.ExactArgs(1),
		RunE:    runPayoutGet,
	}
}

func runPayoutGet(cmd *cobra.Command, args []string) error {
	client, err := newClientFromFlags(cmd)
	if err != nil {
		return err
	}

	payout, err := client.GetPayout(cmd.Context(), args[0])
	if err != nil {
		return scopeHint(err, payoutScopes)
	}

	return printResult(cmd, payout, []model.PayoutView{payout.View()})
}
