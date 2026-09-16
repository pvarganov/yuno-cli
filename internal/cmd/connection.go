package cmd

import (
	"github.com/spf13/cobra"

	"github.com/pvarganov/yuno-cli/internal/model"
)

// connectionScopes are the scopes a 403 on these commands usually asks for.
const connectionScopes = "connections:read/connections:write"

// connectionFields are the named body fields of `connection create`.
var connectionFields = []FieldFlag{
	{Flag: "account-id", Field: "account_id", Kind: FieldString, Usage: "account the connection belongs to"},
	{Flag: "merchant-connection-id", Field: "merchant_connection_id", Kind: FieldString, Usage: "your own id for the connection"},
	{Flag: "provider-id", Field: "provider_id", Kind: FieldString, Usage: "provider to connect to, e.g. ADYEN"},
	{Flag: "flow-type", Field: "flow_type", Kind: FieldString, Usage: "connection flow type, e.g. PAYIN"},
	{Flag: "payment-method", Field: "payment_methods", Kind: FieldStringSlice, Usage: "payment method the connection serves (repeatable)"},
	{Flag: "params", Field: "params", Kind: FieldJSON, Usage: "provider parameters as a JSON array"},
}

// newConnectionCommand builds the `connection` command tree.
func newConnectionCommand() *cobra.Command {
	connection := &cobra.Command{
		Use:     "connection",
		Short:   "Inspect and create provider connections",
		Aliases: []string{"connections"},
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return cmd.Help()
		},
	}

	connection.AddCommand(
		newConnectionCatalogCommand(),
		newConnectionGetCommand(),
		newConnectionCreateCommand(),
	)

	return connection
}

func newConnectionCatalogCommand() *cobra.Command {
	return &cobra.Command{
		Annotations: apiOperations("GET /connections/catalog/{provider_id}"),
		Use:         "catalog <provider_id>",
		Short:       "Show what a provider supports and which parameters it needs",
		Example:     "  yuno-cli connection catalog STRIPE",
		Args:        cobra.ExactArgs(1),
		RunE:        runConnectionCatalog,
	}
}

func runConnectionCatalog(cmd *cobra.Command, args []string) error {
	client, err := newClientFromFlags(cmd)
	if err != nil {
		return err
	}

	catalog, err := client.GetProviderCatalog(cmd.Context(), args[0])
	if err != nil {
		return scopeHint(err, connectionScopes)
	}

	return printResult(cmd, catalog, catalog.Views())
}

func newConnectionGetCommand() *cobra.Command {
	return &cobra.Command{
		Annotations: apiOperations("GET /connections/{connection_id}"),
		Use:         "get <connection_id>",
		Short:       "Retrieve one provider connection",
		Example:     "  yuno-cli connection get f1a3c4d5 --json",
		Args:        cobra.ExactArgs(1),
		RunE:        runConnectionGet,
	}
}

func runConnectionGet(cmd *cobra.Command, args []string) error {
	client, err := newClientFromFlags(cmd)
	if err != nil {
		return err
	}

	connection, err := client.GetConnection(cmd.Context(), args[0])
	if err != nil {
		return scopeHint(err, connectionScopes)
	}

	return printResult(cmd, connection, []model.ConnectionView{connection.View()})
}

func newConnectionCreateCommand() *cobra.Command {
	command := &cobra.Command{
		Annotations: apiOperations("POST /connections"),
		Use:         "create",
		Short:       "Create a provider connection",
		Example:     "  yuno-cli connection create --file connection.json",
		Args:        cobra.NoArgs,
		RunE:        runConnectionCreate,
	}

	registerWriteFlags(command, connectionFields)

	return command
}

func runConnectionCreate(cmd *cobra.Command, _ []string) error {
	body, err := requireBody(cmd, connectionFields)
	if err != nil {
		return err
	}

	client, err := newClientFromFlags(cmd)
	if err != nil {
		return err
	}

	connection, err := client.CreateConnection(cmd.Context(), body)
	if err != nil {
		return scopeHint(err, connectionScopes)
	}

	return printResult(cmd, connection, []model.ConnectionView{connection.View()})
}
