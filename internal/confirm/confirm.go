// Package confirm gates write operations behind an interactive yes/no prompt,
// so that a production key cannot silently create a payment.
package confirm

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"golang.org/x/term"
)

// Sentinel errors returned by the gate.
var (
	// ErrDeclined means the user answered anything other than yes.
	ErrDeclined = errors.New("confirmation declined")
	// ErrNotATerminal means there was nobody to ask: prompting would hang.
	ErrNotATerminal = errors.New("stdin is not a terminal: rerun with --yes to confirm write operations")
)

// EnvAssumeYes skips the prompt when set to a truthy value.
const EnvAssumeYes = "YUNO_ASSUME_YES"

// truthy lists the values of EnvAssumeYes that count as "yes".
var truthy = map[string]struct{}{
	"1":    {},
	"y":    {},
	"yes":  {},
	"true": {},
	"on":   {},
}

// Confirm prints the preview to w and reads an answer from r. Only "y" and
// "yes" (in any case) approve; everything else, including EOF, declines.
func Confirm(w io.Writer, r io.Reader, preview string) error {
	if _, err := fmt.Fprintf(w, "About to send:\n%s\nContinue? [y/N]: ", preview); err != nil {
		return fmt.Errorf("write confirmation prompt: %w", err)
	}

	answer, err := readLine(r)
	if err != nil {
		return err
	}

	switch strings.ToLower(strings.TrimSpace(answer)) {
	case "y", "yes":
		return nil
	default:
		_, _ = fmt.Fprintln(w, "aborted")

		return ErrDeclined
	}
}

// readLine reads one answer. A clean EOF is an empty answer, not a failure.
func readLine(r io.Reader) (string, error) {
	line, err := bufio.NewReader(r).ReadString('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		return "", fmt.Errorf("read confirmation answer: %w", err)
	}

	return line, nil
}

// Gate decides whether a write operation may proceed.
type Gate struct {
	out       io.Writer
	in        io.Reader
	assumeYes bool
}

// New builds a gate that prompts on in/out. It never prompts when assumeYes is
// set by the --yes flag or by EnvAssumeYes.
func New(out io.Writer, in io.Reader, assumeYes bool) *Gate {
	return &Gate{
		out:       out,
		in:        in,
		assumeYes: assumeYes || assumeYesFromEnv(),
	}
}

// Confirm asks for approval of the previewed operation. A nil gate approves
// everything, which keeps non-interactive callers simple.
func (g *Gate) Confirm(preview string) error {
	if g == nil || g.assumeYes {
		return nil
	}

	if !isTerminal(g.in) {
		return ErrNotATerminal
	}

	return Confirm(g.out, g.in, preview)
}

// assumeYesFromEnv reports whether the environment pre-approves writes.
func assumeYesFromEnv() bool {
	_, ok := truthy[strings.ToLower(strings.TrimSpace(os.Getenv(EnvAssumeYes)))]

	return ok
}

// isTerminal reports whether r is an interactive terminal that can answer.
func isTerminal(r io.Reader) bool {
	f, ok := r.(interface{ Fd() uintptr })
	if !ok {
		return false
	}

	return term.IsTerminal(int(f.Fd()))
}
