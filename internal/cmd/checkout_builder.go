package cmd

import (
	"net/url"

	"github.com/spf13/cobra"

	"github.com/pvarganov/yuno-cli/internal/model"
)

// checkoutBuilderScopes are the scopes a 403 on the checkout builder commands
// usually asks for.
const checkoutBuilderScopes = "checkouts:read/checkouts:write"

// checkoutCreateFields are the named body fields of `checkout-builder create`.
var checkoutCreateFields = []FieldFlag{
	{Flag: "name", Field: "name", Kind: FieldString, Usage: "checkout name, unique per account"},
	{Flag: "description", Field: "description", Kind: FieldString, Usage: "free text description"},
}

// checkoutPublishFields are the named body fields of `checkout-builder publish`.
// Both are whole documents: the payment method list and the SDK styling are too
// deep to deserve one flag per leaf, so they take JSON or come from --file.
var checkoutPublishFields = []FieldFlag{
	{
		Flag: "config", Field: "config", Kind: FieldJSON,
		Usage: "checkout configuration as a JSON object: payment_methods and general_settings",
	},
	{
		Flag: "styling", Field: "styling", Kind: FieldJSON,
		Usage: "checkout styling as a JSON object: styles, settings, flags and external_fonts",
	},
}

// checkoutUpdateFields are the named body fields of `checkout-builder update`.
var checkoutUpdateFields = []FieldFlag{
	{Flag: "name", Field: "name", Kind: FieldString, Usage: "new checkout name, unique per account"},
	{Flag: "description", Field: "description", Kind: FieldString, Usage: "new description"},
	{
		Flag: "status", Field: "status", Kind: FieldString,
		Usage: "target lifecycle status: PUBLISHED, NOT_PUBLISHED or ARCHIVED",
	},
	{
		Flag: "default", Field: "is_default", Kind: FieldBool,
		Usage: "promote this checkout to the account default; only true is accepted",
	},
}

// checkoutListFilters maps the filter flags of `checkout-builder list` onto the
// query parameters of the endpoint.
var checkoutListFilters = map[string]string{
	"name":           "name",
	"id":             "id",
	"status":         "status",
	"created-after":  "created_after",
	"created-before": "created_before",
	"updated-after":  "updated_after",
	"updated-before": "updated_before",
}

// newCheckoutBuilderCommand builds the `checkout-builder` command tree.
func newCheckoutBuilderCommand() *cobra.Command {
	builder := &cobra.Command{
		Use:     "checkout-builder",
		Short:   "Build, publish and archive checkout configurations",
		Aliases: []string{"checkouts"},
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return cmd.Help()
		},
	}

	builder.AddCommand(
		newCheckoutBuilderCreateCommand(),
		newCheckoutBuilderListCommand(),
		newCheckoutBuilderGetCommand(),
		newCheckoutBuilderPublishCommand(),
		newCheckoutBuilderUpdateCommand(),
	)

	return builder
}

func newCheckoutBuilderCreateCommand() *cobra.Command {
	command := &cobra.Command{
		Annotations: apiOperations("POST /checkouts"),
		Use:         "create",
		Short:       "Create an empty checkout configuration",
		Long: "Create an empty checkout configuration.\n\n" +
			"The id it answers with is the checkout_code every other command takes. A new\n" +
			"checkout starts unpublished: fill it with `checkout-builder publish`, then flip\n" +
			"it live with `checkout-builder update --status PUBLISHED`.",
		Example: "  yuno-cli checkout-builder create --name 'Promo Checkout' --description 'Seasonal promo'",
		Args:    cobra.NoArgs,
		RunE:    runCheckoutBuilderCreate,
	}

	registerWriteFlags(command, checkoutCreateFields)

	return command
}

func runCheckoutBuilderCreate(cmd *cobra.Command, _ []string) error {
	body, err := requireBody(cmd, checkoutCreateFields)
	if err != nil {
		return err
	}

	client, err := newClientFromFlags(cmd)
	if err != nil {
		return err
	}

	checkout, err := client.CreateCheckout(cmd.Context(), body)
	if err != nil {
		return scopeHint(err, checkoutBuilderScopes)
	}

	return printResult(cmd, checkout, []model.CheckoutView{checkout.View()})
}

func newCheckoutBuilderListCommand() *cobra.Command {
	command := &cobra.Command{
		Annotations: apiOperations("GET /checkouts"),
		Use:         "list",
		Short:       "List the checkouts of the account",
		Example:     "  yuno-cli checkout-builder list --status PUBLISHED --limit 20",
		Args:        cobra.NoArgs,
		RunE:        runCheckoutBuilderList,
	}

	flags := command.Flags()
	flags.String("name", "", "only list checkouts whose name matches")
	flags.String("id", "", "only list the checkout with this id")
	flags.String("status", "", "only list checkouts in this status: PUBLISHED, NOT_PUBLISHED or ARCHIVED")
	flags.String("created-after", "", "only list checkouts created at or after this timestamp")
	flags.String("created-before", "", "only list checkouts created at or before this timestamp")
	flags.String("updated-after", "", "only list checkouts updated at or after this timestamp")
	flags.String("updated-before", "", "only list checkouts updated at or before this timestamp")
	registerPagingFlags(command)

	return command
}

func runCheckoutBuilderList(cmd *cobra.Command, _ []string) error {
	client, err := newClientFromFlags(cmd)
	if err != nil {
		return err
	}

	query := url.Values{}

	for flag, param := range checkoutListFilters {
		if value := flagString(cmd, flag); value != "" {
			query.Set(param, value)
		}
	}

	limit, pageSize := pagingFlags(cmd)

	checkouts, err := client.ListCheckouts(cmd.Context(), query, limit, pageSize)
	if err != nil {
		return scopeHint(err, checkoutBuilderScopes)
	}

	return printResult(cmd, checkouts, model.CheckoutViews(checkouts))
}

func newCheckoutBuilderGetCommand() *cobra.Command {
	return &cobra.Command{
		Annotations: apiOperations("GET /checkouts/{checkout_code}"),
		Use:         "get <checkout_code>",
		Short:       "Retrieve one checkout configuration",
		Long: "Retrieve one checkout configuration.\n\n" +
			"The table shows the lifecycle only; the payment methods, the general settings\n" +
			"and the styling live under `config` and `styling`, so use --json to read them.",
		Example: "  yuno-cli checkout-builder get 2a8d6472-3117-4e85-8b28-d51ba3abab77 --json",
		Args:    cobra.ExactArgs(1),
		RunE:    runCheckoutBuilderGet,
	}
}

func runCheckoutBuilderGet(cmd *cobra.Command, args []string) error {
	client, err := newClientFromFlags(cmd)
	if err != nil {
		return err
	}

	checkout, err := client.GetCheckout(cmd.Context(), args[0])
	if err != nil {
		return scopeHint(err, checkoutBuilderScopes)
	}

	return printResult(cmd, checkout, []model.CheckoutView{checkout.View()})
}

func newCheckoutBuilderPublishCommand() *cobra.Command {
	command := &cobra.Command{
		Annotations: apiOperations("PUT /checkouts/{checkout_code}"),
		Use:         "publish <checkout_code>",
		Short:       "Write the configuration and the styling of a checkout",
		Long: "Write the configuration and the styling of a checkout.\n\n" +
			"Yuno exposes this as a PUT: the body replaces what it touches, so send every\n" +
			"payment method that should survive the call, not only the ones that change.\n" +
			"`checkout-builder get --json` is the easiest starting point for the body.\n\n" +
			"This writes the draft; `checkout-builder update --status PUBLISHED` is what\n" +
			"makes it serve traffic.",
		Example: "  yuno-cli checkout-builder publish 2a8d6472-3117-4e85-8b28-d51ba3abab77 --file checkout.json",
		Args:    cobra.ExactArgs(1),
		RunE:    runCheckoutBuilderPublish,
	}

	registerWriteFlags(command, checkoutPublishFields)

	return command
}

func runCheckoutBuilderPublish(cmd *cobra.Command, args []string) error {
	body, err := requireBody(cmd, checkoutPublishFields)
	if err != nil {
		return err
	}

	client, err := newClientFromFlags(cmd)
	if err != nil {
		return err
	}

	data, err := client.PublishCheckout(cmd.Context(), args[0], body)
	if err != nil {
		return scopeHint(err, checkoutBuilderScopes)
	}

	return printJSON(cmd, data)
}

func newCheckoutBuilderUpdateCommand() *cobra.Command {
	command := &cobra.Command{
		Annotations: apiOperations("PATCH /checkouts/{checkout_code}"),
		Use:         "update <checkout_code>",
		Short:       "Rename a checkout or move it through its lifecycle",
		Long: "Rename a checkout or move it through its lifecycle.\n\n" +
			"PUBLISHED is only reachable from NOT_PUBLISHED, and --default only accepts\n" +
			"true: a checkout is demoted by promoting another one.",
		Example: "  yuno-cli checkout-builder update 2a8d6472-3117-4e85-8b28-d51ba3abab77 --status PUBLISHED",
		Args:    cobra.ExactArgs(1),
		RunE:    runCheckoutBuilderUpdate,
	}

	registerWriteFlags(command, checkoutUpdateFields)

	return command
}

func runCheckoutBuilderUpdate(cmd *cobra.Command, args []string) error {
	body, err := requireBody(cmd, checkoutUpdateFields)
	if err != nil {
		return err
	}

	client, err := newClientFromFlags(cmd)
	if err != nil {
		return err
	}

	checkout, err := client.UpdateCheckout(cmd.Context(), args[0], body)
	if err != nil {
		return scopeHint(err, checkoutBuilderScopes)
	}

	return printResult(cmd, checkout, []model.CheckoutView{checkout.View()})
}
