package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/pvarganov/yuno-cli/internal/api"
	"github.com/pvarganov/yuno-cli/internal/config"
	"github.com/pvarganov/yuno-cli/internal/confirm"
	"github.com/pvarganov/yuno-cli/internal/output"
)

// stdinMarker is the --data value that means "read the body from stdin".
const stdinMarker = "@-"

// Body source flags shared by `raw` and by every generated resource command.
const (
	flagFile = "file"
	flagData = "data"
)

// FieldKind tells the body builder how to read a flag and how to encode it in
// the JSON request body.
type FieldKind int

// Supported body field kinds. Anything more exotic goes through --file.
const (
	FieldString FieldKind = iota
	FieldInt
	FieldBool
	FieldNumber
	FieldStringSlice
	// FieldJSON takes a raw JSON fragment, for nested objects and arrays that
	// deserve a named flag.
	FieldJSON
)

// FieldFlag maps one cobra flag onto one field of the JSON request body. Field
// may be dotted (`customer.email`) to reach a nested object.
type FieldFlag struct {
	Flag     string
	Field    string
	Kind     FieldKind
	Usage    string
	Required bool
}

// registerFieldFlags declares the body flags on a command.
func registerFieldFlags(flags *pflag.FlagSet, fields []FieldFlag) {
	for _, f := range fields {
		switch f.Kind {
		case FieldInt:
			flags.Int(f.Flag, 0, f.Usage)
		case FieldBool:
			flags.Bool(f.Flag, false, f.Usage)
		case FieldNumber:
			flags.Float64(f.Flag, 0, f.Usage)
		case FieldStringSlice:
			flags.StringArray(f.Flag, nil, f.Usage)
		case FieldString, FieldJSON:
			flags.String(f.Flag, "", f.Usage)
		}
	}
}

// registerBodyFlags declares the --file/--data pair used to send a full payload.
func registerBodyFlags(flags *pflag.FlagSet) {
	flags.String(flagFile, "", "read the JSON request body from this file")
	flags.String(flagData, "", "JSON request body, or @- to read it from stdin")
}

// bodyFromFlags builds the request body from --file, --data or stdin and then
// overlays the per-field flags the user actually set. Only changed flags land in
// the body, so a PATCH keeps "absent" and "empty" distinguishable.
func bodyFromFlags(cmd *cobra.Command, fields []FieldFlag) (any, error) {
	payload, err := readBodySource(cmd)
	if err != nil {
		return nil, err
	}

	overlay, err := fieldValues(cmd, fields)
	if err != nil {
		return nil, err
	}

	base, err := decodeBody(payload, len(overlay) > 0)
	if err != nil {
		return nil, err
	}

	if len(overlay) == 0 {
		if base == nil {
			return nil, nil
		}

		return base, nil
	}

	if base == nil {
		base = map[string]any{}
	}

	body, ok := base.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("request body is not a json object: cannot merge field flags into it")
	}

	for _, field := range overlay {
		if err := setField(body, field.path, field.value); err != nil {
			return nil, err
		}
	}

	return body, nil
}

// decodeBody parses the raw payload. A body that only has to be forwarded stays
// verbatim; one that has to be merged with flags is decoded into a map.
func decodeBody(payload []byte, merge bool) (any, error) {
	if len(payload) == 0 {
		return nil, nil
	}

	var decoded any
	if err := json.Unmarshal(payload, &decoded); err != nil {
		return nil, fmt.Errorf("request body is not valid json: %w", err)
	}

	if merge {
		return decoded, nil
	}

	return json.RawMessage(payload), nil
}

// changedField is one body field the user set on the command line.
type changedField struct {
	path  []string
	value any
}

// fieldValues collects the values of every field flag the user changed.
func fieldValues(cmd *cobra.Command, fields []FieldFlag) ([]changedField, error) {
	var changed []changedField

	flags := cmd.Flags()

	for _, f := range fields {
		if !flags.Changed(f.Flag) {
			continue
		}

		value, err := fieldValue(cmd, f)
		if err != nil {
			return nil, err
		}

		path := strings.Split(f.Field, ".")
		changed = append(changed, changedField{path: path, value: value})
	}

	return changed, nil
}

// fieldValue reads one flag according to its declared kind.
func fieldValue(cmd *cobra.Command, f FieldFlag) (any, error) {
	flags := cmd.Flags()

	switch f.Kind {
	case FieldInt:
		return flags.GetInt(f.Flag)
	case FieldBool:
		return flags.GetBool(f.Flag)
	case FieldNumber:
		return flags.GetFloat64(f.Flag)
	case FieldStringSlice:
		values, err := flags.GetStringArray(f.Flag)
		if err != nil {
			return nil, err
		}

		return toAnySlice(values), nil
	case FieldJSON:
		raw := flagString(cmd, f.Flag)

		var decoded any
		if err := json.Unmarshal([]byte(raw), &decoded); err != nil {
			return nil, fmt.Errorf("--%s is not valid json: %w", f.Flag, err)
		}

		return decoded, nil
	case FieldString:
		return flagString(cmd, f.Flag), nil
	default:
		return nil, fmt.Errorf("--%s: unsupported field kind", f.Flag)
	}
}

// toAnySlice widens a string slice so it marshals as a JSON array.
func toAnySlice(values []string) []any {
	out := make([]any, 0, len(values))
	for _, v := range values {
		out = append(out, v)
	}

	return out
}

// setField writes value at a dotted path, creating the intermediate objects.
func setField(body map[string]any, path []string, value any) error {
	current := body

	for i, key := range path {
		if i == len(path)-1 {
			current[key] = value

			return nil
		}

		next, ok := current[key]
		if !ok {
			child := map[string]any{}
			current[key] = child
			current = child

			continue
		}

		child, ok := next.(map[string]any)
		if !ok {
			return fmt.Errorf("field %s: %s is not a json object", strings.Join(path, "."), key)
		}

		current = child
	}

	return nil
}

// readBodySource resolves the payload from --file, --data or stdin. --file wins.
func readBodySource(cmd *cobra.Command) ([]byte, error) {
	if path := flagString(cmd, flagFile); path != "" {
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("read body file %s: %w", path, err)
		}

		return data, nil
	}

	data := strings.TrimSpace(flagString(cmd, flagData))

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
		api.WithTimeout(flagDuration(cmd, "timeout")),
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

// printJSON writes an API response through the formatter. Responses without a
// table shape are always printed as JSON; an empty body prints nothing.
func printJSON(cmd *cobra.Command, data []byte) error {
	if strings.TrimSpace(string(data)) == "" {
		return nil
	}

	var decoded any
	if err := json.Unmarshal(data, &decoded); err != nil {
		return printNonJSON(cmd, data)
	}

	unmask := output.WithUnmask(flagBool(cmd, "unmask"))
	if err := output.NewFormatter(true, unmask).Format(cmd.OutOrStdout(), decoded); err != nil {
		return fmt.Errorf("print response: %w", err)
	}

	return nil
}

// printNonJSON prints a response body that failed to parse as JSON, e.g. from
// `raw` hitting an endpoint that returns plain text or HTML. Because
// internal/mask can only redact fields of a decoded JSON tree, such a body
// cannot be selectively masked, so it is withheld by default like any other
// sensitive output and only printed verbatim with --unmask.
func printNonJSON(cmd *cobra.Command, data []byte) error {
	if !flagBool(cmd, "unmask") {
		if _, err := fmt.Fprintln(cmd.OutOrStdout(), "<non-json response withheld, pass --unmask to print it verbatim>"); err != nil {
			return fmt.Errorf("print response: %w", err)
		}

		return nil
	}

	if _, err := fmt.Fprintln(cmd.OutOrStdout(), string(data)); err != nil {
		return fmt.Errorf("print response: %w", err)
	}

	return nil
}

// printResult renders a typed API result: the full payload as JSON when --json
// is set, an aligned table of the flat view rows otherwise. Either way,
// card-like fields are masked unless --unmask was given.
func printResult(cmd *cobra.Command, payload, views any) error {
	unmask := output.WithUnmask(flagBool(cmd, "unmask"))
	formatter := output.NewFormatter(true, unmask)
	data := payload

	if !flagBool(cmd, "json") {
		formatter = output.NewFormatter(false, unmask)
		data = views
	}

	if err := formatter.Format(cmd.OutOrStdout(), data); err != nil {
		return fmt.Errorf("print response: %w", err)
	}

	return nil
}

// registerWriteFlags declares the body source flags of a mutating command.
func registerWriteFlags(cmd *cobra.Command, fields []FieldFlag) {
	flags := cmd.Flags()

	registerBodyFlags(flags)
	registerFieldFlags(flags, fields)
	flags.String("idempotency-key", "", "pin the X-Idempotency-Key of the request")
}

// requireBody builds the JSON body of a mutating command and refuses an empty
// one, since Yuno rejects a bodyless POST or PATCH anyway.
func requireBody(cmd *cobra.Command, fields []FieldFlag) (any, error) {
	body, err := bodyFromFlags(cmd, fields)
	if err != nil {
		return nil, err
	}

	if body == nil {
		return nil, fmt.Errorf("request body is required (use --file, --data or the field flags)")
	}

	return body, nil
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

// scopeHint appends the scope a 403 is usually missing, so the user knows what
// to ask Yuno for instead of re-reading the raw API error.
func scopeHint(err error, scopes string) error {
	if err == nil || !errors.Is(err, api.ErrForbidden) {
		return err
	}

	return fmt.Errorf("%w: ask Yuno to grant this api key the %s scope", err, scopes)
}

// flagString reads a string flag, treating a missing flag as empty.
func flagString(cmd *cobra.Command, name string) string {
	value, err := cmd.Flags().GetString(name)
	if err != nil {
		return ""
	}

	return value
}

// flagDuration reads a duration flag, treating a missing flag as zero. The PCI
// proxy commands declare their own integer --timeout, which shadows the
// persistent one; there the client keeps its default timeout.
func flagDuration(cmd *cobra.Command, name string) time.Duration {
	value, err := cmd.Flags().GetDuration(name)
	if err != nil {
		return 0
	}

	return value
}

// flagBool reads a bool flag, treating a missing flag as false.
func flagBool(cmd *cobra.Command, name string) bool {
	value, err := cmd.Flags().GetBool(name)
	if err != nil {
		return false
	}

	return value
}

// flagStringArray reads a repeatable string flag, treating a missing flag as empty.
func flagStringArray(cmd *cobra.Command, name string) []string {
	value, err := cmd.Flags().GetStringArray(name)
	if err != nil {
		return nil
	}

	return value
}

// flagIntString renders an int flag as a query value.
func flagIntString(cmd *cobra.Command, name string) string {
	value, err := cmd.Flags().GetInt(name)
	if err != nil {
		return ""
	}

	return strconv.Itoa(value)
}

// flagBoolString renders a bool flag as a query value.
func flagBoolString(cmd *cobra.Command, name string) string {
	return strconv.FormatBool(flagBool(cmd, name))
}

// registerPagingFlags declares the paging flags of a list command.
func registerPagingFlags(cmd *cobra.Command) {
	flags := cmd.Flags()

	flags.Int("limit", 0, "stop after this many items (0 fetches every page)")
	flags.Int("page-size", 0, "items requested per page")
}

// pagingFlags reads the paging flags of a list command.
func pagingFlags(cmd *cobra.Command) (limit, pageSize int) {
	return flagInt(cmd, "limit"), flagInt(cmd, "page-size")
}

// flagInt reads an int flag, treating a missing flag as zero.
func flagInt(cmd *cobra.Command, name string) int {
	value, err := cmd.Flags().GetInt(name)
	if err != nil {
		return 0
	}

	return value
}
