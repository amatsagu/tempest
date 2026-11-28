package rest

import (
	"bytes"
	"context"
	json "encoding/json/v2"
	"errors"
	"fmt"
	"io"
	"math/rand/v2"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"
)

func (c *RestClient) send(ctx context.Context, method, route string, body io.ReadSeeker, contentType, auditLogReason string, headers http.Header, into any) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	if c.maxWaitTime > 0 {
		if deadline, ok := ctx.Deadline(); !ok || time.Until(deadline) > c.maxWaitTime {
			var cancel context.CancelFunc
			ctx, cancel = context.WithTimeout(ctx, c.maxWaitTime)
			defer cancel()
		}
	}

	var (
		lastErr     error
		retryDelay  time.Duration
		retrying    bool
		rateLimited bool
	)

	defer func() {
		if retrying {
			c.retryCounter.Add(-1)
		}
	}()

	for i := uint8(0); i < c.maxRetries; i++ {
		if err := ctx.Err(); err != nil {
			return err
		}

		if body != nil {
			if _, err := body.Seek(0, io.SeekStart); err != nil {
				return fmt.Errorf("failed to seek request body: %w", err)
			}
		}

		if c.isTripped() {
			return ErrTooManyRetries
		}

		req, err := c.buildRequest(ctx, method, route, body, contentType, auditLogReason, headers)
		if err != nil {
			return fmt.Errorf("failed to create request: %w", err)
		}

		rateLimited, retryDelay, err = c.executeOnce(req, into)
		if err == nil {
			return nil
		}

		lastErr = err
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return err
		}

		// 429s were rejected cleanly by Discord, so all methods can retry;
		// transport/5xx errors retry only for idempotent methods.
		retryable := rateLimited || (errors.Is(err, errRetryable) && isIdempotentMethod(method))
		if !retryable {
			return err
		}

		if !rateLimited && !retrying {
			retrying = true

			if c.retryCounter.Add(1) > c.retryThreshold {
				c.trippedUntil.Store(time.Now().Add(5 * time.Second).UnixNano())
				return fmt.Errorf("%w: concurrent retry threshold (%d) exceeded", ErrTooManyRetries, c.retryThreshold)
			}
		}

		if rateLimited {
			if retryDelay > 0 {
				if err := sleepContext(ctx, retryDelay); err != nil {
					return err
				}
			}

			continue
		}

		baseDelay := time.Millisecond * time.Duration(250*int64(i+1))
		jitter := time.Duration(rand.Int64N(int64(100 * time.Millisecond)))

		if err := sleepContext(ctx, baseDelay+jitter); err != nil {
			return err
		}
	}

	return fmt.Errorf("request failed after %d attempts on %s %s: %w", c.maxRetries, method, route, lastErr)
}

func (c *RestClient) sendFiles(ctx context.Context, method, route string, jsonPayload any, files []File, auditLogReason string, headers http.Header, into any) error {
	if len(files) == 0 {
		return c.send(ctx, method, route, nil, ContentTypeJSON, auditLogReason, headers, into)
	}

	if err := ctx.Err(); err != nil {
		return err
	}

	requestBody := c.acquireMultipartBuffer()
	defer c.releaseMultipartBuffer(requestBody)

	copyBuf := c.acquireCopyBuf()
	defer c.releaseCopyBuf(copyBuf)

	writer := multipart.NewWriter(requestBody)
	if err := writeMultipartContext(ctx, writer, jsonPayload, files, c.maxFileUploadSize, *copyBuf); err != nil {
		return err
	}

	if err := ctx.Err(); err != nil {
		return err
	}

	r := c.acquireReader(requestBody.Bytes())
	defer c.releaseReader(r)

	return c.send(ctx, method, route, r, writer.FormDataContentType(), auditLogReason, headers, into)
}

func writeMultipartContext(ctx context.Context, writer *multipart.Writer, jsonPayload any, files []File, maxFileUploadSize int64, copyBuf []byte) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	mimeHeader := make(textproto.MIMEHeader, 2)
	dispSlice := make([]string, 1)
	ctSlice := make([]string, 1)
	mimeHeader["Content-Disposition"] = dispSlice
	mimeHeader["Content-Type"] = ctSlice

	if jsonPayload != nil {
		dispSlice[0] = multipartJSONPartHeader
		ctSlice[0] = ContentTypeJSON

		jsonPart, err := writer.CreatePart(mimeHeader)
		if err != nil {
			return fmt.Errorf("failed to create payload_json part: %w", err)
		}

		if err := json.MarshalWrite(jsonPart, jsonPayload); err != nil {
			return fmt.Errorf("failed to encode payload_json: %w", err)
		}
	}

	var dispBuf [128]byte
	for i, file := range files {
		if err := ctx.Err(); err != nil {
			return err
		}

		if seeker, ok := file.Reader.(io.ReadSeeker); ok {
			if _, err := seeker.Seek(0, io.SeekStart); err != nil {
				return fmt.Errorf("failed to seek file [%s]: %w", file.Name, err)
			}
		}

		dispSlice[0] = formatFileDisposition(dispBuf[:0], i, file.Name)

		ct := file.ContentType
		if ct == "" {
			ct = ContentTypeOctetStream
		}
		ctSlice[0] = ct

		filePart, err := writer.CreatePart(mimeHeader)
		if err != nil {
			return fmt.Errorf("failed to create file part [%d]: %w", i, err)
		}

		src := contextReader{ctx: ctx, reader: file.Reader}
		var n int64

		if maxFileUploadSize > 0 {
			src.reader = io.LimitReader(file.Reader, maxFileUploadSize+1)
			n, err = io.CopyBuffer(filePart, src, copyBuf)
		} else {
			n, err = io.CopyBuffer(filePart, src, copyBuf)
		}

		if err != nil {
			return fmt.Errorf("failed to stream file [%s]: %w", file.Name, err)
		}

		if maxFileUploadSize > 0 && n > maxFileUploadSize {
			return fmt.Errorf("file [%s] exceeds maximum upload size of %d bytes", file.Name, maxFileUploadSize)
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

func (c *RestClient) buildRequest(ctx context.Context, method, route string, body io.Reader, contentType, auditLogReason string, headers http.Header) (*http.Request, error) {
	var parsedURL *url.URL
	if !strings.ContainsAny(route, "?#%") {
		parsedURL = &url.URL{
			Scheme: c.baseScheme,
			Host:   c.baseHost,
			Path:   c.basePath + route,
		}
	} else {
		var err error
		parsedURL, err = url.Parse(c.baseURL + route)
		if err != nil {
			return nil, err
		}
	}

	req, err := http.NewRequestWithContext(ctx, method, "", body)
	if err != nil {
		return nil, err
	}

	req.URL = parsedURL
	req.Host = c.baseHost

	hdr := req.Header
	hdr["Authorization"] = c.authHeader
	hdr["User-Agent"] = c.uaHeader

	if contentType == ContentTypeJSON {
		hdr["Content-Type"] = c.jsonHeader
	} else if contentType != "" {
		hdr["Content-Type"] = []string{contentType}
	}

	if auditLogReason != "" {
		hdr["X-Audit-Log-Reason"] = []string{url.PathEscape(auditLogReason)}
	}

	for k, vals := range headers {
		hdr[textproto.CanonicalMIMEHeaderKey(k)] = vals
	}

	return req, nil
}

func (c *RestClient) executeOnce(req *http.Request, into any) (bool, time.Duration, error) {
	var start time.Time
	if c.trace {
		start = time.Now()
	}

	res, err := c.HTTPClient.Do(req)
	if c.trace && err == nil {
		c.tracef("%s %s - Status: %s (took %v)", req.Method, req.URL.Path, res.Status, time.Since(start))
	}

	if err != nil {
		if ctxErr := req.Context().Err(); ctxErr != nil {
			return false, 0, ctxErr
		}

		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return false, 0, err
		}

		return false, 0, fmt.Errorf("%w: http request execution failed: %s", errRetryable, err.Error())
	}
	defer func() { _ = res.Body.Close() }()

	if res.StatusCode >= http.StatusOK && res.StatusCode < http.StatusMultipleChoices {
		return false, 0, c.decodeResponse(req.Context(), res, into)
	}

	responseBody, err := io.ReadAll(io.LimitReader(res.Body, 1<<20))
	_, _ = io.Copy(io.Discard, res.Body)
	if err != nil {
		if ctxErr := req.Context().Err(); ctxErr != nil {
			return false, 0, ctxErr
		}

		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return false, 0, err
		}

		return false, 0, fmt.Errorf("failed to read response body: %w", err)
	}

	if res.StatusCode == http.StatusTooManyRequests {
		return c.handleRateLimitResponse(res, req, responseBody)
	}

	if res.StatusCode >= http.StatusInternalServerError {
		return false, 0, fmt.Errorf("%w: discord API internal server error: %s", errRetryable, res.Status)
	}

	return false, 0, newHTTPError(res.StatusCode, res.Status, responseBody)
}

func (c *RestClient) decodeResponse(ctx context.Context, res *http.Response, into any) error {
	if into == nil {
		if _, err := io.Copy(io.Discard, res.Body); err != nil {
			if ctxErr := ctx.Err(); ctxErr != nil {
				return ctxErr
			}
		}

		return nil
	}

	if rawTarget, ok := into.(*[]byte); ok {
		b, err := readResponseBytes(res)
		if err != nil {
			return contextOrWrapped(ctx, "failed to read response body", err)
		}

		*rawTarget = b
		return nil
	}

	if w, ok := into.(io.Writer); ok {
		if _, err := io.Copy(w, res.Body); err != nil {
			return contextOrWrapped(ctx, "failed to copy response body", err)
		}

		return nil
	}

	if res.ContentLength > 0 && res.ContentLength <= maxPooledJSONBufferCap {
		buf := c.acquireBuffer()
		defer c.releaseBuffer(buf)

		buf.Grow(int(res.ContentLength))
		if _, err := buf.ReadFrom(res.Body); err != nil {
			return contextOrWrapped(ctx, "failed to read response body", err)
		}

		if err := json.Unmarshal(buf.Bytes(), into); err != nil {
			return fmt.Errorf("failed to decode response json: %w", err)
		}

		return nil
	}

	if err := json.UnmarshalRead(res.Body, into); err != nil {
		return contextOrWrapped(ctx, "failed to decode response json", err)
	}

	return nil
}

func (c *RestClient) handleRateLimitResponse(res *http.Response, req *http.Request, responseBody []byte) (bool, time.Duration, error) {
	if firstHeaderValue(res.Header, headerGlobal) == "true" || firstHeaderValue(res.Header, headerScope) == "global" {
		if secs, ok := parsePositiveSeconds(firstHeaderValue(res.Header, headerRetryAfter)); ok {
			retryAfter := time.Duration(secs * float64(time.Second))
			return true, 0, fmt.Errorf("%w: retrying after %s", errGlobalRateLimit, retryAfter.String())
		}

		return true, 0, errGlobalRateLimit
	}

	var rateErr rateLimitError
	hasJSONRateErr := len(responseBody) > 0 && json.Unmarshal(responseBody, &rateErr) == nil
	if hasJSONRateErr && rateErr.Global {
		retryAfter := time.Duration(rateErr.RetryAfter * float64(time.Second))
		c.limiter.setGlobalPause(time.Now().Add(retryAfter + 250*time.Millisecond).UnixNano())
		return true, 0, fmt.Errorf("%w: retrying after %s", errGlobalRateLimit, retryAfter.String())
	}

	// Non-JSON 429 without Via or X-RateLimit-Bucket indicates a Cloudflare edge limit; pause globally.
	if !hasJSONRateErr && firstHeaderValue(res.Header, headerBucket) == "" && firstHeaderValue(res.Header, headerVia) == "" {
		cfDelay := bucketRetryFloor
		if secs, ok := parsePositiveSeconds(firstHeaderValue(res.Header, headerRetryAfter)); ok {
			cfDelay = time.Duration(secs * float64(time.Second))
		}

		c.limiter.setGlobalPause(time.Now().Add(cfDelay + 250*time.Millisecond).UnixNano())
		return true, 0, fmt.Errorf("%w: edge rate limit (retrying after %s)", errGlobalRateLimit, cfDelay.String())
	}

	var retryDelay time.Duration
	if firstHeaderValue(res.Header, headerResetAfter) == "" && firstHeaderValue(res.Header, headerReset) == "" {
		if secs, ok := parsePositiveSeconds(firstHeaderValue(res.Header, headerRetryAfter)); ok {
			retryDelay = time.Duration(secs * float64(time.Second))
		} else if hasJSONRateErr && rateErr.RetryAfter > 0 {
			retryDelay = time.Duration(rateErr.RetryAfter * float64(time.Second))
		} else {
			retryDelay = bucketRetryFloor
		}
	}

	return true, retryDelay, fmt.Errorf("hit per-route rate limit (429) on %s %s: %s", req.Method, req.URL.Path, string(responseBody))
}

func (c *RestClient) isTripped() bool {
	return c.trippedUntil.Load() > time.Now().UnixNano()
}

func (c *RestClient) acquireBuffer() *bytes.Buffer {
	return getPooledBuffer(c.bufferPool, 4096)
}

func (c *RestClient) releaseBuffer(buf *bytes.Buffer) {
	putPooledBuffer(c.bufferPool, buf, maxPooledJSONBufferCap)
}

func (c *RestClient) acquireMultipartBuffer() *bytes.Buffer {
	return getPooledBuffer(c.multipartPool, 64*1024)
}

func (c *RestClient) releaseMultipartBuffer(buf *bytes.Buffer) {
	putPooledBuffer(c.multipartPool, buf, maxPooledMultipartBufferCap)
}

func getPooledBuffer(pool *sync.Pool, initialCap int) *bytes.Buffer {
	if pool == nil {
		b := new(bytes.Buffer)
		b.Grow(initialCap)
		return b
	}

	buf := pool.Get().(*bytes.Buffer)
	buf.Reset()
	return buf
}

func putPooledBuffer(pool *sync.Pool, buf *bytes.Buffer, maxCap int) {
	if buf == nil {
		return
	}

	if pool != nil && buf.Cap() <= maxCap {
		buf.Reset()
		pool.Put(buf)
	}
}

func (c *RestClient) acquireReader(b []byte) *bytes.Reader {
	if c.readerPool == nil {
		return bytes.NewReader(b)
	}

	r := c.readerPool.Get().(*bytes.Reader)
	r.Reset(b)
	return r
}

func (c *RestClient) releaseReader(r *bytes.Reader) {
	if c.readerPool != nil && r != nil {
		r.Reset(nil)
		c.readerPool.Put(r)
	}
}

func (c *RestClient) acquireCopyBuf() *[]byte {
	if c.copyBufPool == nil {
		b := make([]byte, 32*1024)
		return &b
	}

	return c.copyBufPool.Get().(*[]byte)
}

func (c *RestClient) releaseCopyBuf(b *[]byte) {
	if c.copyBufPool != nil && b != nil {
		c.copyBufPool.Put(b)
	}
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

func formatFileDisposition(dst []byte, idx int, name string) string {
	dst = append(dst, `form-data; name="files[`...)
	dst = strconv.AppendInt(dst, int64(idx), 10)
	dst = append(dst, `]"; filename="`...)
	dst = append(dst, escapeMultipartFilename(name)...)
	dst = append(dst, '"')
	return string(dst)
}

func escapeMultipartFilename(name string) string {
	if !strings.ContainsAny(name, "\"\\\r\n") {
		return name
	}

	return multipartFilenameReplacer.Replace(name)
}

func parsePositiveSeconds(s string) (float64, bool) {
	if s == "" {
		return 0, false
	}

	secs, err := strconv.ParseFloat(s, 64)
	return secs, err == nil && secs > 0
}

func contextOrWrapped(ctx context.Context, msg string, err error) error {
	if ctxErr := ctx.Err(); ctxErr != nil {
		return ctxErr
	}

	return fmt.Errorf("%s: %w", msg, err)
}

func readResponseBytes(res *http.Response) ([]byte, error) {
	if res.ContentLength >= 0 && res.ContentLength <= 16*1024*1024 {
		out := make([]byte, res.ContentLength)
		if _, err := io.ReadFull(res.Body, out); err != nil {
			return nil, err
		}

		_, _ = io.Copy(io.Discard, res.Body)
		return out, nil
	}

	return io.ReadAll(res.Body)
}

func isIdempotentMethod(method string) bool {
	switch method {
	case http.MethodGet, http.MethodHead, http.MethodOptions, http.MethodPut, http.MethodDelete:
		return true
	default:
		return false
	}
}
