package cmd

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"os"

	"github.com/spf13/cobra"

	"github.com/pvarganov/yuno-cli/internal/model"
)

// reportScopes are the scopes a 403 on the report commands usually asks for.
const reportScopes = "reports:read/reports:write"

// stdoutTarget is the --output value that streams the report to stdout.
const stdoutTarget = "-"

// reportFileMode keeps a downloaded report readable by its owner only: it
// carries payment data.
const reportFileMode = 0o600

// reportFields are the named body fields of `report create`.
var reportFields = []FieldFlag{
	{
		Flag: "type", Field: "type", Kind: FieldString,
		Usage: "report type: PAYMENTS, PAYOUTS, TRANSACTIONS, SETTLEMENT_FEES, " +
			"RECONCILIATION_OVERVIEW or COMMUNICATIONS",
	},
	{Flag: "start-date", Field: "start_date", Kind: FieldString, Usage: "first timestamp to include, ISO 8601"},
	{Flag: "end-date", Field: "end_date", Kind: FieldString, Usage: "last timestamp to include, ISO 8601"},
	{
		Flag: "merchant-reference-id", Field: "merchant_reference_id", Kind: FieldString,
		Usage: "your own identifier for the report",
	},
	{
		Flag: "account-id", Field: "account_id", Kind: FieldString,
		Usage: "account to report on, comma separated for several, all accounts when omitted",
	},
	{Flag: "user-code", Field: "user_code", Kind: FieldString, Usage: "user identifier to associate with the run"},
	{Flag: "acquirer", Field: "acquirer", Kind: FieldString, Usage: "filter a settlement report by acquirer"},
	{
		Flag: "payment-status", Field: "payment_status", Kind: FieldString,
		Usage: "comma separated payment statuses to include, all when omitted",
	},
	{
		Flag: "payment-sub-status", Field: "payment_sub_status", Kind: FieldString,
		Usage: "comma separated payment sub statuses to include, all when omitted",
	},
	{
		Flag: "payment-method", Field: "payment_method", Kind: FieldString,
		Usage: "comma separated payment methods to include, all when omitted",
	},
	{Flag: "currency", Field: "currency", Kind: FieldString, Usage: "comma separated currencies to include"},
	{Flag: "country", Field: "country", Kind: FieldString, Usage: "comma separated countries to include"},
}

// newReportCommand builds the `report` command tree.
func newReportCommand() *cobra.Command {
	report := &cobra.Command{
		Use:     "report",
		Short:   "Run and download reports",
		Aliases: []string{"reports"},
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return cmd.Help()
		},
	}

	report.AddCommand(
		newReportCreateCommand(),
		newReportListCommand(),
		newReportGetCommand(),
		newReportDownloadCommand(),
	)

	return report
}

func newReportCreateCommand() *cobra.Command {
	command := &cobra.Command{
		Use:   "create",
		Short: "Schedule a report run",
		Long: "Schedule a report run.\n\n" +
			"The run is asynchronous: it starts as IN_PROCESS and becomes downloadable once\n" +
			"`report get` shows it as SUCCEEDED.",
		Example: "  yuno-cli report create --type PAYMENTS " +
			"--start-date 2026-09-01T00:00:00Z --end-date 2026-09-15T00:00:00Z",
		Args: cobra.NoArgs,
		RunE: runReportCreate,
	}

	registerWriteFlags(command, reportFields)

	return command
}

func runReportCreate(cmd *cobra.Command, _ []string) error {
	body, err := requireBody(cmd, reportFields)
	if err != nil {
		return err
	}

	client, err := newClientFromFlags(cmd)
	if err != nil {
		return err
	}

	report, err := client.CreateReport(cmd.Context(), body)
	if err != nil {
		return scopeHint(err, reportScopes)
	}

	return printResult(cmd, report, []model.ReportView{report.View()})
}

func newReportListCommand() *cobra.Command {
	command := &cobra.Command{
		Use:     "list",
		Short:   "List the report runs",
		Example: "  yuno-cli report list --account-id acc-1 --limit 20",
		Args:    cobra.NoArgs,
		RunE:    runReportList,
	}

	flags := command.Flags()
	flags.String("account-id", "", "only list the reports of this account")
	flags.String("start-date", "", "only list reports starting at or after this timestamp")
	flags.String("end-date", "", "only list reports ending at or before this timestamp")
	registerPagingFlags(command)

	return command
}

func runReportList(cmd *cobra.Command, _ []string) error {
	client, err := newClientFromFlags(cmd)
	if err != nil {
		return err
	}

	query := url.Values{}

	for flag, param := range map[string]string{
		"account-id": "account_id",
		"start-date": "start_date",
		"end-date":   "end_date",
	} {
		if value := flagString(cmd, flag); value != "" {
			query.Set(param, value)
		}
	}

	limit, pageSize := pagingFlags(cmd)

	reports, err := client.ListReports(cmd.Context(), query, limit, pageSize)
	if err != nil {
		return scopeHint(err, reportScopes)
	}

	return printResult(cmd, reports, model.ReportViews(reports))
}

func newReportGetCommand() *cobra.Command {
	return &cobra.Command{
		Use:     "get <report_id>",
		Short:   "Retrieve one report run",
		Example: "  yuno-cli report get 6d905149-b388-4522-8c0a-759fed1f39da --json",
		Args:    cobra.ExactArgs(1),
		RunE:    runReportGet,
	}
}

func runReportGet(cmd *cobra.Command, args []string) error {
	client, err := newClientFromFlags(cmd)
	if err != nil {
		return err
	}

	report, err := client.GetReport(cmd.Context(), args[0])
	if err != nil {
		return scopeHint(err, reportScopes)
	}

	return printResult(cmd, report, []model.ReportView{report.View()})
}

func newReportDownloadCommand() *cobra.Command {
	command := &cobra.Command{
		Use:   "download <report_id>",
		Short: "Retrieve the download link of a report, or the file behind it",
		Long: "Retrieve the download link of a report, or the file behind it.\n\n" +
			"Without --output the pre-signed link is printed. With --output the file is\n" +
			"streamed to that path, or to stdout when the path is `-`. The link carries its\n" +
			"own credentials, so the Yuno api keys are never sent to the storage host.",
		Example: "  yuno-cli report download 6d905149-b388-4522-8c0a-759fed1f39da --output report.csv",
		Args:    cobra.ExactArgs(1),
		RunE:    runReportDownload,
	}

	command.Flags().StringP("output", "o", "", "write the report file here, or `-` for stdout")

	return command
}

func runReportDownload(cmd *cobra.Command, args []string) error {
	client, err := newClientFromFlags(cmd)
	if err != nil {
		return err
	}

	download, err := client.DownloadReport(cmd.Context(), args[0])
	if err != nil {
		return scopeHint(err, reportScopes)
	}

	target := flagString(cmd, "output")
	if target == "" {
		return printResult(cmd, download, []model.ReportDownloadView{download.View()})
	}

	if download.DownloadLink == "" {
		return fmt.Errorf("report %s: no download link yet, its status is %s", args[0], download.Status)
	}

	return streamReport(cmd, client.FetchFile, download.DownloadLink, target)
}

// fetcher streams a pre-signed link into a writer, returning the byte count.
type fetcher func(ctx context.Context, rawURL string, w io.Writer) (int64, error)

// streamReport writes the report file to target, which may be `-` for stdout.
func streamReport(cmd *cobra.Command, fetch fetcher, link, target string) error {
	if target == stdoutTarget {
		_, err := fetch(cmd.Context(), link, cmd.OutOrStdout())

		return err
	}

	file, err := os.OpenFile(target, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, reportFileMode)
	if err != nil {
		return fmt.Errorf("create report file %s: %w", target, err)
	}

	written, fetchErr := fetch(cmd.Context(), link, file)
	if closeErr := file.Close(); closeErr != nil && fetchErr == nil {
		return fmt.Errorf("close report file %s: %w", target, closeErr)
	}

	if fetchErr != nil {
		return fetchErr
	}

	if _, err := fmt.Fprintf(cmd.ErrOrStderr(), "wrote %d bytes to %s\n", written, target); err != nil {
		return fmt.Errorf("report progress: %w", err)
	}

	return nil
}
