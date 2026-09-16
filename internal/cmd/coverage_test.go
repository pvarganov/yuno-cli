package cmd_test

import (
	"bufio"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/pvarganov/yuno-cli/internal/cmd"
)

// operationsFile is the checked-in listing of every operation of the Yuno API.
// It is the authoritative coverage checklist of the CLI.
const operationsFile = "../../docs/yuno-operations.txt"

// operationLine matches one `METHOD /path` entry of the checklist.
var operationLine = regexp.MustCompile(`^(GET|POST|PUT|PATCH|DELETE) (/\S*)$`)

// pathParam matches the `{payment_id}` placeholders of a spec path.
var pathParam = regexp.MustCompile(`\{[^}]*\}`)

// normalizeOperation drops the names of the path parameters, so that a command
// that calls its path parameter `id` still matches the spec's `{payment_id}`.
func normalizeOperation(op string) string {
	return pathParam.ReplaceAllString(strings.TrimSpace(op), "{}")
}

// readSpecOperations parses the checklist into normalized operations.
func readSpecOperations(t *testing.T) map[string]string {
	t.Helper()

	file, err := os.Open(filepath.Clean(operationsFile))
	if err != nil {
		t.Fatalf("open %s: %v", operationsFile, err)
	}
	defer func() { _ = file.Close() }()

	ops := map[string]string{}
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if !operationLine.MatchString(line) {
			continue
		}

		ops[normalizeOperation(line)] = line
	}

	if err := scanner.Err(); err != nil {
		t.Fatalf("read %s: %v", operationsFile, err)
	}

	return ops
}

// walkCommands calls fn for every command of the tree, the root included.
func walkCommands(root *cobra.Command, fn func(*cobra.Command)) {
	fn(root)

	for _, child := range root.Commands() {
		walkCommands(child, fn)
	}
}

// commandOperations collects the operations declared by the command tree,
// mapped to the command paths that declare them.
func commandOperations(t *testing.T) map[string][]string {
	t.Helper()

	found := map[string][]string{}

	walkCommands(cmd.NewRootCommand("test"), func(c *cobra.Command) {
		for _, op := range cmd.Operations(c) {
			key := normalizeOperation(op)
			found[key] = append(found[key], c.CommandPath())
		}
	})

	return found
}

func TestSpecOperationsCount(t *testing.T) {
	ops := readSpecOperations(t)

	// The spec has 172 operations; a few share a normalized shape, so the map
	// must not be empty and must not shrink unnoticed.
	if len(ops) < 170 {
		t.Fatalf("parsed %d operations from %s, want at least 170", len(ops), operationsFile)
	}
}

func TestEveryOperationHasACommand(t *testing.T) {
	spec := readSpecOperations(t)
	found := commandOperations(t)

	var missing []string

	for key, line := range spec {
		if len(found[key]) == 0 {
			missing = append(missing, line)
		}
	}

	sort.Strings(missing)

	if len(missing) > 0 {
		t.Errorf("%d operations have no command:\n%s", len(missing), strings.Join(missing, "\n"))
	}
}

func TestCommandOperationsExistInSpec(t *testing.T) {
	spec := readSpecOperations(t)
	found := commandOperations(t)

	var unknown []string

	for key, paths := range found {
		if _, ok := spec[key]; !ok {
			unknown = append(unknown, key+" ("+strings.Join(paths, ", ")+")")
		}
	}

	sort.Strings(unknown)

	if len(unknown) > 0 {
		t.Errorf("%d commands call operations absent from the spec:\n%s",
			len(unknown), strings.Join(unknown, "\n"))
	}
}

func TestLeafCommandsDeclareOperations(t *testing.T) {
	// Commands that talk to the user instead of the API: they configure the
	// CLI itself or forward an arbitrary path.
	local := map[string]bool{
		"yuno-cli auth":           true,
		"yuno-cli auth status":    true,
		"yuno-cli profile list":   true,
		"yuno-cli profile use":    true,
		"yuno-cli profile delete": true,
		"yuno-cli raw":            true,
		"yuno-cli completion":     true,
		"yuno-cli help":           true,
	}

	var bare []string

	walkCommands(cmd.NewRootCommand("test"), func(c *cobra.Command) {
		if c.HasSubCommands() || len(cmd.Operations(c)) > 0 {
			return
		}

		path := strings.Fields(c.CommandPath())
		key := strings.Join(path, " ")

		for prefix := range local {
			if strings.HasPrefix(key, prefix) {
				return
			}
		}

		bare = append(bare, key)
	})

	sort.Strings(bare)

	if len(bare) > 0 {
		t.Errorf("%d leaf commands declare no operation:\n%s", len(bare), strings.Join(bare, "\n"))
	}
}
