package cmd

import (
	"github.com/spf13/cobra"

	"github.com/pvarganov/yuno-cli/internal/model"
)

// webhookScopes are the scopes a 403 on the webhook commands usually asks for.
const webhookScopes = "webhooks:read/webhooks:write"

// webhookAuthFields are the delivery credentials shared by create and update.
// Yuno masks them in every response, so they can only be written, never read.
var webhookAuthFields = []FieldFlag{
	{Flag: "api-key", Field: "api_key", Kind: FieldString, Usage: "value of the x-api-key header Yuno sends on delivery"},
	{Flag: "secret", Field: "secret", Kind: FieldString, Usage: "value of the x-secret header Yuno sends on delivery"},
	{
		Flag: "hmac-client-secret", Field: "hmac_client_secret", Kind: FieldString,
		Usage: "key Yuno signs the payload with, sent as x-hmac-signature",
	},
	{
		Flag: "oauth2-authentication-url", Field: "oauth2_authentication_url", Kind: FieldString,
		Usage: "token endpoint Yuno authenticates against",
	},
	{
		Flag: "oauth2-client-id", Field: "oauth2_client_id", Kind: FieldString,
		Usage: "client id Yuno requests the token with",
	},
	{
		Flag: "oauth2-client-secret", Field: "oauth2_client_secret", Kind: FieldString,
		Usage: "client secret Yuno requests the token with",
	},
	{
		Flag: "oauth2-grant-type", Field: "oauth2_grant_type", Kind: FieldString,
		Usage: "grant type Yuno requests the token with, e.g. client_credentials",
	},
	{Flag: "oauth2-scope", Field: "oauth2_scope", Kind: FieldString, Usage: "scope Yuno requests the token with"},
	{
		Flag: "oauth2-authorization-name", Field: "oauth2_authorization_name", Kind: FieldString,
		Usage: "header Yuno sends the token in, Authorization by default",
	},
	{
		Flag: "oauth2-include-client-id", Field: "oauth2_include_client_id", Kind: FieldBool,
		Usage: "also send the client id as a header on the token request",
	},
}

// webhookTriggerFields are the event families a webhook can subscribe to.
var webhookTriggerFields = []FieldFlag{
	{
		Flag: "payment-trigger", Field: "payment_triggers", Kind: FieldStringSlice,
		Usage: "payment event to subscribe to, repeatable",
	},
	{
		Flag: "enrollment-trigger", Field: "enrollment_triggers", Kind: FieldStringSlice,
		Usage: "enrollment event to subscribe to, repeatable",
	},
	{
		Flag: "report-trigger", Field: "report_triggers", Kind: FieldStringSlice,
		Usage: "report event to subscribe to, repeatable",
	},
	{
		Flag: "subscription-trigger", Field: "subscription_triggers", Kind: FieldStringSlice,
		Usage: "subscription event to subscribe to, repeatable",
	},
	{
		Flag: "onboarding-trigger", Field: "onboarding_triggers", Kind: FieldStringSlice,
		Usage: "onboarding event to subscribe to, repeatable",
	},
	{
		Flag: "renewal-days", Field: "renewal_days", Kind: FieldInt,
		Usage: "days before a renewal the CLOSE_TO_RENEWAL event is sent",
	},
}

// webhookCreateFields are the body fields of `webhook create`.
var webhookCreateFields = webhookFields(false)

// webhookUpdateFields are the body fields of `webhook update`, which adds the
// state so a webhook can be paused without losing its configuration.
var webhookUpdateFields = webhookFields(true)

// webhookFields assembles the body flags of the two mutating webhook commands.
func webhookFields(update bool) []FieldFlag {
	fields := []FieldFlag{
		{Flag: "account-id", Field: "account_id", Kind: FieldString, Usage: "account the webhook belongs to"},
		{Flag: "name", Field: "name", Kind: FieldString, Usage: "your name for the webhook, unique within the account"},
		{Flag: "url", Field: "url", Kind: FieldString, Usage: "endpoint Yuno posts the notifications to"},
	}

	if update {
		fields = append(fields, FieldFlag{
			Flag: "state", Field: "state", Kind: FieldString,
			Usage: "ACTIVE to deliver events, INACTIVE to pause the deliveries",
		})
	}

	fields = append(fields, webhookAuthFields...)
	fields = append(fields, webhookTriggerFields...)

	return fields
}

// newWebhookCommand builds the `webhook` command tree.
func newWebhookCommand() *cobra.Command {
	webhook := &cobra.Command{
		Use:     "webhook",
		Short:   "Manage the webhooks of an account",
		Aliases: []string{"webhooks"},
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return cmd.Help()
		},
	}

	webhook.AddCommand(
		newWebhookCreateCommand(),
		newWebhookListCommand(),
		newWebhookGetCommand(),
		newWebhookUpdateCommand(),
		newWebhookDeleteCommand(),
	)

	return webhook
}

func newWebhookCreateCommand() *cobra.Command {
	command := &cobra.Command{
		Annotations: apiOperations("POST /webhooks"),
		Use:         "create",
		Short:       "Register a webhook",
		Example: "  yuno-cli webhook create --account-id acc-1 --name 'payments listener' " +
			"--url https://api.acme.com/yuno --payment-trigger AUTHORIZE --payment-trigger REFUND",
		Args: cobra.NoArgs,
		RunE: runWebhookCreate,
	}

	registerWriteFlags(command, webhookCreateFields)

	return command
}

func runWebhookCreate(cmd *cobra.Command, _ []string) error {
	body, err := requireBody(cmd, webhookCreateFields)
	if err != nil {
		return err
	}

	client, err := newClientFromFlags(cmd)
	if err != nil {
		return err
	}

	webhook, err := client.CreateWebhook(cmd.Context(), body)
	if err != nil {
		return scopeHint(err, webhookScopes)
	}

	return printResult(cmd, webhook, []model.WebhookView{webhook.View()})
}

func newWebhookListCommand() *cobra.Command {
	command := &cobra.Command{
		Annotations: apiOperations("GET /webhooks"),
		Use:         "list",
		Short:       "List the webhooks of an account",
		Example:     "  yuno-cli webhook list --account-id acc-1 --state ACTIVE",
		Args:        cobra.NoArgs,
		RunE:        runWebhookList,
	}

	flags := command.Flags()
	flags.String("account-id", "", "account the webhooks belong to")
	flags.String("state", "", "only list webhooks in this state, ACTIVE or INACTIVE")
	_ = command.MarkFlagRequired("account-id")

	return command
}

func runWebhookList(cmd *cobra.Command, _ []string) error {
	client, err := newClientFromFlags(cmd)
	if err != nil {
		return err
	}

	webhooks, err := client.ListWebhooks(cmd.Context(), flagString(cmd, "account-id"), flagString(cmd, "state"))
	if err != nil {
		return scopeHint(err, webhookScopes)
	}

	return printResult(cmd, webhooks, model.WebhookViews(webhooks))
}

func newWebhookGetCommand() *cobra.Command {
	command := &cobra.Command{
		Annotations: apiOperations("GET /webhooks/{webhook_id}"),
		Use:         "get <webhook_id>",
		Short:       "Retrieve one webhook",
		Example:     "  yuno-cli webhook get 12345 --account-id acc-1 --json",
		Args:        cobra.ExactArgs(1),
		RunE:        runWebhookGet,
	}

	command.Flags().String("account-id", "", "account the webhook belongs to")
	_ = command.MarkFlagRequired("account-id")

	return command
}

func runWebhookGet(cmd *cobra.Command, args []string) error {
	client, err := newClientFromFlags(cmd)
	if err != nil {
		return err
	}

	webhook, err := client.GetWebhook(cmd.Context(), args[0], flagString(cmd, "account-id"))
	if err != nil {
		return scopeHint(err, webhookScopes)
	}

	return printResult(cmd, webhook, []model.WebhookView{webhook.View()})
}

func newWebhookUpdateCommand() *cobra.Command {
	command := &cobra.Command{
		Annotations: apiOperations("PATCH /webhooks/{webhook_id}"),
		Use:         "update <webhook_id>",
		Short:       "Update a webhook",
		Long: "Update a webhook.\n\n" +
			"Only the flags you pass are sent, so an omitted field keeps its current value.\n" +
			"Yuno requires the account id on every update, even when it does not change.",
		Example: "  yuno-cli webhook update 12345 --account-id acc-1 --state INACTIVE",
		Args:    cobra.ExactArgs(1),
		RunE:    runWebhookUpdate,
	}

	registerWriteFlags(command, webhookUpdateFields)

	return command
}

func runWebhookUpdate(cmd *cobra.Command, args []string) error {
	body, err := requireBody(cmd, webhookUpdateFields)
	if err != nil {
		return err
	}

	client, err := newClientFromFlags(cmd)
	if err != nil {
		return err
	}

	webhook, err := client.UpdateWebhook(cmd.Context(), args[0], body)
	if err != nil {
		return scopeHint(err, webhookScopes)
	}

	return printResult(cmd, webhook, []model.WebhookView{webhook.View()})
}

func newWebhookDeleteCommand() *cobra.Command {
	command := &cobra.Command{
		Annotations: apiOperations("DELETE /webhooks/{webhook_id}"),
		Use:         "delete <webhook_id>",
		Short:       "Delete a webhook",
		Example:     "  yuno-cli webhook delete 12345 --account-id acc-1",
		Args:        cobra.ExactArgs(1),
		RunE:        runWebhookDelete,
	}

	command.Flags().String("account-id", "", "account the webhook belongs to")
	_ = command.MarkFlagRequired("account-id")

	return command
}

func runWebhookDelete(cmd *cobra.Command, args []string) error {
	client, err := newClientFromFlags(cmd)
	if err != nil {
		return err
	}

	data, err := client.DeleteWebhook(cmd.Context(), args[0], flagString(cmd, "account-id"))
	if err != nil {
		return scopeHint(err, webhookScopes)
	}

	return printJSON(cmd, data)
}
