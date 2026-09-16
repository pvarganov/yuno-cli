package output_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/pvarganov/yuno-cli/internal/output"
)

type row struct {
	ID     string  `json:"id"`
	Status string  `json:"status"`
	Amount *string `json:"amount"`
	Number string  `json:"number"`
	hidden string  //nolint:unused // proves unexported fields are skipped
	Skip   string  `json:"-"`
	NoTag  string
}

func ptr(s string) *string { return &s }

func TestNewFormatterJSONMode(t *testing.T) {
	if _, ok := output.NewFormatter(true).(*output.JSONFormatter); !ok {
		t.Errorf("NewFormatter(true) = %T, want *output.JSONFormatter", output.NewFormatter(true))
	}

	if _, ok := output.NewFormatter(false).(*output.TableFormatter); !ok {
		t.Errorf("NewFormatter(false) = %T, want *output.TableFormatter", output.NewFormatter(false))
	}
}

func TestJSONFormatterPrintsIndentedJSON(t *testing.T) {
	var buf bytes.Buffer

	data := map[string]any{"id": "pay_1", "amount": map[string]any{"value": 10.5}}
	if err := output.NewFormatter(true).Format(&buf, data); err != nil {
		t.Fatalf("Format() error = %v", err)
	}

	want := "{\n  \"amount\": {\n    \"value\": 10.5\n  },\n  \"id\": \"pay_1\"\n}\n"
	if buf.String() != want {
		t.Errorf("Format() = %q, want %q", buf.String(), want)
	}
}

func TestJSONFormatterDoesNotMask(t *testing.T) {
	var buf bytes.Buffer

	if err := output.NewFormatter(true).Format(&buf, map[string]any{"number": "4111111111111111"}); err != nil {
		t.Fatalf("Format() error = %v", err)
	}

	if !strings.Contains(buf.String(), "4111111111111111") {
		t.Errorf("Format() = %q, want the raw value so that jq pipelines keep working", buf.String())
	}
}

func TestJSONFormatterUnsupportedValue(t *testing.T) {
	err := output.NewFormatter(true).Format(&bytes.Buffer{}, make(chan int))
	if err == nil {
		t.Fatal("Format() error = nil, want a marshal error")
	}
}

func TestTableFormatterAlignsColumns(t *testing.T) {
	var buf bytes.Buffer

	data := []row{
		{ID: "pay_1", Status: "SUCCEEDED", Amount: ptr("10.50")},
		{ID: "pay_longer_id", Status: "READY", Amount: ptr("7.00")},
	}

	if err := output.NewFormatter(false).Format(&buf, data); err != nil {
		t.Fatalf("Format() error = %v", err)
	}

	want := strings.Join([]string{
		"ID             STATUS     AMOUNT  NUMBER",
		"pay_1          SUCCEEDED  10.50",
		"pay_longer_id  READY      7.00",
		"",
	}, "\n")

	if buf.String() != want {
		t.Errorf("Format() = %q, want %q", buf.String(), want)
	}
}

func TestTableFormatterEmptySlicePrintsNothing(t *testing.T) {
	var buf bytes.Buffer

	if err := output.NewFormatter(false).Format(&buf, []row{}); err != nil {
		t.Fatalf("Format() error = %v", err)
	}

	if buf.Len() != 0 {
		t.Errorf("Format() = %q, want no output", buf.String())
	}
}

func TestTableFormatterNilPointersRenderEmpty(t *testing.T) {
	var buf bytes.Buffer

	data := []*row{
		{ID: "pay_1", Status: "READY"},
		nil,
	}

	if err := output.NewFormatter(false).Format(&buf, data); err != nil {
		t.Fatalf("Format() error = %v", err)
	}

	lines := strings.Split(strings.TrimSuffix(buf.String(), "\n"), "\n")
	if len(lines) != 3 {
		t.Fatalf("Format() produced %d lines, want 3", len(lines))
	}

	if strings.TrimSpace(lines[1]) != "pay_1  READY" {
		t.Errorf("row = %q, want the nil amount rendered as an empty cell", lines[1])
	}

	if strings.TrimSpace(lines[2]) != "" {
		t.Errorf("nil element row = %q, want only empty cells", lines[2])
	}
}

func TestTableFormatterTruncatesByRunes(t *testing.T) {
	var buf bytes.Buffer

	long := strings.Repeat("я", 80)
	if err := output.NewFormatter(false).Format(&buf, []row{{ID: long}}); err != nil {
		t.Fatalf("Format() error = %v", err)
	}

	cell := strings.Fields(strings.Split(buf.String(), "\n")[1])[0]
	if got := len([]rune(cell)); got != output.MaxCellWidth {
		t.Errorf("cell rune length = %d, want %d", got, output.MaxCellWidth)
	}

	if !strings.HasSuffix(cell, "...") {
		t.Errorf("cell = %q, want a truncation marker", cell)
	}
}

func TestTableFormatterMasksCardFieldsByDefault(t *testing.T) {
	var buf bytes.Buffer

	if err := output.NewFormatter(false).Format(&buf, []row{{ID: "pay_1", Number: "4111111111111111"}}); err != nil {
		t.Fatalf("Format() error = %v", err)
	}

	if strings.Contains(buf.String(), "4111111111111111") {
		t.Fatalf("Format() = %q, leaks the card number", buf.String())
	}

	if !strings.Contains(buf.String(), "4111****") {
		t.Errorf("Format() = %q, want the masked card number", buf.String())
	}
}

func TestTableFormatterUnmaskOptsOut(t *testing.T) {
	var buf bytes.Buffer

	f := output.NewFormatter(false, output.WithUnmask(true))
	if err := f.Format(&buf, []row{{ID: "pay_1", Number: "4111111111111111"}}); err != nil {
		t.Fatalf("Format() error = %v", err)
	}

	if !strings.Contains(buf.String(), "4111111111111111") {
		t.Errorf("Format() = %q, want the raw card number under --unmask", buf.String())
	}
}

func TestWithUnmaskFalseKeepsMasking(t *testing.T) {
	var buf bytes.Buffer

	f := output.NewFormatter(false, output.WithUnmask(false))
	if err := f.Format(&buf, []row{{Number: "4111111111111111"}}); err != nil {
		t.Fatalf("Format() error = %v", err)
	}

	if strings.Contains(buf.String(), "4111111111111111") {
		t.Errorf("Format() = %q, leaks the card number", buf.String())
	}
}

func TestTableFormatterEmbeddedStructsAreFlattened(t *testing.T) {
	type base struct {
		ID string `json:"id"`
	}

	type detail struct {
		base
		Status string `json:"status"`
	}

	var buf bytes.Buffer

	if err := output.NewFormatter(false).Format(&buf, []detail{{base: base{ID: "r_1"}, Status: "READY"}}); err != nil {
		t.Fatalf("Format() error = %v", err)
	}

	if !strings.HasPrefix(buf.String(), "ID   STATUS") {
		t.Errorf("header = %q, want the embedded field flattened in", buf.String())
	}
}

func TestTableFormatterErrors(t *testing.T) {
	tests := []struct {
		name string
		data any
	}{
		{"not a slice", row{ID: "pay_1"}},
		{"slice of scalars", []string{"a", "b"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := output.NewFormatter(false).Format(&bytes.Buffer{}, tt.data); err == nil {
				t.Errorf("Format(%T) error = nil, want an error", tt.data)
			}
		})
	}
}

func TestTableFormatterStructWithoutJSONTags(t *testing.T) {
	type untagged struct {
		Name string
	}

	var buf bytes.Buffer

	if err := output.NewFormatter(false).Format(&buf, []untagged{{Name: "x"}}); err != nil {
		t.Fatalf("Format() error = %v", err)
	}

	if buf.Len() != 0 {
		t.Errorf("Format() = %q, want no output when no field carries a json tag", buf.String())
	}
}

func TestTableFormatter_DoesNotMaskABooleanSecretFlag(t *testing.T) {
	t.Parallel()

	type catalogRow struct {
		ParamID string `json:"param_id"`
		Secret  bool   `json:"secret"`
		APIKey  string `json:"api_key"`
	}

	var buf bytes.Buffer
	if err := output.NewFormatter(false).Format(&buf, []catalogRow{{ParamID: "p-1", Secret: true, APIKey: "abcdefgh"}}); err != nil {
		t.Fatalf("format: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "true") {
		t.Errorf("a boolean flag must stay readable, got:\n%s", out)
	}

	if strings.Contains(out, "abcdefgh") {
		t.Errorf("the api key must be masked, got:\n%s", out)
	}
}
