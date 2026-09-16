package cmd_test

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/pvarganov/yuno-cli/internal/cmd"
)

// apiDocFile documents the command-to-operation map. It is generated from the
// command tree, so it has to be regenerated whenever a command is added.
const apiDocFile = "../../docs/yuno-api.md"

// docOperation matches one `METHOD /path` entry inside the documentation table.
var docOperation = regexp.MustCompile(`(GET|POST|PUT|PATCH|DELETE) (/[^ |<]*)`)

// readDoc returns the contents of a checked-in documentation file.
func readDoc(t *testing.T, path string) string {
	t.Helper()

	data, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}

	return string(data)
}

func TestAPIDocListsEveryCommand(t *testing.T) {
	doc := readDoc(t, apiDocFile)

	var missing []string

	walkCommands(cmd.NewRootCommand("test"), func(c *cobra.Command) {
		if len(cmd.Operations(c)) == 0 {
			return
		}

		path := strings.Join(strings.Fields(c.CommandPath()), " ")
		if !strings.Contains(doc, "`"+path+"`") {
			missing = append(missing, path)
		}
	})

	sort.Strings(missing)

	if len(missing) > 0 {
		t.Errorf("%d commands are absent from %s:\n%s",
			len(missing), apiDocFile, strings.Join(missing, "\n"))
	}
}

func TestAPIDocListsEveryOperation(t *testing.T) {
	doc := readDoc(t, apiDocFile)
	spec := readSpecOperations(t)

	documented := map[string]bool{}
	for _, op := range docOperation.FindAllString(doc, -1) {
		documented[normalizeOperation(op)] = true
	}

	var missing []string

	for key, line := range spec {
		if !documented[key] {
			missing = append(missing, line)
		}
	}

	sort.Strings(missing)

	if len(missing) > 0 {
		t.Errorf("%d operations are absent from %s:\n%s",
			len(missing), apiDocFile, strings.Join(missing, "\n"))
	}
}

func TestAPIDocPointsAtTheSpec(t *testing.T) {
	doc := readDoc(t, apiDocFile)

	for _, want := range []string{"docs/yuno-openapi.json", "docs/yuno-operations.txt"} {
		if !strings.Contains(doc, want) {
			t.Errorf("%s does not point at %s", apiDocFile, want)
		}
	}
}

func TestReadmeCoversTheBasics(t *testing.T) {
	doc := readDoc(t, "../../README.md")

	for _, want := range []string{
		"yuno-cli auth",
		"--profile",
		"--yes",
		"--json",
		"jq",
		"yuno-cli raw",
		"YUNO_PUBLIC_API_KEY",
	} {
		if !strings.Contains(doc, want) {
			t.Errorf("README.md does not document %q", want)
		}
	}
}
