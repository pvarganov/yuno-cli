package cmd

import (
	"github.com/spf13/cobra"

	"github.com/pvarganov/yuno-cli/internal/model"
)

// conversionRateScopes are the scopes a 403 on `conversion-rate` usually asks for.
const conversionRateScopes = "payments:read/payments:write"

// conversionRateFields are the named body fields of `conversion-rate get`. Raw
// card data is deliberately not exposed as a flag: it would end up in the shell
// history, so a PCI merchant passes it through --file.
var conversionRateFields = []FieldFlag{
	{Flag: "account-id", Field: "account_id", Kind: FieldString, Usage: "account the conversion is quoted for"},
	{Flag: "currency", Field: "amount.currency", Kind: FieldString, Usage: "currency of the amount to convert"},
	{Flag: "amount", Field: "amount.value", Kind: FieldNumber, Usage: "amount to convert"},
	{
		Flag: "cardholder-currency", Field: "amount.currency_conversion.cardholder_currency", Kind: FieldString,
		Usage: "currency to convert the amount to",
	},
	{Flag: "provider", Field: "provider_data.id", Kind: FieldString, Usage: "provider quoting the rate, e.g. CIBC"},
	{
		Flag: "token", Field: "payment_method.token", Kind: FieldString,
		Usage: "one time token of the card the rate is quoted for",
	},
	{
		Flag: "vaulted-token", Field: "payment_method.vaulted_token", Kind: FieldString,
		Usage: "vaulted token of the card the rate is quoted for",
	},
}

// newConversionRateCommand builds the `conversion-rate` command tree.
func newConversionRateCommand() *cobra.Command {
	rate := &cobra.Command{
		Use:     "conversion-rate",
		Short:   "Quote the currency conversion rate of an amount",
		Aliases: []string{"currency-conversion"},
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return cmd.Help()
		},
	}

	rate.AddCommand(newConversionRateGetCommand())

	return rate
}

func newConversionRateGetCommand() *cobra.Command {
	command := &cobra.Command{
		Use:   "get",
		Short: "Quote the conversion of an amount into the cardholder currency",
		Example: "  yuno-cli conversion-rate get --account-id acc-1 --currency COP --amount 10000 " +
			"--cardholder-currency USD --provider CIBC --vaulted-token vt-1 --yes",
		Args: cobra.NoArgs,
		RunE: runConversionRateGet,
	}

	registerWriteFlags(command, conversionRateFields)

	return command
}

func runConversionRateGet(cmd *cobra.Command, _ []string) error {
	body, err := requireBody(cmd, conversionRateFields)
	if err != nil {
		return err
	}

	client, err := newClientFromFlags(cmd)
	if err != nil {
		return err
	}

	rate, err := client.GetConversionRate(cmd.Context(), body)
	if err != nil {
		return scopeHint(err, conversionRateScopes)
	}

	return printResult(cmd, rate, []model.ConversionRateView{rate.View()})
}
