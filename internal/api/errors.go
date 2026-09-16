package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// Sentinel errors, comparable with errors.Is against an *Error.
var (
	ErrUnauthorized = errors.New("unauthorized")
	ErrForbidden    = errors.New("forbidden")
	ErrNotFound     = errors.New("not found")
	ErrRateLimited  = errors.New("rate limited")
	ErrValidation   = errors.New("validation failed")
	ErrServer       = errors.New("server error")
)

// maxBodyInError caps how much of an unparseable error body is kept.
const maxBodyInError = 512

// Error is a Yuno API error response.
type Error struct {
	StatusCode int
	Code       string
	Message    string
	Type       string
	Body       string
	// RetryAfter carries the parsed Retry-After header, zero when absent.
	RetryAfter time.Duration
}

var statusSummaries = map[int]string{
	http.StatusBadRequest:          "bad request",
	http.StatusUnauthorized:        "unauthorized: check your api keys",
	http.StatusForbidden:           "forbidden: the api key lacks the required scope",
	http.StatusNotFound:            "not found",
	http.StatusConflict:            "conflict",
	http.StatusUnprocessableEntity: "validation failed",
	http.StatusTooManyRequests:     "rate limited",
}

var statusSentinels = map[int]error{
	http.StatusBadRequest:          ErrValidation,
	http.StatusUnauthorized:        ErrUnauthorized,
	http.StatusForbidden:           ErrForbidden,
	http.StatusNotFound:            ErrNotFound,
	http.StatusUnprocessableEntity: ErrValidation,
	http.StatusTooManyRequests:     ErrRateLimited,
}

func (e *Error) Error() string {
	var b strings.Builder

	b.WriteString(summary(e.StatusCode))

	if e.Code != "" {
		fmt.Fprintf(&b, " [%s]", e.Code)
	}

	switch {
	case e.Message != "":
		fmt.Fprintf(&b, ": %s", e.Message)
	case e.Code == "" && e.Body != "":
		fmt.Fprintf(&b, ": %s", e.Body)
	}

	return b.String()
}

// Unwrap maps the status code onto a sentinel so callers can use errors.Is.
func (e *Error) Unwrap() error {
	if sentinel, ok := statusSentinels[e.StatusCode]; ok {
		return sentinel
	}

	if e.StatusCode >= http.StatusInternalServerError {
		return ErrServer
	}

	return nil
}

// Retryable reports whether the request may be sent again.
func (e *Error) Retryable() bool {
	return e.StatusCode == http.StatusTooManyRequests || e.StatusCode >= http.StatusInternalServerError
}

func summary(status int) string {
	if s, ok := statusSummaries[status]; ok {
		return s
	}

	if status >= http.StatusInternalServerError {
		return fmt.Sprintf("server error %d", status)
	}

	return fmt.Sprintf("api error %d", status)
}

// errorBody is the shape Yuno uses for error responses. The code is sometimes a
// string and sometimes a number, so it is decoded loosely.
type errorBody struct {
	Code    json.RawMessage `json:"code"`
	Message string          `json:"message"`
	Type    string          `json:"type"`
	Error   string          `json:"error"`
}

func newError(resp *http.Response, body []byte) *Error {
	apiErr := &Error{
		StatusCode: resp.StatusCode,
		Body:       truncate(strings.TrimSpace(string(body)), maxBodyInError),
		RetryAfter: parseRetryAfter(resp.Header.Get("Retry-After")),
	}

	var parsed errorBody
	if err := json.Unmarshal(body, &parsed); err != nil {
		return apiErr
	}

	apiErr.Code = rawToString(parsed.Code)
	apiErr.Message = parsed.Message
	apiErr.Type = parsed.Type

	if apiErr.Message == "" {
		apiErr.Message = parsed.Error
	}

	return apiErr
}

func rawToString(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}

	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		return s
	}

	return strings.Trim(string(raw), `"`)
}

func parseRetryAfter(value string) time.Duration {
	if value == "" {
		return 0
	}

	if seconds, err := strconv.Atoi(value); err == nil {
		if seconds < 0 {
			return 0
		}

		return time.Duration(seconds) * time.Second
	}

	if date, err := http.ParseTime(value); err == nil {
		if d := time.Until(date); d > 0 {
			return d
		}
	}

	return 0
}

func truncate(s string, limit int) string {
	if len(s) <= limit {
		return s
	}

	return s[:limit] + "…"
}
