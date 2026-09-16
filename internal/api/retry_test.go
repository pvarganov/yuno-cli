package api

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

func TestRetryOnRateLimitAndServerErrors(t *testing.T) {
	tests := []struct {
		name   string
		status int
	}{
		{"rate limited", http.StatusTooManyRequests},
		{"server error", http.StatusInternalServerError},
		{"bad gateway", http.StatusBadGateway},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var calls atomic.Int32

			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				if calls.Add(1) < 3 {
					w.WriteHeader(tt.status)

					return
				}

				_, _ = w.Write([]byte(`{"id":"ok"}`))
			}))
			defer srv.Close()

			c, slept := newTestClient(t, srv, testProfile())

			data, err := c.DoRaw(t.Context(), Request{Method: http.MethodGet, Path: "/routing"})
			if err != nil {
				t.Fatalf("DoRaw: %v", err)
			}

			if string(data) != `{"id":"ok"}` {
				t.Errorf("body = %q", data)
			}

			if calls.Load() != 3 {
				t.Errorf("calls = %d, want 3", calls.Load())
			}

			if len(*slept) != 2 {
				t.Errorf("sleeps = %d, want 2", len(*slept))
			}
		})
	}
}

func TestRetryExhausted(t *testing.T) {
	var calls atomic.Int32

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls.Add(1)
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer srv.Close()

	c, slept := newTestClient(t, srv, testProfile())

	_, err := c.DoRaw(t.Context(), Request{Method: http.MethodGet, Path: "/routing"})
	if !errors.Is(err, ErrServer) {
		t.Fatalf("err = %v, want ErrServer", err)
	}

	if calls.Load() != 4 {
		t.Errorf("calls = %d, want 4 (1 + 3 retries)", calls.Load())
	}

	if len(*slept) != 3 {
		t.Errorf("sleeps = %d, want 3", len(*slept))
	}
}

func TestNoRetryOnClientErrors(t *testing.T) {
	for _, status := range []int{http.StatusBadRequest, http.StatusUnauthorized, http.StatusForbidden, http.StatusNotFound} {
		t.Run(fmt.Sprint(status), func(t *testing.T) {
			var calls atomic.Int32

			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				calls.Add(1)
				w.WriteHeader(status)
			}))
			defer srv.Close()

			c, slept := newTestClient(t, srv, testProfile())

			if _, err := c.DoRaw(t.Context(), Request{Method: http.MethodGet, Path: "/x"}); err == nil {
				t.Fatal("want error, got nil")
			}

			if calls.Load() != 1 {
				t.Errorf("calls = %d, want 1", calls.Load())
			}

			if len(*slept) != 0 {
				t.Errorf("sleeps = %d, want 0", len(*slept))
			}
		})
	}
}

func TestRetryHonoursRetryAfter(t *testing.T) {
	var calls atomic.Int32

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if calls.Add(1) == 1 {
			w.Header().Set("Retry-After", "7")
			w.WriteHeader(http.StatusTooManyRequests)

			return
		}

		_, _ = w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	c, slept := newTestClient(t, srv, testProfile())

	if _, err := c.DoRaw(t.Context(), Request{Method: http.MethodGet, Path: "/x"}); err != nil {
		t.Fatalf("DoRaw: %v", err)
	}

	if len(*slept) != 1 || (*slept)[0] != 7*time.Second {
		t.Errorf("sleeps = %v, want [7s]", *slept)
	}
}

func TestRetryDisabled(t *testing.T) {
	var calls atomic.Int32

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls.Add(1)
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	c, _ := newTestClient(t, srv, testProfile(), WithMaxRetries(0))

	if _, err := c.DoRaw(t.Context(), Request{Method: http.MethodGet, Path: "/x"}); err == nil {
		t.Fatal("want error, got nil")
	}

	if calls.Load() != 1 {
		t.Errorf("calls = %d, want 1", calls.Load())
	}
}

func TestRetryStopsWhenContextCancelledDuringBackoff(t *testing.T) {
	var calls atomic.Int32

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls.Add(1)
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	c, _ := NewClient(testProfile(), WithEndpoint(srv.URL))

	ctx, cancel := context.WithCancel(t.Context())

	c.sleep = func(_ context.Context, _ time.Duration) error {
		cancel()

		return context.Canceled
	}

	_, err := c.DoRaw(ctx, Request{Method: http.MethodGet, Path: "/x"})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("err = %v, want context.Canceled", err)
	}

	if calls.Load() != 1 {
		t.Errorf("calls = %d, want 1", calls.Load())
	}
}

func TestBackoff(t *testing.T) {
	t.Run("server hint wins and is capped", func(t *testing.T) {
		if got := backoff(1, 5*time.Second); got != 5*time.Second {
			t.Errorf("got %v, want 5s", got)
		}

		if got := backoff(1, 10*time.Minute); got != maxBackoff {
			t.Errorf("got %v, want %v", got, maxBackoff)
		}
	})

	t.Run("grows and stays capped", func(t *testing.T) {
		for attempt := 1; attempt <= 20; attempt++ {
			got := backoff(attempt, 0)
			if got <= 0 || got > maxBackoff {
				t.Fatalf("attempt %d: got %v, want (0, %v]", attempt, got, maxBackoff)
			}
		}
	})

	t.Run("jitter varies", func(t *testing.T) {
		seen := map[time.Duration]bool{}
		for range 50 {
			seen[backoff(4, 0)] = true
		}

		if len(seen) < 2 {
			t.Error("backoff produced no jitter")
		}
	})
}

func TestRetryablePredicate(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{"nil", nil, false},
		{"transport", &transportError{err: errors.New("dial tcp")}, true},
		{"wrapped transport", fmt.Errorf("x: %w", &transportError{err: errors.New("dial")}), true},
		{"429", &Error{StatusCode: http.StatusTooManyRequests}, true},
		{"503", &Error{StatusCode: http.StatusServiceUnavailable}, true},
		{"404", &Error{StatusCode: http.StatusNotFound}, false},
		{"canceled", context.Canceled, false},
		{"deadline", context.DeadlineExceeded, false},
		{"plain", errors.New("boom"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := retryable(tt.err); got != tt.want {
				t.Errorf("retryable(%v) = %v, want %v", tt.err, got, tt.want)
			}
		})
	}
}

func TestSleepContext(t *testing.T) {
	if err := sleepContext(t.Context(), time.Millisecond); err != nil {
		t.Errorf("sleepContext: %v", err)
	}

	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	if err := sleepContext(ctx, time.Hour); !errors.Is(err, context.Canceled) {
		t.Errorf("err = %v, want context.Canceled", err)
	}
}
