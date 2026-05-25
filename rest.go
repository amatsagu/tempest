package tempest

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net"
	"net/http"
	"net/textproto"
	"strings"
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
	Name   string // File's display name
	Reader io.Reader
}

type rateLimitError struct {
	Message    string  `json:"message"`
	RetryAfter float64 `json:"retry_after"`
	Global     bool    `json:"global"`
}

type Rest struct {
	HTTPClient     http.Client
	maxRetries     uint8
	maxWaitTime    time.Duration
	token          string
	limiter        *RateLimiter
	retryCounter   atomic.Uint64
	retryThreshold uint64
}

type RestOptions struct {
	Token              string
	MaxRetries         uint8         // By default: 3
	MaxWaitTime        time.Duration // Max duration it can take for each request.
	RetryThreshold     uint64        // Max number of concurrent retries allowed before failing fast. Default: 100.
	RateLimiterOptions RateLimiterOptions
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
			Timeout:   5 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		ForceAttemptHTTP2: false,
		// MaxConnsPerHost:   256,
		MaxIdleConns:    256,
		IdleConnTimeout: 90 * time.Second,
	}

	maxTimeout := 10 * time.Second
	if opt.MaxWaitTime != 0 {
		maxTimeout = opt.MaxWaitTime
	}

	maxRetries := opt.MaxRetries
	if opt.MaxRetries == 0 {
		maxRetries = 3
	}

	retryThreshold := opt.RetryThreshold
	if retryThreshold == 0 {
		retryThreshold = 100
	}

	return &Rest{
		HTTPClient: http.Client{
			Transport: &rateLimitTransport{
				limiter:        NewRateLimiter(opt.RateLimiterOptions),
				innerTransport: transport,
			},
			Timeout: maxTimeout,
		},
		token:          prefixedToken,
		maxRetries:     maxRetries,
		maxWaitTime:    maxTimeout,
		retryThreshold: retryThreshold,
	}
}

func (rest *Rest) Request(method, route string, jsonPayload any) ([]byte, error) {
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

	return rest.request(method, route, body, CONTENT_TYPE_JSON)
}

// Internal handler for buffered requests. It's used by Request and RequestWithFiles (for PATCH only).
func (rest *Rest) request(method, route string, body io.ReadSeeker, contentType string) ([]byte, error) {
	var (
		responseBody []byte
		lastErr      error
	)

	for i := uint8(0); i < rest.maxRetries; i++ {
		if body != nil {
			if _, err := body.Seek(0, io.SeekStart); err != nil {
				return nil, fmt.Errorf("failed to seek request body: %w", err)
			}
		}

		// #nosec G704
		req, err := http.NewRequest(method, DiscordAPIBaseURL()+route, body)
		if err != nil {
			return nil, fmt.Errorf("failed to create request: %w", err)
		}
		req.Header.Set("Content-Type", contentType)

		responseBody, err = rest.executeOnce(req)
		if err != nil {
			lastErr = err

			if errors.Is(err, errGlobalRateLimit) || errors.Is(err, errRetryable) {
				currentRetries := rest.retryCounter.Add(1)
				defer rest.retryCounter.Add(^uint64(0)) // Decrement counter after loop/retry.

				if currentRetries > rest.retryThreshold {
					return nil, fmt.Errorf("%w: current retries (%d) exceed limit (%d)", errTooManyRetries, currentRetries, rest.retryThreshold)
				}

				if errors.Is(err, errGlobalRateLimit) {
					continue
				}

				time.Sleep(time.Millisecond * time.Duration(250*int64(i+1))) // Backoff on network/server errors
				continue
			}

			return nil, err // Not a retryable error.
		}
		return responseBody, nil
	}

	return nil, fmt.Errorf("request failed after %d retries on %s %s: %w", rest.maxRetries, method, route, lastErr)
}

func (rest *Rest) RequestWithFiles(method string, route string, jsonPayload any, files []File) ([]byte, error) {
	if len(files) == 0 {
		return rest.Request(method, route, jsonPayload)
	}

	// To avoid issues with retrying requests that have streaming bodies (which can't be re-read),
	// we pre-buffer the multipart request into memory. This ensures that the request body can be
	// reliably sent multiple times in case of retries.
	var requestBody bytes.Buffer
	writer := multipart.NewWriter(&requestBody)
	if err := writeMultipart(writer, jsonPayload, files); err != nil {
		return nil, err
	}

	return rest.request(method, route, bytes.NewReader(requestBody.Bytes()), writer.FormDataContentType())
}

// Writes the JSON payload and files to a multipart writer.
func writeMultipart(writer *multipart.Writer, jsonPayload any, files []File) error {
	defer writer.Close() //nolint:errcheck

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
		if seeker, ok := file.Reader.(io.ReadSeeker); ok {
			if _, err := seeker.Seek(0, io.SeekStart); err != nil {
				return fmt.Errorf("failed to seek file [%s]: %w", file.Name, err)
			}
		}

		filePart, err := writer.CreatePart(textproto.MIMEHeader{
			"Content-Disposition": []string{fmt.Sprintf(`form-data; name="files[%d]"; filename="%s"`, i, file.Name)},
			"Content-Type":        []string{CONTENT_TYPE_OCTET_STREAM},
		})
		if err != nil {
			return fmt.Errorf("failed to create file part [%d]: %w", i, err)
		}

		lr := io.LimitReader(file.Reader, MAX_FILE_UPLOAD_SIZE+1)
		n, err := io.Copy(filePart, lr)
		if err != nil {
			return fmt.Errorf("failed to stream file [%s]: %w", file.Name, err)
		}
		if n > MAX_FILE_UPLOAD_SIZE {
			return fmt.Errorf("file [%s] exceeds maximum upload size of 10MB", file.Name)
		}
	}

	if err := writer.Close(); err != nil {
		return fmt.Errorf("failed to close multipart writer: %w", err)
	}

	return nil
}

// executeOnce handles the lifecycle of a single request attempt.
func (rest *Rest) executeOnce(req *http.Request) ([]byte, error) {
	req.Header.Set("User-Agent", USER_AGENT)
	req.Header.Set("Authorization", rest.token)

	res, err := rest.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: http request execution failed: %s", errRetryable, err.Error())
	}
	defer res.Body.Close() //nolint:errcheck

	responseBody, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if res.StatusCode >= http.StatusOK && res.StatusCode < http.StatusMultipleChoices {
		return responseBody, nil
	}

	if res.StatusCode == http.StatusTooManyRequests {
		var rateErr rateLimitError
		err = json.Unmarshal(responseBody, &rateErr)
		if err != nil {
			return nil, fmt.Errorf("failed to parse rate limit details: %w", err)
		}

		if rateErr.Global {
			retryAfter := time.Duration(rateErr.RetryAfter * float64(time.Second))
			return nil, fmt.Errorf("%w: retrying after %s", errGlobalRateLimit, retryAfter.String())
		}

		return nil, fmt.Errorf("%w: hit per-route rate limit (429) on %s %s: %s", errRetryable, req.Method, req.URL.Path, string(responseBody))
	}

	if res.StatusCode >= http.StatusInternalServerError {
		return nil, fmt.Errorf("%w: discord API internal server error: %s", errRetryable, res.Status)
	}

	return nil, fmt.Errorf("request failed with status %s: %s", res.Status, string(responseBody))
}

func extractUserIDFromToken(token string) (Snowflake, error) {
	strs := strings.Split(token, ".")
	if len(strs) == 0 {
		return 0, errors.New("token is not in a valid format")
	}

	hexID := strings.Replace(strs[0], "Bot ", "", 1)

	byteID, err := base64.RawStdEncoding.DecodeString(hexID)
	if err != nil {
		return 0, err
	}

	return StringToSnowflake(string(byteID))
}
