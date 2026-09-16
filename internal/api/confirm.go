package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/pvarganov/yuno-cli/internal/mask"
)

// maxPreviewBody caps how much of a non-JSON body the preview shows.
const maxPreviewBody = 2048

// readOnlyMethods never change state and therefore never prompt.
var readOnlyMethods = map[string]struct{}{
	http.MethodGet:     {},
	http.MethodHead:    {},
	http.MethodOptions: {},
}

// Confirmer approves a mutating request before it leaves the process. Wiring it
// into the client is what makes the gate cover every write operation.
type Confirmer interface {
	Confirm(preview string) error
}

// WithConfirmer routes every non-GET request through c for approval.
func WithConfirmer(c Confirmer) Option {
	return func(cl *Client) {
		cl.confirmer = c
	}
}

// needsConfirmation reports whether a request of this method must be approved.
func needsConfirmation(method string) bool {
	_, ok := readOnlyMethods[strings.ToUpper(method)]

	return !ok
}

// confirm asks the gate for approval once per logical call, before any attempt.
func (c *Client) confirm(method, path, target string, payload []byte) error {
	if c.confirmer == nil || !needsConfirmation(method) {
		return nil
	}

	if err := c.confirmer.Confirm(previewRequest(method, target, payload)); err != nil {
		return fmt.Errorf("%s %s: %w", method, path, err)
	}

	return nil
}

// previewRequest renders what is about to be sent, with card data and
// credentials masked.
func previewRequest(method, target string, payload []byte) string {
	var b strings.Builder

	fmt.Fprintf(&b, "%s %s", strings.ToUpper(method), target)

	if len(bytes.TrimSpace(payload)) > 0 {
		b.WriteString("\n")
		b.WriteString(previewBody(payload))
	}

	return b.String()
}

// previewBody pretty-prints a masked JSON body, falling back to the truncated
// raw payload when it is not JSON.
func previewBody(payload []byte) string {
	var decoded any
	if err := json.Unmarshal(payload, &decoded); err != nil {
		return truncatePreview(string(payload))
	}

	out, err := json.MarshalIndent(mask.JSON(decoded), "", "  ")
	if err != nil {
		return truncatePreview(string(payload))
	}

	return truncatePreview(string(out))
}

// truncatePreview caps a preview body by runes, so a huge payload cannot flood
// the terminal before the user answers.
func truncatePreview(s string) string {
	runes := []rune(s)
	if len(runes) <= maxPreviewBody {
		return s
	}

	return string(runes[:maxPreviewBody]) + "… (truncated)"
}
