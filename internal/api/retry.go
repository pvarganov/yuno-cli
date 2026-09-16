package api

import (
	"context"
	"errors"
	"math/rand/v2"
	"time"
)

const (
	baseBackoff = 500 * time.Millisecond
	maxBackoff  = 60 * time.Second
	maxShift    = 16
)

// transportError marks a network-level failure, which is always safe to retry.
type transportError struct{ err error }

func (e *transportError) Error() string { return e.err.Error() }

func (e *transportError) Unwrap() error { return e.err }

// retryable reports whether err justifies another attempt. Context errors never do.
func retryable(err error) bool {
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return false
	}

	var transport *transportError
	if errors.As(err, &transport) {
		return true
	}

	var apiErr *Error
	if errors.As(err, &apiErr) {
		return apiErr.Retryable()
	}

	return false
}

// retryAfter extracts the server-requested delay, zero when none was sent.
func retryAfterOf(err error) time.Duration {
	var apiErr *Error
	if errors.As(err, &apiErr) {
		return apiErr.RetryAfter
	}

	return 0
}

// backoff returns the delay before attempt n (1-based), jittered and capped.
// A server-supplied Retry-After wins over the computed value.
func backoff(attempt int, serverHint time.Duration) time.Duration {
	if serverHint > 0 {
		return min(serverHint, maxBackoff)
	}

	shift := min(attempt-1, maxShift)
	delay := baseBackoff << shift

	if delay > maxBackoff {
		delay = maxBackoff
	}

	// Full jitter in [delay/2, delay) keeps retries from synchronising.
	jitter := time.Duration(rand.Int64N(int64(delay / 2))) //nolint:gosec // jitter, not a secret

	return delay/2 + jitter
}
