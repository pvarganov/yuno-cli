package cmd

import (
	"github.com/spf13/cobra"

	"github.com/pvarganov/yuno-cli/internal/model"
)

// dryRunScopes are the scopes a 403 on `dry-run` usually asks for.
const dryRunScopes = "payments:write"

// dryRunFields are the named body fields of `dry-run provider-event`. The
// events themselves are a nested array, so they come from --events or --file.
var dryRunFields = []FieldFlag{
	{
		Flag: "merchant-reference", Field: "merchant_reference", Kind: FieldString,
		Usage: "your own reference for the simulated event",
	},
	{Flag: "payment-id", Field: "payment_id", Kind: FieldString, Usage: "payment the event is simulated for"},
	{Flag: "account-id", Field: "account_id", Kind: FieldString, Usage: "account the event is simulated for"},
	{Flag: "provider-id", Field: "provider_id", Kind: FieldString, Usage: "provider the event is simulated for"},
	{
		Flag: "payment-method-type", Field: "payment_method_type", Kind: FieldString,
		Usage: "payment method type of the simulated event",
	},
	{
		Flag: "events", Field: "events", Kind: FieldJSON,
		Usage: `simulated events as a JSON array, e.g. '[{"type":"WEBHOOK","http_method":"POST",` +
			`"operation_type":"CREATE_PAYMENT","headers":"{}"}]'`,
	},
}

// newDryRunCommand builds the `dry-run` command tree.
func newDryRunCommand() *cobra.Command {
	dryRun := &cobra.Command{
		Use:     "dry-run",
		Short:   "Replay provider events without the provider",
		Aliases: []string{"dryrun"},
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return cmd.Help()
		},
	}

	dryRun.AddCommand(newDryRunProviderEventCommand())

	return dryRun
}

func newDryRunProviderEventCommand() *cobra.Command {
	command := &cobra.Command{
		Annotations: apiOperations("POST /dry-run/provider-events"),
		Use:         "provider-event",
		Short:       "Register a simulated provider event",
		Aliases:     []string{"provider-events"},
		Example:     "  yuno-cli dry-run provider-event --merchant-reference ref-1 --file events.json",
		Args:        cobra.NoArgs,
		RunE:        runDryRunProviderEvent,
	}

	registerWriteFlags(command, dryRunFields)

	return command
}

func runDryRunProviderEvent(cmd *cobra.Command, _ []string) error {
	body, err := requireBody(cmd, dryRunFields)
	if err != nil {
		return err
	}

	client, err := newClientFromFlags(cmd)
	if err != nil {
		return err
	}

	event, err := client.CreateDryRunProviderEvent(cmd.Context(), body)
	if err != nil {
		return scopeHint(err, dryRunScopes)
	}

	return printResult(cmd, event, []model.DryRunProviderEventView{event.View()})
}
