package cmd

import (
	"github.com/spf13/cobra"

	"github.com/pvarganov/yuno-cli/internal/model"
)

// sellerScopes are the scopes a 403 on the seller commands usually asks for.
const sellerScopes = "sellers:read/sellers:write"

// sellerFields are the named body fields of the seller commands. The nested
// document, phone, address and payment method objects come from --file or from
// their JSON flags.
var sellerFields = []FieldFlag{
	{Flag: "account-id", Field: "account_id", Kind: FieldString, Usage: "account the seller belongs to"},
	{
		Flag: "merchant-seller-id", Field: "merchant_seller_id", Kind: FieldString,
		Usage: "your own identifier for the seller, set on create only",
	},
	{Flag: "name", Field: "name", Kind: FieldString, Usage: "name of the seller"},
	{Flag: "email", Field: "email", Kind: FieldString, Usage: "email of the seller"},
	{Flag: "country", Field: "country", Kind: FieldString, Usage: "country of the seller, ISO 3166-1 alpha-2"},
	{Flag: "website", Field: "website", Kind: FieldString, Usage: "website of the seller"},
	{Flag: "industry", Field: "industry", Kind: FieldString, Usage: "industry of the seller"},
	{
		Flag: "merchant-category-code", Field: "merchant_category_code", Kind: FieldString,
		Usage: "four digit merchant category code",
	},
	{Flag: "document", Field: "document", Kind: FieldJSON, Usage: "document as a JSON object"},
	{Flag: "phone", Field: "phone", Kind: FieldJSON, Usage: "phone as a JSON object"},
	{Flag: "address", Field: "address", Kind: FieldJSON, Usage: "address as a JSON object"},
	{
		Flag: "payment-methods", Field: "payment_methods", Kind: FieldJSON,
		Usage: "payment method configuration as a JSON array",
	},
}

// newSellerCommand builds the `seller` command tree.
func newSellerCommand() *cobra.Command {
	seller := &cobra.Command{
		Use:     "seller",
		Short:   "Manage marketplace sellers",
		Aliases: []string{"sellers"},
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return cmd.Help()
		},
	}

	seller.AddCommand(
		newSellerCreateCommand(),
		newSellerGetCommand(),
		newSellerUpdateCommand(),
		newSellerDeleteCommand(),
	)

	return seller
}

func newSellerCreateCommand() *cobra.Command {
	command := &cobra.Command{
		Annotations: apiOperations("POST /sellers"),
		Use:         "create",
		Short:       "Create a seller",
		Long: "Create a seller.\n\n" +
			"Yuno has no seller list endpoint: a seller is read back by the merchant_seller_id\n" +
			"you assign here, so keep it.",
		Example: "  yuno-cli seller create --account-id acc-1 --merchant-seller-id shop-1 " +
			"--name 'Acme Shop' --country US",
		Args: cobra.NoArgs,
		RunE: runSellerCreate,
	}

	registerWriteFlags(command, sellerFields)

	return command
}

func runSellerCreate(cmd *cobra.Command, _ []string) error {
	body, err := requireBody(cmd, sellerFields)
	if err != nil {
		return err
	}

	client, err := newClientFromFlags(cmd)
	if err != nil {
		return err
	}

	seller, err := client.CreateSeller(cmd.Context(), body)
	if err != nil {
		return scopeHint(err, sellerScopes)
	}

	return printResult(cmd, seller, []model.SellerView{seller.View()})
}

func newSellerGetCommand() *cobra.Command {
	return &cobra.Command{
		Annotations: apiOperations("GET /sellers/{merchant_seller_id}"),
		Use:         "get <merchant_seller_id>",
		Short:       "Retrieve one seller",
		Example:     "  yuno-cli seller get shop-1 --json",
		Args:        cobra.ExactArgs(1),
		RunE:        runSellerGet,
	}
}

func runSellerGet(cmd *cobra.Command, args []string) error {
	client, err := newClientFromFlags(cmd)
	if err != nil {
		return err
	}

	seller, err := client.GetSeller(cmd.Context(), args[0])
	if err != nil {
		return scopeHint(err, sellerScopes)
	}

	return printResult(cmd, seller, []model.SellerView{seller.View()})
}

func newSellerUpdateCommand() *cobra.Command {
	command := &cobra.Command{
		Annotations: apiOperations("PUT /sellers/{merchant_seller_id}"),
		Use:         "update <merchant_seller_id>",
		Short:       "Update a seller",
		Long: "Update a seller.\n\n" +
			"Yuno exposes this as a PUT: the body replaces the seller, so send every field\n" +
			"that should survive the call, not only the ones that change.",
		Example: "  yuno-cli seller update shop-1 --account-id acc-1 --name 'Acme Shop EU' --country DE",
		Args:    cobra.ExactArgs(1),
		RunE:    runSellerUpdate,
	}

	registerWriteFlags(command, sellerFields)

	return command
}

func runSellerUpdate(cmd *cobra.Command, args []string) error {
	body, err := requireBody(cmd, sellerFields)
	if err != nil {
		return err
	}

	client, err := newClientFromFlags(cmd)
	if err != nil {
		return err
	}

	seller, err := client.UpdateSeller(cmd.Context(), args[0], body)
	if err != nil {
		return scopeHint(err, sellerScopes)
	}

	return printResult(cmd, seller, []model.SellerView{seller.View()})
}

func newSellerDeleteCommand() *cobra.Command {
	return &cobra.Command{
		Annotations: apiOperations("DELETE /sellers/{merchant_seller_id}"),
		Use:         "delete <merchant_seller_id>",
		Short:       "Delete a seller",
		Example:     "  yuno-cli seller delete shop-1",
		Args:        cobra.ExactArgs(1),
		RunE:        runSellerDelete,
	}
}

func runSellerDelete(cmd *cobra.Command, args []string) error {
	client, err := newClientFromFlags(cmd)
	if err != nil {
		return err
	}

	data, err := client.DeleteSeller(cmd.Context(), args[0])
	if err != nil {
		return scopeHint(err, sellerScopes)
	}

	return printJSON(cmd, data)
}
