package api

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// recordingConfirmer captures the previews it is asked to approve and answers
// with a fixed verdict.
type recordingConfirmer struct {
	previews []string
	err      error
}

func (r *recordingConfirmer) Confirm(preview string) error {
	r.previews = append(r.previews, preview)

	return r.err
}

var errDeclined = errors.New("confirmation declined")

func TestConfirmGateSkipsReadOnlyRequests(t *testing.T) {
	for _, method := range []string{http.MethodGet, http.MethodHead, http.MethodOptions} {
		t.Run(method, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				_, _ = w.Write([]byte(`{"ok":true}`))
			}))
			defer srv.Close()

			gate := &recordingConfirmer{err: errDeclined}

			c, _ := newTestClient(t, srv, testProfile(), WithConfirmer(gate))

			if _, err := c.DoRaw(t.Context(), Request{Method: method, Path: "/routing"}); err != nil {
				t.Fatalf("DoRaw: %v", err)
			}

			if len(gate.previews) != 0 {
				t.Errorf("%s prompted: %v", method, gate.previews)
			}
		})
	}
}

func TestConfirmGatePromptsOnMutatingRequests(t *testing.T) {
	for _, method := range []string{http.MethodPost, http.MethodPatch, http.MethodPut, http.MethodDelete} {
		t.Run(method, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				_, _ = w.Write([]byte(`{"ok":true}`))
			}))
			defer srv.Close()

			gate := &recordingConfirmer{}

			c, _ := newTestClient(t, srv, testProfile(), WithConfirmer(gate))

			if _, err := c.DoRaw(t.Context(), Request{Method: method, Path: "/payments"}); err != nil {
				t.Fatalf("DoRaw: %v", err)
			}

			if len(gate.previews) != 1 {
				t.Fatalf("%s prompts = %d, want 1", method, len(gate.previews))
			}

			if !strings.HasPrefix(gate.previews[0], method+" "+srv.URL+"/payments") {
				t.Errorf("preview = %q, want method and url", gate.previews[0])
			}
		})
	}
}

func TestConfirmGateDeclineAbortsBeforeSending(t *testing.T) {
	var calls int

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls++
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()

	gate := &recordingConfirmer{err: errDeclined}

	c, _ := newTestClient(t, srv, testProfile(), WithConfirmer(gate))

	_, err := c.DoRaw(t.Context(), Request{Method: http.MethodPost, Path: "/payments", Body: map[string]any{"amount": 10}})
	if !errors.Is(err, errDeclined) {
		t.Fatalf("DoRaw = %v, want errDeclined", err)
	}

	if !strings.Contains(err.Error(), "POST /payments") {
		t.Errorf("error missing request context: %v", err)
	}

	if calls != 0 {
		t.Errorf("server calls = %d, want 0", calls)
	}
}

func TestConfirmGatePreviewMasksCardData(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()

	gate := &recordingConfirmer{}

	c, _ := newTestClient(t, srv, testProfile(), WithConfirmer(gate))

	body := map[string]any{
		"amount": map[string]any{"value": 10, "currency": "USD"},
		"payment_method": map[string]any{
			"card": map[string]any{"number": "4111111111111111", "security_code": "123"},
		},
	}

	if _, err := c.DoRaw(t.Context(), Request{Method: http.MethodPost, Path: "/payments", Body: body}); err != nil {
		t.Fatalf("DoRaw: %v", err)
	}

	preview := gate.previews[0]

	for _, secret := range []string{"4111111111111111", "123"} {
		if strings.Contains(preview, secret) {
			t.Errorf("preview leaks %q:\n%s", secret, preview)
		}
	}

	for _, want := range []string{"4111****", "USD"} {
		if !strings.Contains(preview, want) {
			t.Errorf("preview missing %q:\n%s", want, preview)
		}
	}
}

func TestConfirmGatePromptsOncePerCallDespiteRetries(t *testing.T) {
	var calls int

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls++
		if calls < 3 {
			w.WriteHeader(http.StatusInternalServerError)

			return
		}

		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()

	gate := &recordingConfirmer{}

	c, _ := newTestClient(t, srv, testProfile(), WithConfirmer(gate))
	c.sleep = func(context.Context, time.Duration) error { return nil }

	if _, err := c.DoRaw(t.Context(), Request{Method: http.MethodPost, Path: "/payments"}); err != nil {
		t.Fatalf("DoRaw: %v", err)
	}

	if len(gate.previews) != 1 {
		t.Errorf("prompts = %d, want 1 for %d attempts", len(gate.previews), calls)
	}
}

func TestPreviewRequestWithoutBody(t *testing.T) {
	got := previewRequest(http.MethodDelete, "https://api-sandbox.y.uno/v1/customers/1", nil)

	if got != "DELETE https://api-sandbox.y.uno/v1/customers/1" {
		t.Errorf("previewRequest = %q", got)
	}
}

func TestPreviewRequestNonJSONBodyIsTruncated(t *testing.T) {
	body := strings.Repeat("x", maxPreviewBody+100)

	got := previewRequest(http.MethodPost, "https://example.test/raw", []byte(body))

	if !strings.HasSuffix(got, "(truncated)") {
		t.Errorf("long body not truncated: %q", got[len(got)-30:])
	}

	if len([]rune(got)) > maxPreviewBody+100 {
		t.Errorf("preview length = %d, want capped", len([]rune(got)))
	}
}
