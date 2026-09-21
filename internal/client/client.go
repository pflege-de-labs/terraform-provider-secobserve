// Package client is a hand-written transport layer for the SecObserve REST API.
//
// The generated OpenAPI models (see ../../api) cover request and response
// shapes; everything that the schema cannot express lives here: the APIToken
// auth scheme, page-number pagination, throttle backoff and the mapping of
// Django REST Framework error bodies onto typed errors.
package client

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	// SecObserve accepts both user and product API tokens under this prefix.
	authPrefix = "APIToken"

	// SchemaVersion is the SecObserve release the vendored OpenAPI schema was
	// generated from. Compared against /api/status/version/ at configure time.
	SchemaVersion = "1.59.2"

	defaultTimeout = 30 * time.Second

	// The backend throttles authenticated callers at 100 req/s.
	maxRetries = 4
)

// Config holds everything needed to talk to a SecObserve instance.
type Config struct {
	BaseURL   string
	APIToken  string
	Insecure  bool
	Timeout   time.Duration
	UserAgent string
}

// Client is a SecObserve API client. Safe for concurrent use.
type Client struct {
	baseURL   *url.URL
	token     string
	userAgent string
	http      *http.Client
}

// New validates the configuration and builds a Client. It performs no I/O.
func New(cfg Config) (*Client, error) {
	if cfg.BaseURL == "" {
		return nil, fmt.Errorf("base_url must not be empty")
	}
	if cfg.APIToken == "" {
		return nil, fmt.Errorf("api_token must not be empty")
	}

	// Trailing slashes would double up against the "api/..." reference paths.
	parsed, err := url.Parse(strings.TrimRight(cfg.BaseURL, "/"))
	if err != nil {
		return nil, fmt.Errorf("base_url is not a valid URL: %w", err)
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return nil, fmt.Errorf("base_url must use http or https, got %q", parsed.Scheme)
	}

	timeout := cfg.Timeout
	if timeout <= 0 {
		timeout = defaultTimeout
	}

	transport := http.DefaultTransport.(*http.Transport).Clone()
	if cfg.Insecure {
		transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true} //nolint:gosec // opt-in, dev only
	}

	userAgent := cfg.UserAgent
	if userAgent == "" {
		userAgent = "terraform-provider-secobserve"
	}

	return &Client{
		baseURL:   parsed,
		token:     cfg.APIToken,
		userAgent: userAgent,
		http:      &http.Client{Timeout: timeout, Transport: transport},
	}, nil
}

// Request describes a single API call.
type Request struct {
	Method string
	// Path is relative to the base URL, e.g. "api/products/1/".
	Path  string
	Query url.Values
	// Body is JSON-encoded when non-nil.
	Body any
	// Out receives the decoded response body when non-nil.
	Out any
}

// Do executes req, retrying on throttling and transient upstream failures.
func (c *Client) Do(ctx context.Context, req Request) error {
	var lastErr error

	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			delay := backoff(attempt, lastErr)
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(delay):
			}
		}

		err := c.do(ctx, req)
		if err == nil {
			return nil
		}
		if !isRetryable(err) {
			return err
		}
		lastErr = err
	}

	return lastErr
}

func (c *Client) do(ctx context.Context, req Request) error {
	endpoint := c.baseURL.JoinPath(req.Path)
	if req.Query != nil {
		endpoint.RawQuery = req.Query.Encode()
	}

	var body io.Reader
	if req.Body != nil {
		encoded, err := json.Marshal(req.Body)
		if err != nil {
			return fmt.Errorf("encoding request body for %s %s: %w", req.Method, req.Path, err)
		}
		body = bytes.NewReader(encoded)
	}

	httpReq, err := http.NewRequestWithContext(ctx, req.Method, endpoint.String(), body)
	if err != nil {
		return fmt.Errorf("building request for %s %s: %w", req.Method, req.Path, err)
	}

	httpReq.Header.Set("Authorization", authPrefix+" "+c.token)
	httpReq.Header.Set("Accept", "application/json")
	httpReq.Header.Set("User-Agent", c.userAgent)
	if req.Body != nil {
		httpReq.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.http.Do(httpReq)
	if err != nil {
		return fmt.Errorf("calling %s %s: %w", req.Method, req.Path, err)
	}
	defer func() {
		_, _ = io.Copy(io.Discard, resp.Body)
		_ = resp.Body.Close()
	}()

	if resp.StatusCode >= 400 {
		return newAPIError(req.Method, req.Path, resp)
	}

	// 204 responses are common: approvals, member deletions, bulk actions.
	if req.Out == nil || resp.StatusCode == http.StatusNoContent {
		return nil
	}

	if err := json.NewDecoder(resp.Body).Decode(req.Out); err != nil {
		return fmt.Errorf("decoding response of %s %s: %w", req.Method, req.Path, err)
	}
	return nil
}

// Get issues a GET and decodes the body into out.
func (c *Client) Get(ctx context.Context, path string, query url.Values, out any) error {
	return c.Do(ctx, Request{Method: http.MethodGet, Path: path, Query: query, Out: out})
}

// Post issues a POST with a JSON body.
func (c *Client) Post(ctx context.Context, path string, body, out any) error {
	return c.Do(ctx, Request{Method: http.MethodPost, Path: path, Body: body, Out: out})
}

// Patch issues a PATCH with a JSON body. SecObserve accepts partial updates
// everywhere, so the provider never needs PUT.
func (c *Client) Patch(ctx context.Context, path string, body, out any) error {
	return c.Do(ctx, Request{Method: http.MethodPatch, Path: path, Body: body, Out: out})
}

// Delete issues a DELETE. Products and product groups additionally require a
// name confirmation query parameter; pass it via query.
func (c *Client) Delete(ctx context.Context, path string, query url.Values) error {
	return c.Do(ctx, Request{Method: http.MethodDelete, Path: path, Query: query})
}

func backoff(attempt int, err error) time.Duration {
	// Honour Retry-After when the backend throttled us.
	var apiErr *APIError
	if errors.As(err, &apiErr) && apiErr.RetryAfter > 0 {
		return apiErr.RetryAfter
	}
	return time.Duration(1<<attempt) * 100 * time.Millisecond
}
