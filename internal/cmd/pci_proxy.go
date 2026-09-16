package cmd

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/pvarganov/yuno-cli/internal/api"
	"github.com/pvarganov/yuno-cli/internal/model"
)

// pciProxyScopes are the scopes a 403 on the PCI proxy commands usually asks for.
const pciProxyScopes = "pci-proxy:read/pci-proxy:write"

// pciProxyDestinationFields are the named body fields of `pci-proxy destination create`.
var pciProxyDestinationFields = []FieldFlag{
	{
		Flag: "hostname", Field: "hostname", Kind: FieldString,
		Usage: "public HTTPS hostname to allow, without scheme or path",
	},
	{Flag: "purpose", Field: "purpose", Kind: FieldString, Usage: "label describing what the destination is for"},
	{
		Flag: "account-id", Field: "account_id", Kind: FieldString,
		Usage: "scope the destination to one account instead of the whole organization",
	},
}

// forwardMethods are the methods the forward proxy passes through verbatim.
var forwardMethods = map[string]struct{}{
	http.MethodGet:    {},
	http.MethodPost:   {},
	http.MethodPut:    {},
	http.MethodPatch:  {},
	http.MethodDelete: {},
}

// newPCIProxyCommand builds the `pci-proxy` command tree.
func newPCIProxyCommand() *cobra.Command {
	proxy := &cobra.Command{
		Use:     "pci-proxy",
		Short:   "Manage the PCI forward proxy and its destination allowlist",
		Aliases: []string{"proxy"},
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return cmd.Help()
		},
	}

	proxy.AddCommand(
		newPCIProxyDestinationCommand(),
		newPCIProxyForwardCommand(),
	)

	return proxy
}

// newPCIProxyDestinationCommand builds the `pci-proxy destination` subtree.
func newPCIProxyDestinationCommand() *cobra.Command {
	destination := &cobra.Command{
		Use:     "destination",
		Short:   "Manage the forward proxy destination allowlist",
		Aliases: []string{"destinations"},
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return cmd.Help()
		},
	}

	destination.AddCommand(
		newPCIProxyDestinationListCommand(),
		newPCIProxyDestinationCreateCommand(),
		newPCIProxyDestinationSwitchCommand("enable"),
		newPCIProxyDestinationSwitchCommand("disable"),
		newPCIProxyDestinationDeleteCommand(),
	)

	return destination
}

func newPCIProxyDestinationListCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List the allowlisted destinations",
		Long: "List the allowlisted destinations.\n\n" +
			"The allowlist is not paginated: Yuno answers with the whole list at once.",
		Example: "  yuno-cli pci-proxy destination list",
		Aliases: []string{"ls"},
		Args:    cobra.NoArgs,
		RunE:    runPCIProxyDestinationList,
	}
}

func runPCIProxyDestinationList(cmd *cobra.Command, _ []string) error {
	client, err := newClientFromFlags(cmd)
	if err != nil {
		return err
	}

	destinations, err := client.ListPCIProxyDestinations(cmd.Context())
	if err != nil {
		return scopeHint(err, pciProxyScopes)
	}

	return printResult(cmd, destinations, model.PCIProxyDestinationViews(destinations))
}

func newPCIProxyDestinationCreateCommand() *cobra.Command {
	command := &cobra.Command{
		Use:   "create",
		Short: "Register a destination on the allowlist",
		Long: "Register a destination on the allowlist.\n\n" +
			"Only registered hostnames can be reached through `pci-proxy forward`; IP addresses\n" +
			"and internal networks are rejected.",
		Example: "  yuno-cli pci-proxy destination create --hostname api.processor.com --purpose 'card charges'",
		Args:    cobra.NoArgs,
		RunE:    runPCIProxyDestinationCreate,
	}

	registerWriteFlags(command, pciProxyDestinationFields)

	return command
}

func runPCIProxyDestinationCreate(cmd *cobra.Command, _ []string) error {
	body, err := requireBody(cmd, pciProxyDestinationFields)
	if err != nil {
		return err
	}

	client, err := newClientFromFlags(cmd)
	if err != nil {
		return err
	}

	destination, err := client.CreatePCIProxyDestination(cmd.Context(), body)
	if err != nil {
		return scopeHint(err, pciProxyScopes)
	}

	return printResult(cmd, destination, []model.PCIProxyDestinationView{destination.View()})
}

// newPCIProxyDestinationSwitchCommand builds `enable` or `disable`, which differ
// only in the path segment they post to.
func newPCIProxyDestinationSwitchCommand(action string) *cobra.Command {
	return &cobra.Command{
		Use:     action + " <id>",
		Short:   strings.ToUpper(action[:1]) + action[1:] + " a destination",
		Example: "  yuno-cli pci-proxy destination " + action + " d7e8f9a0-1234-4b5c-8d6e-7f8091a2b3c4",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runPCIProxyDestinationSwitch(cmd, action, args[0])
		},
	}
}

func runPCIProxyDestinationSwitch(cmd *cobra.Command, action, id string) error {
	client, err := newClientFromFlags(cmd)
	if err != nil {
		return err
	}

	switchFn := client.EnablePCIProxyDestination
	if action == "disable" {
		switchFn = client.DisablePCIProxyDestination
	}

	destination, err := switchFn(cmd.Context(), id)
	if err != nil {
		return scopeHint(err, pciProxyScopes)
	}

	return printResult(cmd, destination, []model.PCIProxyDestinationView{destination.View()})
}

func newPCIProxyDestinationDeleteCommand() *cobra.Command {
	return &cobra.Command{
		Use:     "delete <id>",
		Short:   "Remove a destination from the allowlist",
		Aliases: []string{"remove"},
		Example: "  yuno-cli pci-proxy destination delete d7e8f9a0-1234-4b5c-8d6e-7f8091a2b3c4",
		Args:    cobra.ExactArgs(1),
		RunE:    runPCIProxyDestinationDelete,
	}
}

func runPCIProxyDestinationDelete(cmd *cobra.Command, args []string) error {
	client, err := newClientFromFlags(cmd)
	if err != nil {
		return err
	}

	data, err := client.DeletePCIProxyDestination(cmd.Context(), args[0])
	if err != nil {
		return scopeHint(err, pciProxyScopes)
	}

	return printJSON(cmd, data)
}

func newPCIProxyForwardCommand() *cobra.Command {
	command := &cobra.Command{
		Use:   "forward",
		Short: "Forward a request to a destination with card data injected",
		Long: "Forward a request to a destination with card data injected.\n\n" +
			"`{{vaulted_token.<TOKEN>.<field>}}` expressions in the body and in the extra\n" +
			"headers are replaced with real card data inside Yuno's PCI environment. The\n" +
			"destination's status code, headers and body come back unchanged, so the response\n" +
			"is printed verbatim.",
		Example: "  yuno-cli pci-proxy forward --destination-url https://api.processor.com/charges " +
			"--file charge.json\n" +
			"  yuno-cli pci-proxy forward --method GET --destination-url https://api.processor.com/charges/ch_1",
		Args: cobra.NoArgs,
		RunE: runPCIProxyForward,
	}

	flags := command.Flags()
	flags.String("destination-url", "", "full destination URL, including its path and query string")
	flags.String("method", http.MethodPost, "HTTP method to forward: GET, POST, PUT, PATCH or DELETE")
	flags.Int("timeout", 0, "destination timeout in seconds, 1 to 120")
	flags.String("proxy-auth", "", "signing scheme for the destination, e.g. DLOCAL_HMAC")
	flags.String("proxy-auth-secret-key", "", "signing secret of the --proxy-auth scheme")
	flags.String("account-id", "", "narrow the allowlist this request may reach to one account")
	flags.StringArray("header", nil, "extra destination header as 'Name: value' (repeatable)")
	registerBodyFlags(flags)
	flags.String("idempotency-key", "", "pin the X-Idempotency-Key of the request")

	_ = command.MarkFlagRequired("destination-url")

	return command
}

func runPCIProxyForward(cmd *cobra.Command, _ []string) error {
	method, err := forwardMethod(flagString(cmd, "method"))
	if err != nil {
		return err
	}

	headers, err := forwardHeaders(cmd)
	if err != nil {
		return err
	}

	body, err := bodyFromFlags(cmd, nil)
	if err != nil {
		return err
	}

	if body == nil && methodNeedsBody(method) {
		return fmt.Errorf("%s: request body is required (use --file or --data)", method)
	}

	client, err := newClientFromFlags(cmd)
	if err != nil {
		return err
	}

	data, err := client.ForwardPCIProxy(cmd.Context(), method, body, headers)
	if err != nil {
		return scopeHint(err, pciProxyScopes)
	}

	return printJSON(cmd, data)
}

// forwardMethod validates the method the proxy is asked to pass through.
func forwardMethod(arg string) (string, error) {
	method := strings.ToUpper(strings.TrimSpace(arg))
	if _, ok := forwardMethods[method]; !ok {
		return "", fmt.Errorf("unsupported method %q: want one of GET, POST, PUT, PATCH, DELETE", arg)
	}

	return method, nil
}

// forwardHeaders builds the yuno-proxy-* headers plus the destination headers
// the user passed with --header.
func forwardHeaders(cmd *cobra.Command) (map[string]string, error) {
	headers, err := parseHeaders(flagStringArray(cmd, "header"))
	if err != nil {
		return nil, err
	}

	if headers == nil {
		headers = map[string]string{}
	}

	headers[api.HeaderProxyDestinationURL] = flagString(cmd, "destination-url")

	if timeout := flagInt(cmd, "timeout"); timeout > 0 {
		headers[api.HeaderProxyTimeout] = strconv.Itoa(timeout)
	}

	for header, flag := range map[string]string{
		api.HeaderProxyAuth:          "proxy-auth",
		api.HeaderProxyAuthSecretKey: "proxy-auth-secret-key",
		api.HeaderProxyAccountID:     "account-id",
	} {
		if value := flagString(cmd, flag); value != "" {
			headers[header] = value
		}
	}

	return headers, nil
}
