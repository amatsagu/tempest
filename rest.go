package tempest

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net"
	"net/http"
	"net/textproto"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

var (
	errRetryable       = errors.New("a retryable error occurred")
	errGlobalRateLimit = errors.New("hit global rate limit")
	errTooManyRetries  = errors.New("internal retry threshold exceeded - your code logic is probably unsafe to use at scale")
)

// Represents file you can attach to message on Discord.
type File struct {
	Reader io.Reader
	Name   string // File's display name
}

type rateLimitError struct {
	Message    string  `json:"message"`
	RetryAfter float64 `json:"retry_after"`
	Global     bool    `json:"global"`
}

type Rest struct {
	baseCtx     context.Context
	limiter     *RateLimiter
	traceLogger *log.Logger
	// Timeout is intentionally left unset; Rest methods apply MaxWaitTime
	// through the request context so retries share one deadline.
	HTTPClient     http.Client
	token          string
	maxWaitTime    time.Duration
	retryCounter   atomic.Int64
	retryThreshold int64
	trippedUntil   atomic.Int64 // UnixNano
	baseCtxMu      sync.RWMutex
	maxRetries     uint8
	trace          bool
}

type RestOptions struct {
	// Context is the lifecycle context used by context-free Rest methods
	// (Request, DirectRequest, RequestWithFiles). Gateway clients replace it
	// with the context passed to ShardManager.Start. HTTP clients should set
	// this to their server/application context so shutdown also stops
	// post-response REST work. If nil, a background fallback is used.
	Context            context.Context
	TraceLogger        *log.Logger
	Token              string
	RateLimiterOptions RateLimiterOptions
	MaxWaitTime        time.Duration // Max duration for one logical request, including retries. By default: 10 seconds.
	RetryThreshold     uint32        // Max number of concurrent retries allowed before failing all ongoing requests (emergency breaks). By default: 60.
	MaxRetries         uint8         // By default: 3
	Trace              bool
}

func NewRest(opt RestOptions) *Rest {
	_, err := extractUserIDFromToken(opt.Token)
	if err != nil {
		panic("failed to extract bot user ID from bot token: " + err.Error())
	}

	prefixedToken := opt.Token
	if !strings.HasPrefix(prefixedToken, "Bot ") {
		prefixedToken = "Bot " + prefixedToken
	}

	transport := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   REST_DIAL_TIMEOUT,
			KeepAlive: REST_KEEP_ALIVE,
		}).DialContext,
		ForceAttemptHTTP2:     REST_FORCE_ATTEMPT_HTTP2,
		MaxIdleConns:          REST_MAX_IDLE_CONNS,
		MaxIdleConnsPerHost:   REST_MAX_IDLE_CONNS_PER_HOST,
		IdleConnTimeout:       REST_IDLE_CONN_TIMEOUT,
		TLSHandshakeTimeout:   REST_TLS_HANDSHAKE_TIMEOUT,
		ExpectContinueTimeout: REST_EXPECT_CONTINUE_TIMEOUT,
	}

	maxTimeout := REST_DEFAULT_MAX_WAIT_TIME
	if opt.MaxWaitTime != 0 {
		maxTimeout = opt.MaxWaitTime
	}

	maxRetries := opt.MaxRetries
	if opt.MaxRetries == 0 {
		maxRetries = REST_DEFAULT_MAX_RETRIES
	}

	retryThreshold := opt.RetryThreshold
	if retryThreshold == 0 {
		retryThreshold = REST_DEFAULT_RETRY_THRESHOLD
	}

	traceLogger := opt.TraceLogger
	if traceLogger == nil {
		traceLogger = log.New(io.Discard, "[TEMPEST] ", log.LstdFlags)
	}

	limiterOptions := opt.RateLimiterOptions
	limiterOptions.Trace = opt.Trace
	if limiterOptions.TraceLogger == nil {
		limiterOptions.TraceLogger = traceLogger
	}
	limiter := NewRateLimiter(limiterOptions)

	return &Rest{
		limiter: limiter,
		HTTPClient: http.Client{
			Transport: &rateLimitTransport{
				limiter:        limiter,
				innerTransport: transport,
			},
		},
		baseCtx:        opt.Context,
		token:          prefixedToken,
		maxRetries:     maxRetries,
		maxWaitTime:    maxTimeout,
		retryThreshold: int64(retryThreshold),
		traceLogger:    traceLogger,
		trace:          traceLogger.Writer() != io.Discard,
	}
}

func (rest *Rest) setBaseContext(ctx context.Context) {
	if ctx == nil {
		return
	}

	rest.baseCtxMu.Lock()
	rest.baseCtx = ctx
	rest.baseCtxMu.Unlock()
}

func (rest *Rest) baseContext() context.Context {
	rest.baseCtxMu.RLock()
	ctx := rest.baseCtx
	rest.baseCtxMu.RUnlock()

	if ctx == nil {
		return context.Background()
	}
	return ctx
}

func (rest *Rest) tracef(format string, v ...any) {
	if !rest.trace {
		return
	}

	rest.traceLogger.Printf("[(REST) CLIENT] "+format, v...)
}

func (rest *Rest) Request(method, route string, jsonPayload any) ([]byte, error) {
	return rest.RequestContext(rest.baseContext(), method, route, jsonPayload)
}

// RequestContext is like Request, but it honors ctx for limiter waits, retries,
// and the HTTP request itself.
func (rest *Rest) RequestContext(ctx context.Context, method, route string, jsonPayload any) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	body, err := rest.BufferJSON(jsonPayload)
	if err != nil {
		return nil, err
	}

	return rest.DirectRequestContext(ctx, method, route, body, CONTENT_TYPE_JSON, "")
}

// Prepares new json buffered request payload that can be used by DirectRequest() method.
// Please use regular Request() or RequestWithFiles() if you're not sure what you're doing.
func (rest *Rest) BufferJSON(jsonPayload any) (io.ReadSeeker, error) {
	var body io.ReadSeeker
	if jsonPayload != nil {
		var buf bytes.Buffer
		encoder := json.NewEncoder(&buf)
		encoder.SetEscapeHTML(false)
		if err := encoder.Encode(jsonPayload); err != nil {
			return nil, fmt.Errorf("failed to encode JSON payload: %w", err)
		}
		body = bytes.NewReader(buf.Bytes())
	}

	return body, nil
}

func (rest *Rest) isTripped() bool {
	return rest.trippedUntil.Load() > time.Now().UnixNano()
}

// DirectRequest is the context-free wrapper for DirectRequestContext.
func (rest *Rest) DirectRequest(method, route string, body io.ReadSeeker, contentType string, auditLogReason string) ([]byte, error) {
	return rest.DirectRequestContext(rest.baseContext(), method, route, body, contentType, auditLogReason)
}

// DirectRequestContext is like DirectRequest, but it honors ctx for limiter
// waits, retry backoff, and the HTTP request itself. When ctx has no deadline,
// or a deadline later than MaxWaitTime, the request is capped at MaxWaitTime.
func (rest *Rest) DirectRequestContext(ctx context.Context, method, route string, body io.ReadSeeker, contentType string, auditLogReason string) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if contentType == "" {
		return nil, errors.New("requests must have content type provided - for most Discord API requests, you probably want tempest.CONTENT_TYPE_JSON")
	}

	if rest.maxWaitTime > 0 {
		if deadline, ok := ctx.Deadline(); !ok || time.Until(deadline) > rest.maxWaitTime {
			var cancel context.CancelFunc
			ctx, cancel = context.WithTimeout(ctx, rest.maxWaitTime)
			defer cancel()
		}
	}

	var (
		responseBody []byte
		lastErr      error
		retrying     bool
		rateLimited  bool
	)

	defer func() {
		if retrying {
			rest.retryCounter.Add(-1)
		}
	}()

	for i := uint8(0); i < rest.maxRetries; i++ {
		if err := ctx.Err(); err != nil {
			return nil, err
		}

		if body != nil {
			if _, err := body.Seek(0, io.SeekStart); err != nil {
				return nil, fmt.Errorf("failed to seek request body: %w", err)
			}
		}

		if rest.isTripped() {
			return nil, errTooManyRetries
		}

		// #nosec G704
		req, err := http.NewRequestWithContext(ctx, method, DiscordAPIBaseURL()+route, body)
		if err != nil {
			return nil, fmt.Errorf("failed to create request: %w", err)
		}

		req.Header.Set("Content-Type", contentType)

		if auditLogReason != "" {
			req.Header.Set("X-Audit-Log-Reason", url.PathEscape(auditLogReason))
		}

		responseBody, rateLimited, err = rest.executeOnce(req)
		if err != nil {
			lastErr = err

			if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				return nil, err
			}

			// 429 responses are safe to retry because Discord rejected the
			// request. Transport errors and 5xx responses may have already
			// produced side effects, so only retry those for idempotent methods.
			retryable := rateLimited || (errors.Is(err, errRetryable) && isIdempotentMethod(method))
			if retryable {
				if !retrying {
					retrying = true
					if rest.retryCounter.Add(1) > rest.retryThreshold {
						rest.trippedUntil.Store(time.Now().Add(5 * time.Second).UnixNano())
						return nil, fmt.Errorf("%w: global retry threshold (%d) exceeded", errTooManyRetries, rest.retryThreshold)
					}
				}

				if rateLimited && errors.Is(err, errGlobalRateLimit) {
					continue
				}

				if err := sleepContext(ctx, time.Millisecond*time.Duration(250*int64(i+1))); err != nil {
					return nil, err
				}
				continue
			}

			return nil, err // Not a retryable error.
		}
		return responseBody, nil
	}

	return nil, fmt.Errorf("request failed after %d retries on %s %s: %w", rest.maxRetries, method, route, lastErr)
}

func (rest *Rest) RequestWithFiles(method string, route string, jsonPayload any, files []File) ([]byte, error) {
	return rest.RequestWithFilesContext(rest.baseContext(), method, route, jsonPayload, files)
}

// RequestWithFilesContext is like RequestWithFiles, but it honors ctx for
// limiter waits, retry backoff, multipart buffering, and the HTTP request.
func (rest *Rest) RequestWithFilesContext(ctx context.Context, method string, route string, jsonPayload any, files []File) ([]byte, error) {
	if len(files) == 0 {
		return rest.RequestContext(ctx, method, route, jsonPayload)
	}

	if err := ctx.Err(); err != nil {
		return nil, err
	}

	// To avoid issues with retrying requests that have streaming bodies (which can't be re-read),
	// we pre-buffer the multipart request into memory. This ensures that the request body can be
	// reliably sent multiple times in case of retries.
	var requestBody bytes.Buffer
	writer := multipart.NewWriter(&requestBody)
	if err := writeMultipartContext(ctx, writer, jsonPayload, files); err != nil {
		return nil, err
	}

	if err := ctx.Err(); err != nil {
		return nil, err
	}

	return rest.DirectRequestContext(ctx, method, route, bytes.NewReader(requestBody.Bytes()), writer.FormDataContentType(), "")
}

// Writes the JSON payload and files to a multipart writer.
func writeMultipartContext(ctx context.Context, writer *multipart.Writer, jsonPayload any, files []File) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	// Write JSON payload part.
	jsonPart, err := writer.CreatePart(textproto.MIMEHeader{
		"Content-Disposition": []string{`form-data; name="payload_json"`},
		"Content-Type":        []string{CONTENT_TYPE_JSON},
	})
	if err != nil {
		return fmt.Errorf("failed to create payload_json part: %w", err)
	}
	encoder := json.NewEncoder(jsonPart)
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(jsonPayload); err != nil {
		return fmt.Errorf("failed to encode payload_json: %w", err)
	}

	// Write file parts.
	for i, file := range files {
		if err := ctx.Err(); err != nil {
			return err
		}

		if seeker, ok := file.Reader.(io.ReadSeeker); ok {
			if _, err := seeker.Seek(0, io.SeekStart); err != nil {
				return fmt.Errorf("failed to seek file [%s]: %w", file.Name, err)
			}
		}

		filePart, err := writer.CreatePart(textproto.MIMEHeader{
			"Content-Disposition": []string{fmt.Sprintf(`form-data; name="files[%d]"; filename="%s"`, i, escapeMultipartFilename(file.Name))},
			"Content-Type":        []string{CONTENT_TYPE_OCTET_STREAM},
		})
		if err != nil {
			return fmt.Errorf("failed to create file part [%d]: %w", i, err)
		}

		lr := io.LimitReader(file.Reader, MAX_FILE_UPLOAD_SIZE+1)
		n, err := io.Copy(filePart, contextReader{ctx: ctx, reader: lr})
		if err != nil {
			return fmt.Errorf("failed to stream file [%s]: %w", file.Name, err)
		}
		if n > MAX_FILE_UPLOAD_SIZE {
			return fmt.Errorf("file [%s] exceeds maximum upload size of 10MB", file.Name)
		}
	}

	if err := ctx.Err(); err != nil {
		return err
	}
	if err := writer.Close(); err != nil {
		return fmt.Errorf("failed to close multipart writer: %w", err)
	}

	return nil
}

type contextReader struct {
	ctx    context.Context
	reader io.Reader
}

func (r contextReader) Read(p []byte) (int, error) {
	if err := r.ctx.Err(); err != nil {
		return 0, err
	}
	return r.reader.Read(p)
}

var multipartFilenameReplacer = strings.NewReplacer(
	"\\", "\\\\",
	`"`, `\"`,
	"\r", "",
	"\n", "",
)

// escapeMultipartFilename makes a filename safe for a quoted MIME parameter.
func escapeMultipartFilename(name string) string {
	return multipartFilenameReplacer.Replace(name)
}

// executeOnce handles the lifecycle of a single request attempt. The boolean
// reports a 429 response, which is safe to retry for any HTTP method.
func (rest *Rest) executeOnce(req *http.Request) ([]byte, bool, error) {
	req.Header.Set("User-Agent", USER_AGENT)
	req.Header.Set("Authorization", rest.token)

	var start time.Time
	if rest.trace {
		start = time.Now()
	}

	res, err := rest.HTTPClient.Do(req)
	if rest.trace && err == nil {
		rest.tracef("%s %s - Status: %s (took %v)", req.Method, req.URL.Path, res.Status, time.Since(start))
	}

	if err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return nil, false, err
		}
		return nil, false, fmt.Errorf("%w: http request execution failed: %s", errRetryable, err.Error())
	}
	defer res.Body.Close() //nolint:errcheck

	responseBody, err := io.ReadAll(res.Body)
	if err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return nil, false, err
		}
		return nil, false, fmt.Errorf("failed to read response body: %w", err)
	}

	if res.StatusCode >= http.StatusOK && res.StatusCode < http.StatusMultipleChoices {
		return responseBody, false, nil
	}

	if res.StatusCode == http.StatusTooManyRequests {
		var rateErr rateLimitError
		err = json.Unmarshal(responseBody, &rateErr)
		if err != nil {
			return nil, false, fmt.Errorf("failed to parse rate limit details: %w", err)
		}

		if rateErr.Global {
			retryAfter := time.Duration(rateErr.RetryAfter * float64(time.Second))
			return nil, true, fmt.Errorf("%w: retrying after %s", errGlobalRateLimit, retryAfter.String())
		}

		return nil, true, fmt.Errorf("hit per-route rate limit (429) on %s %s: %s", req.Method, req.URL.Path, string(responseBody))
	}

	if res.StatusCode >= http.StatusInternalServerError {
		return nil, false, fmt.Errorf("%w: discord API internal server error: %s", errRetryable, res.Status)
	}

	return nil, false, fmt.Errorf("request failed with status %s: %s", res.Status, string(responseBody))
}

func isIdempotentMethod(method string) bool {
	switch method {
	case http.MethodGet, http.MethodHead, http.MethodOptions, http.MethodPut, http.MethodDelete:
		return true
	default:
		return false
	}
}

func extractUserIDFromToken(token string) (Snowflake, error) {
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

	id, err := StringToSnowflake(string(byteID))
	if err != nil {
		return 0, err
	}
	if id == 0 {
		return 0, errors.New("token contains an invalid bot user ID")
	}
	return id, nil
}
