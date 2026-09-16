package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/pvarganov/yuno-cli/internal/api"
	"github.com/pvarganov/yuno-cli/internal/config"
	"github.com/pvarganov/yuno-cli/internal/confirm"
	"github.com/pvarganov/yuno-cli/internal/output"
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

// stdinMarker is the --data value that means "read the body from stdin".
const stdinMarker = "@-"

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
	flags.String("file", "", "read the JSON request body from this file")
	flags.String("data", "", "JSON request body, or @- to read it from stdin")
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

	return printRaw(cmd, data)
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
	payload, err := readRawBody(cmd)
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

// readRawBody resolves the body source; --file wins over --data.
func readRawBody(cmd *cobra.Command) ([]byte, error) {
	if path := flagString(cmd, "file"); path != "" {
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("read body file %s: %w", path, err)
		}

		return data, nil
	}

	data := strings.TrimSpace(flagString(cmd, "data"))

	switch {
	case data == "":
		return nil, nil
	case data == stdinMarker:
		payload, err := io.ReadAll(cmd.InOrStdin())
		if err != nil {
			return nil, fmt.Errorf("read body from stdin: %w", err)
		}

		return payload, nil
	case strings.HasPrefix(data, "@"):
		path := strings.TrimPrefix(data, "@")

		payload, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("read body file %s: %w", path, err)
		}

		return payload, nil
	default:
		return []byte(data), nil
	}
}

// methodNeedsBody reports whether Yuno expects a payload for this method.
func methodNeedsBody(method string) bool {
	switch method {
	case http.MethodPost, http.MethodPut, http.MethodPatch:
		return true
	default:
		return false
	}
}

// printRaw writes the response through the formatter. An arbitrary endpoint has
// no table shape, so the raw output is always JSON; an empty body prints nothing.
func printRaw(cmd *cobra.Command, data []byte) error {
	if strings.TrimSpace(string(data)) == "" {
		return nil
	}

	var decoded any
	if err := json.Unmarshal(data, &decoded); err != nil {
		if _, err := fmt.Fprintln(cmd.OutOrStdout(), string(data)); err != nil {
			return fmt.Errorf("print response: %w", err)
		}

		return nil
	}

	if err := output.NewFormatter(true).Format(cmd.OutOrStdout(), decoded); err != nil {
		return fmt.Errorf("print response: %w", err)
	}

	return nil
}

// newClientFromFlags resolves the profile from the flags and the environment
// and builds a client wired to the confirmation gate and the verbose dump.
func newClientFromFlags(cmd *cobra.Command) (*api.Client, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, err
	}

	profile, err := cfg.Profile(flagString(cmd, "profile"))
	if err != nil {
		return nil, err
	}

	opts := []api.Option{
		api.WithConfirmer(confirm.New(cmd.ErrOrStderr(), cmd.InOrStdin(), flagBool(cmd, "yes"))),
		api.WithIdempotencyKey(flagString(cmd, "idempotency-key")),
	}

	if flagBool(cmd, "verbose") {
		opts = append(opts, api.WithVerbose(cmd.ErrOrStderr()))
	}

	client, err := api.NewClient(&profile, opts...)
	if err != nil {
		return nil, err
	}

	return client, nil
}

// flagStringArray reads a repeatable string flag, treating a missing flag as empty.
func flagStringArray(cmd *cobra.Command, name string) []string {
	value, err := cmd.Flags().GetStringArray(name)
	if err != nil {
		return nil
	}

	return value
}
