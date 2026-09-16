package cmd

import (
	"strings"

	"github.com/spf13/cobra"
)

// annotationOperations is the cobra annotation key under which a command
// records the Yuno API operations it calls, as `METHOD /path` entries joined
// by commas. The coverage test walks the command tree and checks these
// annotations against `docs/yuno-operations.txt`, so every leaf command that
// talks to the API must carry one.
const annotationOperations = "yuno.operations"

// apiOperations builds the annotation map of a command from its operations.
func apiOperations(ops ...string) map[string]string {
	return map[string]string{annotationOperations: strings.Join(ops, ",")}
}

// commandOperations returns the operations a single command declares.
func commandOperations(cmd *cobra.Command) []string {
	value := cmd.Annotations[annotationOperations]
	if value == "" {
		return nil
	}

	return strings.Split(value, ",")
}

// Operations reports the Yuno API operations a command declares. It is
// exported for the coverage test, which walks the command tree and matches
// these against the checked-in operation listing.
func Operations(cmd *cobra.Command) []string {
	return commandOperations(cmd)
}
