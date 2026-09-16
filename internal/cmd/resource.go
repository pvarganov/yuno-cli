package cmd

import (
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strings"

	"github.com/spf13/cobra"

	"github.com/pvarganov/yuno-cli/internal/api"
)

// pathParamPattern matches the `{payment_id}` placeholders of the OpenAPI paths.
var pathParamPattern = regexp.MustCompile(`\{([a-zA-Z_][a-zA-Z0-9_]*)\}`)

// QueryFlag maps one cobra flag onto one query parameter.
type QueryFlag struct {
	Flag     string
	Param    string
	Kind     FieldKind
	Usage    string
	Required bool
}

// Operation describes one Yuno API operation as a CLI subcommand.
type Operation struct {
	Name    string
	Short   string
	Long    string
	Example string
	Method  string
	// Path is the spec path, relative to the API base URL, with `{param}`
	// placeholders that become positional arguments in declaration order.
	Path   string
	Query  []QueryFlag
	Fields []FieldFlag
	// Body registers --file/--data so a full payload can be sent as is.
	Body bool
}

// Resource groups the operations of one Yuno resource under a parent command.
type Resource struct {
	Use     string
	Short   string
	Long    string
	Aliases []string
	Ops     []Operation
}

// newResourceCommand turns a resource description into a cobra command tree.
func newResourceCommand(res *Resource) *cobra.Command {
	parent := &cobra.Command{
		Use:     res.Use,
		Short:   res.Short,
		Long:    res.Long,
		Aliases: res.Aliases,
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return cmd.Help()
		},
	}

	for i := range res.Ops {
		parent.AddCommand(newOperationCommand(&res.Ops[i]))
	}

	return parent
}

// newOperationCommand builds the subcommand for one operation.
func newOperationCommand(op *Operation) *cobra.Command {
	params := pathParams(op.Path)

	command := &cobra.Command{
		Use:     operationUse(op.Name, params),
		Short:   op.Short,
		Long:    op.Long,
		Example: op.Example,
		Args:    cobra.ExactArgs(len(params)),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runOperation(cmd, op, params, args)
		},
	}

	flags := command.Flags()

	for i := range op.Query {
		q := &op.Query[i]

		switch q.Kind {
		case FieldInt:
			flags.Int(q.Flag, 0, q.Usage)
		case FieldBool:
			flags.Bool(q.Flag, false, q.Usage)
		case FieldStringSlice:
			flags.StringArray(q.Flag, nil, q.Usage)
		case FieldString, FieldNumber, FieldJSON:
			flags.String(q.Flag, "", q.Usage)
		}

		if q.Required {
			_ = command.MarkFlagRequired(q.Flag)
		}
	}

	registerFieldFlags(flags, op.Fields)

	for i := range op.Fields {
		if f := &op.Fields[i]; f.Required {
			_ = command.MarkFlagRequired(f.Flag)
		}
	}

	if op.Body {
		registerBodyFlags(flags)
		flags.String("idempotency-key", "", "pin the X-Idempotency-Key of the request")
	}

	return command
}

// pathParams lists the placeholders of a spec path in declaration order.
func pathParams(path string) []string {
	matches := pathParamPattern.FindAllStringSubmatch(path, -1)

	params := make([]string, 0, len(matches))
	for _, m := range matches {
		params = append(params, m[1])
	}

	return params
}

// operationUse renders the usage line, e.g. `get <payment_id>`.
func operationUse(name string, params []string) string {
	parts := make([]string, 0, len(params)+1)
	parts = append(parts, name)

	for _, p := range params {
		parts = append(parts, "<"+p+">")
	}

	return strings.Join(parts, " ")
}

// runOperation builds the request described by the operation and prints the
// response.
func runOperation(cmd *cobra.Command, op *Operation, params, args []string) error {
	path, err := resolvePath(op.Path, params, args)
	if err != nil {
		return err
	}

	query, err := queryFromFlags(cmd, op.Query)
	if err != nil {
		return err
	}

	body, err := operationBody(cmd, op)
	if err != nil {
		return err
	}

	client, err := newClientFromFlags(cmd)
	if err != nil {
		return err
	}

	data, err := client.DoRaw(cmd.Context(), api.Request{
		Method: op.Method,
		Path:   path,
		Query:  query,
		Body:   body,
	})
	if err != nil {
		return fmt.Errorf("%s %s: %w", strings.ToLower(op.Method), op.Path, err)
	}

	return printJSON(cmd, data)
}

// operationBody builds the payload and refuses a mutating call without one.
func operationBody(cmd *cobra.Command, op *Operation) (any, error) {
	if !op.Body && len(op.Fields) == 0 {
		return nil, nil
	}

	body, err := bodyFromFlags(cmd, op.Fields)
	if err != nil {
		return nil, err
	}

	if body == nil && methodNeedsBody(op.Method) {
		return nil, fmt.Errorf("%s: request body is required (use --file or --data)", op.Method)
	}

	return body, nil
}

// resolvePath substitutes the positional arguments into the spec path.
func resolvePath(path string, params, args []string) (string, error) {
	if len(params) != len(args) {
		return "", fmt.Errorf("path %s: want %d arguments, got %d", path, len(params), len(args))
	}

	resolved := path

	for i, param := range params {
		value := strings.TrimSpace(args[i])
		if value == "" {
			return "", fmt.Errorf("%s: must not be empty", param)
		}

		resolved = strings.Replace(resolved, "{"+param+"}", url.PathEscape(value), 1)
	}

	return resolved, nil
}

// queryFromFlags collects the query parameters the user actually set.
func queryFromFlags(cmd *cobra.Command, params []QueryFlag) (url.Values, error) {
	var values url.Values

	flags := cmd.Flags()

	for i := range params {
		q := &params[i]

		if !flags.Changed(q.Flag) {
			continue
		}

		if values == nil {
			values = url.Values{}
		}

		if err := addQueryValue(cmd, values, q); err != nil {
			return nil, err
		}
	}

	return values, nil
}

// addQueryValue renders one flag into the query string.
func addQueryValue(cmd *cobra.Command, values url.Values, q *QueryFlag) error {
	switch q.Kind {
	case FieldInt:
		values.Set(q.Param, flagIntString(cmd, q.Flag))
	case FieldBool:
		values.Set(q.Param, flagBoolString(cmd, q.Flag))
	case FieldStringSlice:
		for _, v := range flagStringArray(cmd, q.Flag) {
			values.Add(q.Param, v)
		}
	case FieldString, FieldNumber, FieldJSON:
		values.Set(q.Param, flagString(cmd, q.Flag))
	default:
		return fmt.Errorf("--%s: unsupported query kind", q.Flag)
	}

	return nil
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
