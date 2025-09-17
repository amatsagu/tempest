package tempest

import (
	"bytes"
	"encoding/json"
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

	// This transport is aggressively tuned for low-latency communication with the Discord API.
	transport := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   5 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		ForceAttemptHTTP2: false, // Discord's API runs on HTTP/1.1, disabling HTTP/2 may improve performance.
		MaxConnsPerHost:   256,   // Allow a high number of concurrent connections to the API.
		MaxIdleConns:      256,   // Keep a large pool of idle connections ready for reuse.
		IdleConnTimeout:   90 * time.Second,
	}

	return &Rest{
		HTTPClient: http.Client{Transport: transport, Timeout: 10 * time.Second},
		token:      t,
		MaxRetries: 3,
	}
}

func (rest *Rest) Request(method, route string, jsonPayload any) ([]byte, error) {
	var body io.Reader

	if jsonPayload != nil {
		var buf bytes.Buffer
		encoder := json.NewEncoder(&buf)
		encoder.SetEscapeHTML(false)
		if err := encoder.Encode(jsonPayload); err != nil {
			return nil, fmt.Errorf("failed to encode JSON payload: %w", err)
		}
		body = &buf
	}

	return rest.execute(method, route, body, CONTENT_TYPE_JSON)
}

func (rest *Rest) RequestWithFiles(method string, route string, jsonPayload any, files []File) ([]byte, error) {
	if len(files) == 0 {
		return rest.Request(method, route, jsonPayload)
	}

	pr, pw := io.Pipe()
	writer := multipart.NewWriter(pw)

	go func() {
		defer pw.Close()
		defer writer.Close()

		jsonPart, err := writer.CreatePart(textproto.MIMEHeader{
			"Content-Disposition": []string{`form-data; name="payload_json"`},
			"Content-Type":        []string{CONTENT_TYPE_JSON},
		})
		if err != nil {
			pw.CloseWithError(fmt.Errorf("failed to create payload_json part: %w", err))
			return
		}
		encoder := json.NewEncoder(jsonPart)
		encoder.SetEscapeHTML(false)
		if err := encoder.Encode(jsonPayload); err != nil {
			pw.CloseWithError(fmt.Errorf("failed to encode payload_json: %w", err))
			return
		}

		for i, file := range files {
			filePart, err := writer.CreatePart(textproto.MIMEHeader{
				"Content-Disposition": []string{fmt.Sprintf(`form-data; name="files[%d]"; filename="%s"`, i, file.Name)},
				"Content-Type":        []string{CONTENT_TYPE_OCTET_STREAM},
			})
			if err != nil {
				pw.CloseWithError(fmt.Errorf("failed to create file part [%d]: %w", i, err))
				return
			}
			if _, err := io.Copy(filePart, file.Reader); err != nil {
				pw.CloseWithError(fmt.Errorf("failed to stream file [%s]: %w", file.Name, err))
				return
			}
		}
	}()

	return rest.execute(method, route, pr, writer.FormDataContentType())
}

// Handles the full lifecycle of a request, including global rate limiting and retries.
func (rest *Rest) execute(method, route string, body io.Reader, contentType string) ([]byte, error) {
	var lastErr error

	for i := uint8(0); i < rest.MaxRetries; i++ {
		rest.globalMu.RLock()
		sleepDuration := time.Until(rest.globalReset)
		rest.globalMu.RUnlock()
		if sleepDuration > 0 {
			time.Sleep(sleepDuration)
		}

		req, err := http.NewRequest(method, DISCORD_API_URL+route, body)
		if err != nil {
			return nil, fmt.Errorf("failed to create request: %w", err)
		}

		req.Header.Set("Content-Type", contentType)
		req.Header.Set("User-Agent", USER_AGENT)
		req.Header.Set("Authorization", rest.token)

		res, err := rest.HTTPClient.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("http request execution failed: %w", err)
			time.Sleep(time.Millisecond * time.Duration(250*int64(i+1))) // Backoff on network errors
			continue
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
				lastErr = fmt.Errorf("hit global rate limit, retrying after: %s", retryAfter.String())
				continue // Retry after the global cooldown.
			}

			// For any other per-route rate limit, fail fast and return the error.
			// The developer is responsible for handling this specific rate limit.
			return nil, fmt.Errorf("hit per-route rate limit (429) on %s %s: %s", method, route, string(responseBody))
		}

		if res.StatusCode >= http.StatusInternalServerError {
			lastErr = fmt.Errorf("discord API internal server error: %s", res.Status)
			time.Sleep(time.Millisecond * time.Duration(500*int64(i+1))) // Backoff on server errors
			continue
		}

		// For any other client-side error (4xx), fail immediately.
		return nil, fmt.Errorf("request failed with status %s: %s", res.Status, string(responseBody))
	}

	return nil, fmt.Errorf("request failed after %d retries on %s %s: %w", rest.MaxRetries, method, route, lastErr)
}
