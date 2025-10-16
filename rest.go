package tempest

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net"
	"net/http"
	"net/textproto"
	"strings"
	"sync"
	"time"
)

var (
	errRetryable       = errors.New("a retryable error occurred")
	errGlobalRateLimit = errors.New("hit global rate limit")
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
	HTTPClient  http.Client
	MaxRetries  uint8
	token       string
	globalMu    sync.RWMutex
	globalReset time.Time
}

func NewRest(token string) *Rest {
	t := token
	if !strings.HasPrefix(t, "Bot ") {
		t = "Bot " + t
	}

	transport := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   5 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		ForceAttemptHTTP2: false,
		MaxConnsPerHost:   256,
		MaxIdleConns:      256,
		IdleConnTimeout:   90 * time.Second,
	}

	return &Rest{
		HTTPClient: http.Client{Transport: transport, Timeout: 10 * time.Second},
		token:      t,
		MaxRetries: 5,
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

	for i := uint8(0); i < rest.MaxRetries; i++ {
		if body != nil {
			body.Seek(0, io.SeekStart)
		}

		req, err := http.NewRequest(method, DISCORD_API_URL+route, body)
		if err != nil {
			return nil, fmt.Errorf("failed to create request: %w", err)
		}
		req.Header.Set("Content-Type", contentType)

		responseBody, err = rest.executeOnce(req)
		if err != nil {
			lastErr = err
			if errors.Is(err, errGlobalRateLimit) {
				continue
			}
			if errors.Is(err, errRetryable) {
				time.Sleep(time.Millisecond * time.Duration(250*int64(i+1))) // Backoff on network/server errors
				continue
			}
			return nil, err // Not a retryable error.
		}
		return responseBody, nil
	}

	return nil, fmt.Errorf("request failed after %d retries on %s %s: %w", rest.MaxRetries, method, route, lastErr)
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
	defer writer.Close()

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
			seeker.Seek(0, io.SeekStart)
		}

		filePart, err := writer.CreatePart(textproto.MIMEHeader{
			"Content-Disposition": []string{fmt.Sprintf(`form-data; name="files[%d]"; filename="%s"`, i, file.Name)},
			"Content-Type":        []string{CONTENT_TYPE_OCTET_STREAM},
		})
		if err != nil {
			return fmt.Errorf("failed to create file part [%d]: %w", i, err)
		}
		if _, err := io.Copy(filePart, file.Reader); err != nil {
			return fmt.Errorf("failed to stream file [%s]: %w", file.Name, err)
		}
	}

	return nil
}

// executeOnce handles the lifecycle of a single request attempt.
func (rest *Rest) executeOnce(req *http.Request) ([]byte, error) {
	rest.globalMu.RLock()
	sleepDuration := time.Until(rest.globalReset)
	rest.globalMu.RUnlock()
	if sleepDuration > 0 {
		time.Sleep(sleepDuration)
	}

	req.Header.Set("User-Agent", USER_AGENT)
	req.Header.Set("Authorization", rest.token)

	res, err := rest.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: http request execution failed: %s", errRetryable, err.Error())
	}
	defer res.Body.Close()

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
			rest.globalMu.Lock()
			rest.globalReset = time.Now().Add(retryAfter)
			rest.globalMu.Unlock()
			return nil, fmt.Errorf("%w: retrying after %s", errGlobalRateLimit, retryAfter.String())
		}

		return nil, fmt.Errorf("hit per-route rate limit (429) on %s %s: %s", req.Method, req.URL.Path, string(responseBody))
	}

	if res.StatusCode >= http.StatusInternalServerError {
		return nil, fmt.Errorf("%w: discord API internal server error: %s", errRetryable, res.Status)
	}

	return nil, fmt.Errorf("request failed with status %s: %s", res.Status, string(responseBody))
}
