package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httputil"
	"strings"

	"github.com/pvarganov/yuno-cli/internal/mask"
)

// headerBodySeparator ends the header block of an HTTP/1.1 message.
var headerBodySeparator = []byte("\r\n\r\n")

// dumpRequest writes a masked dump of the outgoing request to the verbose
// output. Dump failures are reported inline: a broken dump must never abort the
// call it is only observing.
func (c *Client) dumpRequest(req *http.Request) {
	if !c.verbose {
		return
	}

	dump, err := httputil.DumpRequestOut(req, true)
	if err != nil {
		_, _ = fmt.Fprintf(c.verboseOut, "> dump request failed: %v\n", err)

		return
	}

	c.writeDump("> ", dump)
}

// dumpResponse writes a masked dump of the response. The body is passed in
// separately because it has already been consumed by the caller.
func (c *Client) dumpResponse(resp *http.Response, body []byte) {
	if !c.verbose {
		return
	}

	dump, err := httputil.DumpResponse(resp, false)
	if err != nil {
		_, _ = fmt.Fprintf(c.verboseOut, "< dump response failed: %v\n", err)

		return
	}

	c.writeDump("< ", append(dump, body...))
}

func (c *Client) writeDump(prefix string, dump []byte) {
	sanitised := sanitizeDump(dump)

	for _, line := range strings.Split(strings.TrimRight(string(sanitised), "\r\n"), "\n") {
		_, _ = fmt.Fprintf(c.verboseOut, "%s%s\n", prefix, strings.TrimSuffix(line, "\r"))
	}

	_, _ = fmt.Fprintln(c.verboseOut, strings.TrimSpace(prefix))
}

// sanitizeDump masks credential headers and any sensitive field of a JSON body,
// so that a --verbose run can be pasted into a ticket as is.
func sanitizeDump(dump []byte) []byte {
	head, body, found := bytes.Cut(dump, headerBodySeparator)
	if !found {
		return maskHeaderBlock(dump)
	}

	out := append(maskHeaderBlock(head), headerBodySeparator...)

	return append(out, maskJSONBody(body)...)
}

func maskHeaderBlock(head []byte) []byte {
	lines := strings.Split(string(head), "\r\n")
	for i, line := range lines {
		name, value, ok := strings.Cut(line, ":")
		if !ok || !mask.IsSensitive(name) {
			continue
		}

		lines[i] = name + ": " + mask.Secret(strings.TrimSpace(value))
	}

	return []byte(strings.Join(lines, "\r\n"))
}

func maskJSONBody(body []byte) []byte {
	if len(bytes.TrimSpace(body)) == 0 {
		return body
	}

	var decoded any
	if err := json.Unmarshal(body, &decoded); err != nil {
		return body
	}

	masked, err := json.Marshal(mask.JSON(decoded))
	if err != nil {
		return body
	}

	return masked
}

// verboseWriter falls back to a discarding writer so a verbose client is never
// nil-dereferenced.
func verboseWriter(w io.Writer) io.Writer {
	if w == nil {
		return io.Discard
	}

	return w
}
