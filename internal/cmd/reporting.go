package cmd

import (
	"github.com/spf13/cobra"
)

// reportingScopes are the scopes a 403 on the reporting commands usually asks for.
const reportingScopes = "reporting:write"

// reportingFields are the named body fields of `reporting transactions`. The
// events are a batch of up to 500 objects, so they come from --file, from stdin
// or from the --events JSON flag.
var reportingFields = []FieldFlag{
	{Flag: "account-id", Field: "account_id", Kind: FieldString, Usage: "account the reported payments belong to"},
	{Flag: "events", Field: "events", Kind: FieldJSON, Usage: "events to ingest as a JSON array"},
}

// newReportingCommand builds the `reporting` command tree.
func newReportingCommand() *cobra.Command {
	reporting := &cobra.Command{
		Use:   "reporting",
		Short: "Report transactions processed outside Yuno",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return cmd.Help()
		},
	}

	reporting.AddCommand(newReportingTransactionsCommand())

	return reporting
}

func newReportingTransactionsCommand() *cobra.Command {
	command := &cobra.Command{
		Annotations: apiOperations("POST /reporting/transactions"),
		Use:         "transactions",
		Short:       "Ingest off-Yuno transactions",
		Long: "Ingest off-Yuno transactions.\n\n" +
			"Up to 500 events per call, deduplicated by their report_id, which makes a retry\n" +
			"safe. Yuno answers 202 with a per-event result: one invalid event never rejects\n" +
			"the batch, so the answer is printed in full.",
		Example: "  yuno-cli reporting transactions --file events.json",
		Args:    cobra.NoArgs,
		RunE:    runReportingTransactions,
	}

	registerWriteFlags(command, reportingFields)

	return command
}

func runReportingTransactions(cmd *cobra.Command, _ []string) error {
	body, err := requireBody(cmd, reportingFields)
	if err != nil {
		return err
	}

	client, err := newClientFromFlags(cmd)
	if err != nil {
		return err
	}

	data, err := client.ReportTransactions(cmd.Context(), body)
	if err != nil {
		return scopeHint(err, reportingScopes)
	}

	return printJSON(cmd, data)
}
