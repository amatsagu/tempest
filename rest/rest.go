// Package rest is a small, fast Discord REST client: connection reuse, json/v2,
// rate-limit awareness via the ratelimit package, and bounded retries.
package rest

import (
	"bytes"
	"context"
	"encoding/json/v2"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net"
	"net/http"
	"net/textproto"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/amatsagu/tempest/discord"
	"github.com/amatsagu/tempest/ratelimit"
)

// Canonicalized header names, computed once so header lookups never allocate.
var (
	headerRetryAfter = textproto.CanonicalMIMEHeaderKey("Retry-After")
	headerResetAfter = textproto.CanonicalMIMEHeaderKey("X-RateLimit-Reset-After")
	headerGlobal     = textproto.CanonicalMIMEHeaderKey("X-RateLimit-Global")
	headerScope      = textproto.CanonicalMIMEHeaderKey("X-RateLimit-Scope")
)

func headerValue(h http.Header, key string) string {
	if v := h[key]; len(v) > 0 {
		return v[0]
	}
	return ""
}

// DefaultBaseURL is the Discord API base used when Options.BaseURL is empty.
const DefaultBaseURL = "https://discord.com/api/v10"

const defaultUserAgent = "Tempest (https://github.com/amatsagu/tempest)"

// Defaults mirror upstream Tempest: a 10s request budget, a 5s dial, and up to
// five retries with a growing backoff.
const (
	defaultTimeout    = 10 * time.Second
	defaultMaxRetries = 5
	dialTimeout       = 5 * time.Second
	dialKeepAlive     = 30 * time.Second
	defaultIdleConns  = 32
)

// Options tunes a Client. Zero values are replaced with defaults.
type Options struct {
	Limiter *ratelimit.Limiter
	// BaseURL overrides DefaultBaseURL.
	BaseURL string
	// UserAgent overrides the default Tempest user agent.
	UserAgent string
	// MaxIdleConnsPerHost tunes the connection pool per Discord host.
	MaxIdleConnsPerHost int
	// Timeout is the whole-request budget; zero uses 10s.
	Timeout time.Duration
	// MaxRetries is how many times a 429 or 5xx is retried; zero uses 5.
	MaxRetries int
}

// Client is a rate-limit aware Discord REST client.
type Client struct {
	limiter    *ratelimit.Limiter
	http       *http.Client
	base       string
	token      string
	userAgent  string
	maxRetries int
}

// New builds a REST client; pass a shared limiter so several clients cooperate.
func New(token string, opts Options) *Client {
	base := opts.BaseURL
	if base == "" {
		base = DefaultBaseURL
	}
	base = strings.TrimSuffix(base, "/")

	ua := opts.UserAgent
	if ua == "" {
		ua = defaultUserAgent
	}

	limiter := opts.Limiter
	if limiter == nil {
		limiter = ratelimit.New(ratelimit.Options{})
	}

	idle := opts.MaxIdleConnsPerHost
	if idle <= 0 {
		idle = defaultIdleConns
	}

	retries := opts.MaxRetries
	if retries <= 0 {
		retries = defaultMaxRetries
	}

	timeout := opts.Timeout
	if timeout <= 0 {
		timeout = defaultTimeout
	}

	transport := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   dialTimeout,
			KeepAlive: dialKeepAlive,
		}).DialContext,
		MaxIdleConns:          idle * 4,
		MaxIdleConnsPerHost:   idle,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: time.Second,
		ForceAttemptHTTP2:     true,
	}

	return &Client{
		limiter: limiter,
		http: &http.Client{
			Transport: transport,
			Timeout:   timeout,
		},
		base:       base,
		token:      token,
		userAgent:  ua,
		maxRetries: retries,
	}
}

// HTTPError is returned for non-2xx responses.
type HTTPError struct {
	Body       []byte
	StatusCode int
}

func (e *HTTPError) Error() string {
	return fmt.Sprintf("discord rest: status %d: %s", e.StatusCode, strings.TrimSpace(string(e.Body)))
}

// File is one attachment in a multipart request; the data model defines it so
// interaction replies can carry files without importing this package.
type File = discord.File

// MaxFileBytes is the largest single attachment RequestWithFiles will send. Discord
// rejects larger uploads, so it is caught before the multipart body is built.
const MaxFileBytes = 20 << 20 // 20 MiB

// Request performs one REST call, honoring rate limits and retrying 429s and 5xx
// responses up to the configured limit, and returns the raw body. payload may be nil.
// A 5xx is retried only for idempotent methods, since a non-idempotent request may
// already have been processed.
func (c *Client) Request(ctx context.Context, method, path string, payload any, auditReason string) ([]byte, error) {
	body, err := marshalPayload(payload)
	if err != nil {
		return nil, err
	}

	return c.send(ctx, method, path, body, contentTypeJSON, auditReason)
}

// RequestHeaders is Request that also returns the response headers, for endpoints
// the caller drives from them, such as pagination or rate-limit introspection.
func (c *Client) RequestHeaders(ctx context.Context, method, path string, payload any, auditReason string) ([]byte, http.Header, error) {
	body, err := marshalPayload(payload)
	if err != nil {
		return nil, nil, err
	}

	resp, err := c.do(ctx, method, path, body, contentTypeJSON, auditReason)
	if err != nil {
		return nil, nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, nil, err
	}
	return data, resp.Header, nil
}

// RequestWithFiles is Request for a multipart body: the payload in a "payload_json"
// field and each file in a "files[n]" part. The body is built once, so retries resend
// identical bytes.
func (c *Client) RequestWithFiles(ctx context.Context, method, path string, payload any, files []File, auditReason string) ([]byte, error) {
	for _, file := range files {
		if len(file.Data) > MaxFileBytes {
			return nil, fmt.Errorf("discord rest: file %q is %d bytes, over the %d byte limit",
				file.Name, len(file.Data), MaxFileBytes)
		}
	}

	body, contentType, err := marshalMultipart(payload, files)
	if err != nil {
		return nil, err
	}

	return c.send(ctx, method, path, body, contentType, auditReason)
}

func (c *Client) send(ctx context.Context, method, path string, body []byte, contentType, auditReason string) ([]byte, error) {
	resp, err := c.do(ctx, method, path, body, contentType, auditReason)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	return io.ReadAll(resp.Body)
}

// RequestInto is Request with the response decoded straight into v, avoiding a raw
// copy for large payloads. Pass nil for v to discard the body.
func (c *Client) RequestInto(ctx context.Context, method, path string, payload any, auditReason string, v any) error {
	body, err := marshalPayload(payload)
	if err != nil {
		return err
	}

	resp, err := c.do(ctx, method, path, body, contentTypeJSON, auditReason)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()

	if v == nil {
		_, err = io.Copy(io.Discard, resp.Body)
		return err
	}
	return json.UnmarshalRead(resp.Body, v)
}

// do runs one request with rate limiting and retries, returning the live response
// for a 2xx call. The caller owns resp.Body; non-2xx bodies are drained into an *HTTPError.
func (c *Client) do(ctx context.Context, method, path string, body []byte, contentType, auditReason string) (*http.Response, error) {
	key := ratelimit.RouteKey(method, path)

	// Interaction responses are exempt from the global limit and must reach Discord
	// within three seconds, so they skip the global gate and cannot be delayed by a
	// global pause elsewhere.
	globalExempt := strings.HasPrefix(path, "/interactions/")

	for attempt := 0; ; attempt++ {
		var waitErr error
		if globalExempt {
			waitErr = c.limiter.WaitRoute(ctx, key)
		} else {
			waitErr = c.limiter.Wait(ctx, key)
		}

		if waitErr != nil {
			return nil, waitErr
		}

		req, err := c.newRequest(ctx, method, path, body, contentType, auditReason)
		if err != nil {
			return nil, err
		}

		resp, err := c.http.Do(req)
		if err != nil {
			return nil, err
		}

		c.limiter.Update(key, resp.Header)

		switch {
		case resp.StatusCode == http.StatusTooManyRequests:
			retryAfter := parseRetryAfter(resp.Header)
			if headerValue(resp.Header, headerGlobal) != "" || headerValue(resp.Header, headerScope) == "global" {
				c.limiter.Pause(retryAfter)
			}

			if attempt >= c.maxRetries {
				return nil, &HTTPError{StatusCode: resp.StatusCode, Body: drain(resp)}
			}
			discard(resp)
			if err := sleepCtx(ctx, retryAfter); err != nil {
				return nil, err
			}
			continue

		case resp.StatusCode >= 500:
			// A server error on a non-idempotent request may still have been
			// processed, so those are returned instead of replayed.
			if !idempotent(method) || attempt >= c.maxRetries {
				return nil, &HTTPError{StatusCode: resp.StatusCode, Body: drain(resp)}
			}
			discard(resp)
			if err := sleepCtx(ctx, backoff(attempt)); err != nil {
				return nil, err
			}
			continue

		case resp.StatusCode >= 300:
			return nil, &HTTPError{StatusCode: resp.StatusCode, Body: drain(resp)}
		}

		return resp, nil
	}
}

// newRequest builds one request. A non-nil body becomes a *bytes.Reader so net/http
// can set Content-Length instead of chunking; this is where the shared headers are set.
func (c *Client) newRequest(ctx context.Context, method, path string, body []byte, contentType, auditReason string) (*http.Request, error) {
	var reader io.Reader
	if body != nil {
		reader = bytes.NewReader(body)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.base+path, reader)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bot "+c.token)
	req.Header.Set("User-Agent", c.userAgent)
	if body != nil && contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}

	if auditReason != "" {
		req.Header.Set("X-Audit-Log-Reason", url.PathEscape(auditReason))
	}

	return req, nil
}

const contentTypeJSON = "application/json"

func marshalPayload(payload any) ([]byte, error) {
	if payload == nil {
		return nil, nil
	}
	return json.Marshal(payload)
}

// marshalMultipart builds Discord's multipart body (payload_json plus files[n]),
// buffered once so the retry loop can resend identical bytes.
func marshalMultipart(payload any, files []File) ([]byte, string, error) {
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)

	if payload != nil {
		data, err := json.Marshal(payload)
		if err != nil {
			return nil, "", err
		}

		if err := w.WriteField("payload_json", string(data)); err != nil {
			return nil, "", err
		}
	}

	for i, file := range files {
		header := textproto.MIMEHeader{}
		header.Set("Content-Disposition",
			`form-data; name="files[`+strconv.Itoa(i)+`]"; filename="`+escapeQuotes(file.Name)+`"`)
		contentType := file.ContentType
		if contentType == "" {
			contentType = "application/octet-stream"
		}
		header.Set("Content-Type", contentType)

		part, err := w.CreatePart(header)
		if err != nil {
			return nil, "", err
		}

		if _, err := part.Write(file.Data); err != nil {
			return nil, "", err
		}
	}

	if err := w.Close(); err != nil {
		return nil, "", err
	}
	return buf.Bytes(), w.FormDataContentType(), nil
}

// escapeQuotes escapes characters that would end a quoted multipart parameter early,
// so a file name cannot break out of the Content-Disposition header.
func escapeQuotes(s string) string {
	if !strings.ContainsAny(s, `"\`) {
		return s
	}

	var b strings.Builder
	b.Grow(len(s) + 2)
	for i := 0; i < len(s); i++ {
		if c := s[i]; c == '"' || c == '\\' {
			b.WriteByte('\\')
		}
		b.WriteByte(s[i])
	}
	return b.String()
}

func drain(resp *http.Response) []byte {
	data, _ := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	return data
}

func discard(resp *http.Response) {
	_, _ = io.Copy(io.Discard, resp.Body)
	_ = resp.Body.Close()
}

// Limiter exposes the shared limiter for callers that need to coordinate custom
// requests or react to global limits.
func (c *Client) Limiter() *ratelimit.Limiter { return c.limiter }

func parseRetryAfter(header http.Header) time.Duration {
	if v := headerValue(header, headerRetryAfter); v != "" {
		if secs, err := strconv.ParseFloat(v, 64); err == nil && secs >= 0 {
			return time.Duration(secs * float64(time.Second))
		}
	}

	if v := headerValue(header, headerResetAfter); v != "" {
		if secs, err := strconv.ParseFloat(v, 64); err == nil && secs >= 0 {
			return time.Duration(secs * float64(time.Second))
		}
	}
	return 250 * time.Millisecond
}

// idempotent reports whether a method may be replayed after a server error, where
// the request might already have been processed.
func idempotent(method string) bool {
	switch method {
	case http.MethodGet, http.MethodHead, http.MethodOptions, http.MethodTrace, http.MethodPut, http.MethodDelete:
		return true
	default:
		return false
	}
}

// backoff is the wait before retrying a retryable failure, growing each attempt
// from 250ms so a failing endpoint is not hammered.
func backoff(attempt int) time.Duration {
	d := time.Duration(attempt+1) * 250 * time.Millisecond
	if d > 5*time.Second {
		d = 5 * time.Second
	}
	return d
}

func sleepCtx(ctx context.Context, d time.Duration) error {
	if d <= 0 {
		return ctx.Err()
	}

	timer := time.NewTimer(d)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

// ErrEmptyBody is returned by helpers that expect a body but receive none.
var ErrEmptyBody = errors.New("discord rest: empty response body")
