package cmd

import (
	"net/url"

	"github.com/spf13/cobra"

	"github.com/pvarganov/yuno-cli/internal/model"
)

// recipientScopes are the scopes a 403 on the recipient commands usually asks for.
const recipientScopes = "recipients:read/recipients:write"

// recipientFilters maps the list flags onto the query parameters of
// `GET /recipients`.
var recipientFilters = []struct {
	Flag  string
	Query string
	Usage string
}{
	{Flag: "national-entity", Query: "national_entity", Usage: "only list recipients of this national entity"},
	{Flag: "country", Query: "country", Usage: "only list recipients of this country"},
}

// recipientFields are the named body fields of `recipient create`. The nested
// legal representatives, documentation, withdrawal methods and onboardings are
// arrays of objects, so they come from --file or from their JSON flags.
var recipientFields = []FieldFlag{
	{Flag: "account-id", Field: "account_id", Kind: FieldString, Usage: "account the recipient belongs to"},
	{
		Flag: "merchant-recipient-id", Field: "merchant_recipient_id", Kind: FieldString,
		Usage: "your own reference for the recipient",
	},
	{
		Flag: "national-entity", Field: "national_entity", Kind: FieldString,
		Usage: "national entity of the recipient, INDIVIDUAL or BUSINESS",
	},
	{Flag: "entity-type", Field: "entity_type", Kind: FieldString, Usage: "entity type of the recipient"},
	{Flag: "first-name", Field: "first_name", Kind: FieldString, Usage: "first name of the recipient"},
	{Flag: "last-name", Field: "last_name", Kind: FieldString, Usage: "last name of the recipient"},
	{Flag: "legal-name", Field: "legal_name", Kind: FieldString, Usage: "legal name of a business recipient"},
	{Flag: "email", Field: "email", Kind: FieldString, Usage: "email of the recipient"},
	{Flag: "date-of-birth", Field: "date_of_birth", Kind: FieldString, Usage: "date of birth, YYYY-MM-DD"},
	{Flag: "country", Field: "country", Kind: FieldString, Usage: "country of the recipient, ISO 3166-1 alpha-2"},
	{Flag: "website", Field: "website", Kind: FieldString, Usage: "website of the recipient"},
	{Flag: "industry", Field: "industry", Kind: FieldString, Usage: "industry of the recipient"},
	{
		Flag: "merchant-category-code", Field: "merchant_category_code", Kind: FieldString,
		Usage: "merchant category code of the recipient",
	},
	{
		Flag: "document-number", Field: "document.document_number", Kind: FieldString,
		Usage: "identity document number of the recipient",
	},
	{
		Flag: "document-type", Field: "document.document_type", Kind: FieldString,
		Usage: "identity document type, e.g. CPF",
	},
	{
		Flag: "phone-country-code", Field: "phone.country_code", Kind: FieldString,
		Usage: "country code of the phone number",
	},
	{Flag: "phone-number", Field: "phone.number", Kind: FieldString, Usage: "phone number of the recipient"},
	{Flag: "address", Field: "address", Kind: FieldJSON, Usage: "whole address as a JSON object"},
	{
		Flag: "split-configuration", Field: "split_configuration", Kind: FieldJSON,
		Usage: "split configuration as a JSON object",
	},
	{
		Flag: "legal-representatives", Field: "legal_representatives", Kind: FieldJSON,
		Usage: "legal representatives as a JSON array",
	},
	{
		Flag: "withdrawal-methods", Field: "withdrawal_methods", Kind: FieldJSON,
		Usage: "withdrawal methods as a JSON object",
	},
	{Flag: "documentation", Field: "documentation", Kind: FieldJSON, Usage: "documentation as a JSON array"},
	{Flag: "onboardings", Field: "onboardings", Kind: FieldJSON, Usage: "onboardings as a JSON array"},
}

// recipientUpdateFields are the named body fields of `recipient update`. The
// national entity and the country cannot be patched, and an onboarding may be
// pinned so the patch applies to the right provider.
var recipientUpdateFields = append(
	[]FieldFlag{{
		Flag: "onboarding-id", Field: "onboarding_id", Kind: FieldString,
		Usage: "onboarding the update applies to",
	}},
	patchableRecipientFields()...,
)

// patchableRecipientFields drops the create-only fields from the recipient
// fields, so `recipient update` only offers what Yuno accepts in a PATCH.
func patchableRecipientFields() []FieldFlag {
	skip := map[string]bool{"account-id": true, "national-entity": true, "country": true, "legal-name": true}

	fields := make([]FieldFlag, 0, len(recipientFields))

	for _, f := range recipientFields {
		if !skip[f.Flag] {
			fields = append(fields, f)
		}
	}

	return fields
}

// newRecipientCommand builds the `recipient` command tree.
func newRecipientCommand() *cobra.Command {
	recipient := &cobra.Command{
		Use:     "recipient",
		Short:   "Manage marketplace recipients, their onboardings and their transfers",
		Aliases: []string{"recipients"},
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return cmd.Help()
		},
	}

	recipient.AddCommand(
		newRecipientCreateCommand(),
		newRecipientListCommand(),
		newRecipientGetCommand(),
		newRecipientUpdateCommand(),
		newRecipientDeleteCommand(),
		newRecipientOnboardingCommand(),
		newRecipientTransferCommand(),
	)

	return recipient
}

func newRecipientCreateCommand() *cobra.Command {
	command := &cobra.Command{
		Annotations: apiOperations("POST /recipients"),
		Use:         "create",
		Short:       "Create a marketplace recipient",
		Example: "  yuno-cli recipient create --account-id acc-1 --merchant-recipient-id rec-1 " +
			"--national-entity INDIVIDUAL --first-name Ada --last-name Lovelace --country BR",
		Args: cobra.NoArgs,
		RunE: runRecipientCreate,
	}

	registerWriteFlags(command, recipientFields)

	return command
}

func runRecipientCreate(cmd *cobra.Command, _ []string) error {
	body, err := requireBody(cmd, recipientFields)
	if err != nil {
		return err
	}

	client, err := newClientFromFlags(cmd)
	if err != nil {
		return err
	}

	recipient, err := client.CreateRecipient(cmd.Context(), body)
	if err != nil {
		return scopeHint(err, recipientScopes)
	}

	return printRecipient(cmd, recipient)
}

func newRecipientListCommand() *cobra.Command {
	command := &cobra.Command{
		Annotations: apiOperations("GET /recipients"),
		Use:         "list",
		Short:       "List the marketplace recipients",
		Example:     "  yuno-cli recipient list --country BR --limit 20",
		Args:        cobra.NoArgs,
		RunE:        runRecipientList,
	}

	for _, filter := range recipientFilters {
		command.Flags().String(filter.Flag, "", filter.Usage)
	}

	registerPagingFlags(command)

	return command
}

func runRecipientList(cmd *cobra.Command, _ []string) error {
	client, err := newClientFromFlags(cmd)
	if err != nil {
		return err
	}

	filters := url.Values{}

	for _, filter := range recipientFilters {
		if value := flagString(cmd, filter.Flag); value != "" {
			filters.Set(filter.Query, value)
		}
	}

	limit, pageSize := pagingFlags(cmd)

	recipients, err := client.ListRecipients(cmd.Context(), filters, limit, pageSize)
	if err != nil {
		return scopeHint(err, recipientScopes)
	}

	return printResult(cmd, recipients, model.RecipientViews(recipients))
}

func newRecipientGetCommand() *cobra.Command {
	return &cobra.Command{
		Annotations: apiOperations("GET /recipients/{recipient_id}"),
		Use:         "get <recipient_id>",
		Short:       "Retrieve one marketplace recipient",
		Example:     "  yuno-cli recipient get rec-1 --json",
		Args:        cobra.ExactArgs(1),
		RunE:        runRecipientGet,
	}
}

func runRecipientGet(cmd *cobra.Command, args []string) error {
	client, err := newClientFromFlags(cmd)
	if err != nil {
		return err
	}

	recipient, err := client.GetRecipient(cmd.Context(), args[0])
	if err != nil {
		return scopeHint(err, recipientScopes)
	}

	return printRecipient(cmd, recipient)
}

func newRecipientUpdateCommand() *cobra.Command {
	command := &cobra.Command{
		Annotations: apiOperations("PATCH /recipients/{recipient_id}"),
		Use:         "update <recipient_id>",
		Short:       "Update a marketplace recipient",
		Example:     "  yuno-cli recipient update rec-1 --email ada@example.com --yes",
		Args:        cobra.ExactArgs(1),
		RunE:        runRecipientUpdate,
	}

	registerWriteFlags(command, recipientUpdateFields)

	return command
}

func runRecipientUpdate(cmd *cobra.Command, args []string) error {
	body, err := requireBody(cmd, recipientUpdateFields)
	if err != nil {
		return err
	}

	client, err := newClientFromFlags(cmd)
	if err != nil {
		return err
	}

	recipient, err := client.UpdateRecipient(cmd.Context(), args[0], body)
	if err != nil {
		return scopeHint(err, recipientScopes)
	}

	return printRecipient(cmd, recipient)
}

func newRecipientDeleteCommand() *cobra.Command {
	command := &cobra.Command{
		Annotations: apiOperations("DELETE /recipients/{recipient_id}"),
		Use:         "delete <recipient_id>",
		Short:       "Delete a marketplace recipient",
		Example:     "  yuno-cli recipient delete rec-1 --yes",
		Args:        cobra.ExactArgs(1),
		RunE:        runRecipientDelete,
	}

	return command
}

func runRecipientDelete(cmd *cobra.Command, args []string) error {
	client, err := newClientFromFlags(cmd)
	if err != nil {
		return err
	}

	data, err := client.DeleteRecipient(cmd.Context(), args[0])
	if err != nil {
		return scopeHint(err, recipientScopes)
	}

	return printJSON(cmd, data)
}

// printRecipient renders one recipient: the whole response as JSON, a single
// table row otherwise.
func printRecipient(cmd *cobra.Command, recipient *model.Recipient) error {
	return printResult(cmd, recipient, []model.RecipientView{recipient.View()})
}
