package confirm_test

import (
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/pvarganov/yuno-cli/internal/confirm"
)

func TestConfirmAnswers(t *testing.T) {
	tests := []struct {
		name    string
		answer  string
		wantErr error
	}{
		{name: "lowercase yes", answer: "y\n"},
		{name: "spelled out", answer: "yes\n"},
		{name: "uppercase", answer: "  Y  \n"},
		{name: "no", answer: "n\n", wantErr: confirm.ErrDeclined},
		{name: "empty line", answer: "\n", wantErr: confirm.ErrDeclined},
		{name: "garbage", answer: "maybe\n", wantErr: confirm.ErrDeclined},
		{name: "eof", answer: "", wantErr: confirm.ErrDeclined},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var out strings.Builder

			err := confirm.Confirm(&out, strings.NewReader(tt.answer), "POST /payments")
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("Confirm = %v, want %v", err, tt.wantErr)
			}

			if !strings.Contains(out.String(), "POST /payments") {
				t.Errorf("prompt missing preview: %q", out.String())
			}

			if !strings.Contains(out.String(), "[y/N]") {
				t.Errorf("prompt missing the y/N hint: %q", out.String())
			}
		})
	}
}

func TestConfirmPrintsAbortedOnDecline(t *testing.T) {
	var out strings.Builder

	if err := confirm.Confirm(&out, strings.NewReader("n\n"), "DELETE /customers/1"); !errors.Is(err, confirm.ErrDeclined) {
		t.Fatalf("Confirm = %v, want ErrDeclined", err)
	}

	if !strings.Contains(out.String(), "aborted") {
		t.Errorf("decline not reported: %q", out.String())
	}
}

func TestGateAssumeYesSkipsPrompt(t *testing.T) {
	t.Setenv(confirm.EnvAssumeYes, "")

	var out strings.Builder

	gate := confirm.New(&out, strings.NewReader("n\n"), true)

	if err := gate.Confirm("POST /payments"); err != nil {
		t.Fatalf("Confirm = %v, want nil", err)
	}

	if out.String() != "" {
		t.Errorf("prompt written despite --yes: %q", out.String())
	}
}

func TestGateAssumeYesFromEnv(t *testing.T) {
	for _, value := range []string{"1", "true", "YES", "on"} {
		t.Run(value, func(t *testing.T) {
			t.Setenv(confirm.EnvAssumeYes, value)

			var out strings.Builder

			if err := confirm.New(&out, strings.NewReader(""), false).Confirm("POST /payments"); err != nil {
				t.Fatalf("Confirm = %v, want nil", err)
			}
		})
	}
}

func TestGateEnvIgnoresOtherValues(t *testing.T) {
	t.Setenv(confirm.EnvAssumeYes, "0")

	var out strings.Builder

	err := confirm.New(&out, strings.NewReader("y\n"), false).Confirm("POST /payments")
	if !errors.Is(err, confirm.ErrNotATerminal) {
		t.Fatalf("Confirm = %v, want ErrNotATerminal", err)
	}
}

func TestGateNonTTYFailsInsteadOfHanging(t *testing.T) {
	t.Setenv(confirm.EnvAssumeYes, "")

	var out strings.Builder

	// A pipe never becomes a terminal, so prompting here would block forever.
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe: %v", err)
	}

	defer func() { _ = r.Close(); _ = w.Close() }()

	if err := confirm.New(&out, r, false).Confirm("POST /payments"); !errors.Is(err, confirm.ErrNotATerminal) {
		t.Fatalf("Confirm = %v, want ErrNotATerminal", err)
	}

	if out.String() != "" {
		t.Errorf("prompt written to a non-terminal: %q", out.String())
	}
}

func TestNilGateApproves(t *testing.T) {
	var gate *confirm.Gate

	if err := gate.Confirm("POST /payments"); err != nil {
		t.Fatalf("Confirm = %v, want nil", err)
	}
}
