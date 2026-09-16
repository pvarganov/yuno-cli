package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/pvarganov/yuno-cli/internal/model"
)

// customerScopes are the scopes a 403 on the customer commands usually asks for.
const customerScopes = "customers:read/customers:write"

// customerFields are the named body fields of `customer create` and
// `customer update`. Addresses and metadata come from --file.
var customerFields = []FieldFlag{
	{Flag: "merchant-customer-id", Field: "merchant_customer_id", Kind: FieldString, Usage: "your own id for the customer"},
	{Flag: "merchant-customer-created-at", Field: "merchant_customer_created_at", Kind: FieldString, Usage: "when the customer was created on your side"},
	{Flag: "first-name", Field: "first_name", Kind: FieldString, Usage: "first name of the customer"},
	{Flag: "last-name", Field: "last_name", Kind: FieldString, Usage: "last name of the customer"},
	{Flag: "email", Field: "email", Kind: FieldString, Usage: "email of the customer"},
	{Flag: "gender", Field: "gender", Kind: FieldString, Usage: "gender of the customer, e.g. M, F or NB"},
	{Flag: "date-of-birth", Field: "date_of_birth", Kind: FieldString, Usage: "date of birth, YYYY-MM-DD"},
	{Flag: "nationality", Field: "nationality", Kind: FieldString, Usage: "nationality of the customer, e.g. CO"},
	{Flag: "country", Field: "country", Kind: FieldString, Usage: "country of the customer, e.g. CO"},
	{Flag: "document-number", Field: "document.document_number", Kind: FieldString, Usage: "document number of the customer"},
	{Flag: "document-type", Field: "document.document_type", Kind: FieldString, Usage: "document type of the customer, e.g. CC"},
	{Flag: "phone-number", Field: "phone.number", Kind: FieldString, Usage: "phone number of the customer"},
	{Flag: "phone-country-code", Field: "phone.country_code", Kind: FieldString, Usage: "phone country code of the customer, e.g. 57"},
	{Flag: "billing-address", Field: "billing_address", Kind: FieldJSON, Usage: "billing address as a JSON object"},
	{Flag: "shipping-address", Field: "shipping_address", Kind: FieldJSON, Usage: "shipping address as a JSON object"},
	{Flag: "metadata", Field: "metadata", Kind: FieldJSON, Usage: "metadata as a JSON array"},
}

// customerSessionFields are the body fields of `customer session create`.
var customerSessionFields = []FieldFlag{
	{Flag: "account-id", Field: "account_id", Kind: FieldString, Usage: "account the session belongs to"},
	{Flag: "country", Field: "country", Kind: FieldString, Usage: "country of the session, e.g. CO"},
	{Flag: "customer-id", Field: "customer_id", Kind: FieldString, Usage: "customer the session belongs to"},
	{Flag: "callback-url", Field: "callback_url", Kind: FieldString, Usage: "url the customer returns to"},
	{Flag: "checkout-id", Field: "checkout_id", Kind: FieldString, Usage: "checkout the session is opened for"},
}

// networkTokenFields are the body fields of `network-token cryptogram`.
var networkTokenFields = []FieldFlag{
	{Flag: "vaulted-token", Field: "vaulted_token", Kind: FieldString, Usage: "vaulted token of the enrolled card"},
	{Flag: "country", Field: "country", Kind: FieldString, Usage: "country the token will be used in, e.g. US"},
	{Flag: "currency", Field: "amount.currency", Kind: FieldString, Usage: "currency bound to the cryptogram, e.g. USD"},
	{Flag: "amount", Field: "amount.value", Kind: FieldNumber, Usage: "amount bound to the cryptogram"},
	{Flag: "include-network-token", Field: "include_network_token", Kind: FieldBool, Usage: "return the network token next to the cryptogram"},
}

// newCustomerCommand builds the `customer` command tree.
func newCustomerCommand() *cobra.Command {
	customer := &cobra.Command{
		Use:     "customer",
		Short:   "Create, inspect and delete customers",
		Aliases: []string{"customers"},
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return cmd.Help()
		},
	}

	customer.AddCommand(
		newCustomerCreateCommand(),
		newCustomerGetCommand(),
		newCustomerListCommand(),
		newCustomerByMerchantIDCommand(),
		newCustomerUpdateCommand(),
		newCustomerDeleteCommand(),
		newCustomerSessionCommand(),
	)

	return customer
}

func newCustomerCreateCommand() *cobra.Command {
	command := &cobra.Command{
		Annotations: apiOperations("POST /customers"),
		Use:         "create",
		Short:       "Create a customer",
		Example:     "  yuno-cli customer create --merchant-customer-id user-42 --email user@example.com",
		Args:        cobra.NoArgs,
		RunE:        runCustomerCreate,
	}

	registerWriteFlags(command, customerFields)

	return command
}

func runCustomerCreate(cmd *cobra.Command, _ []string) error {
	body, err := requireBody(cmd, customerFields)
	if err != nil {
		return err
	}

	client, err := newClientFromFlags(cmd)
	if err != nil {
		return err
	}

	customer, err := client.CreateCustomer(cmd.Context(), body)
	if err != nil {
		return scopeHint(err, customerScopes)
	}

	return printCustomer(cmd, customer)
}

func newCustomerGetCommand() *cobra.Command {
	return &cobra.Command{
		Annotations: apiOperations("GET /customers/{customer_id}"),
		Use:         "get <customer_id>",
		Short:       "Retrieve one customer",
		Example:     "  yuno-cli customer get 8f3c9a1e --json",
		Args:        cobra.ExactArgs(1),
		RunE:        runCustomerGet,
	}
}

func runCustomerGet(cmd *cobra.Command, args []string) error {
	client, err := newClientFromFlags(cmd)
	if err != nil {
		return err
	}

	customer, err := client.GetCustomer(cmd.Context(), args[0])
	if err != nil {
		return scopeHint(err, customerScopes)
	}

	return printCustomer(cmd, customer)
}

// newCustomerListCommand exposes `GET /customers`, which Yuno only serves
// scoped to one merchant customer id - there is no unfiltered customer list.
func newCustomerListCommand() *cobra.Command {
	command := &cobra.Command{
		Annotations: apiOperations("GET /customers"),
		Use:         "list",
		Short:       "Look a customer up by your own customer id",
		Example:     "  yuno-cli customer list --merchant-customer-id user-42",
		Args:        cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runCustomerByMerchantID(cmd, flagString(cmd, "merchant-customer-id"))
		},
	}

	command.Flags().String("merchant-customer-id", "", "your own customer id to look up")
	_ = command.MarkFlagRequired("merchant-customer-id")

	return command
}

func newCustomerByMerchantIDCommand() *cobra.Command {
	return &cobra.Command{
		Annotations: apiOperations("GET /customers"),
		Use:         "get-by-merchant-id <merchant_customer_id>",
		Short:       "Retrieve a customer by your own customer id",
		Example:     "  yuno-cli customer get-by-merchant-id user-42",
		Args:        cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCustomerByMerchantID(cmd, args[0])
		},
	}
}

func runCustomerByMerchantID(cmd *cobra.Command, merchantCustomerID string) error {
	if merchantCustomerID == "" {
		return fmt.Errorf("merchant customer id must not be empty")
	}

	client, err := newClientFromFlags(cmd)
	if err != nil {
		return err
	}

	customer, err := client.GetCustomerByMerchantCustomerID(cmd.Context(), merchantCustomerID)
	if err != nil {
		return scopeHint(err, customerScopes)
	}

	return printCustomer(cmd, customer)
}

func newCustomerUpdateCommand() *cobra.Command {
	command := &cobra.Command{
		Annotations: apiOperations("PATCH /customers/{customer_id}"),
		Use:         "update <customer_id>",
		Short:       "Update a customer",
		Example:     "  yuno-cli customer update 8f3c9a1e --email new@example.com",
		Args:        cobra.ExactArgs(1),
		RunE:        runCustomerUpdate,
	}

	registerWriteFlags(command, customerFields)

	return command
}

func runCustomerUpdate(cmd *cobra.Command, args []string) error {
	body, err := requireBody(cmd, customerFields)
	if err != nil {
		return err
	}

	client, err := newClientFromFlags(cmd)
	if err != nil {
		return err
	}

	customer, err := client.UpdateCustomer(cmd.Context(), args[0], body)
	if err != nil {
		return scopeHint(err, customerScopes)
	}

	return printCustomer(cmd, customer)
}

func newCustomerDeleteCommand() *cobra.Command {
	command := &cobra.Command{
		Annotations: apiOperations("DELETE /customers/{customer_id}"),
		Use:         "delete <customer_id>",
		Short:       "Delete a customer",
		Example:     "  yuno-cli customer delete 8f3c9a1e --yes",
		Args:        cobra.ExactArgs(1),
		RunE:        runCustomerDelete,
	}

	return command
}

func runCustomerDelete(cmd *cobra.Command, args []string) error {
	client, err := newClientFromFlags(cmd)
	if err != nil {
		return err
	}

	data, err := client.DeleteCustomer(cmd.Context(), args[0])
	if err != nil {
		return scopeHint(err, customerScopes)
	}

	return printJSON(cmd, data)
}

// newCustomerSessionCommand builds `customer session`, the enrollment session
// group.
func newCustomerSessionCommand() *cobra.Command {
	session := &cobra.Command{
		Use:     "session",
		Short:   "Manage customer enrollment sessions",
		Aliases: []string{"sessions"},
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return cmd.Help()
		},
	}

	create := &cobra.Command{
		Annotations: apiOperations("POST /customers/sessions"),
		Use:         "create",
		Short:       "Open an enrollment session for a customer",
		Example:     "  yuno-cli customer session create --account-id acc-1 --country CO --customer-id cus-1",
		Args:        cobra.NoArgs,
		RunE:        runCustomerSessionCreate,
	}

	registerWriteFlags(create, customerSessionFields)
	session.AddCommand(create)

	return session
}

func runCustomerSessionCreate(cmd *cobra.Command, _ []string) error {
	body, err := requireBody(cmd, customerSessionFields)
	if err != nil {
		return err
	}

	client, err := newClientFromFlags(cmd)
	if err != nil {
		return err
	}

	session, err := client.CreateCustomerSession(cmd.Context(), body)
	if err != nil {
		return scopeHint(err, customerScopes)
	}

	return printResult(cmd, session, []model.CustomerSessionView{session.View()})
}

// newNetworkTokenCommand builds the `network-token` command tree.
func newNetworkTokenCommand() *cobra.Command {
	networkToken := &cobra.Command{
		Use:     "network-token",
		Short:   "Work with network tokens of vaulted cards",
		Aliases: []string{"network-tokens"},
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return cmd.Help()
		},
	}

	cryptogram := &cobra.Command{
		Annotations: apiOperations("POST /network-tokens/cryptograms"),
		Use:         "cryptogram",
		Short:       "Generate a cryptogram for the network token of a vaulted card",
		Example:     "  yuno-cli network-token cryptogram --vaulted-token 9b2f4b1c --country US",
		Args:        cobra.NoArgs,
		RunE:        runNetworkTokenCryptogram,
	}

	registerWriteFlags(cryptogram, networkTokenFields)
	networkToken.AddCommand(cryptogram)

	return networkToken
}

func runNetworkTokenCryptogram(cmd *cobra.Command, _ []string) error {
	body, err := requireBody(cmd, networkTokenFields)
	if err != nil {
		return err
	}

	client, err := newClientFromFlags(cmd)
	if err != nil {
		return err
	}

	cryptogram, err := client.GenerateNetworkTokenCryptogram(cmd.Context(), body)
	if err != nil {
		return scopeHint(err, customerScopes)
	}

	return printResult(cmd, cryptogram, []model.NetworkTokenCryptogramView{cryptogram.View()})
}

// printCustomer renders one customer: the whole response as JSON, a single
// table row otherwise.
func printCustomer(cmd *cobra.Command, customer *model.Customer) error {
	return printResult(cmd, customer, []model.CustomerView{customer.View()})
}
