// Package rest provides a standalone, rate-limit-aware Discord REST HTTP client.
// It can be used independently of the rest of Tempest and requires no other internal packages.
package rest

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json/jsontext"
	json "encoding/json/v2"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

const (
	ContentTypeJSON         = "application/json"
	ContentTypeOctetStream  = "application/octet-stream"
	multipartJSONPartHeader = `form-data; name="payload_json"`

	maxPooledJSONBufferCap      = 64 * 1024
	maxPooledMultipartBufferCap = 512 * 1024
)

// Attachment uploaded via RequestWithFiles or CustomRequest.
type File struct {
	Reader      io.Reader
	Name        string
	ContentType string // Defaults to "application/octet-stream".
}

// Options for custom, raw, or header-customized REST calls via Do.
type CustomRequest struct {
	Headers        http.Header
	Body           any // nil, jsontext.Value, []byte, string, io.Reader/io.ReadSeeker, or any JSON value.
	Into           any // nil, *[]byte, io.Writer, or JSON pointer target.
	Method         string
	Route          string
	ContentType    string // Inferred from Body when empty.
	AuditLogReason string // Optional X-Audit-Log-Reason header (URL-escaped automatically).
	Files          []File // Multipart attachments; encodes Body as payload_json when non-empty.
}

// Configuration for RestClient. Only Token is required; zero fields use built-in defaults.
type RestClientConfig struct {
	HTTPClient            *http.Client      // Optional custom http.Client; nil builds a pooled Transport.
	HTTP2                 *http.HTTP2Config // Optional HTTP/2 settings; nil uses PING health checks when ForceAttemptHTTP2 is enabled.
	Limiter               *RateLimiter      // Optional shared rate limiter; nil creates one from RateLimiterConfig.
	TraceLogger           *log.Logger
	ForceAttemptHTTP2     *bool // Attempts HTTP/2 on default transport (defaults to true).
	Token                 string
	BaseURL               string // Default: "https://discord.com/api/v10".
	UserAgent             string // Default: "DiscordApp https://github.com/amatsagu/tempest".
	RateLimiterConfig     RateLimiterConfig
	MaxFileUploadSize     int64         // Max upload size per file in bytes (< 0 disables). Default: 10 MiB.
	DialTimeout           time.Duration // Default: 5s.
	KeepAlive             time.Duration // Default: 30s.
	IdleConnTimeout       time.Duration // Default: 60s.
	ResponseHeaderTimeout time.Duration // Header wait timeout (< 0 disables). Default: 10s.
	TLSHandshakeTimeout   time.Duration // Default: 10s.
	ExpectContinueTimeout time.Duration // Default: 1s.
	MaxWaitTime           time.Duration // Per-call deadline budget (< 0 disables). Default: 10s.
	MaxIdleConns          int           // Default: 512.
	MaxIdleConnsPerHost   int           // Default: 256.
	RetryThreshold        uint32        // Max concurrent retries before tripping breaker. Default: 512.
	MaxRetries            uint8         // Default: 3.
	Trace                 bool
}

// Standalone Discord REST client with sharded rate limiting, pooled buffers, and bounded retries.
type RestClient struct {
	limiter           *RateLimiter
	traceLogger       *log.Logger
	bufferPool        *sync.Pool
	multipartPool     *sync.Pool
	readerPool        *sync.Pool
	copyBufPool       *sync.Pool
	HTTPClient        http.Client
	baseURL           string
	baseScheme        string
	baseHost          string
	basePath          string
	authHeader        []string
	uaHeader          []string
	jsonHeader        []string
	maxWaitTime       time.Duration
	maxFileUploadSize int64
	retryThreshold    int64
	retryCounter      atomic.Int64
	trippedUntil      atomic.Int64
	maxRetries        uint8
	trace             bool
}

func defaultRestClientConfig() RestClientConfig {
	return RestClientConfig{
		ForceAttemptHTTP2:     new(true),
		BaseURL:               "https://discord.com/api/v10",
		UserAgent:             "DiscordApp https://github.com/amatsagu/tempest",
		RateLimiterConfig:     defaultRateLimiterConfig(),
		MaxFileUploadSize:     10 * 1024 * 1024,
		DialTimeout:           5 * time.Second,
		KeepAlive:             30 * time.Second,
		IdleConnTimeout:       60 * time.Second,
		ResponseHeaderTimeout: 10 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: time.Second,
		MaxWaitTime:           10 * time.Second,
		MaxIdleConns:          512,
		MaxIdleConnsPerHost:   256,
		RetryThreshold:        512,
		MaxRetries:            3,
	}
}

// Creates a configured Discord REST client.
func NewRestClient(cfg RestClientConfig) *RestClient {
	if _, err := ExtractUserIDFromToken(cfg.Token); err != nil {
		panic("failed to extract bot user ID from bot token: " + err.Error())
	}

	def := defaultRestClientConfig()

	prefixedToken := cfg.Token
	if !strings.HasPrefix(prefixedToken, "Bot ") {
		prefixedToken = "Bot " + prefixedToken
	}

	baseURL := cfg.BaseURL
	if baseURL == "" {
		baseURL = def.BaseURL
	}

	baseURL = strings.TrimSuffix(baseURL, "/")

	parsedBase, err := url.Parse(baseURL)
	if err != nil || parsedBase.Scheme == "" || parsedBase.Host == "" {
		panic("invalid REST BaseURL: " + baseURL)
	}

	userAgent := cfg.UserAgent
	if userAgent == "" {
		userAgent = def.UserAgent
	}

	maxFileUploadSize := cfg.MaxFileUploadSize
	if maxFileUploadSize == 0 {
		maxFileUploadSize = def.MaxFileUploadSize
	}

	dialTimeout := cfg.DialTimeout
	if dialTimeout <= 0 {
		dialTimeout = def.DialTimeout
	}

	keepAlive := cfg.KeepAlive
	if keepAlive <= 0 {
		keepAlive = def.KeepAlive
	}

	idleConnTimeout := cfg.IdleConnTimeout
	if idleConnTimeout <= 0 {
		idleConnTimeout = def.IdleConnTimeout
	}

	responseHeaderTimeout := cfg.ResponseHeaderTimeout
	if responseHeaderTimeout == 0 {
		responseHeaderTimeout = def.ResponseHeaderTimeout
	} else if responseHeaderTimeout < 0 {
		responseHeaderTimeout = 0
	}

	tlsHandshakeTimeout := cfg.TLSHandshakeTimeout
	if tlsHandshakeTimeout <= 0 {
		tlsHandshakeTimeout = def.TLSHandshakeTimeout
	}

	expectContinueTimeout := cfg.ExpectContinueTimeout
	if expectContinueTimeout <= 0 {
		expectContinueTimeout = def.ExpectContinueTimeout
	}

	maxWaitTime := cfg.MaxWaitTime
	if maxWaitTime == 0 {
		maxWaitTime = def.MaxWaitTime
	} else if maxWaitTime < 0 {
		maxWaitTime = 0
	}

	maxRetries := cfg.MaxRetries
	if maxRetries == 0 {
		maxRetries = def.MaxRetries
	}

	retryThreshold := cfg.RetryThreshold
	if retryThreshold == 0 {
		retryThreshold = def.RetryThreshold
	}

	idlePerHost := cfg.MaxIdleConnsPerHost
	if idlePerHost <= 0 {
		idlePerHost = def.MaxIdleConnsPerHost
	}

	maxIdleConns := cfg.MaxIdleConns
	if maxIdleConns <= 0 {
		maxIdleConns = max(def.MaxIdleConns, idlePerHost*2)
	}

	attemptHTTP2 := true
	if cfg.ForceAttemptHTTP2 != nil {
		attemptHTTP2 = *cfg.ForceAttemptHTTP2
	}

	traceLogger := cfg.TraceLogger
	if traceLogger == nil {
		if cfg.Trace {
			traceLogger = log.New(os.Stdout, "[TEMPEST] ", log.LstdFlags)
		} else {
			traceLogger = log.New(io.Discard, "[TEMPEST] ", log.LstdFlags)
		}
	}

	trace := cfg.Trace || traceLogger.Writer() != io.Discard

	limiter := cfg.Limiter
	if limiter == nil {
		limCfg := cfg.RateLimiterConfig
		limCfg.Trace = limCfg.Trace || trace

		if limCfg.TraceLogger == nil {
			if cfg.RateLimiterConfig.Trace && cfg.TraceLogger == nil {
				limCfg.TraceLogger = log.New(os.Stdout, "[TEMPEST] ", log.LstdFlags)
			} else {
				limCfg.TraceLogger = traceLogger
			}
		}

		limiter = NewRateLimiter(limCfg)
	}

	var httpClient http.Client
	if cfg.HTTPClient != nil {
		httpClient = *cfg.HTTPClient
		inner := httpClient.Transport
		if inner == nil {
			inner = http.DefaultTransport
		}

		httpClient.Transport = &rateLimitTransport{
			limiter:        limiter,
			innerTransport: inner,
		}
	} else {
		var http2Cfg *http.HTTP2Config
		if cfg.HTTP2 != nil {
			cloned := *cfg.HTTP2
			http2Cfg = &cloned
		} else if attemptHTTP2 {
			// Proactively probe long-idle HTTP/2 connections via PING so dead peer sockets are detected early.
			http2Cfg = &http.HTTP2Config{
				SendPingTimeout:  15 * time.Second,
				PingTimeout:      10 * time.Second,
				WriteByteTimeout: 15 * time.Second,
			}
		}

		transport := &http.Transport{
			Proxy: http.ProxyFromEnvironment,
			DialContext: (&net.Dialer{
				Timeout:   dialTimeout,
				KeepAlive: keepAlive,
			}).DialContext,
			ForceAttemptHTTP2:     attemptHTTP2,
			HTTP2:                 http2Cfg,
			MaxIdleConns:          maxIdleConns,
			MaxIdleConnsPerHost:   idlePerHost,
			IdleConnTimeout:       idleConnTimeout,
			ResponseHeaderTimeout: responseHeaderTimeout,
			TLSHandshakeTimeout:   tlsHandshakeTimeout,
			ExpectContinueTimeout: expectContinueTimeout,
		}

		httpClient = http.Client{
			Transport: &rateLimitTransport{
				limiter:        limiter,
				innerTransport: transport,
			},
		}
	}

	return &RestClient{
		limiter: limiter,
		bufferPool: &sync.Pool{
			New: func() any {
				b := new(bytes.Buffer)
				b.Grow(4096)
				return b
			},
		},
		multipartPool: &sync.Pool{
			New: func() any {
				b := new(bytes.Buffer)
				b.Grow(64 * 1024)
				return b
			},
		},
		readerPool: &sync.Pool{
			New: func() any {
				return bytes.NewReader(nil)
			},
		},
		copyBufPool: &sync.Pool{
			New: func() any {
				b := make([]byte, 32*1024)
				return &b
			},
		},
		HTTPClient:        httpClient,
		baseURL:           baseURL,
		baseScheme:        parsedBase.Scheme,
		baseHost:          parsedBase.Host,
		basePath:          parsedBase.Path,
		authHeader:        []string{prefixedToken},
		uaHeader:          []string{userAgent},
		jsonHeader:        []string{ContentTypeJSON},
		maxWaitTime:       maxWaitTime,
		maxFileUploadSize: maxFileUploadSize,
		maxRetries:        maxRetries,
		retryThreshold:    int64(retryThreshold),
		traceLogger:       traceLogger,
		trace:             trace,
	}
}

// Returns the underlying sharded rate limiter.
func (c *RestClient) Limiter() *RateLimiter {
	return c.limiter
}

func (c *RestClient) tracef(format string, v ...any) {
	if !c.trace {
		return
	}

	c.traceLogger.Printf("[(REST) CLIENT] "+format, v...)
}

// Sends a JSON (or bodyless) request and decodes into into (or captures *[]byte / io.Writer).
func (c *RestClient) Request(ctx context.Context, method, route string, payload, into any) error {
	if ctx == nil {
		return errors.New("rest: nil context")
	}

	if err := ctx.Err(); err != nil {
		return err
	}

	if payload == nil {
		return c.send(ctx, method, route, nil, ContentTypeJSON, "", nil, into)
	}

	if raw, ok := payload.(jsontext.Value); ok {
		r := c.acquireReader(raw)
		defer c.releaseReader(r)

		return c.send(ctx, method, route, r, ContentTypeJSON, "", nil, into)
	}

	buf := c.acquireBuffer()
	defer c.releaseBuffer(buf)

	if err := json.MarshalWrite(buf, payload); err != nil {
		return fmt.Errorf("failed to encode JSON payload: %w", err)
	}

	r := c.acquireReader(buf.Bytes())
	defer c.releaseReader(r)

	return c.send(ctx, method, route, r, ContentTypeJSON, "", nil, into)
}

// Sends a multipart request with JSON payload and file attachments.
func (c *RestClient) RequestWithFiles(ctx context.Context, method, route string, jsonPayload any, files []File, into any) error {
	if ctx == nil {
		return errors.New("rest: nil context")
	}

	return c.sendFiles(ctx, method, route, jsonPayload, files, "", nil, into)
}

// Executes a custom REST call with optional headers, audit reason, files, or raw body.
func (c *RestClient) Do(ctx context.Context, req CustomRequest) error {
	if ctx == nil {
		return errors.New("rest: nil context")
	}

	if err := ctx.Err(); err != nil {
		return err
	}

	if len(req.Files) > 0 {
		return c.sendFiles(ctx, req.Method, req.Route, req.Body, req.Files, req.AuditLogReason, req.Headers, req.Into)
	}

	ct := req.ContentType

	switch v := req.Body.(type) {
	case nil:
		if ct == "" {
			ct = ContentTypeJSON
		}

		return c.send(ctx, req.Method, req.Route, nil, ct, req.AuditLogReason, req.Headers, req.Into)

	case jsontext.Value:
		if ct == "" {
			ct = ContentTypeJSON
		}

		r := c.acquireReader(v)
		defer c.releaseReader(r)

		return c.send(ctx, req.Method, req.Route, r, ct, req.AuditLogReason, req.Headers, req.Into)

	case []byte:
		if ct == "" {
			ct = ContentTypeOctetStream
		}

		r := c.acquireReader(v)
		defer c.releaseReader(r)

		return c.send(ctx, req.Method, req.Route, r, ct, req.AuditLogReason, req.Headers, req.Into)

	case string:
		if ct == "" {
			ct = ContentTypeOctetStream
		}

		return c.send(ctx, req.Method, req.Route, strings.NewReader(v), ct, req.AuditLogReason, req.Headers, req.Into)

	case io.ReadSeeker:
		if ct == "" {
			ct = ContentTypeOctetStream
		}

		return c.send(ctx, req.Method, req.Route, v, ct, req.AuditLogReason, req.Headers, req.Into)

	case io.Reader:
		if ct == "" {
			ct = ContentTypeOctetStream
		}

		buf := c.acquireBuffer()
		defer c.releaseBuffer(buf)

		if _, err := buf.ReadFrom(contextReader{ctx: ctx, reader: v}); err != nil {
			return contextOrWrapped(ctx, "failed to read request body", err)
		}

		r := c.acquireReader(buf.Bytes())
		defer c.releaseReader(r)

		return c.send(ctx, req.Method, req.Route, r, ct, req.AuditLogReason, req.Headers, req.Into)

	default:
		if ct == "" {
			ct = ContentTypeJSON
		}

		buf := c.acquireBuffer()
		defer c.releaseBuffer(buf)

		if err := json.MarshalWrite(buf, v); err != nil {
			return fmt.Errorf("failed to encode JSON payload: %w", err)
		}

		r := c.acquireReader(buf.Bytes())
		defer c.releaseReader(r)

		return c.send(ctx, req.Method, req.Route, r, ct, req.AuditLogReason, req.Headers, req.Into)
	}
}

// Extracts the bot application ID from a Discord token.
func ExtractUserIDFromToken(token string) (uint64, error) {
	token = strings.TrimPrefix(token, "Bot ")

	dot := strings.IndexByte(token, '.')
	if dot <= 0 {
		return 0, errors.New("token is not in a valid format")
	}

	encodedID := token[:dot]
	byteID, err := base64.RawURLEncoding.DecodeString(encodedID)
	if err != nil {
		byteID, err = base64.RawStdEncoding.DecodeString(encodedID)
	}
	if err != nil {
		byteID, err = base64.StdEncoding.DecodeString(encodedID)
	}
	if err != nil {
		return 0, err
	}

	id, err := strconv.ParseUint(string(byteID), 10, 64)
	if err != nil {
		return 0, err
	}

	if id == 0 {
		return 0, errors.New("token contains an invalid bot user ID")
	}

	return id, nil
}
