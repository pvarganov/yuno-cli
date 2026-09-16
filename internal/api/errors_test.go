package api

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestErrorDecodedFromBody(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnprocessableEntity)
		_, _ = w.Write([]byte(`{"code":"VALIDATION_ERROR","message":"amount is required","type":"REQUEST_ERROR"}`))
	}))
	defer srv.Close()

	c, _ := newTestClient(t, srv, testProfile())

	_, err := c.DoRaw(t.Context(), Request{Method: http.MethodPost, Path: "/payments", Body: map[string]any{}})

	var apiErr *Error
	if !errors.As(err, &apiErr) {
		t.Fatalf("err = %v, want *api.Error", err)
	}

	if apiErr.StatusCode != http.StatusUnprocessableEntity {
		t.Errorf("status = %d", apiErr.StatusCode)
	}

	if apiErr.Code != "VALIDATION_ERROR" || apiErr.Message != "amount is required" || apiErr.Type != "REQUEST_ERROR" {
		t.Errorf("parsed = %+v", apiErr)
	}

	if !errors.Is(err, ErrValidation) {
		t.Error("want errors.Is ErrValidation")
	}
}

func TestUnauthorizedNotRetried(t *testing.T) {
	var calls int

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls++

		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	c, slept := newTestClient(t, srv, testProfile())

	_, err := c.DoRaw(t.Context(), Request{Method: http.MethodGet, Path: "/routing"})
	if !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("err = %v, want ErrUnauthorized", err)
	}

	if !strings.Contains(err.Error(), "unauthorized: check your api keys") {
		t.Errorf("message = %q", err.Error())
	}

	if calls != 1 {
		t.Errorf("calls = %d, want 1", calls)
	}

	if len(*slept) != 0 {
		t.Errorf("sleeps = %d, want 0", len(*slept))
	}
}

func TestErrorSentinels(t *testing.T) {
	tests := []struct {
		status int
		want   error
	}{
		{http.StatusBadRequest, ErrValidation},
		{http.StatusUnauthorized, ErrUnauthorized},
		{http.StatusForbidden, ErrForbidden},
		{http.StatusNotFound, ErrNotFound},
		{http.StatusUnprocessableEntity, ErrValidation},
		{http.StatusTooManyRequests, ErrRateLimited},
		{http.StatusInternalServerError, ErrServer},
		{http.StatusBadGateway, ErrServer},
	}

	for _, tt := range tests {
		err := &Error{StatusCode: tt.status}
		if !errors.Is(err, tt.want) {
			t.Errorf("status %d: want %v", tt.status, tt.want)
		}
	}

	if got := (&Error{StatusCode: http.StatusTeapot}).Unwrap(); got != nil {
		t.Errorf("unmapped status unwrapped to %v, want nil", got)
	}
}

func TestErrorMessageFormats(t *testing.T) {
	tests := []struct {
		name string
		err  *Error
		want string
	}{
		{"code and message", &Error{StatusCode: 404, Code: "NOT_FOUND", Message: "no such payment"}, "not found [NOT_FOUND]: no such payment"},
		{"status only", &Error{StatusCode: 409}, "conflict"},
		{"raw body fallback", &Error{StatusCode: 500, Body: "<html>"}, "server error 500: <html>"},
		{"unmapped status", &Error{StatusCode: 418}, "api error 418"},
		{"forbidden mentions scope", &Error{StatusCode: 403}, "forbidden: the api key lacks the required scope"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.err.Error(); got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestNewErrorNonJSONBody(t *testing.T) {
	resp := &http.Response{StatusCode: http.StatusBadGateway, Header: http.Header{}}

	err := newError(resp, []byte("upstream unavailable"))
	if err.Code != "" || err.Message != "" {
		t.Errorf("parsed fields from non-JSON body: %+v", err)
	}

	if err.Body != "upstream unavailable" {
		t.Errorf("body = %q", err.Body)
	}
}

func TestNewErrorMasksSensitiveFieldsInTheRawBody(t *testing.T) {
	resp := &http.Response{StatusCode: http.StatusBadGateway, Header: http.Header{}}

	err := newError(resp, []byte(`{"upstream":"rejected card","number":"4111111111111111"}`))
	if strings.Contains(err.Body, "4111111111111111") {
		t.Errorf("body leaks the card number unmasked: %q", err.Body)
	}

	if !strings.Contains(err.Error(), "rejected card") {
		t.Errorf("expected the non-sensitive field to survive masking, got: %q", err.Error())
	}
}

func TestNewErrorNumericCodeAndErrorField(t *testing.T) {
	resp := &http.Response{StatusCode: http.StatusBadRequest, Header: http.Header{}}

	err := newError(resp, []byte(`{"code":1234,"error":"bad input"}`))
	if err.Code != "1234" {
		t.Errorf("code = %q, want 1234", err.Code)
	}

	if err.Message != "bad input" {
		t.Errorf("message = %q, want bad input", err.Message)
	}
}

func TestNewErrorTruncatesLongBody(t *testing.T) {
	resp := &http.Response{StatusCode: http.StatusInternalServerError, Header: http.Header{}}

	err := newError(resp, []byte(strings.Repeat("x", maxBodyInError+100)))
	if len([]rune(err.Body)) != maxBodyInError+1 {
		t.Errorf("body length = %d, want %d", len([]rune(err.Body)), maxBodyInError+1)
	}
}

func TestParseRetryAfter(t *testing.T) {
	tests := []struct {
		name  string
		value string
		want  time.Duration
	}{
		{"empty", "", 0},
		{"seconds", "12", 12 * time.Second},
		{"negative", "-5", 0},
		{"garbage", "soon", 0},
		{"past date", "Mon, 02 Jan 2006 15:04:05 GMT", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := parseRetryAfter(tt.value); got != tt.want {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}

	future := time.Now().Add(30 * time.Second).UTC().Format(http.TimeFormat)
	if got := parseRetryAfter(future); got <= 0 || got > 31*time.Second {
		t.Errorf("future date = %v, want ~30s", got)
	}
}

func TestRetryAfterOf(t *testing.T) {
	if got := retryAfterOf(errors.New("plain")); got != 0 {
		t.Errorf("got %v, want 0", got)
	}

	if got := retryAfterOf(&Error{RetryAfter: 3 * time.Second}); got != 3*time.Second {
		t.Errorf("got %v, want 3s", got)
	}
}
