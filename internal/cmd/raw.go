package cmd

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/spf13/cobra"

	"github.com/pvarganov/yuno-cli/internal/api"
)

// rawMethods are the HTTP methods `raw` accepts, so that a typo cannot reach
// the API as an unexpected verb.
var rawMethods = map[string]struct{}{
	http.MethodGet:     {},
	http.MethodPost:    {},
	http.MethodPut:     {},
	http.MethodPatch:   {},
	http.MethodDelete:  {},
	http.MethodHead:    {},
	http.MethodOptions: {},
}

// newRawCommand builds `raw`, the escape hatch that calls any Yuno endpoint,
// including the ones without a dedicated subcommand yet.
func newRawCommand() *cobra.Command {
	rawCmd := &cobra.Command{
		Use:   "raw <method> <path>",
		Short: "Call any Yuno API endpoint directly",
		Long: "Send an arbitrary request to the Yuno API using the credentials of the " +
			"selected profile. The path is relative to the API base URL, e.g. `/routing`.",
		Example: strings.Join([]string{
			"  yuno-cli raw GET /routing --query account_id=<uuid>",
			"  yuno-cli raw POST /customers --file body.json",
			`  echo '{"email":"a@b.c"}' | yuno-cli raw POST /customers --data @- --yes`,
		}, "\n"),
		Args: cobra.ExactArgs(2),
		RunE: runRaw,
	}

	flags := rawCmd.Flags()
	flags.StringArray("query", nil, "query parameter as key=value (repeatable)")
	flags.StringArray("header", nil, "extra request header as 'Name: value' (repeatable)")
	registerBodyFlags(flags)
	flags.String("idempotency-key", "", "pin the X-Idempotency-Key of the request")

	return rawCmd
}

// runRaw builds one API request from the arguments and prints the response.
func runRaw(cmd *cobra.Command, args []string) error {
	method, err := rawMethod(args[0])
	if err != nil {
		return err
	}

	query, err := parseQuery(flagStringArray(cmd, "query"))
	if err != nil {
		return err
	}

	headers, err := parseHeaders(flagStringArray(cmd, "header"))
	if err != nil {
		return err
	}

	body, err := rawBody(cmd, method)
	if err != nil {
		return err
	}

	client, err := newClientFromFlags(cmd)
	if err != nil {
		return err
	}

	data, err := client.DoRaw(cmd.Context(), api.Request{
		Method:  method,
		Path:    args[1],
		Query:   query,
		Body:    body,
		Headers: headers,
	})
	if err != nil {
		return err
	}

	return printJSON(cmd, data)
}

// rawMethod validates and normalises the method argument.
func rawMethod(arg string) (string, error) {
	method := strings.ToUpper(strings.TrimSpace(arg))
	if _, ok := rawMethods[method]; !ok {
		return "", fmt.Errorf("unsupported method %q: want one of GET, POST, PUT, PATCH, DELETE, HEAD, OPTIONS", arg)
	}

	return method, nil
}

// parseQuery turns repeated key=value flags into query parameters.
func parseQuery(pairs []string) (url.Values, error) {
	if len(pairs) == 0 {
		return nil, nil
	}

	values := url.Values{}

	for _, pair := range pairs {
		key, value, ok := strings.Cut(pair, "=")
		if !ok || strings.TrimSpace(key) == "" {
			return nil, fmt.Errorf("invalid query parameter %q: want key=value", pair)
		}

		values.Add(strings.TrimSpace(key), value)
	}

	return values, nil
}

// parseHeaders turns repeated "Name: value" flags into request headers.
func parseHeaders(pairs []string) (map[string]string, error) {
	if len(pairs) == 0 {
		return nil, nil
	}

	headers := make(map[string]string, len(pairs))

	for _, pair := range pairs {
		name, value, ok := strings.Cut(pair, ":")
		if !ok || strings.TrimSpace(name) == "" {
			return nil, fmt.Errorf("invalid header %q: want 'Name: value'", pair)
		}

		headers[strings.TrimSpace(name)] = strings.TrimSpace(value)
	}

	return headers, nil
}

// rawBody reads the request body from --file, --data or stdin and checks that
// it is valid JSON before anything is sent. Methods that carry no body are
// allowed to have none; the ones that do must be given one.
func rawBody(cmd *cobra.Command, method string) (any, error) {
	payload, err := readBodySource(cmd)
	if err != nil {
		return nil, err
	}

	if len(payload) == 0 {
		if methodNeedsBody(method) {
			return nil, fmt.Errorf("%s: request body is required (use --file or --data)", method)
		}

		return nil, nil
	}

	var decoded any
	if err := json.Unmarshal(payload, &decoded); err != nil {
		return nil, fmt.Errorf("request body is not valid json: %w", err)
	}

	return json.RawMessage(payload), nil
}
