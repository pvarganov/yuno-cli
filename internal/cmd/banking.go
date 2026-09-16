package cmd

import (
	"github.com/spf13/cobra"

	"github.com/pvarganov/yuno-cli/internal/model"
)

// bankingScopes are the scopes a 403 on the banking commands usually asks for.
const bankingScopes = "banking:read/banking:write"

// bankingEntityFields are the named body fields of the entity commands. The
// national entity, phone, address and entity detail objects are deep enough to
// come from --file or from their JSON flags.
var bankingEntityFields = []FieldFlag{
	{Flag: "account-id", Field: "account_id", Kind: FieldString, Usage: "account the entity belongs to"},
	{
		Flag: "merchant-entity-id", Field: "merchant_entity_id", Kind: FieldString,
		Usage: "your own identifier for the entity",
	},
	{
		Flag: "national-entity", Field: "national_entity", Kind: FieldString,
		Usage: "country of incorporation, ISO 3166-1 alpha-2",
	},
	{Flag: "phone", Field: "phone", Kind: FieldJSON, Usage: "phone as a JSON object"},
	{Flag: "address", Field: "address", Kind: FieldJSON, Usage: "address as a JSON object"},
	{
		Flag: "entity-detail", Field: "entity_detail", Kind: FieldJSON,
		Usage: "business or individual detail as a JSON object",
	},
}

// bankingOnboardingFields are the named body fields of the onboarding commands.
var bankingOnboardingFields = []FieldFlag{
	{Flag: "account-id", Field: "account_id", Kind: FieldString, Usage: "account the onboarding belongs to"},
	{
		Flag: "yuno-connection-id", Field: "yuno_connection_id", Kind: FieldString,
		Usage: "provider connection the entity is onboarded with",
	},
	{
		Flag: "onboarding-type", Field: "onboarding_type", Kind: FieldString,
		Usage: "what the entity is being onboarded for",
	},
	{
		Flag: "compliance-declaration", Field: "compliance_declaration", Kind: FieldJSON,
		Usage: "compliance declaration as a JSON object",
	},
	{Flag: "risk-assessment", Field: "risk_assessment", Kind: FieldJSON, Usage: "risk assessment as a JSON object"},
	{Flag: "source-of-funds", Field: "source_of_funds", Kind: FieldJSON, Usage: "source of funds as a JSON object"},
	{Flag: "documentation", Field: "documentation", Kind: FieldJSON, Usage: "supporting documents as a JSON array"},
}

// bankingAccountFields are the named body fields of the account commands.
var bankingAccountFields = []FieldFlag{
	{Flag: "account-id", Field: "account_id", Kind: FieldString, Usage: "account the bank account belongs to"},
	{
		Flag: "onboarding-id", Field: "onboarding_id", Kind: FieldString,
		Usage: "completed onboarding the bank account is opened for",
	},
	{Flag: "account-type", Field: "account_type", Kind: FieldString, Usage: "type of bank account to open"},
	{Flag: "currency", Field: "currency", Kind: FieldString, Usage: "currency of the account, ISO 4217"},
}

// bankingTransferFields are the named body fields of `banking transfer create`.
var bankingTransferFields = []FieldFlag{
	{Flag: "account-id", Field: "account_id", Kind: FieldString, Usage: "account the transfer belongs to"},
	{Flag: "source-account-id", Field: "source_account_id", Kind: FieldString, Usage: "bank account to debit"},
	{
		Flag: "destination-account-id", Field: "destination_account_id", Kind: FieldString,
		Usage: "bank account to credit, for book transfers between your own accounts",
	},
	{Flag: "direction", Field: "direction", Kind: FieldString, Usage: "direction of the transfer"},
	{Flag: "payment-rail", Field: "payment_rail", Kind: FieldString, Usage: "rail to send the transfer on, e.g. ACH"},
	{Flag: "description", Field: "description", Kind: FieldString, Usage: "description shown on the statement"},
	{
		Flag: "merchant-transfer-id", Field: "merchant_transfer_id", Kind: FieldString,
		Usage: "your own identifier for the transfer",
	},
	{Flag: "amount", Field: "amount", Kind: FieldJSON, Usage: "amount as a JSON object: value and currency"},
	{
		Flag: "destination-account", Field: "destination_account", Kind: FieldJSON,
		Usage: "external destination account as a JSON object",
	},
}

// newBankingCommand builds the `banking` command tree.
func newBankingCommand() *cobra.Command {
	banking := &cobra.Command{
		Use:   "banking",
		Short: "Manage banking connectivity entities, accounts and transfers",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return cmd.Help()
		},
	}

	banking.AddCommand(
		newBankingEntityCommand(),
		newBankingAccountCommand(),
		newBankingTransferCommand(),
	)

	return banking
}

// newBankingEntityCommand builds the `banking entity` subtree, onboardings included.
func newBankingEntityCommand() *cobra.Command {
	entity := &cobra.Command{
		Use:     "entity",
		Short:   "Manage banking connectivity entities",
		Aliases: []string{"entities"},
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return cmd.Help()
		},
	}

	create := &cobra.Command{
		Annotations: apiOperations("POST /banking/entities"),
		Use:         "create",
		Short:       "Register a legal entity for banking connectivity",
		Example:     "  yuno-cli banking entity create --file entity.json",
		Args:        cobra.NoArgs,
		RunE:        runBankingEntityCreate,
	}
	registerWriteFlags(create, bankingEntityFields)

	update := &cobra.Command{
		Annotations: apiOperations("PATCH /banking/entities/{entity_id}"),
		Use:         "update <entity_id>",
		Short:       "Update an entity",
		Example:     "  yuno-cli banking entity update be-1 --account-id acc-1 --phone '{\"number\":\"5550000\"}'",
		Args:        cobra.ExactArgs(1),
		RunE:        runBankingEntityUpdate,
	}
	registerWriteFlags(update, bankingEntityFields)

	entity.AddCommand(
		create,
		&cobra.Command{
			Annotations: apiOperations("GET /banking/entities/{entity_id}"),
			Use:         "get <entity_id>",
			Short:       "Retrieve one entity",
			Example:     "  yuno-cli banking entity get be-1 --json",
			Args:        cobra.ExactArgs(1),
			RunE:        runBankingEntityGet,
		},
		update,
		newBankingOnboardingCommand(),
	)

	return entity
}

func runBankingEntityCreate(cmd *cobra.Command, _ []string) error {
	body, err := requireBody(cmd, bankingEntityFields)
	if err != nil {
		return err
	}

	client, err := newClientFromFlags(cmd)
	if err != nil {
		return err
	}

	entity, err := client.CreateBankingEntity(cmd.Context(), body)
	if err != nil {
		return scopeHint(err, bankingScopes)
	}

	return printResult(cmd, entity, []model.BankingEntityView{entity.View()})
}

func runBankingEntityGet(cmd *cobra.Command, args []string) error {
	client, err := newClientFromFlags(cmd)
	if err != nil {
		return err
	}

	entity, err := client.GetBankingEntity(cmd.Context(), args[0])
	if err != nil {
		return scopeHint(err, bankingScopes)
	}

	return printResult(cmd, entity, []model.BankingEntityView{entity.View()})
}

func runBankingEntityUpdate(cmd *cobra.Command, args []string) error {
	body, err := requireBody(cmd, bankingEntityFields)
	if err != nil {
		return err
	}

	client, err := newClientFromFlags(cmd)
	if err != nil {
		return err
	}

	entity, err := client.UpdateBankingEntity(cmd.Context(), args[0], body)
	if err != nil {
		return scopeHint(err, bankingScopes)
	}

	return printResult(cmd, entity, []model.BankingEntityView{entity.View()})
}

// newBankingOnboardingCommand builds the `banking entity onboarding` subtree.
func newBankingOnboardingCommand() *cobra.Command {
	onboarding := &cobra.Command{
		Use:     "onboarding",
		Short:   "Manage the onboarding of an entity with a banking provider",
		Aliases: []string{"onboardings"},
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return cmd.Help()
		},
	}

	create := &cobra.Command{
		Annotations: apiOperations("POST /banking/entities/{entity_id}/onboardings"),
		Use:         "create <entity_id>",
		Short:       "Start the onboarding of an entity",
		Example:     "  yuno-cli banking entity onboarding create be-1 --account-id acc-1 --onboarding-type BANK_ACCOUNT",
		Args:        cobra.ExactArgs(1),
		RunE:        runBankingOnboardingCreate,
	}
	registerWriteFlags(create, bankingOnboardingFields)

	update := &cobra.Command{
		Annotations: apiOperations("PATCH /banking/entities/{entity_id}/onboardings/{onboarding_id}"),
		Use:         "update <entity_id> <onboarding_id>",
		Short:       "Update an onboarding",
		Long: "Update an onboarding.\n\n" +
			"This is how a pending requirement is satisfied: send the document or the\n" +
			"declaration the onboarding status asked for.",
		Example: "  yuno-cli banking entity onboarding update be-1 bo-1 --file requirement.json",
		Args:    cobra.ExactArgs(2),
		RunE:    runBankingOnboardingUpdate,
	}
	registerWriteFlags(update, bankingOnboardingFields)

	onboarding.AddCommand(
		create,
		&cobra.Command{
			Annotations: apiOperations("GET /banking/entities/{entity_id}/onboardings/{onboarding_id}"),
			Use:         "get <entity_id> <onboarding_id>",
			Short:       "Retrieve the status of an onboarding",
			Example:     "  yuno-cli banking entity onboarding get be-1 bo-1 --json",
			Args:        cobra.ExactArgs(2),
			RunE:        runBankingOnboardingGet,
		},
		update,
		&cobra.Command{
			Annotations: apiOperations("POST /banking/entities/{entity_id}/onboardings/{onboarding_id}/cancel"),
			Use:         "cancel <entity_id> <onboarding_id>",
			Short:       "Cancel an onboarding that has not completed",
			Example:     "  yuno-cli banking entity onboarding cancel be-1 bo-1",
			Args:        cobra.ExactArgs(2),
			RunE:        runBankingOnboardingCancel,
		},
	)

	return onboarding
}

func runBankingOnboardingCreate(cmd *cobra.Command, args []string) error {
	body, err := requireBody(cmd, bankingOnboardingFields)
	if err != nil {
		return err
	}

	client, err := newClientFromFlags(cmd)
	if err != nil {
		return err
	}

	onboarding, err := client.CreateBankingOnboarding(cmd.Context(), args[0], body)
	if err != nil {
		return scopeHint(err, bankingScopes)
	}

	return printResult(cmd, onboarding, []model.BankingOnboardingView{onboarding.View()})
}

func runBankingOnboardingGet(cmd *cobra.Command, args []string) error {
	client, err := newClientFromFlags(cmd)
	if err != nil {
		return err
	}

	onboarding, err := client.GetBankingOnboarding(cmd.Context(), args[0], args[1])
	if err != nil {
		return scopeHint(err, bankingScopes)
	}

	return printResult(cmd, onboarding, []model.BankingOnboardingView{onboarding.View()})
}

func runBankingOnboardingUpdate(cmd *cobra.Command, args []string) error {
	body, err := requireBody(cmd, bankingOnboardingFields)
	if err != nil {
		return err
	}

	client, err := newClientFromFlags(cmd)
	if err != nil {
		return err
	}

	onboarding, err := client.UpdateBankingOnboarding(cmd.Context(), args[0], args[1], body)
	if err != nil {
		return scopeHint(err, bankingScopes)
	}

	return printResult(cmd, onboarding, []model.BankingOnboardingView{onboarding.View()})
}

func runBankingOnboardingCancel(cmd *cobra.Command, args []string) error {
	client, err := newClientFromFlags(cmd)
	if err != nil {
		return err
	}

	onboarding, err := client.CancelBankingOnboarding(cmd.Context(), args[0], args[1])
	if err != nil {
		return scopeHint(err, bankingScopes)
	}

	return printResult(cmd, onboarding, []model.BankingOnboardingView{onboarding.View()})
}

// newBankingAccountCommand builds the `banking account` subtree.
func newBankingAccountCommand() *cobra.Command {
	account := &cobra.Command{
		Use:     "account",
		Short:   "Manage the bank accounts of onboarded entities",
		Aliases: []string{"accounts"},
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return cmd.Help()
		},
	}

	create := &cobra.Command{
		Annotations: apiOperations("POST /banking/accounts"),
		Use:         "create",
		Short:       "Open a bank account for a completed onboarding",
		Example: "  yuno-cli banking account create --account-id acc-1 --onboarding-id bo-1 " +
			"--account-type CHECKING --currency USD",
		Args: cobra.NoArgs,
		RunE: runBankingAccountCreate,
	}
	registerWriteFlags(create, bankingAccountFields)

	update := &cobra.Command{
		Annotations: apiOperations("PATCH /banking/accounts/{account_id}"),
		Use:         "update <account_id>",
		Short:       "Update a bank account",
		Example:     "  yuno-cli banking account update ba-1 --account-id acc-1 --account-type SAVINGS",
		Args:        cobra.ExactArgs(1),
		RunE:        runBankingAccountUpdate,
	}
	registerWriteFlags(update, bankingAccountFields)

	account.AddCommand(
		create,
		&cobra.Command{
			Annotations: apiOperations("GET /banking/accounts/{account_id}"),
			Use:         "get <account_id>",
			Short:       "Retrieve one bank account with its balance",
			Example:     "  yuno-cli banking account get ba-1 --json",
			Args:        cobra.ExactArgs(1),
			RunE:        runBankingAccountGet,
		},
		update,
		&cobra.Command{
			Annotations: apiOperations("DELETE /banking/accounts/{account_id}"),
			Use:         "close <account_id>",
			Short:       "Close a bank account",
			Long: "Close a bank account.\n\n" +
				"Yuno exposes this as a DELETE but answers with the closed account, not an\n" +
				"empty body.",
			Aliases: []string{"delete"},
			Example: "  yuno-cli banking account close ba-1",
			Args:    cobra.ExactArgs(1),
			RunE:    runBankingAccountClose,
		},
	)

	return account
}

func runBankingAccountCreate(cmd *cobra.Command, _ []string) error {
	body, err := requireBody(cmd, bankingAccountFields)
	if err != nil {
		return err
	}

	client, err := newClientFromFlags(cmd)
	if err != nil {
		return err
	}

	account, err := client.CreateBankingAccount(cmd.Context(), body)
	if err != nil {
		return scopeHint(err, bankingScopes)
	}

	return printResult(cmd, account, []model.BankingAccountView{account.View()})
}

func runBankingAccountGet(cmd *cobra.Command, args []string) error {
	client, err := newClientFromFlags(cmd)
	if err != nil {
		return err
	}

	account, err := client.GetBankingAccount(cmd.Context(), args[0])
	if err != nil {
		return scopeHint(err, bankingScopes)
	}

	return printResult(cmd, account, []model.BankingAccountView{account.View()})
}

func runBankingAccountUpdate(cmd *cobra.Command, args []string) error {
	body, err := requireBody(cmd, bankingAccountFields)
	if err != nil {
		return err
	}

	client, err := newClientFromFlags(cmd)
	if err != nil {
		return err
	}

	account, err := client.UpdateBankingAccount(cmd.Context(), args[0], body)
	if err != nil {
		return scopeHint(err, bankingScopes)
	}

	return printResult(cmd, account, []model.BankingAccountView{account.View()})
}

func runBankingAccountClose(cmd *cobra.Command, args []string) error {
	client, err := newClientFromFlags(cmd)
	if err != nil {
		return err
	}

	account, err := client.CloseBankingAccount(cmd.Context(), args[0])
	if err != nil {
		return scopeHint(err, bankingScopes)
	}

	return printResult(cmd, account, []model.BankingAccountView{account.View()})
}

// newBankingTransferCommand builds the `banking transfer` subtree. A transfer is
// created on /banking/transfers but read back through its source account.
func newBankingTransferCommand() *cobra.Command {
	transfer := &cobra.Command{
		Use:     "transfer",
		Short:   "Move money between banking connectivity accounts",
		Aliases: []string{"transfers"},
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return cmd.Help()
		},
	}

	create := &cobra.Command{
		Annotations: apiOperations("POST /banking/transfers"),
		Use:         "create",
		Short:       "Initiate a transfer",
		Example: "  yuno-cli banking transfer create --account-id acc-1 --source-account-id ba-1 " +
			"--direction OUTBOUND --payment-rail ACH --amount '{\"value\":250,\"currency\":\"USD\"}' " +
			"--destination-account '{\"account_number\":\"9876543210\"}'",
		Args: cobra.NoArgs,
		RunE: runBankingTransferCreate,
	}
	registerWriteFlags(create, bankingTransferFields)

	transfer.AddCommand(
		create,
		&cobra.Command{
			Annotations: apiOperations("GET /banking/accounts/{account_id}/transfers/{transfer_id}"),
			Use:         "get <account_id> <transfer_id>",
			Short:       "Retrieve the status of a transfer",
			Long: "Retrieve the status of a transfer.\n\n" +
				"The read side hangs off the source account, so the account id comes first.",
			Example: "  yuno-cli banking transfer get ba-1 bt-1 --json",
			Args:    cobra.ExactArgs(2),
			RunE:    runBankingTransferGet,
		},
		&cobra.Command{
			Annotations: apiOperations("POST /banking/accounts/{account_id}/transfers/{transfer_id}/cancel"),
			Use:         "cancel <account_id> <transfer_id>",
			Short:       "Cancel a transfer that has not settled",
			Example:     "  yuno-cli banking transfer cancel ba-1 bt-1",
			Args:        cobra.ExactArgs(2),
			RunE:        runBankingTransferCancel,
		},
	)

	return transfer
}

func runBankingTransferCreate(cmd *cobra.Command, _ []string) error {
	body, err := requireBody(cmd, bankingTransferFields)
	if err != nil {
		return err
	}

	client, err := newClientFromFlags(cmd)
	if err != nil {
		return err
	}

	transfer, err := client.CreateBankingTransfer(cmd.Context(), body)
	if err != nil {
		return scopeHint(err, bankingScopes)
	}

	return printResult(cmd, transfer, []model.BankingTransferView{transfer.View()})
}

func runBankingTransferGet(cmd *cobra.Command, args []string) error {
	client, err := newClientFromFlags(cmd)
	if err != nil {
		return err
	}

	transfer, err := client.GetBankingTransfer(cmd.Context(), args[0], args[1])
	if err != nil {
		return scopeHint(err, bankingScopes)
	}

	return printResult(cmd, transfer, []model.BankingTransferView{transfer.View()})
}

func runBankingTransferCancel(cmd *cobra.Command, args []string) error {
	client, err := newClientFromFlags(cmd)
	if err != nil {
		return err
	}

	transfer, err := client.CancelBankingTransfer(cmd.Context(), args[0], args[1])
	if err != nil {
		return scopeHint(err, bankingScopes)
	}

	return printResult(cmd, transfer, []model.BankingTransferView{transfer.View()})
}
