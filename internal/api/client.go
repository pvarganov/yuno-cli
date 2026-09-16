// Package api implements the HTTP client for the Yuno REST API: authentication
// headers, typed errors and retries.
package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/pvarganov/yuno-cli/internal/config"
)

// Request headers required by the Yuno API.
const (
	HeaderPublicAPIKey     = "public-api-key"
	HeaderPrivateSecretKey = "private-secret-key" //nolint:gosec // header name, not a credential
	HeaderAccountCode      = "X-Account-Code"
	HeaderOrganizationCode = "X-Organization-Code"
	HeaderIdempotencyKey   = "X-Idempotency-Key"
)

const (
	defaultTimeout    = 30 * time.Second
	defaultMaxRetries = 3
	userAgent         = "yuno-cli"
)

// Client talks to one Yuno environment with one set of credentials.
type Client struct {
	profile    config.Profile
	endpoint   string
	httpClient *http.Client
	maxRetries int
	// idempotencyKey pins the X-Idempotency-Key of every mutating request when
	// the caller wants to retry a known call instead of issuing a new one.
	idempotencyKey string
	verbose        bool
	verboseOut     io.Writer
	// confirmer gates every mutating request; nil means no prompting.
	confirmer Confirmer
	// sleep is swapped out in tests to keep retry cases fast.
	sleep func(context.Context, time.Duration) error
}

// Option customises a Client.
type Option func(*Client)

// WithEndpoint overrides the base URL derived from the profile environment.
func WithEndpoint(endpoint string) Option {
	return func(c *Client) {
		if endpoint != "" {
			c.endpoint = strings.TrimRight(endpoint, "/")
		}
	}
}

// WithHTTPClient replaces the underlying *http.Client.
func WithHTTPClient(hc *http.Client) Option {
	return func(c *Client) {
		if hc != nil {
			c.httpClient = hc
		}
	}
}

// WithMaxRetries sets how many extra attempts a retryable failure gets.
func WithMaxRetries(n int) Option {
	return func(c *Client) {
		if n >= 0 {
			c.maxRetries = n
		}
	}
}

// WithTimeout sets the per-request timeout.
func WithTimeout(d time.Duration) Option {
	return func(c *Client) {
		if d > 0 {
			c.httpClient.Timeout = d
		}
	}
}

// WithIdempotencyKey pins the key sent on mutating requests instead of
// generating a fresh UUID per call.
func WithIdempotencyKey(key string) Option {
	return func(c *Client) {
		if key != "" {
			c.idempotencyKey = key
		}
	}
}

// WithVerbose dumps every request and response to w, with credentials and card
// data masked. A nil writer disables the dump.
func WithVerbose(w io.Writer) Option {
	return func(c *Client) {
		c.verbose = w != nil
		c.verboseOut = verboseWriter(w)
	}
}

// NewClient validates the profile and builds a client for its environment.
func NewClient(profile *config.Profile, opts ...Option) (*Client, error) {
	if err := profile.Validate(); err != nil {
		return nil, fmt.Errorf("profile: %w", err)
	}

	endpoint, err := profile.Endpoint()
	if err != nil {
		return nil, fmt.Errorf("resolve endpoint: %w", err)
	}

	c := &Client{
		profile:    *profile,
		endpoint:   endpoint,
		httpClient: &http.Client{Timeout: defaultTimeout},
		maxRetries: defaultMaxRetries,
		verboseOut: io.Discard,
		sleep:      sleepContext,
	}

	for _, opt := range opts {
		opt(c)
	}

	return c, nil
}

// Endpoint returns the base URL the client sends requests to.
func (c *Client) Endpoint() string { return c.endpoint }

// AccountID returns the account the profile is scoped to, empty when the
// profile does not pin one.
func (c *Client) AccountID() string { return c.profile.AccountID }

// Request describes one API call.
type Request struct {
	Method string
	Path   string
	Query  url.Values
	// Body is marshalled to JSON. Raw []byte is sent verbatim.
	Body any
	// Headers are merged on top of the auth headers.
	Headers map[string]string
}

// DoRaw performs the request and returns the raw response body.
func (c *Client) DoRaw(ctx context.Context, req Request) ([]byte, error) {
	payload, err := encodeBody(req.Body)
	if err != nil {
		return nil, err
	}

	target, err := c.resolveURL(req.Path, req.Query)
	if err != nil {
		return nil, err
	}

	if err := c.confirm(req.Method, req.Path, target, payload); err != nil {
		return nil, err
	}

	// One key per logical call: retries of the same call must not create a
	// second payment.
	idemKey := c.idempotencyKeyFor(req.Method)

	var lastErr error

	for attempt := 0; attempt <= c.maxRetries; attempt++ {
		if attempt > 0 {
			if err := c.sleep(ctx, backoff(attempt, retryAfterOf(lastErr))); err != nil {
				return nil, fmt.Errorf("%s %s: %w", req.Method, req.Path, err)
			}
		}

		data, err := c.attempt(ctx, req, target, payload, idemKey)
		if err == nil {
			return data, nil
		}

		lastErr = err

		if !retryable(err) {
			return nil, err
		}
	}

	return nil, fmt.Errorf("%s %s: giving up after %d attempts: %w",
		req.Method, req.Path, c.maxRetries+1, lastErr)
}

func (c *Client) attempt(ctx context.Context, req Request, target string, payload []byte, idemKey string) ([]byte, error) {
	var body io.Reader
	if payload != nil {
		body = bytes.NewReader(payload)
	}

	httpReq, err := http.NewRequestWithContext(ctx, req.Method, target, body)
	if err != nil {
		return nil, fmt.Errorf("build request %s %s: %w", req.Method, req.Path, err)
	}

	c.setHeaders(httpReq, payload != nil, idemKey, req.Headers)
	c.dumpRequest(httpReq)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return nil, fmt.Errorf("%s %s: %w", req.Method, req.Path, ctxErr)
		}

		return nil, &transportError{err: fmt.Errorf("%s %s: %w", req.Method, req.Path, err)}
	}
	defer func() { _ = resp.Body.Close() }()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, &transportError{err: fmt.Errorf("read %s %s response: %w", req.Method, req.Path, err)}
	}

	c.dumpResponse(resp, data)

	if resp.StatusCode >= http.StatusBadRequest {
		return nil, newError(resp, data)
	}

	return data, nil
}

func (c *Client) setHeaders(req *http.Request, hasBody bool, idemKey string, extra map[string]string) {
	req.Header.Set(HeaderPublicAPIKey, c.profile.PublicAPIKey)
	req.Header.Set(HeaderPrivateSecretKey, c.profile.PrivateSecretKey)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", userAgent)

	if hasBody {
		req.Header.Set("Content-Type", "application/json")
	}

	if idemKey != "" {
		req.Header.Set(HeaderIdempotencyKey, idemKey)
	}

	if c.profile.AccountCode != "" {
		req.Header.Set(HeaderAccountCode, c.profile.AccountCode)
	}

	if c.profile.OrganizationCode != "" {
		req.Header.Set(HeaderOrganizationCode, c.profile.OrganizationCode)
	}

	for k, v := range extra {
		req.Header.Set(k, v)
	}
}

func (c *Client) resolveURL(path string, query url.Values) (string, error) {
	u, err := url.Parse(c.endpoint + "/" + strings.TrimPrefix(path, "/"))
	if err != nil {
		return "", fmt.Errorf("build url for %s: %w", path, err)
	}

	if len(query) > 0 {
		u.RawQuery = query.Encode()
	}

	return u.String(), nil
}

func encodeBody(body any) ([]byte, error) {
	switch v := body.(type) {
	case nil:
		return nil, nil
	case []byte:
		return v, nil
	case json.RawMessage:
		return v, nil
	}

	data, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("encode request body: %w", err)
	}

	return data, nil
}

func sleepContext(ctx context.Context, d time.Duration) error {
	timer := time.NewTimer(d)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

// Do performs the request and decodes the JSON response into T. An empty body
// leaves the zero value untouched, which is what Yuno returns for 204s.
func Do[T any](ctx context.Context, c *Client, req Request) (T, error) {
	var out T

	data, err := c.DoRaw(ctx, req)
	if err != nil {
		return out, err
	}

	if len(bytes.TrimSpace(data)) == 0 {
		return out, nil
	}

	if err := json.Unmarshal(data, &out); err != nil {
		return out, fmt.Errorf("decode %s %s response: %w", req.Method, req.Path, err)
	}

	return out, nil
}
