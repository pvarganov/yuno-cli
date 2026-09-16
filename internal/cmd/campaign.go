package cmd

import (
	"net/url"

	"github.com/spf13/cobra"

	"github.com/pvarganov/yuno-cli/internal/model"
)

// campaignScopes are the scopes a 403 on the campaign commands usually asks for.
const campaignScopes = "campaigns:read/campaigns:write"

// campaignCreateFields are the named body fields of `campaign create`.
var campaignCreateFields = []FieldFlag{
	{Flag: "name", Field: "name", Kind: FieldString, Usage: "campaign name"},
	{Flag: "account-id", Field: "account_id", Kind: FieldString, Usage: "account the campaign runs on"},
	{
		Flag: "organization-code", Field: "organization_code", Kind: FieldString,
		Usage: "organization identifier, a UUID",
	},
	{Flag: "country", Field: "country", Kind: FieldString, Usage: "country of the campaign, ISO 3166-1 alpha-2"},
	{Flag: "channel", Field: "channel", Kind: FieldString, Usage: "communication channel: WHATSAPP_MESSAGE or PHONE_CALL"},
	{Flag: "focus", Field: "focus", Kind: FieldString, Usage: "campaign focus descriptor, e.g. payment_recovery"},
	{
		Flag: "daily-start-time", Field: "schedule.daily_start_time", Kind: FieldString,
		Usage: "start of the daily send window, HH:MM",
	},
	{
		Flag: "daily-end-time", Field: "schedule.daily_end_time", Kind: FieldString,
		Usage: "end of the daily send window, HH:MM",
	},
	{
		Flag: "time-zone", Field: "schedule.time_zone", Kind: FieldString,
		Usage: "IANA timezone of the send window, e.g. America/Bogota",
	},
	{Flag: "start-at", Field: "duration.start_at", Kind: FieldString, Usage: "campaign start date, ISO 8601"},
	{Flag: "end-at", Field: "duration.end_at", Kind: FieldString, Usage: "campaign end date, ISO 8601"},
}

// campaignStatusFields is the one-field body the status endpoints take.
var campaignStatusFields = []FieldFlag{
	{Flag: "status", Field: "status", Kind: FieldString, Usage: "target status of the campaign"},
}

// campaignRuleFields are the named body fields of one campaign rule.
var campaignRuleFields = []FieldFlag{
	{Flag: "rule-type", Field: "rule_type", Kind: FieldString, Usage: "type of the rule, e.g. METADATA or USER_COMMS_PER_DAY"},
	{
		Flag: "value", Field: "values", Kind: FieldStringSlice,
		Usage: "value to compare against, repeat for several",
	},
	{Flag: "conditional", Field: "conditional", Kind: FieldString, Usage: "comparison operator of the rule"},
	{
		Flag: "metadata-key", Field: "metadata_key", Kind: FieldString,
		Usage: "metadata field to read, required when --rule-type is METADATA",
	},
}

// campaignRuleStatusFields is the one-field body `campaign rule status` takes.
var campaignRuleStatusFields = []FieldFlag{
	{Flag: "status", Field: "status", Kind: FieldString, Usage: "target status of the rule"},
}

// newCampaignCommand builds the `campaign` command tree.
func newCampaignCommand() *cobra.Command {
	campaign := &cobra.Command{
		Use:     "campaign",
		Short:   "Manage communications campaigns and their rules",
		Aliases: []string{"campaigns"},
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return cmd.Help()
		},
	}

	campaign.AddCommand(
		newCampaignCreateCommand(),
		newCampaignListCommand(),
		newCampaignGetCommand(),
		newCampaignUpdateCommand(),
		newCampaignRuleCommand(),
	)

	return campaign
}

func newCampaignCreateCommand() *cobra.Command {
	command := &cobra.Command{
		Annotations: apiOperations("POST /campaigns"),
		Use:         "create",
		Short:       "Create a communications campaign",
		Long: "Create a communications campaign.\n\n" +
			"A campaign reaches nobody until it has rules: add them with `campaign rule create`.",
		Example: "  yuno-cli campaign create --name 'Recovery CO' --account-id acc-1 --country CO \\\n" +
			"    --channel PHONE_CALL --daily-start-time 09:00 --daily-end-time 18:00 \\\n" +
			"    --time-zone America/Bogota --start-at 2026-10-01 --end-at 2026-12-31",
		Args: cobra.NoArgs,
		RunE: runCampaignCreate,
	}

	registerWriteFlags(command, campaignCreateFields)

	return command
}

func runCampaignCreate(cmd *cobra.Command, _ []string) error {
	body, err := requireBody(cmd, campaignCreateFields)
	if err != nil {
		return err
	}

	client, err := newClientFromFlags(cmd)
	if err != nil {
		return err
	}

	campaign, err := client.CreateCampaign(cmd.Context(), body)
	if err != nil {
		return scopeHint(err, campaignScopes)
	}

	return printResult(cmd, campaign, []model.CampaignView{campaign.View()})
}

func newCampaignListCommand() *cobra.Command {
	command := &cobra.Command{
		Annotations: apiOperations("GET /campaigns"),
		Use:         "list",
		Short:       "List the communications campaigns",
		Example:     "  yuno-cli campaign list --start-date 2026-09-01 --limit 20",
		Args:        cobra.NoArgs,
		RunE:        runCampaignList,
	}

	flags := command.Flags()
	flags.String("start-date", "", "only list campaigns starting at or after this date")
	flags.String("end-date", "", "only list campaigns ending at or before this date")
	registerPagingFlags(command)

	return command
}

func runCampaignList(cmd *cobra.Command, _ []string) error {
	client, err := newClientFromFlags(cmd)
	if err != nil {
		return err
	}

	query := url.Values{}

	for flag, param := range map[string]string{"start-date": "start_date", "end-date": "end_date"} {
		if value := flagString(cmd, flag); value != "" {
			query.Set(param, value)
		}
	}

	limit, pageSize := pagingFlags(cmd)

	campaigns, err := client.ListCampaigns(cmd.Context(), query, limit, pageSize)
	if err != nil {
		return scopeHint(err, campaignScopes)
	}

	return printResult(cmd, campaigns, model.CampaignViews(campaigns))
}

func newCampaignGetCommand() *cobra.Command {
	return &cobra.Command{
		Annotations: apiOperations("GET /campaigns/{campaign_id}"),
		Use:         "get <campaign_id>",
		Short:       "Retrieve one campaign",
		Long: "Retrieve one campaign.\n\n" +
			"The table shows the campaign itself; its rules come along under `rules`, so use\n" +
			"--json to read them.",
		Example: "  yuno-cli campaign get 4f0d1f6e-2f8a-4c71-9d2e-1b1f0a6c7e55 --json",
		Args:    cobra.ExactArgs(1),
		RunE:    runCampaignGet,
	}
}

func runCampaignGet(cmd *cobra.Command, args []string) error {
	client, err := newClientFromFlags(cmd)
	if err != nil {
		return err
	}

	campaign, err := client.GetCampaign(cmd.Context(), args[0])
	if err != nil {
		return scopeHint(err, campaignScopes)
	}

	return printResult(cmd, campaign, []model.CampaignView{campaign.View()})
}

func newCampaignUpdateCommand() *cobra.Command {
	command := &cobra.Command{
		Annotations: apiOperations("PATCH /campaigns/{campaign_id}"),
		Use:         "update <campaign_id>",
		Short:       "Change the status of a campaign",
		Example:     "  yuno-cli campaign update 4f0d1f6e-2f8a-4c71-9d2e-1b1f0a6c7e55 --status ACTIVE",
		Args:        cobra.ExactArgs(1),
		RunE:        runCampaignUpdate,
	}

	registerWriteFlags(command, campaignStatusFields)

	return command
}

func runCampaignUpdate(cmd *cobra.Command, args []string) error {
	body, err := requireBody(cmd, campaignStatusFields)
	if err != nil {
		return err
	}

	client, err := newClientFromFlags(cmd)
	if err != nil {
		return err
	}

	campaign, err := client.UpdateCampaign(cmd.Context(), args[0], body)
	if err != nil {
		return scopeHint(err, campaignScopes)
	}

	return printResult(cmd, campaign, []model.CampaignView{campaign.View()})
}

// newCampaignRuleCommand builds the `campaign rule` subtree.
func newCampaignRuleCommand() *cobra.Command {
	rule := &cobra.Command{
		Use:     "rule",
		Short:   "Manage the targeting rules of a campaign",
		Aliases: []string{"rules"},
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return cmd.Help()
		},
	}

	rule.AddCommand(
		newCampaignRuleCreateCommand(),
		newCampaignRuleGetCommand(),
		newCampaignRuleUpdateCommand(),
		newCampaignRuleStatusCommand(),
	)

	return rule
}

func newCampaignRuleCreateCommand() *cobra.Command {
	command := &cobra.Command{
		Annotations: apiOperations("POST /campaigns/{campaign_id}/rules"),
		Use:         "create <campaign_id>",
		Short:       "Add targeting rules to a campaign",
		Long: "Add targeting rules to a campaign.\n\n" +
			"The endpoint takes a `rules` array. The field flags describe one rule and are\n" +
			"wrapped into that array; send several at once with --file.",
		Example: "  yuno-cli campaign rule create 4f0d1f6e-2f8a-4c71-9d2e-1b1f0a6c7e55 \\\n" +
			"    --rule-type PAYMENT_METHOD --conditional IN --value CARD --value PAYPAL",
		Args: cobra.ExactArgs(1),
		RunE: runCampaignRuleCreate,
	}

	registerWriteFlags(command, campaignRuleFields)

	return command
}

func runCampaignRuleCreate(cmd *cobra.Command, args []string) error {
	body, err := campaignRulesBody(cmd)
	if err != nil {
		return err
	}

	client, err := newClientFromFlags(cmd)
	if err != nil {
		return err
	}

	rules, err := client.CreateCampaignRules(cmd.Context(), args[0], body)
	if err != nil {
		return scopeHint(err, campaignScopes)
	}

	return printResult(cmd, rules, model.CampaignRuleViews(rules))
}

// campaignRulesBody builds the `rules` array the create endpoint takes. A rule
// assembled from the field flags is wrapped, so the common case of adding one
// rule needs no --file; a body that already carries `rules` is left alone.
func campaignRulesBody(cmd *cobra.Command) (any, error) {
	body, err := requireBody(cmd, campaignRuleFields)
	if err != nil {
		return nil, err
	}

	rule, ok := body.(map[string]any)
	if !ok {
		// A verbatim --file or --data payload is forwarded as it was written.
		return body, nil
	}

	if _, wrapped := rule["rules"]; wrapped {
		return rule, nil
	}

	return map[string]any{"rules": []any{rule}}, nil
}

func newCampaignRuleGetCommand() *cobra.Command {
	return &cobra.Command{
		Annotations: apiOperations("GET /campaigns/{campaign_id}/rules/{rule_id}"),
		Use:         "get <campaign_id> <rule_id>",
		Short:       "Retrieve one rule of a campaign",
		Example:     "  yuno-cli campaign rule get 4f0d1f6e-2f8a-4c71-9d2e-1b1f0a6c7e55 rule-1 --json",
		Args:        cobra.ExactArgs(2),
		RunE:        runCampaignRuleGet,
	}
}

func runCampaignRuleGet(cmd *cobra.Command, args []string) error {
	client, err := newClientFromFlags(cmd)
	if err != nil {
		return err
	}

	rule, err := client.GetCampaignRule(cmd.Context(), args[0], args[1])
	if err != nil {
		return scopeHint(err, campaignScopes)
	}

	return printResult(cmd, rule, []model.CampaignRuleView{rule.View()})
}

func newCampaignRuleUpdateCommand() *cobra.Command {
	command := &cobra.Command{
		Annotations: apiOperations("PATCH /campaigns/{campaign_id}/rules/{rule_id}"),
		Use:         "update <campaign_id> <rule_id>",
		Short:       "Change the definition of one rule of a campaign",
		Example:     "  yuno-cli campaign rule update 4f0d1f6e-2f8a-4c71-9d2e-1b1f0a6c7e55 rule-1 --value CARD",
		Args:        cobra.ExactArgs(2),
		RunE:        runCampaignRuleUpdate,
	}

	registerWriteFlags(command, campaignRuleFields)

	return command
}

func runCampaignRuleUpdate(cmd *cobra.Command, args []string) error {
	body, err := requireBody(cmd, campaignRuleFields)
	if err != nil {
		return err
	}

	client, err := newClientFromFlags(cmd)
	if err != nil {
		return err
	}

	rule, err := client.UpdateCampaignRule(cmd.Context(), args[0], args[1], body)
	if err != nil {
		return scopeHint(err, campaignScopes)
	}

	return printResult(cmd, rule, []model.CampaignRuleView{rule.View()})
}

func newCampaignRuleStatusCommand() *cobra.Command {
	command := &cobra.Command{
		Annotations: apiOperations("PATCH /campaigns/{campaign_id}/rules/{rule_id}/status"),
		Use:         "status <campaign_id> <rule_id>",
		Short:       "Enable or disable one rule of a campaign",
		Long: "Enable or disable one rule of a campaign.\n\n" +
			"This leaves the definition of the rule alone: only its status changes.",
		Example: "  yuno-cli campaign rule status 4f0d1f6e-2f8a-4c71-9d2e-1b1f0a6c7e55 rule-1 --status INACTIVE",
		Args:    cobra.ExactArgs(2),
		RunE:    runCampaignRuleStatus,
	}

	registerWriteFlags(command, campaignRuleStatusFields)

	return command
}

func runCampaignRuleStatus(cmd *cobra.Command, args []string) error {
	body, err := requireBody(cmd, campaignRuleStatusFields)
	if err != nil {
		return err
	}

	client, err := newClientFromFlags(cmd)
	if err != nil {
		return err
	}

	rule, err := client.UpdateCampaignRuleStatus(cmd.Context(), args[0], args[1], body)
	if err != nil {
		return scopeHint(err, campaignScopes)
	}

	return printResult(cmd, rule, []model.CampaignRuleView{rule.View()})
}
