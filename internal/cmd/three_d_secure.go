package cmd

import (
	"github.com/spf13/cobra"

	"github.com/pvarganov/yuno-cli/internal/model"
)

// threeDSecureScopes are the scopes a 403 on `three-d-secure` usually asks for.
const threeDSecureScopes = "payments:write"

// threeDSecureFields are the named body fields of `three-d-secure setup`.
var threeDSecureFields = []FieldFlag{
	{Flag: "account-id", Field: "account_id", Kind: FieldString, Usage: "account the setup belongs to"},
	{Flag: "type", Field: "type", Kind: FieldString, Usage: "setup type, e.g. BROWSER"},
	{
		Flag: "user-agent", Field: "browser_info.user_agent", Kind: FieldString,
		Usage: "user agent of the shopper browser",
	},
	{
		Flag: "accept-header", Field: "browser_info.accept_header", Kind: FieldString,
		Usage: "accept header of the shopper browser",
	},
	{Flag: "color-depth", Field: "browser_info.color_depth", Kind: FieldString, Usage: "browser color depth"},
	{Flag: "screen-height", Field: "browser_info.screen_height", Kind: FieldString, Usage: "browser screen height"},
	{Flag: "screen-width", Field: "browser_info.screen_width", Kind: FieldString, Usage: "browser screen width"},
	{Flag: "language", Field: "browser_info.language", Kind: FieldString, Usage: "browser language, e.g. en-US"},
	{
		Flag: "javascript-enabled", Field: "browser_info.javascript_enabled", Kind: FieldBool,
		Usage: "whether the browser runs javascript",
	},
	{
		Flag: "java-enabled", Field: "browser_info.java_enabled", Kind: FieldBool,
		Usage: "whether the browser runs java",
	},
	{
		Flag: "time-difference", Field: "browser_info.browser_time_difference", Kind: FieldString,
		Usage: "browser time zone offset in minutes",
	},
	{Flag: "platform", Field: "browser_info.platform", Kind: FieldString, Usage: "platform of the shopper browser"},
	{
		Flag: "browser-info", Field: "browser_info", Kind: FieldJSON,
		Usage: "whole browser info as a JSON object",
	},
	{
		Flag: "device-fingerprints", Field: "device_fingerprints", Kind: FieldJSON,
		Usage: `device fingerprints as a JSON array, e.g. '[{"provider_id":"CYBERSOURCE","id":"fp-1"}]'`,
	},
}

// newThreeDSecureCommand builds the `three-d-secure` command tree.
func newThreeDSecureCommand() *cobra.Command {
	secure := &cobra.Command{
		Use:     "three-d-secure",
		Short:   "Create 3-D Secure setups",
		Aliases: []string{"3ds"},
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return cmd.Help()
		},
	}

	secure.AddCommand(newThreeDSecureSetupCommand())

	return secure
}

func newThreeDSecureSetupCommand() *cobra.Command {
	command := &cobra.Command{
		Use:   "setup",
		Short: "Create a 3-D Secure setup and collect the device fingerprints",
		Example: "  yuno-cli three-d-secure setup --account-id acc-1 --type BROWSER " +
			"--user-agent Mozilla/5.0 --accept-header '*/*' --language en-US " +
			"--screen-width 1920 --screen-height 1080 --color-depth 24",
		Args: cobra.NoArgs,
		RunE: runThreeDSecureSetup,
	}

	registerWriteFlags(command, threeDSecureFields)

	return command
}

func runThreeDSecureSetup(cmd *cobra.Command, _ []string) error {
	body, err := requireBody(cmd, threeDSecureFields)
	if err != nil {
		return err
	}

	client, err := newClientFromFlags(cmd)
	if err != nil {
		return err
	}

	setup, err := client.CreateThreeDSecureSetup(cmd.Context(), body)
	if err != nil {
		return scopeHint(err, threeDSecureScopes)
	}

	return printResult(cmd, setup, []model.ThreeDSecureSetupView{setup.View()})
}
