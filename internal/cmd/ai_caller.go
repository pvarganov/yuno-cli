package cmd

import (
	"github.com/spf13/cobra"

	"github.com/pvarganov/yuno-cli/internal/model"
)

// aiCallerScopes are the scopes a 403 on the AI caller commands usually asks for.
const aiCallerScopes = "smart-support:write"

// aiCallerDeclinedFields are the named body fields of `ai-caller declined-payments`.
// Both are whole documents: the AI agent configuration and the payment context
// are too deep to deserve one flag per leaf, so they take JSON or come from --file.
var aiCallerDeclinedFields = []FieldFlag{
	{
		Flag: "settings", Field: "settings", Kind: FieldJSON,
		Usage: "AI agent configuration as a JSON object, including contact_details",
	},
	{
		Flag: "additional-information", Field: "additional_information", Kind: FieldJSON,
		Usage: "context for the AI as a JSON object: seller_details and the declined payment",
	},
}

// aiCallerRecoverFields are the named body fields of `ai-caller recover`.
var aiCallerRecoverFields = []FieldFlag{
	{Flag: "customer", Field: "customer", Kind: FieldJSON, Usage: "customer to reach out to, as a JSON object"},
	{Flag: "session", Field: "session", Kind: FieldJSON, Usage: "abandoned session as a JSON object"},
	{Flag: "cart", Field: "cart", Kind: FieldJSON, Usage: "abandoned cart as a JSON object"},
	{
		Flag: "engagement", Field: "engagement", Kind: FieldJSON,
		Usage: "engagement preferences as a JSON object: preferred_channel, reason and metadata",
	},
}

// newAICallerCommand builds the `ai-caller` command tree.
func newAICallerCommand() *cobra.Command {
	caller := &cobra.Command{
		Use:     "ai-caller",
		Short:   "Ask the AI caller to reach out to a customer",
		Aliases: []string{"smart-support"},
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return cmd.Help()
		},
	}

	caller.AddCommand(
		newAICallerDeclinedPaymentsCommand(),
		newAICallerRecoverCommand(),
	)

	return caller
}

func newAICallerDeclinedPaymentsCommand() *cobra.Command {
	command := &cobra.Command{
		Use:     "declined-payments",
		Short:   "Call the payer of a declined payment to recover it",
		Aliases: []string{"declined"},
		Long: "Call the payer of a declined payment to recover it.\n\n" +
			"The body carries the AI agent configuration under `settings` and the payment it\n" +
			"should talk about under `additional_information`. It is deep enough that --file\n" +
			"is usually the better way to send it.",
		Example: "  yuno-cli ai-caller declined-payments --file declined.json",
		Args:    cobra.NoArgs,
		RunE:    runAICallerDeclinedPayments,
	}

	registerWriteFlags(command, aiCallerDeclinedFields)

	return command
}

func runAICallerDeclinedPayments(cmd *cobra.Command, _ []string) error {
	body, err := requireBody(cmd, aiCallerDeclinedFields)
	if err != nil {
		return err
	}

	client, err := newClientFromFlags(cmd)
	if err != nil {
		return err
	}

	outreach, err := client.AICallerDeclinedPayments(cmd.Context(), body)
	if err != nil {
		return scopeHint(err, aiCallerScopes)
	}

	return printResult(cmd, outreach, []model.AICallerOutreachView{outreach.View()})
}

func newAICallerRecoverCommand() *cobra.Command {
	command := &cobra.Command{
		Use:     "recover",
		Short:   "Reach out to a customer who abandoned a checkout flow",
		Aliases: []string{"abandoned"},
		Long: "Reach out to a customer who abandoned a checkout flow.\n\n" +
			"Unlike `declined-payments`, this one is about a flow that never produced a\n" +
			"payment: the body describes the customer, the session they dropped, the cart\n" +
			"they left behind and how they prefer to be contacted.\n\n" +
			"Yuno answers 200 with no body, so a silent exit means the outreach was accepted.",
		Example: "  yuno-cli ai-caller recover --file abandoned.json",
		Args:    cobra.NoArgs,
		RunE:    runAICallerRecover,
	}

	registerWriteFlags(command, aiCallerRecoverFields)

	return command
}

func runAICallerRecover(cmd *cobra.Command, _ []string) error {
	body, err := requireBody(cmd, aiCallerRecoverFields)
	if err != nil {
		return err
	}

	client, err := newClientFromFlags(cmd)
	if err != nil {
		return err
	}

	data, err := client.AICallerRecover(cmd.Context(), body)
	if err != nil {
		return scopeHint(err, aiCallerScopes)
	}

	return printJSON(cmd, data)
}
