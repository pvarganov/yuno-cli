package cmd_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/pvarganov/yuno-cli/internal/cmd"
)

func TestCompletionCommand_Shells(t *testing.T) {
	t.Parallel()

	tests := []struct {
		shell  string
		marker string
	}{
		{shell: "bash", marker: "yuno-cli"},
		{shell: "zsh", marker: "#compdef"},
		{shell: "fish", marker: "complete -c yuno-cli"},
		{shell: "powershell", marker: "Register-ArgumentCompleter"},
	}

	for _, tt := range tests {
		t.Run(tt.shell, func(t *testing.T) {
			t.Parallel()

			var buf bytes.Buffer
			root := cmd.NewRootCommand("test")
			root.SetOut(&buf)
			root.SetErr(&buf)
			root.SetArgs([]string{"completion", tt.shell})

			if err := root.Execute(); err != nil {
				t.Fatalf("completion %s: unexpected error %v", tt.shell, err)
			}

			out := buf.String()
			if out == "" {
				t.Fatalf("completion %s: empty output", tt.shell)
			}

			if !strings.Contains(out, tt.marker) {
				t.Errorf("completion %s: output should contain %q, got %q", tt.shell, tt.marker, head(out))
			}
		})
	}
}

func TestCompletionCommand_UnknownShell(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	root := cmd.NewRootCommand("test")
	root.SetOut(&buf)
	root.SetErr(&buf)
	root.SetArgs([]string{"completion", "tcsh"})

	err := root.Execute()
	if err == nil {
		t.Fatal("expected an error for an unsupported shell")
	}

	if !strings.Contains(err.Error(), "tcsh") {
		t.Errorf("error should name the shell, got %v", err)
	}
}

func TestCompletionCommand_NoArgs(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	root := cmd.NewRootCommand("test")
	root.SetOut(&buf)
	root.SetErr(&buf)
	root.SetArgs([]string{"completion"})

	if err := root.Execute(); err == nil {
		t.Fatal("expected an error when no shell is given")
	}
}

// head trims long completion scripts down to a readable failure message.
func head(s string) string {
	const limit = 200
	if len(s) > limit {
		return s[:limit]
	}

	return s
}
