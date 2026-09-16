// Package output renders API results either as indented JSON for piping into
// jq or as an aligned table for reading in a terminal.
package output

import (
	"encoding/json"
	"fmt"
	"io"
	"reflect"
	"strings"

	"github.com/pvarganov/yuno-cli/internal/mask"
)

// MaxCellWidth caps a table cell, counted in runes so that multi-byte values
// are cut on a character boundary. Longer values end with ellipsis.
const MaxCellWidth = 48

// ellipsis marks a cell whose value did not fit into MaxCellWidth.
const ellipsis = "..."

// columnSep separates two table columns.
const columnSep = "  "

// Formatter writes data to a writer in a specific format.
type Formatter interface {
	Format(w io.Writer, data any) error
}

// Option configures a Formatter.
type Option func(*options)

// options holds the settings shared by the formatters.
type options struct {
	unmask bool
}

// WithUnmask disables the masking of card-like fields in the table output. It
// backs the `--unmask` flag and has no effect on the JSON output, which is
// never masked so that jq pipelines keep seeing the API response verbatim.
func WithUnmask(unmask bool) Option {
	return func(o *options) {
		o.unmask = unmask
	}
}

// NewFormatter returns a JSON formatter when jsonMode is true, otherwise a
// table formatter.
func NewFormatter(jsonMode bool, opts ...Option) Formatter {
	var o options

	for _, opt := range opts {
		opt(&o)
	}

	if jsonMode {
		return &JSONFormatter{}
	}

	return &TableFormatter{unmask: o.unmask}
}

// JSONFormatter renders data as indented JSON.
type JSONFormatter struct{}

// Format writes data as indented JSON followed by a newline.
func (f *JSONFormatter) Format(w io.Writer, data any) error {
	b, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal json: %w", err)
	}

	if _, err := w.Write(append(b, '\n')); err != nil {
		return fmt.Errorf("write json: %w", err)
	}

	return nil
}

// TableFormatter renders a slice of structs as an aligned text table whose
// headers are the uppercased JSON tags of the element type.
type TableFormatter struct {
	unmask bool
}

// column is one table column, addressed by its field path in the element type.
type column struct {
	index []int
	name  string
	tag   string
}

// Format writes data as an aligned table. An empty slice produces no output.
func (f *TableFormatter) Format(w io.Writer, data any) error {
	elem, err := elemStructType(data)
	if err != nil {
		return err
	}

	value := reflect.ValueOf(data)
	if value.Len() == 0 {
		return nil
	}

	cols := collectColumns(elem, nil)
	if len(cols) == 0 {
		return nil
	}

	rows := make([][]string, value.Len())
	for i := range rows {
		rows[i] = f.row(value.Index(i), cols)
	}

	return writeTable(w, cols, rows)
}

// row renders one slice element as the string cells of a table row.
func (f *TableFormatter) row(elem reflect.Value, cols []column) []string {
	cells := make([]string, len(cols))

	if elem.Kind() == reflect.Pointer {
		if elem.IsNil() {
			return cells
		}

		elem = elem.Elem()
	}

	for i, col := range cols {
		cells[i] = truncate(f.cell(elem.FieldByIndex(col.index), col))
	}

	return cells
}

// cell renders a single field, masking it when the column carries card data or
// a credential and masking was not disabled.
func (f *TableFormatter) cell(field reflect.Value, col column) string {
	if field.Kind() == reflect.Pointer {
		if field.IsNil() {
			return ""
		}

		field = field.Elem()
	}

	text := fmt.Sprintf("%v", field.Interface())

	// A bool never carries a credential: `secret: true` on a provider parameter
	// says the parameter holds one, it is not the value itself.
	if !f.unmask && field.Kind() != reflect.Bool && mask.IsSensitive(col.tag) {
		return mask.Secret(text)
	}

	return text
}

// elemStructType validates that data is a slice of structs and returns the
// struct type of its elements.
func elemStructType(data any) (reflect.Type, error) {
	value := reflect.ValueOf(data)
	if value.Kind() != reflect.Slice {
		return nil, fmt.Errorf("format table: want a slice, got %T", data)
	}

	elem := value.Type().Elem()
	if elem.Kind() == reflect.Pointer {
		elem = elem.Elem()
	}

	if elem.Kind() != reflect.Struct {
		return nil, fmt.Errorf("format table: want a slice of structs, got %T", data)
	}

	return elem, nil
}

// collectColumns walks a struct type and returns one column per exported field
// carrying a JSON tag, flattening embedded structs into the same row.
func collectColumns(t reflect.Type, prefix []int) []column {
	var cols []column

	for i := range t.NumField() {
		field := t.Field(i)
		if !field.IsExported() && !field.Anonymous {
			continue
		}

		path := append(append([]int{}, prefix...), i)
		if field.Anonymous && field.Type.Kind() == reflect.Struct {
			cols = append(cols, collectColumns(field.Type, path)...)

			continue
		}

		tag, _, _ := strings.Cut(field.Tag.Get("json"), ",")
		if tag == "" || tag == "-" {
			continue
		}

		cols = append(cols, column{index: path, name: strings.ToUpper(tag), tag: tag})
	}

	return cols
}

// writeTable prints the header and the rows padded to a common column width.
func writeTable(w io.Writer, cols []column, rows [][]string) error {
	widths := make([]int, len(cols))
	headers := make([]string, len(cols))

	for i, col := range cols {
		widths[i] = len([]rune(col.name))
		headers[i] = col.name
	}

	for _, row := range rows {
		for i, cell := range row {
			if n := len([]rune(cell)); n > widths[i] {
				widths[i] = n
			}
		}
	}

	if err := writeRow(w, headers, widths); err != nil {
		return err
	}

	for _, row := range rows {
		if err := writeRow(w, row, widths); err != nil {
			return err
		}
	}

	return nil
}

// writeRow prints one row, padding every cell to its column width and dropping
// the trailing padding so that lines carry no invisible whitespace.
func writeRow(w io.Writer, cells []string, widths []int) error {
	parts := make([]string, len(cells))
	for i, cell := range cells {
		parts[i] = cell + strings.Repeat(" ", widths[i]-len([]rune(cell)))
	}

	line := strings.TrimRight(strings.Join(parts, columnSep), " ")
	if _, err := fmt.Fprintln(w, line); err != nil {
		return fmt.Errorf("write row: %w", err)
	}

	return nil
}

// truncate shortens a cell to MaxCellWidth runes, ellipsis included.
func truncate(s string) string {
	runes := []rune(s)
	if len(runes) <= MaxCellWidth {
		return s
	}

	return string(runes[:MaxCellWidth-len(ellipsis)]) + ellipsis
}
