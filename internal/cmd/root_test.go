package cmd_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/pvarganov/yuno-cli/internal/cmd"
)

func TestRootCommand_Help(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	root := cmd.NewRootCommand("test")
	root.SetOut(&buf)
	root.SetErr(&buf)
	root.SetArgs([]string{"--help"})

	if err := root.Execute(); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "yuno-cli") {
		t.Errorf("help output should contain 'yuno-cli', got: %s", out)
	}
}

func TestRootCommand_NoArgs(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	root := cmd.NewRootCommand("test")
	root.SetOut(&buf)
	root.SetErr(&buf)
	root.SetArgs([]string{})

	if err := root.Execute(); err != nil {
		t.Fatalf("expected no error running with no args, got %v", err)
	}

	if !strings.Contains(buf.String(), "Usage:") {
		t.Errorf("no-args output should print help, got: %s", buf.String())
	}
}

func TestRootCommand_Version(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	root := cmd.NewRootCommand("1.2.3")
	root.SetOut(&buf)
	root.SetErr(&buf)
	root.SetArgs([]string{"--version"})

	if err := root.Execute(); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if !strings.Contains(buf.String(), "1.2.3") {
		t.Errorf("version output should contain '1.2.3', got: %s", buf.String())
	}
}

func TestRootCommand_UnknownCommand(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	root := cmd.NewRootCommand("test")
	root.SetOut(&buf)
	root.SetErr(&buf)
	root.SetArgs([]string{"nope"})

	err := root.Execute()
	if err == nil {
		t.Fatal("expected an error for an unknown command, got nil")
	}

	if !strings.Contains(err.Error(), "nope") {
		t.Errorf("error should mention the unknown command, got: %v", err)
	}
}

func TestRootCommand_PersistentFlags(t *testing.T) {
	t.Parallel()

	root := cmd.NewRootCommand("test")
	for _, name := range []string{"json", "profile", "verbose", "yes"} {
		if root.PersistentFlags().Lookup(name) == nil {
			t.Errorf("expected persistent flag %q to be registered", name)
		}
	}
}
